package main

import (
	"context"
	"fmt"
	"github.com/gliderlabs/ssh"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"log"
	"net/url"
	"time"
)

type RunnerInstance struct {
	Pod *corev1.Pod
}

type AppContextOptions struct {
	KubeConfig           string
	UseKubeConfig        bool
	Namespace            string
	MaxConcurrentRunners int
}

type AppContext struct {
	clientSet            *kubernetes.Clientset
	kubeConfig           *rest.Config
	namespace            string
	maxConcurrentRunners int
}

func NewAppContext(config AppContextOptions) (*AppContext, error) {
	var kubeConfig *rest.Config

	if config.UseKubeConfig {
		configData, err := ioutil.ReadFile(config.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to read kubeconfig file: %s", err)
		}

		result, err := clientcmd.RESTConfigFromKubeConfig(configData)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %s", err)
		}

		kubeConfig = result
	} else {
		result, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to make in-cluster config: %s", err)
		}

		kubeConfig = result
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %s", err)
	}

	return &AppContext{
		clientSet:            clientset,
		kubeConfig:           kubeConfig,
		namespace:            config.Namespace,
		maxConcurrentRunners: config.MaxConcurrentRunners,
	}, nil
}

func (c *AppContext) Init() error {
	if err := c.CreateAppNamespace(); err != nil {
		return err
	}

	return nil
}

func (c *AppContext) CreateAppNamespace() error {
	namespaceConfig := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.namespace,
		},
	}

	_, err := c.clientSet.CoreV1().Namespaces().Get(
		context.Background(), c.namespace, metav1.GetOptions{})

	if err == nil {
		return nil
	}

	if _, err := c.clientSet.CoreV1().Namespaces().Create(
		context.Background(), &namespaceConfig, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create namespace: %s", err)
	}

	return nil
}

func (c *AppContext) ProvisionRunner(repoURL, repoBranch, origin string) (*RunnerInstance, error) {
	label := fmt.Sprintf("cli-runner-%s", rand.String(10))
	autoMountToken := false

	podConfig := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: label,
			Labels: map[string]string{
				"linode-cli-autodeploy.type":   "runner",
				"linode-cli-autodeploy.origin": origin,
			},
		},
		Spec: corev1.PodSpec{
			AutomountServiceAccountToken: &autoMountToken,
			RestartPolicy:                corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  "runner",
					Image: "lbgarber/cli-autodeploy-runner:latest",
					Env: []corev1.EnvVar{
						{
							Name:  "GIT_REPO_URL",
							Value: repoURL,
						},
						{
							Name:  "GIT_REPO_BRANCH",
							Value: repoBranch,
						},
					},
					Stdin: true,
					TTY:   true,
				},
			},
		},
	}

	result, err := c.clientSet.CoreV1().Pods(c.namespace).Create(
		context.Background(), &podConfig, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create backing pod: %s", err)
	}

	result, err = c.waitForPodRunning(result)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for pod running: %s", err)
	}

	return &RunnerInstance{
		Pod: result,
	}, nil
}

func (c *AppContext) AttachRunner(runner *RunnerInstance, pipe ssh.Session) error {
	var sizeQueue remotecommand.TerminalSizeQueue

	u, err := url.Parse(
		fmt.Sprintf(
			"%s/api/v1/namespaces/%s/pods/%s/attach?container=%s&stderr=true&stdin=true&stdout=true&tty=true",
			c.kubeConfig.Host,
			c.namespace,
			runner.Pod.Name,
			runner.Pod.Spec.Containers[0].Name,
		),
	)

	exec, err := remotecommand.NewSPDYExecutor(c.kubeConfig, "POST", u)
	if err != nil {
		return fmt.Errorf("failed to create spdy executor: %s", err)
	}

	err = exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:             pipe,
		Stdout:            pipe,
		Stderr:            pipe.Stderr(),
		Tty:               false,
		TerminalSizeQueue: sizeQueue,
	})

	return err
}

func (c *AppContext) DestroyRunner(runner *RunnerInstance) error {
	if err := c.clientSet.CoreV1().Pods(c.namespace).Delete(
		context.Background(), runner.Pod.Name, *metav1.NewDeleteOptions(0)); err != nil {
		return fmt.Errorf("failed to delete pod %s: %s", runner.Pod.Name, err)
	}

	return nil
}

func (c *AppContext) waitForPodRunning(pod *corev1.Pod) (*corev1.Pod, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			result, err := c.clientSet.CoreV1().Pods(c.namespace).Get(
				context.Background(), pod.Name, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to get backing pod: %s", err)
			}

			if result.Status.Phase == corev1.PodRunning {
				return result, nil
			}

		case <-ctx.Done():
			return nil, fmt.Errorf("failed to wait for pod running %s: %s", pod.Name, ctx.Err())
		}
	}
}

func (c *AppContext) RunnerCleanupCron() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.attemptRunnerCleanup(); err != nil {
				log.Printf("[WARN] got error while running cleanup cron: %s", err)
			}
		}
	}
}

func (c *AppContext) attemptRunnerCleanup() error {
	pods, err := c.clientSet.CoreV1().Pods(c.namespace).List(
		context.Background(), metav1.ListOptions{
			LabelSelector: "linode-cli-autodeploy.type=runner",
		})
	if err != nil {
		return fmt.Errorf("failed to list pods: %s", err)
	}

	for _, pod := range pods.Items {
		if !shouldRemovePod(&pod) {
			continue
		}

		log.Printf("[INFO] Deleting stale pod: %s", pod.Name)

		if err := c.clientSet.CoreV1().Pods(c.namespace).Delete(
			context.Background(), pod.Name, *metav1.NewDeleteOptions(0)); err != nil {
			return fmt.Errorf("failed to delete pod %s: %s", pod.Name, err)
		}
	}

	return nil
}

func (c *AppContext) CanOriginProvisionRunner(origin string) (bool, error) {
	pods, err := c.clientSet.CoreV1().Pods(c.namespace).List(
		context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("linode-cli-autodeploy.origin=%s", origin),
		})
	if err != nil {
		return false, fmt.Errorf("unable to list pods with origin %s: %s", origin, err)
	}

	return len(pods.Items) < c.maxConcurrentRunners, nil
}

func shouldRemovePod(pod *corev1.Pod) bool {
	return time.Since(pod.CreationTimestamp.Time).Minutes() >= 15 ||
		pod.Status.Phase == corev1.PodFailed ||
		pod.Status.Phase == corev1.PodSucceeded
}

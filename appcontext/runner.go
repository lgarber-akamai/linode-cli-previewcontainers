package appcontext

import (
	"context"
	"fmt"
	"github.com/gliderlabs/ssh"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/remotecommand"
	"net/url"
)

const LabelRunnerID = "linode-cli-autodeploy.runner-id"
const LabelRunnerType = "linode-cli-autodeploy.type"
const LabelRunnerOrigin = "linode-cli-autodeploy.origin"

type RunnerInstance struct {
	ID string
	Pod *corev1.Pod
	Secret *corev1.Secret
}



type ProvisionRunnerOptions struct {
	RepoCloneURL string
	RepoBranch string
	Origin string
	Token string
}

func (c *AppContext) ProvisionRunner(opts ProvisionRunnerOptions) (*RunnerInstance, error) {
	runnerID := rand.String(10)

	runnerSecrets, err := c.createSecretsForRunner(runnerID, opts.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to provision runner secrets: %s", err)
	}

	autoMountToken := false

	podConfig := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("cli-runner-%s", runnerID),
			Labels: map[string]string{
				LabelRunnerID:     runnerID, // Useful for reverse references
				LabelRunnerType:   "runner",
				LabelRunnerOrigin: opts.Origin,
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
							Value: opts.RepoCloneURL,
						},
						{
							Name:  "GIT_REPO_BRANCH",
							Value: opts.RepoBranch,
						},
						{
							Name: "LINODE_CLI_TOKEN",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: runnerSecrets.Name,
									},
									Key: "linode-api-token",
								},
							},
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
		ID: runnerID,
		Pod: result,
		Secret: runnerSecrets,
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
	if runner.Pod != nil {
		if err := c.clientSet.CoreV1().Pods(c.namespace).Delete(
			context.Background(), runner.Pod.Name, *metav1.NewDeleteOptions(0)); err != nil {
			return fmt.Errorf("failed to delete pod %s: %s", runner.Pod.Name, err)
		}
	}

	if runner.Secret != nil {
		if err := c.clientSet.CoreV1().Secrets(c.namespace).Delete(
			context.Background(), runner.Secret.Name, *metav1.NewDeleteOptions(0)); err != nil {
			return fmt.Errorf("failed to delete secret %s: %s", runner.Secret.Name, err)
		}
	}

	return nil
}

func (c *AppContext) createSecretsForRunner(runnerID, token string) (*corev1.Secret, error) {
	immutable := true

	runnerSecretsOpts := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("runner-secrets-%s", runnerID),
			Labels: map[string]string{
				LabelRunnerID:   runnerID,
				LabelRunnerType: "runner-creds",
			},
		},
		Immutable:  &immutable,
		StringData: map[string]string{
			"linode-api-token": token,
		},
	}

	secret, err := c.clientSet.CoreV1().Secrets(c.namespace).Create(
		context.Background(), &runnerSecretsOpts, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create runner secret: %s", err)
	}

	return secret, nil
}

func (c *AppContext) getRunnerByID(runnerID string) (*RunnerInstance, error) {
	var secret *corev1.Secret
	var pod *corev1.Pod

	listOpts := metav1.ListOptions{
		LabelSelector:        fmt.Sprintf("%s=%s", LabelRunnerID, runnerID),
	}

	pods, err := c.clientSet.CoreV1().Pods(c.namespace).List(
		context.Background(), listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list runner pods: %s", err)
	}

	if len(pods.Items) > 0 {
		pod = &pods.Items[0]
	}

	secrets, err := c.clientSet.CoreV1().Secrets(c.namespace).List(
		context.Background(), listOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list runner secrets: %s", err)
	}

	if len(secrets.Items) > 0 {
		secret = &secrets.Items[0]
	}

	return &RunnerInstance{
		ID: runnerID,
		Pod:    pod,
		Secret: secret,
	}, nil
}

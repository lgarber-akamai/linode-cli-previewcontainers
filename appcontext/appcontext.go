package appcontext

import (
	"context"
	"fmt"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type AppContextOptions struct {
	KubeConfig           string
	Namespace            string
	RunnerImage          string
	RunnerMemoryLimit    string
	RunnerCPULimit       string
	RunnerExpiryTime     int
	MaxConcurrentRunners int
	UseKubeConfig        bool
}

type AppContext struct {
	clientSet         *kubernetes.Clientset
	kubeConfig        *rest.Config
	namespace         string
	runnerImage       string
	runnerMemoryLimit string
	runnerCPULimit    string
	runnerExpiryTime  int

	MaxConcurrentRunners int
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
		MaxConcurrentRunners: config.MaxConcurrentRunners,
		runnerImage:          config.RunnerImage,
		runnerCPULimit:       config.RunnerCPULimit,
		runnerMemoryLimit:    config.RunnerMemoryLimit,
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

package appcontext

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

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

func (c *AppContext) CanOriginProvisionRunner(origin string) (bool, error) {
	pods, err := c.clientSet.CoreV1().Pods(c.namespace).List(
		context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("linode-cli-autodeploy.origin=%s", origin),
		})
	if err != nil {
		return false, fmt.Errorf("unable to list pods with origin %s: %s", origin, err)
	}

	return len(pods.Items) < c.MaxConcurrentRunners, nil
}

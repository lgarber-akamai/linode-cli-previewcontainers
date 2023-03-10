package appcontext

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"time"
)

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
	if err := c.cleanupStalePods(); err != nil {
		return err
	}

	if err := c.cleanupOrphans(); err != nil {
		return err
	}

	return nil
}

func (c *AppContext) cleanupStalePods() error {
	pods, err := c.clientSet.CoreV1().Pods(c.namespace).List(
		context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=runner", LabelRunnerType),
		})
	if err != nil {
		return fmt.Errorf("failed to list pods: %s", err)
	}

	for _, pod := range pods.Items {
		if !c.shouldRemovePod(&pod) {
			continue
		}

		runnerID, ok := pod.Labels[LabelRunnerID]
		if !ok {
			continue
		}

		runner, err := c.getRunnerByID(runnerID)
		if err != nil {
			log.Printf("[ERROR] failed to get runner from id %s: %s\n", runnerID, err)
			continue
		}

		log.Printf("[INFO] Deleting stale runner: %s\n", pod.Name)

		if err := c.DestroyRunner(runner); err != nil {
			log.Printf("[ERROR] failed to destroy runner %s: %s\n", runnerID, err)
			continue
		}
	}

	return nil
}

func (c *AppContext) cleanupOrphans() error {
	runners, err := c.aggregateRunners()
	if err != nil {
		return fmt.Errorf("failed to aggregate runners: %s", err)
	}

	for _, runner := range runners {
		shouldRemove, err := c.isOrphan(runner)
		if err != nil {
			log.Printf("[ERROR] failed to check if runner is orphan: %s\n", err)
			continue
		}

		if !shouldRemove {
			continue
		}

		log.Printf("[INFO] Deleting orphan runner: %s\n", runner.ID)

		if err := c.DestroyRunner(runner); err != nil {
			log.Printf("[ERROR] failed to destroy orphan runner %s: %s\n", runner.ID, err)
			continue
		}
	}

	return nil
}

// This is terrible; I should really aggregate runner IDs instead of dynamically resolving them from labels
func (c *AppContext) aggregateRunners() ([]*RunnerInstance, error) {
	aggregatedIDs := make(map[string]bool)

	// Aggregate runner labels across the cluster
	pods, err := c.clientSet.CoreV1().Pods(c.namespace).List(
		context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=runner", LabelRunnerType),
		})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %s", err)
	}

	secrets, err := c.clientSet.CoreV1().Secrets(c.namespace).List(
		context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=runner-creds", LabelRunnerType),
		})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %s", err)
	}

	for _, pod := range pods.Items {
		runnerID, ok := pod.Labels[LabelRunnerID]
		if !ok {
			continue
		}

		aggregatedIDs[runnerID] = true
	}

	for _, secret := range secrets.Items {
		runnerID, ok := secret.Labels[LabelRunnerID]
		if !ok {
			continue
		}

		aggregatedIDs[runnerID] = true
	}

	result := make([]*RunnerInstance, 0)
	for id := range aggregatedIDs {
		runner, err := c.getRunnerByID(id)
		if err != nil {
			return nil, fmt.Errorf("failed to get runner by id %s: %s", id, err)
		}
		result = append(result, runner)
	}

	return result, nil
}

func (c *AppContext) shouldRemovePod(pod *corev1.Pod) bool {
	return time.Since(pod.CreationTimestamp.Time).Minutes() >= float64(c.runnerExpiryTime) ||
		pod.Status.Phase == corev1.PodFailed ||
		pod.Status.Phase == corev1.PodSucceeded
}

func (c *AppContext) isOrphan(runner *RunnerInstance) (bool, error) {
	return runner.Pod == nil || runner.Secret == nil, nil
}

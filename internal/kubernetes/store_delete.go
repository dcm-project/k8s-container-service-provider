package kubernetes

import (
	"context"
	"fmt"

	"github.com/dcm-project/k8s-container-service-provider/internal/store"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Delete removes a container and its associated Kubernetes resources.
func (s *K8sContainerStore) Delete(ctx context.Context, containerID string) error {
	// 1. Find Deployment by instance ID.
	deploys, err := s.client.AppsV1().Deployments(s.cfg.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: instanceSelector(containerID),
	})
	if err != nil {
		return err
	}
	if len(deploys.Items) == 0 {
		return &store.NotFoundError{ID: containerID}
	}
	if len(deploys.Items) > 1 {
		return &store.ConflictError{Message: fmt.Sprintf("multiple deployments found for container %q", containerID)}
	}
	deploy := &deploys.Items[0]

	// 2. Delete Deployment.
	err = s.client.AppsV1().Deployments(s.cfg.Namespace).Delete(ctx, deploy.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	// 3. Delete Service (ignore not-found).
	err = s.client.CoreV1().Services(s.cfg.Namespace).Delete(ctx, deploy.Name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

package kubernetes

import (
	"context"
	"fmt"

	v1alpha1 "github.com/dcm-project/k8s-container-service-provider/api/v1alpha1"
	"github.com/dcm-project/k8s-container-service-provider/internal/store"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// K8sContainerStore implements store.ContainerRepository backed by Kubernetes
// Deployments, Pods, and Services.
type K8sContainerStore struct {
	client kubernetes.Interface
	cfg    K8sConfig
}

// NewK8sContainerStore creates a new K8sContainerStore with the given client and config.
func NewK8sContainerStore(client kubernetes.Interface, cfg K8sConfig) *K8sContainerStore {
	return &K8sContainerStore{
		client: client,
		cfg:    cfg,
	}
}

// buildContainer reconstructs an API Container from a Deployment and enriches
// it with runtime data from the cluster.
func (s *K8sContainerStore) buildContainer(ctx context.Context, deploy *appsv1.Deployment, instanceID string) (*v1alpha1.Container, error) {
	c := containerFromDeployment(deploy, instanceID)
	if err := s.enrichFromCluster(ctx, &c, deploy, instanceID); err != nil {
		return nil, err
	}
	return &c, nil
}

// enrichFromCluster enriches a Container with runtime data from Pods and Services.
func (s *K8sContainerStore) enrichFromCluster(
	ctx context.Context,
	c *v1alpha1.Container,
	deploy *appsv1.Deployment,
	instanceID string,
) error {
	pods, err := s.client.CoreV1().Pods(s.cfg.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: instanceSelector(instanceID),
	})
	if err != nil {
		return err
	}

	if len(pods.Items) == 1 {
		enrichWithPod(c, &pods.Items[0])
	} else if len(pods.Items) > 1 {
		return &store.ConflictError{
			InstanceRef: fmt.Sprintf("multiple pods found for container %s", instanceID),
		}
	} else {
		pending := v1alpha1.PENDING
		c.Status = &pending
		if t := latestDeploymentTransitionTime(deploy); t != nil {
			c.UpdateTime = t
		}
	}

	svc, err := s.client.CoreV1().Services(s.cfg.Namespace).Get(ctx, deploy.Name, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	} else {
		enrichWithService(c, svc)
	}

	return nil
}

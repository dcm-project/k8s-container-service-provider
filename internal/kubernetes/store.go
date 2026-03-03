package kubernetes

import (
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

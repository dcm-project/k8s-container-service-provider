package kubernetes

import (
	"context"
	"fmt"
	"time"

	v1alpha1 "github.com/dcm-project/k8s-container-service-provider/api/v1alpha1"
	"github.com/dcm-project/k8s-container-service-provider/internal/store"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// shouldCreateService determines whether a Kubernetes Service should be created.
// ProviderHints override the config default when explicitly set.
func shouldCreateService(cfg K8sConfig, hints *v1alpha1.ProviderHints) bool {
	if hints != nil && hints.Kubernetes != nil && hints.Kubernetes.Service != nil && hints.Kubernetes.Service.Enabled != nil {
		return *hints.Kubernetes.Service.Enabled
	}
	return cfg.CreateService
}

// resolveServiceType determines the Kubernetes Service type to use.
// ProviderHints override the config default when explicitly set.
func resolveServiceType(cfg K8sConfig, hints *v1alpha1.ProviderHints) corev1.ServiceType {
	if hints != nil && hints.Kubernetes != nil && hints.Kubernetes.Service != nil && hints.Kubernetes.Service.Type != nil {
		return corev1.ServiceType(*hints.Kubernetes.Service.Type)
	}
	return corev1.ServiceType(cfg.DefaultServiceType)
}

// hasPorts returns true if the container has at least one network port defined.
func hasPorts(container v1alpha1.Container) bool {
	return container.Network != nil && len(container.Network.Ports) > 0
}

// Create creates a new container backed by a Kubernetes Deployment (and
// optionally a Service).
func (s *K8sContainerStore) Create(ctx context.Context, container v1alpha1.Container, id string) (*v1alpha1.Container, error) {
	// Validate user labels don't collide with DCM reserved labels and build merged labels.
	if err := validateUserLabels(container.Metadata.Labels); err != nil {
		return nil, err
	}
	labels := dcmLabels(id)
	if container.Metadata.Labels != nil {
		labels = mergeLabels(labels, *container.Metadata.Labels)
	}

	// Check for duplicate dcm-instance-id.
	existing, err := s.client.AppsV1().Deployments(s.cfg.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: instanceSelector(id),
	})
	if err != nil {
		return nil, err
	}
	if len(existing.Items) > 0 {
		return nil, &store.ConflictError{Message: fmt.Sprintf("container with instance ID %q already exists", id)}
	}

	// Determine if a Service should be created.
	createSvc := shouldCreateService(s.cfg, container.ProviderHints)

	// If Service needed but no ports, fail BEFORE creating Deployment (atomicity).
	if createSvc && !hasPorts(container) {
		return nil, &store.InvalidArgumentError{
			Message: "service creation requires at least one port to be defined",
		}
	}

	// Validate memory format before building Deployment.
	if err := validateMemory(container.Resources.Memory.Min, container.Resources.Memory.Max); err != nil {
		return nil, err
	}

	// Create Deployment.
	deploy := buildDeployment(container, id, s.cfg, labels)
	_, err = s.client.AppsV1().Deployments(s.cfg.Namespace).Create(ctx, deploy, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, &store.ConflictError{Message: fmt.Sprintf("deployment %q already exists", container.Metadata.Name)}
		}
		return nil, err
	}

	// Create Service if needed.
	if createSvc {
		svcType := resolveServiceType(s.cfg, container.ProviderHints)
		svc := buildService(container, id, s.cfg, labels, svcType)
		_, err = s.client.CoreV1().Services(s.cfg.Namespace).Create(ctx, svc, metav1.CreateOptions{})
		if err != nil {
			// Rollback: delete the just-created Deployment.
			_ = s.client.AppsV1().Deployments(s.cfg.Namespace).Delete(ctx, deploy.Name, metav1.DeleteOptions{})
			return nil, err
		}
	}

	return newContainerResult(container, id, s.cfg.Namespace), nil
}

// validateMemory checks that both memory strings can be parsed by ConvertMemory.
func validateMemory(min, max string) error {
	if _, err := ConvertMemory(min); err != nil {
		return &store.InvalidArgumentError{
			Message: fmt.Sprintf("invalid memory.min: %s", err),
		}
	}
	if _, err := ConvertMemory(max); err != nil {
		return &store.InvalidArgumentError{
			Message: fmt.Sprintf("invalid memory.max: %s", err),
		}
	}
	return nil
}

// newContainerResult stamps server-assigned fields onto a user-provided container.
func newContainerResult(container v1alpha1.Container, id, namespace string) *v1alpha1.Container {
	now := time.Now()
	status := v1alpha1.PENDING
	path := fmt.Sprintf("containers/%s", id)

	result := container
	result.Id = &id
	result.Path = &path
	result.Status = &status
	result.CreateTime = &now
	result.UpdateTime = &now
	result.Metadata.Namespace = &namespace

	return &result
}

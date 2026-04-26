package kubernetes

import (
	"context"
	"fmt"
	"time"

	v1alpha1 "github.com/dcm-project/k8s-container-service-provider/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Update modifies an existing container's Deployment (and Service when port
// visibility changes) to match the provided spec.
func (s *K8sContainerStore) Update(ctx context.Context, containerID string, spec v1alpha1.ContainerSpec) (*v1alpha1.Container, error) {
	deploy, err := s.findDeployment(ctx, containerID)
	if err != nil {
		return nil, err
	}

	labels := dcmLabels(containerID)
	if spec.Metadata.Labels != nil {
		labels = mergeLabels(labels, *spec.Metadata.Labels)
	}

	updated := buildDeployment(spec, containerID, s.cfg, labels)
	updated.Name = deploy.Name
	updated.ResourceVersion = deploy.ResourceVersion

	_, err = s.client.AppsV1().Deployments(s.cfg.Namespace).Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	if err := s.reconcileService(ctx, spec, containerID, deploy.Name, labels); err != nil {
		return nil, err
	}

	return s.buildContainer(ctx, deploy, containerID)
}

// reconcileService ensures the Service state matches the desired port visibility:
// create, update, or delete the Service as needed.
func (s *K8sContainerStore) reconcileService(
	ctx context.Context,
	spec v1alpha1.ContainerSpec,
	containerID, deployName string,
	labels map[string]string,
) error {
	servicePorts := portsWithVisibility(spec)
	existingSvc, err := s.client.CoreV1().Services(s.cfg.Namespace).Get(ctx, deployName, metav1.GetOptions{})
	svcExists := err == nil

	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	switch {
	case len(servicePorts) > 0 && !svcExists:
		svcType := resolveServiceType(s.cfg, servicePorts)
		svc := buildService(spec, containerID, s.cfg, labels, svcType, servicePorts)
		_, err = s.client.CoreV1().Services(s.cfg.Namespace).Create(ctx, svc, metav1.CreateOptions{})
		return err

	case len(servicePorts) > 0 && svcExists:
		svcType := resolveServiceType(s.cfg, servicePorts)
		svc := buildService(spec, containerID, s.cfg, labels, svcType, servicePorts)
		svc.ResourceVersion = existingSvc.ResourceVersion
		svc.Spec.ClusterIP = existingSvc.Spec.ClusterIP
		_, err = s.client.CoreV1().Services(s.cfg.Namespace).Update(ctx, svc, metav1.UpdateOptions{})
		return err

	case len(servicePorts) == 0 && svcExists:
		propagation := metav1.DeletePropagationBackground
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		if err := s.client.CoreV1().Services(s.cfg.Namespace).Delete(cleanupCtx, deployName, metav1.DeleteOptions{
			PropagationPolicy: &propagation,
		}); err != nil && !apierrors.IsNotFound(err) {
			s.logger.Error("failed to delete Service during update",
				"service", deployName,
				"namespace", s.cfg.Namespace,
				"error", err,
			)
			return fmt.Errorf("deleting Service %q: %w", deployName, err)
		}
		return nil

	default:
		return nil
	}
}

package kubernetes

import (
	v1alpha1 "github.com/dcm-project/k8s-container-service-provider/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

// buildService creates a Kubernetes Service from a Container spec.
func buildService(container v1alpha1.Container, id string, cfg K8sConfig, labels map[string]string, svcType corev1.ServiceType) *corev1.Service {
	// Selector uses only DCM labels
	selectorLabels := dcmLabels(id)

	svcPorts := make([]corev1.ServicePort, len(container.Network.Ports))
	for i, p := range container.Network.Ports {
		svcPorts[i] = corev1.ServicePort{
			Port:       int32(p.ContainerPort),
			TargetPort: intstr.FromInt32(int32(p.ContainerPort)),
			Protocol:   corev1.ProtocolTCP,
		}
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      container.Metadata.Name,
			Namespace: cfg.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     svcType,
			Selector: selectorLabels,
			Ports:    svcPorts,
		},
	}
}

package kubernetes

import (
	v1alpha1 "github.com/dcm-project/k8s-container-service-provider/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// buildDeployment creates a Kubernetes Deployment from a Container spec.
func buildDeployment(container v1alpha1.Container, id string, cfg K8sConfig, labels map[string]string) *appsv1.Deployment {
	replicas := int32(1)

	// Selector uses only DCM labels (immutable after creation)
	selectorLabels := dcmLabels(id)

	// CPU resources
	cpuReq, cpuLim := ConvertCPU(container.Resources.Cpu)

	// Memory resources — errors handled upstream; safe to ignore here since
	// validation occurs before buildDeployment is called.
	memReq, _ := ConvertMemory(container.Resources.Memory.Min)
	memLim, _ := ConvertMemory(container.Resources.Memory.Max)

	k8sContainer := corev1.Container{
		Name:  container.Metadata.Name,
		Image: container.Image.Reference,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    cpuReq,
				corev1.ResourceMemory: memReq,
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    cpuLim,
				corev1.ResourceMemory: memLim,
			},
		},
	}

	if container.Process != nil {
		if container.Process.Command != nil {
			k8sContainer.Command = *container.Process.Command
		}
		if container.Process.Args != nil {
			k8sContainer.Args = *container.Process.Args
		}
		if container.Process.Env != nil {
			envVars := make([]corev1.EnvVar, len(*container.Process.Env))
			for i, e := range *container.Process.Env {
				envVars[i] = corev1.EnvVar{Name: e.Name, Value: e.Value}
			}
			k8sContainer.Env = envVars
		}
	}

	if container.Network != nil && len(container.Network.Ports) > 0 {
		ports := make([]corev1.ContainerPort, len(container.Network.Ports))
		for i, p := range container.Network.Ports {
			ports[i] = corev1.ContainerPort{
				ContainerPort: int32(p.ContainerPort),
			}
		}
		k8sContainer.Ports = ports
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      container.Metadata.Name,
			Namespace: cfg.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{k8sContainer},
				},
			},
		},
	}
}

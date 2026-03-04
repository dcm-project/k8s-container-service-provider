package kubernetes

import (
	"fmt"
	"strings"
	"time"

	v1alpha1 "github.com/dcm-project/k8s-container-service-provider/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ConvertCPU converts a ContainerCpu spec to Kubernetes resource quantities
// for requests and limits.
func ConvertCPU(cpu v1alpha1.ContainerCpu) (requests, limits resource.Quantity) {
	requests = *resource.NewQuantity(int64(cpu.Min), resource.DecimalSI)
	limits = *resource.NewQuantity(int64(cpu.Max), resource.DecimalSI)
	return requests, limits
}

// unitMapping maps API memory units to Kubernetes binary units.
var unitMapping = map[string]string{
	"MB": "Mi",
	"GB": "Gi",
	"TB": "Ti",
}

// ConvertMemory converts a memory string (e.g., "1GB") to a Kubernetes
// resource quantity.
func ConvertMemory(memoryStr string) (resource.Quantity, error) {
	for suffix, k8sUnit := range unitMapping {
		if strings.HasSuffix(memoryStr, suffix) {
			numStr := strings.TrimSuffix(memoryStr, suffix)
			return resource.ParseQuantity(numStr + k8sUnit)
		}
	}
	return resource.Quantity{}, fmt.Errorf("unsupported memory unit in %q", memoryStr)
}

// MapPodPhaseToStatus maps a Kubernetes Pod phase to a container status.
// Returns the mapped status and true if mapping exists, or ("", false)
// for phases that should be ignored (e.g., Succeeded per DD-020).
func MapPodPhaseToStatus(phase corev1.PodPhase) (v1alpha1.ContainerStatus, bool) {
	switch phase {
	case corev1.PodPending:
		return v1alpha1.PENDING, true
	case corev1.PodRunning:
		return v1alpha1.RUNNING, true
	case corev1.PodFailed:
		return v1alpha1.FAILED, true
	case corev1.PodUnknown:
		return v1alpha1.UNKNOWN, true
	default:
		// Succeeded and any unknown phases are not mapped
		return "", false
	}
}

// containerFromDeployment reconstructs an API Container from a Kubernetes Deployment.
func containerFromDeployment(deploy *appsv1.Deployment, instanceID string) v1alpha1.Container {
	id := instanceID
	path := fmt.Sprintf("containers/%s", instanceID)
	ns := deploy.Namespace
	createTime := deploy.CreationTimestamp.Time
	serviceType := v1alpha1.ContainerServiceTypeContainer

	c := v1alpha1.Container{
		Id:          &id,
		Path:        &path,
		ServiceType: serviceType,
		CreateTime:  &createTime,
		Metadata: v1alpha1.ContainerMetadata{
			Name:      deploy.Name,
			Namespace: &ns,
		},
	}

	k8sC := deploy.Spec.Template.Spec.Containers[0]
	c.Image = v1alpha1.ContainerImage{Reference: k8sC.Image}

	return c
}

// enrichWithPod populates runtime data from a Pod into the Container.
func enrichWithPod(container *v1alpha1.Container, pod *corev1.Pod) {
	if status, ok := MapPodPhaseToStatus(pod.Status.Phase); ok {
		container.Status = &status
	}

	if pod.Status.PodIP != "" {
		if container.Network == nil {
			container.Network = &v1alpha1.ContainerNetwork{}
		}
		container.Network.Ip = &pod.Status.PodIP
	}

	if t := latestPodTransitionTime(pod); t != nil {
		container.UpdateTime = t
	}
}

// enrichWithService populates service info from a Kubernetes Service.
func enrichWithService(container *v1alpha1.Container, svc *corev1.Service) {
	info := &v1alpha1.ServiceInfo{}

	if svc.Spec.ClusterIP != "" {
		info.ClusterIp = &svc.Spec.ClusterIP
	}

	svcType := v1alpha1.ServiceInfoType(svc.Spec.Type)
	info.Type = &svcType

	if len(svc.Spec.Ports) > 0 {
		ports := make([]v1alpha1.ServicePort, len(svc.Spec.Ports))
		for i, p := range svc.Spec.Ports {
			protocol := string(p.Protocol)
			ports[i] = v1alpha1.ServicePort{
				Port:       int(p.Port),
				TargetPort: p.TargetPort.IntValue(),
				Protocol:   &protocol,
			}
		}
		info.Ports = &ports
	}

	if len(svc.Status.LoadBalancer.Ingress) > 0 && svc.Status.LoadBalancer.Ingress[0].IP != "" {
		info.ExternalIp = &svc.Status.LoadBalancer.Ingress[0].IP
	}

	container.Service = info
}

// latestPodTransitionTime returns the most recent LastTransitionTime from Pod conditions.
func latestPodTransitionTime(pod *corev1.Pod) *time.Time {
	var latest *time.Time
	for i := range pod.Status.Conditions {
		t := pod.Status.Conditions[i].LastTransitionTime.Time
		if t.IsZero() {
			continue
		}
		if latest == nil || t.After(*latest) {
			latest = &t
		}
	}
	return latest
}

// latestDeploymentTransitionTime returns the most recent LastTransitionTime from Deployment conditions.
func latestDeploymentTransitionTime(deploy *appsv1.Deployment) *time.Time {
	var latest *time.Time
	for i := range deploy.Status.Conditions {
		t := deploy.Status.Conditions[i].LastTransitionTime.Time
		if t.IsZero() {
			continue
		}
		if latest == nil || t.After(*latest) {
			latest = &t
		}
	}
	return latest
}

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

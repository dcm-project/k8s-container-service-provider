package kubernetes

import (
	"fmt"
	"strings"

	v1alpha1 "github.com/dcm-project/k8s-container-service-provider/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

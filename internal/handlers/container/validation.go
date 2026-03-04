package container

import (
	"fmt"
	"regexp"

	v1alpha1 "github.com/dcm-project/k8s-container-service-provider/api/v1alpha1"
	"github.com/dcm-project/k8s-container-service-provider/internal/dcm"
	"github.com/dcm-project/k8s-container-service-provider/internal/units"
)

// containerIDPattern enforces AEP-122 resource ID format:
// must start with a lowercase letter, contain only lowercase letters, digits,
// and hyphens, and be 1–63 characters long.
var containerIDPattern = regexp.MustCompile(`^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$`)

func validateContainerID(id string) error {
	if !containerIDPattern.MatchString(id) {
		return fmt.Errorf("invalid container ID %q: must match %s", id, containerIDPattern.String())
	}
	return nil
}

func validateResources(res v1alpha1.ContainerResources) error {
	if res.Cpu.Min > res.Cpu.Max {
		return fmt.Errorf("cpu.min (%d) must not exceed cpu.max (%d)", res.Cpu.Min, res.Cpu.Max)
	}

	minMem, err := units.ConvertMemory(res.Memory.Min)
	if err != nil {
		return fmt.Errorf("invalid memory.min %q: %w", res.Memory.Min, err)
	}
	maxMem, err := units.ConvertMemory(res.Memory.Max)
	if err != nil {
		return fmt.Errorf("invalid memory.max %q: %w", res.Memory.Max, err)
	}
	if minMem.Cmp(maxMem) > 0 {
		return fmt.Errorf("memory.min (%s) must not exceed memory.max (%s)", res.Memory.Min, res.Memory.Max)
	}

	return nil
}

func validateUserLabels(labels *map[string]string) error {
	if labels == nil {
		return nil
	}
	for k := range *labels {
		if dcm.ReservedLabelKeys[k] {
			return fmt.Errorf("label %q is reserved by DCM and cannot be set by the user", k)
		}
	}
	return nil
}

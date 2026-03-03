package kubernetes

import (
	"fmt"

	"github.com/dcm-project/k8s-container-service-provider/internal/store"
)

const (
	LabelManagedBy    = "managed-by"
	LabelInstanceID   = "dcm-instance-id"
	LabelServiceType  = "dcm-service-type"
	ValueManagedByDCM = "dcm"
	ValueServiceType  = "container"
)

// reservedLabels is the set of label keys managed by DCM.
var reservedLabels = map[string]bool{
	LabelManagedBy:   true,
	LabelInstanceID:  true,
	LabelServiceType: true,
}

// dcmLabels returns the standard DCM labels for a given instance ID.
func dcmLabels(instanceID string) map[string]string {
	return map[string]string{
		LabelManagedBy:   ValueManagedByDCM,
		LabelInstanceID:  instanceID,
		LabelServiceType: ValueServiceType,
	}
}

// validateUserLabels returns an InvalidArgumentError if any user-supplied label
// key collides with a DCM reserved label.
func validateUserLabels(labels *map[string]string) error {
	if labels == nil {
		return nil
	}
	for k := range *labels {
		if reservedLabels[k] {
			return &store.InvalidArgumentError{
				Message: fmt.Sprintf("label %q is reserved by DCM and cannot be set by the user", k),
			}
		}
	}
	return nil
}

// mergeLabels merges DCM labels with user labels into a new map.
// DCM labels are written first, then user labels are added.
func mergeLabels(dcm, user map[string]string) map[string]string {
	merged := make(map[string]string, len(dcm)+len(user))
	for k, v := range dcm {
		merged[k] = v
	}
	for k, v := range user {
		merged[k] = v
	}
	return merged
}

// instanceSelector returns a label selector string that matches a specific
// DCM instance by ID.
func instanceSelector(instanceID string) string {
	return fmt.Sprintf("%s=%s,%s=%s", LabelInstanceID, instanceID, LabelManagedBy, ValueManagedByDCM)
}

// dcmSelector returns a label selector string that matches all DCM-managed
// container resources.
func dcmSelector() string {
	return fmt.Sprintf("%s=%s,%s=%s", LabelManagedBy, ValueManagedByDCM, LabelServiceType, ValueServiceType)
}

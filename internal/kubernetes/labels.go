package kubernetes

import (
	"fmt"

	"github.com/dcm-project/k8s-container-service-provider/internal/dcm"
)

// dcmLabels returns the standard DCM labels for a given instance ID.
func dcmLabels(instanceID string) map[string]string {
	return map[string]string{
		dcm.LabelManagedBy:   dcm.ValueManagedByDCM,
		dcm.LabelInstanceID:  instanceID,
		dcm.LabelServiceType: dcm.ValueServiceType,
	}
}

// mergeLabels merges DCM base labels with user labels into a new map.
// User labels are written first, then DCM labels overwrite — DCM labels
// always win on collision (defense-in-depth against label corruption).
func mergeLabels(base, user map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(user))
	for k, v := range user {
		merged[k] = v
	}
	for k, v := range base {
		merged[k] = v
	}
	return merged
}

// instanceSelector returns a label selector string that matches a specific
// DCM instance by ID.
func instanceSelector(instanceID string) string {
	return fmt.Sprintf("%s=%s,%s=%s", dcm.LabelInstanceID, instanceID, dcm.LabelManagedBy, dcm.ValueManagedByDCM)
}

// dcmSelector returns a label selector string that matches all DCM-managed
// container resources.
func dcmSelector() string {
	return fmt.Sprintf("%s=%s,%s=%s", dcm.LabelManagedBy, dcm.ValueManagedByDCM, dcm.LabelServiceType, dcm.ValueServiceType)
}

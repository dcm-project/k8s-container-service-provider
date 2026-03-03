package kubernetes_test

import (
	"context"

	v1alpha1 "github.com/dcm-project/k8s-container-service-provider/api/v1alpha1"
	k8sstore "github.com/dcm-project/k8s-container-service-provider/internal/kubernetes"
	"github.com/dcm-project/k8s-container-service-provider/internal/store"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// Compile-time assertion: K8sContainerStore implements ContainerRepository (TC-U024).
var _ store.ContainerRepository = (*k8sstore.K8sContainerStore)(nil)

// newTestStore creates a K8sContainerStore backed by a fake clientset.
func newTestStore(cfg k8sstore.K8sConfig) (*k8sstore.K8sContainerStore, *fake.Clientset) {
	client := fake.NewSimpleClientset()
	s := k8sstore.NewK8sContainerStore(client, cfg)
	return s, client
}

// defaultConfig returns a K8sConfig with reasonable defaults for testing.
func defaultConfig() k8sstore.K8sConfig {
	return k8sstore.K8sConfig{
		Namespace:          "default",
		CreateService:      false,
		DefaultServiceType: "ClusterIP",
	}
}

// serviceEnabledConfig returns a K8sConfig with Service creation enabled.
func serviceEnabledConfig() k8sstore.K8sConfig {
	return k8sstore.K8sConfig{
		Namespace:          "default",
		CreateService:      true,
		DefaultServiceType: "ClusterIP",
	}
}

// minimalContainer creates a container with only the required fields set.
func minimalContainer(name string) v1alpha1.Container {
	return v1alpha1.Container{
		ServiceType: v1alpha1.ContainerServiceTypeContainer,
		Metadata: v1alpha1.ContainerMetadata{
			Name: name,
		},
		Image: v1alpha1.ContainerImage{
			Reference: "nginx:latest",
		},
		Resources: v1alpha1.ContainerResources{
			Cpu: v1alpha1.ContainerCpu{
				Min: 1,
				Max: 2,
			},
			Memory: v1alpha1.ContainerMemory{
				Min: "1GB",
				Max: "2GB",
			},
		},
	}
}

// containerWithPorts creates a container with the specified network ports.
func containerWithPorts(name string, ports ...int) v1alpha1.Container {
	c := minimalContainer(name)
	containerPorts := make([]v1alpha1.ContainerPort, len(ports))
	for i, p := range ports {
		containerPorts[i] = v1alpha1.ContainerPort{ContainerPort: p}
	}
	c.Network = &v1alpha1.ContainerNetwork{
		Ports: containerPorts,
	}
	return c
}

// dcmLabels returns the standard DCM labels for a given instance ID.
func dcmLabels(instanceID string) map[string]string {
	return map[string]string{
		"managed-by":       "dcm",
		"dcm-instance-id":  instanceID,
		"dcm-service-type": "container",
	}
}

// createFakeDeployment creates a Deployment with DCM labels directly in the fake client.
func createFakeDeployment(client kubernetes.Interface, namespace, name, instanceID string) error {
	labels := dcmLabels(instanceID)
	replicas := int32(1)
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}
	_, err := client.AppsV1().Deployments(namespace).Create(context.Background(), deploy, metav1.CreateOptions{})
	return err
}

// createFakePod creates a Pod with DCM labels directly in the fake client.
func createFakePod(client kubernetes.Interface, namespace, name, instanceID string, phase corev1.PodPhase, podIP string) error {
	labels := dcmLabels(instanceID)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Status: corev1.PodStatus{
			Phase: phase,
			PodIP: podIP,
		},
	}
	_, err := client.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	return err
}

// createFakeService creates a Service with DCM labels directly in the fake client.
func createFakeService(client kubernetes.Interface, namespace, name, instanceID string, svcType corev1.ServiceType, ports ...int32) error {
	labels := dcmLabels(instanceID)
	svcPorts := make([]corev1.ServicePort, len(ports))
	for i, p := range ports {
		svcPorts[i] = corev1.ServicePort{
			Port:       p,
			TargetPort: intstr.FromInt32(p),
			Protocol:   corev1.ProtocolTCP,
		}
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     svcType,
			Selector: labels,
			Ports:    svcPorts,
		},
	}
	_, err := client.CoreV1().Services(namespace).Create(context.Background(), svc, metav1.CreateOptions{})
	return err
}

// createFakeServiceWithClusterIP creates a Service with a ClusterIP set in status.
func createFakeServiceWithClusterIP(client kubernetes.Interface, namespace, name, instanceID string, svcType corev1.ServiceType, clusterIP string, ports ...int32) error {
	labels := dcmLabels(instanceID)
	svcPorts := make([]corev1.ServicePort, len(ports))
	for i, p := range ports {
		svcPorts[i] = corev1.ServicePort{
			Port:       p,
			TargetPort: intstr.FromInt32(p),
			Protocol:   corev1.ProtocolTCP,
		}
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:      svcType,
			Selector:  labels,
			Ports:     svcPorts,
			ClusterIP: clusterIP,
		},
	}
	_, err := client.CoreV1().Services(namespace).Create(context.Background(), svc, metav1.CreateOptions{})
	return err
}

// createFakeLBService creates a LoadBalancer Service with an external IP.
func createFakeLBService(client kubernetes.Interface, namespace, name, instanceID, externalIP string, ports ...int32) error {
	labels := dcmLabels(instanceID)
	svcPorts := make([]corev1.ServicePort, len(ports))
	for i, p := range ports {
		svcPorts[i] = corev1.ServicePort{
			Port:       p,
			TargetPort: intstr.FromInt32(p),
			Protocol:   corev1.ProtocolTCP,
		}
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeLoadBalancer,
			Selector: labels,
			Ports:    svcPorts,
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{IP: externalIP},
				},
			},
		},
	}
	_, err := client.CoreV1().Services(namespace).Create(context.Background(), svc, metav1.CreateOptions{})
	return err
}

// createFakeDeploymentWithConditions creates a Deployment with status conditions.
func createFakeDeploymentWithConditions(client kubernetes.Interface, namespace, name, instanceID string, conditions []appsv1.DeploymentCondition) error {
	labels := dcmLabels(instanceID)
	replicas := int32(1)
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: "nginx:latest",
						},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			Conditions: conditions,
		},
	}
	_, err := client.AppsV1().Deployments(namespace).Create(context.Background(), deploy, metav1.CreateOptions{})
	return err
}

// createFakePodWithConditions creates a Pod with status conditions.
func createFakePodWithConditions(client kubernetes.Interface, namespace, name, instanceID string, phase corev1.PodPhase, podIP string, conditions []corev1.PodCondition) error {
	labels := dcmLabels(instanceID)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Status: corev1.PodStatus{
			Phase:      phase,
			PodIP:      podIP,
			Conditions: conditions,
		},
	}
	_, err := client.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	return err
}

// boolPtr returns a pointer to the given bool value.
func boolPtr(b bool) *bool {
	return &b
}

// withServiceHints returns a ProviderHints with Kubernetes service hints.
func withServiceHints(enabled bool, svcType string) *v1alpha1.ProviderHints {
	hints := &v1alpha1.ProviderHints{
		Kubernetes: &v1alpha1.KubernetesProviderHints{
			Service: &v1alpha1.KubernetesServiceHints{
				Enabled: boolPtr(enabled),
			},
		},
	}
	if svcType != "" {
		t := v1alpha1.KubernetesServiceHintsType(svcType)
		hints.Kubernetes.Service.Type = &t
	}
	return hints
}

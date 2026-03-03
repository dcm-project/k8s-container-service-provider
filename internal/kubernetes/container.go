package kubernetes

import (
	"context"
	"fmt"
	"time"

	v1alpha1 "github.com/dcm-project/k8s-container-service-provider/api/v1alpha1"
	"github.com/dcm-project/k8s-container-service-provider/internal/store"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// enrichFromCluster enriches a Container with runtime data from Pods and Services.
func (s *K8sContainerStore) enrichFromCluster(
	ctx context.Context,
	c *v1alpha1.Container,
	deploy *appsv1.Deployment,
	instanceID string,
) error {
	pods, err := s.client.CoreV1().Pods(s.cfg.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: instanceSelector(instanceID),
	})
	if err != nil {
		return err
	}

	if len(pods.Items) == 1 {
		enrichWithPod(c, &pods.Items[0])
	} else if len(pods.Items) > 1 {
		return &store.ConflictError{
			InstanceRef: fmt.Sprintf("multiple pods found for container %s", instanceID),
		}
	} else {
		pending := v1alpha1.PENDING
		c.Status = &pending
		if t := latestDeploymentTransitionTime(deploy); t != nil {
			c.UpdateTime = t
		}
	}

	svc, err := s.client.CoreV1().Services(s.cfg.Namespace).Get(ctx, deploy.Name, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	} else {
		enrichWithService(c, svc)
	}

	return nil
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

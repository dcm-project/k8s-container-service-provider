package kubernetes_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1alpha1 "github.com/dcm-project/k8s-container-service-provider/api/v1alpha1"
	"github.com/dcm-project/k8s-container-service-provider/internal/store"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("K8s Store", func() {
	Describe("Service Creation", func() {

		// TC-I019: Service created when createService config is true
		It("creates Service when createService config is true (TC-I019)", func() {
			s, client := newTestStore(serviceEnabledConfig())
			c := containerWithPorts("my-app", 8080)

			_, err := s.Create(context.Background(), c, "test-id-019")
			Expect(err).NotTo(HaveOccurred())

			svc, err := client.CoreV1().Services("default").Get(context.Background(), "my-app", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(svc.Spec.Ports).To(HaveLen(1))
			Expect(svc.Spec.Ports[0].Port).To(Equal(int32(8080)))
		})

		// TC-I020: Service created when providerHints enables it
		It("creates Service when providerHints enables it (TC-I020)", func() {
			s, client := newTestStore(defaultConfig()) // createService=false
			c := containerWithPorts("my-app", 8080)
			c.ProviderHints = withServiceHints(true, "")

			_, err := s.Create(context.Background(), c, "test-id-020")
			Expect(err).NotTo(HaveOccurred())

			svc, err := client.CoreV1().Services("default").Get(context.Background(), "my-app", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(svc).NotTo(BeNil())
		})

		// TC-I021: Service creation returns error when no ports defined
		It("returns error when Service enabled but no ports defined (TC-I021)", func() {
			s, client := newTestStore(serviceEnabledConfig())
			c := minimalContainer("my-app") // no network.ports

			_, err := s.Create(context.Background(), c, "test-id-021")

			var invalidErr *store.InvalidArgumentError
			Expect(errors.As(err, &invalidErr)).To(BeTrue(), "expected InvalidArgumentError, got: %v", err)

			// Verify atomic: no Deployment created
			_, deployErr := client.AppsV1().Deployments("default").Get(context.Background(), "my-app", metav1.GetOptions{})
			Expect(deployErr).To(HaveOccurred())
		})

		// TC-I022: Service includes all ports in single resource
		It("includes all ports in a single Service (TC-I022)", func() {
			s, client := newTestStore(serviceEnabledConfig())
			c := containerWithPorts("my-app", 8080, 9090)

			_, err := s.Create(context.Background(), c, "test-id-022")
			Expect(err).NotTo(HaveOccurred())

			svcs, err := client.CoreV1().Services("default").List(context.Background(), metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(svcs.Items).To(HaveLen(1))
			Expect(svcs.Items[0].Spec.Ports).To(HaveLen(2))
		})

		// TC-I090: Multi-port Service has named ports for K8s compliance
		It("assigns unique names to each ServicePort (TC-I090)", func() {
			s, client := newTestStore(serviceEnabledConfig())
			c := containerWithPorts("my-app", 8080, 9090, 3000)

			_, err := s.Create(context.Background(), c, "test-id-090")
			Expect(err).NotTo(HaveOccurred())

			svc, err := client.CoreV1().Services("default").Get(context.Background(), "my-app", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(svc.Spec.Ports).To(HaveLen(3))
			Expect(svc.Spec.Ports[0].Name).To(Equal("port-8080"))
			Expect(svc.Spec.Ports[1].Name).To(Equal("port-9090"))
			Expect(svc.Spec.Ports[2].Name).To(Equal("port-3000"))
		})

		// TC-I023: Service uses default type from configuration
		It("uses default service type from config (TC-I023)", func() {
			s, client := newTestStore(serviceEnabledConfig())
			c := containerWithPorts("my-app", 8080)

			_, err := s.Create(context.Background(), c, "test-id-023")
			Expect(err).NotTo(HaveOccurred())

			svc, err := client.CoreV1().Services("default").Get(context.Background(), "my-app", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(svc.Spec.Type)).To(Equal("ClusterIP"))
		})

		// TC-I024: Service type overridden by providerHints
		It("overrides service type via providerHints (TC-I024)", func() {
			s, client := newTestStore(serviceEnabledConfig())
			c := containerWithPorts("my-app", 8080)
			c.ProviderHints = withServiceHints(true, "LoadBalancer")

			_, err := s.Create(context.Background(), c, "test-id-024")
			Expect(err).NotTo(HaveOccurred())

			svc, err := client.CoreV1().Services("default").Get(context.Background(), "my-app", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(svc.Spec.Type)).To(Equal("LoadBalancer"))
		})

		// TC-I025: Service not created when providerHints explicitly disables it
		It("does not create Service when providerHints disables it (TC-I025)", func() {
			s, client := newTestStore(serviceEnabledConfig()) // config says create
			c := containerWithPorts("my-app", 8080)
			c.ProviderHints = withServiceHints(false, "") // hints say don't

			_, err := s.Create(context.Background(), c, "test-id-025")
			Expect(err).NotTo(HaveOccurred())

			svcs, err := client.CoreV1().Services("default").List(context.Background(), metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(svcs.Items).To(BeEmpty())
		})

		// TC-I026: Service not created by default
		It("does not create Service by default (TC-I026)", func() {
			s, client := newTestStore(defaultConfig()) // createService=false
			c := containerWithPorts("my-app", 8080)

			_, err := s.Create(context.Background(), c, "test-id-026")
			Expect(err).NotTo(HaveOccurred())

			svcs, err := client.CoreV1().Services("default").List(context.Background(), metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(svcs.Items).To(BeEmpty())
		})

		// TC-I027: Service carries DCM labels
		It("carries DCM labels on Service (TC-I027)", func() {
			s, client := newTestStore(serviceEnabledConfig())
			c := containerWithPorts("my-app", 8080)

			_, err := s.Create(context.Background(), c, "abc-123")
			Expect(err).NotTo(HaveOccurred())

			svc, err := client.CoreV1().Services("default").Get(context.Background(), "my-app", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			Expect(svc.Labels).To(HaveKeyWithValue("managed-by", "dcm"))
			Expect(svc.Labels).To(HaveKeyWithValue("dcm-instance-id", "abc-123"))
			Expect(svc.Labels).To(HaveKeyWithValue("dcm-service-type", "container"))
		})

		// TC-I072: Empty ports array treated as no ports for Service creation
		It("returns error when Service enabled but ports array is empty (TC-I072)", func() {
			s, client := newTestStore(serviceEnabledConfig())
			c := minimalContainer("my-app")
			c.Network = &v1alpha1.ContainerNetwork{
				Ports: []v1alpha1.ContainerPort{}, // explicit empty
			}

			_, err := s.Create(context.Background(), c, "test-id-072")

			var invalidErr *store.InvalidArgumentError
			Expect(errors.As(err, &invalidErr)).To(BeTrue(), "expected InvalidArgumentError, got: %v", err)

			// Verify atomic: no Deployment created
			_, deployErr := client.AppsV1().Deployments("default").Get(context.Background(), "my-app", metav1.GetOptions{})
			Expect(deployErr).To(HaveOccurred())
		})

		// TC-I073: Service error via providerHints path with no ports
		It("returns error when providerHints enables Service but no ports (TC-I073)", func() {
			s, client := newTestStore(defaultConfig()) // createService=false
			c := minimalContainer("my-app")            // no ports
			c.ProviderHints = withServiceHints(true, "")

			_, err := s.Create(context.Background(), c, "test-id-073")

			var invalidErr *store.InvalidArgumentError
			Expect(errors.As(err, &invalidErr)).To(BeTrue(), "expected InvalidArgumentError, got: %v", err)

			// Verify atomic: no Deployment created
			_, deployErr := client.AppsV1().Deployments("default").Get(context.Background(), "my-app", metav1.GetOptions{})
			Expect(deployErr).To(HaveOccurred())
		})

		// TC-I074: NodePort service type via providerHints
		It("creates NodePort Service via providerHints (TC-I074)", func() {
			s, client := newTestStore(serviceEnabledConfig())
			c := containerWithPorts("my-app", 8080)
			c.ProviderHints = withServiceHints(true, "NodePort")

			_, err := s.Create(context.Background(), c, "test-id-074")
			Expect(err).NotTo(HaveOccurred())

			svc, err := client.CoreV1().Services("default").Get(context.Background(), "my-app", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(svc.Spec.Type)).To(Equal("NodePort"))
		})
	})
})

package container

import (
	v1alpha1 "github.com/dcm-project/k8s-container-service-provider/api/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validation Contract", func() {

	// TC-U068: containerIDPattern stays in sync with the OpenAPI spec.
	It("containerIDPattern matches the OpenAPI id parameter pattern (TC-U068)", func() {
		swagger, err := v1alpha1.GetSwagger()
		Expect(err).NotTo(HaveOccurred())

		createOp := swagger.Paths.Find("/api/v1alpha1/containers").Post
		Expect(createOp).NotTo(BeNil(), "POST /api/v1alpha1/containers must exist")

		var specPattern string
		for _, param := range createOp.Parameters {
			if param.Value != nil && param.Value.Name == "id" {
				specPattern = param.Value.Schema.Value.Pattern
				break
			}
		}
		Expect(specPattern).NotTo(BeEmpty(), "id parameter must have a pattern in the OpenAPI spec")

		Expect(containerIDPattern.String()).To(Equal(specPattern),
			"handler containerIDPattern must match the OpenAPI spec pattern")
	})
})

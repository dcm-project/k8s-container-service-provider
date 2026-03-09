package apiserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	oapigen "github.com/dcm-project/k8s-container-service-provider/internal/api/server"
	"github.com/dcm-project/k8s-container-service-provider/internal/apiserver"
	"github.com/dcm-project/k8s-container-service-provider/internal/config"
	"github.com/dcm-project/k8s-container-service-provider/internal/handlers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Container API Handlers - Request Validation", func() {

	// startValidationServer starts a minimal server for validation tests and
	// returns the base URL. The server is stopped when the test context ends.
	startValidationServer := func() string {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Address:         ":0",
				ShutdownTimeout: 5 * time.Second,
			},
		}
		logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
		h := handlers.New(logger, time.Now(), "0.0.1-test", &oapigen.Unimplemented{})
		srv := apiserver.New(cfg, logger, h)

		ln, err := net.Listen("tcp", ":0")
		Expect(err).NotTo(HaveOccurred())
		addr := ln.Addr().String()

		ctx, cancel := context.WithCancel(context.Background())
		DeferCleanup(cancel)

		errCh := make(chan error, 1)
		go func() {
			errCh <- srv.Run(ctx, ln)
		}()

		// Wait for server to be ready.
		Eventually(func() error {
			resp, reqErr := http.Get(fmt.Sprintf("http://%s/health", addr))
			if reqErr != nil {
				return reqErr
			}
			resp.Body.Close()
			return nil
		}).WithTimeout(5 * time.Second).WithPolling(50 * time.Millisecond).Should(Succeed())

		return fmt.Sprintf("http://%s", addr)
	}

	// TC-U014: validates request body via OpenAPI middleware
	DescribeTable("validates request body via OpenAPI middleware (TC-U014)",
		func(bodyJSON string, description string) {
			baseURL := startValidationServer()

			resp, err := http.Post(
				baseURL+"/api/v1alpha1/containers",
				"application/json",
				strings.NewReader(bodyJSON),
			)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest),
				"expected 400 for: %s", description)
			Expect(resp.Header.Get("Content-Type")).To(Equal("application/problem+json"),
				"expected RFC 7807 content type for: %s", description)

			body, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			var problemJSON map[string]any
			Expect(json.Unmarshal(body, &problemJSON)).To(Succeed(),
				"body should be valid JSON for: %s", description)
			Expect(problemJSON).To(HaveKey("type"),
				"RFC 7807 body must have 'type' for: %s", description)
			Expect(problemJSON["type"]).To(Equal("INVALID_ARGUMENT"))
			Expect(problemJSON).To(HaveKey("title"),
				"RFC 7807 body must have 'title' for: %s", description)
			Expect(problemJSON).To(HaveKey("status"),
				"RFC 7807 body must have 'status' for: %s", description)
		},

		// Missing required top-level fields
		Entry("empty object",
			`{}`,
			"empty object missing all required fields"),
		Entry("missing image",
			`{"service_type":"container","metadata":{"name":"test"},"resources":{"cpu":{"min":1,"max":2},"memory":{"min":"1GB","max":"2GB"}}}`,
			"missing required image field"),
		Entry("missing metadata",
			`{"service_type":"container","image":{"reference":"nginx:latest"},"resources":{"cpu":{"min":1,"max":2},"memory":{"min":"1GB","max":"2GB"}}}`,
			"missing required metadata field"),
		Entry("missing resources",
			`{"service_type":"container","image":{"reference":"nginx:latest"},"metadata":{"name":"test"}}`,
			"missing required resources field"),
		Entry("missing service_type",
			`{"image":{"reference":"nginx:latest"},"metadata":{"name":"test"},"resources":{"cpu":{"min":1,"max":2},"memory":{"min":"1GB","max":"2GB"}}}`,
			"missing required service_type field"),

		// Missing required nested fields
		Entry("missing metadata.name",
			`{"service_type":"container","image":{"reference":"nginx:latest"},"metadata":{},"resources":{"cpu":{"min":1,"max":2},"memory":{"min":"1GB","max":"2GB"}}}`,
			"missing required metadata.name"),
		Entry("missing image.reference",
			`{"service_type":"container","image":{},"metadata":{"name":"test"},"resources":{"cpu":{"min":1,"max":2},"memory":{"min":"1GB","max":"2GB"}}}`,
			"missing required image.reference"),
		Entry("missing resources.cpu",
			`{"service_type":"container","image":{"reference":"nginx:latest"},"metadata":{"name":"test"},"resources":{"memory":{"min":"1GB","max":"2GB"}}}`,
			"missing required resources.cpu"),
		Entry("missing resources.memory",
			`{"service_type":"container","image":{"reference":"nginx:latest"},"metadata":{"name":"test"},"resources":{"cpu":{"min":1,"max":2}}}`,
			"missing required resources.memory"),

		// Invalid types
		Entry("invalid service_type enum",
			`{"service_type":"invalid","image":{"reference":"nginx:latest"},"metadata":{"name":"test"},"resources":{"cpu":{"min":1,"max":2},"memory":{"min":"1GB","max":"2GB"}}}`,
			"invalid service_type enum value"),
		Entry("cpu.min is string instead of int",
			`{"service_type":"container","image":{"reference":"nginx:latest"},"metadata":{"name":"test"},"resources":{"cpu":{"min":"one","max":2},"memory":{"min":"1GB","max":"2GB"}}}`,
			"cpu.min wrong type"),
		Entry("cpu.max is negative",
			`{"service_type":"container","image":{"reference":"nginx:latest"},"metadata":{"name":"test"},"resources":{"cpu":{"min":1,"max":-1},"memory":{"min":"1GB","max":"2GB"}}}`,
			"cpu.max negative value"),

		// TC-U055: cpu.min=0 rejected by OpenAPI minimum: 1
		Entry("cpu.min is 0 (TC-U055)",
			`{"service_type":"container","image":{"reference":"nginx:latest"},"metadata":{"name":"test"},"resources":{"cpu":{"min":0,"max":2},"memory":{"min":"1GB","max":"2GB"}}}`,
			"cpu.min below minimum 1"),

		// Malformed JSON
		Entry("malformed JSON",
			`{not valid json}`,
			"malformed JSON body"),

		// Missing body entirely (empty string)
		Entry("empty body",
			``,
			"empty request body"),

		// Invalid nested object structure
		Entry("network.ports is string instead of array",
			`{"service_type":"container","image":{"reference":"nginx:latest"},"metadata":{"name":"test"},"resources":{"cpu":{"min":1,"max":2},"memory":{"min":"1GB","max":"2GB"}},"network":{"ports":"invalid"}}`,
			"network.ports wrong type"),
		Entry("missing cpu.min",
			`{"service_type":"container","image":{"reference":"nginx:latest"},"metadata":{"name":"test"},"resources":{"cpu":{"max":2},"memory":{"min":"1GB","max":"2GB"}}}`,
			"missing required cpu.min"),

		// TC-U053: invalid metadata.name format
		Entry("metadata.name with invalid characters (TC-U053)",
			`{"service_type":"container","image":{"reference":"nginx:latest"},"metadata":{"name":"Invalid_Name!"},"resources":{"cpu":{"min":1,"max":2},"memory":{"min":"1GB","max":"2GB"}}}`,
			"metadata.name with invalid characters"),

		// TC-U056: port out of range
		Entry("container_port exceeds 65535 (TC-U056)",
			`{"service_type":"container","image":{"reference":"nginx:latest"},"metadata":{"name":"test"},"resources":{"cpu":{"min":1,"max":2},"memory":{"min":"1GB","max":"2GB"}},"network":{"ports":[{"container_port":70000}]}}`,
			"container_port above maximum 65535"),
		Entry("container_port is 0 (TC-U056)",
			`{"service_type":"container","image":{"reference":"nginx:latest"},"metadata":{"name":"test"},"resources":{"cpu":{"min":1,"max":2},"memory":{"min":"1GB","max":"2GB"}},"network":{"ports":[{"container_port":0}]}}`,
			"container_port below minimum 1"),

		// TC-U059: network object without ports field rejected
		Entry("network object without ports (TC-U059)",
			`{"service_type":"container","image":{"reference":"nginx:latest"},"metadata":{"name":"test"},"resources":{"cpu":{"min":1,"max":2},"memory":{"min":"1GB","max":"2GB"}},"network":{}}`,
			"network object without ports field"),
	)

	// TC-U012: rejects invalid client IDs via OpenAPI middleware
	DescribeTable("rejects invalid client IDs (TC-U012)",
		func(invalidID string, description string) {
			baseURL := startValidationServer()

			body := `{"service_type":"container","metadata":{"name":"test"},"image":{"reference":"nginx:latest"},"resources":{"cpu":{"min":1,"max":2},"memory":{"min":"1GB","max":"2GB"}}}`
			resp, err := http.Post(
				baseURL+"/api/v1alpha1/containers?id="+invalidID,
				"application/json",
				strings.NewReader(body),
			)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest),
				"expected 400 for: %s", description)
			Expect(resp.Header.Get("Content-Type")).To(Equal("application/problem+json"),
				"expected RFC 7807 content type for: %s", description)

			respBody, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			var problemJSON map[string]any
			Expect(json.Unmarshal(respBody, &problemJSON)).To(Succeed())
			Expect(problemJSON["type"]).To(Equal("INVALID_ARGUMENT"))
		},
		Entry("leading dash", "-leading-dash", "ID starting with dash"),
		Entry("trailing dash", "trailing-", "ID ending with dash"),
		Entry("has underscore", "has_underscore", "ID containing underscore"),
		Entry("UPPERCASE", "UPPERCASE", "ID with uppercase letters"),
		Entry("too long (64 chars)", strings.Repeat("a", 64), "ID exceeding 63 character limit"),
	)

	// TC-U047: accepts valid boundary IDs via OpenAPI middleware
	DescribeTable("accepts valid boundary IDs (TC-U047)",
		func(validID string, description string) {
			baseURL := startValidationServer()

			body := `{"service_type":"container","metadata":{"name":"test"},"image":{"reference":"nginx:latest"},"resources":{"cpu":{"min":1,"max":2},"memory":{"min":"1GB","max":"2GB"}}}`
			resp, err := http.Post(
				baseURL+"/api/v1alpha1/containers?id="+validID,
				"application/json",
				strings.NewReader(body),
			)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			// Valid ID should pass middleware validation and reach the Unimplemented handler (501)
			Expect(resp.StatusCode).To(Equal(http.StatusNotImplemented),
				"expected 501 (pass-through) for valid ID: %s", description)
		},
		Entry("single char", "a", "minimum length"),
		Entry("two chars", "ab", "two characters"),
		Entry("max length (63 chars)", strings.Repeat("a", 63), "maximum length"),
		Entry("with hyphens", "a-b", "dash in middle"),
		Entry("letters and digits", "a0", "letter followed by digit"),
		Entry("starts with digit", "1abc", "starts with digit"),
		Entry("UUID format", "550e8400-e29b-41d4-a716-446655440000", "UUID format"),
	)

	// TC-U067: valid request passes OpenAPI middleware and reaches handler.
	It("passes a valid request through OpenAPI middleware (TC-U067)", func() {
		baseURL := startValidationServer()

		body := `{"service_type":"container","metadata":{"name":"test"},"image":{"reference":"nginx:latest"},"resources":{"cpu":{"min":1,"max":2},"memory":{"min":"1GB","max":"2GB"}}}`
		resp, err := http.Post(
			baseURL+"/api/v1alpha1/containers",
			"application/json",
			strings.NewReader(body),
		)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusNotImplemented),
			"valid request should pass middleware and reach the Unimplemented handler (501)")
	})
})

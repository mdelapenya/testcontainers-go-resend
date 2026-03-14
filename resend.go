package resend

import (
	"context"
	"embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gopkg.in/yaml.v3"
	microcks "microcks.io/testcontainers-go"
)

//go:embed testdata/resend.yaml
var embeddedSpec embed.FS

const (
	// DefaultImage is the default Microcks image used to mock the Resend API.
	DefaultImage = "quay.io/microcks/microcks-uber:1.12.0"

	// defaultSpecURL is the default URL to the Resend OpenAPI spec.
	defaultSpecURL = "https://raw.githubusercontent.com/resend/resend-openapi/refs/heads/main/resend.yaml"
)

// Container wraps a MicrocksContainer pre-loaded with the Resend OpenAPI spec.
type Container struct {
	*microcks.MicrocksContainer
	serviceName    string
	serviceVersion string
}

// Run creates an instance of the Container type, starting a Microcks container
// pre-loaded with the Resend OpenAPI spec.
func Run(ctx context.Context, img string, opts ...testcontainers.ContainerCustomizer) (*Container, error) {
	ro := defaultOptions()
	var microcksOpts []testcontainers.ContainerCustomizer
	for _, opt := range opts {
		if o, ok := opt.(Option); ok {
			if err := o(&ro); err != nil {
				return nil, fmt.Errorf("apply option: %w", err)
			}
		} else {
			microcksOpts = append(microcksOpts, opt)
		}
	}

	specPath, name, version, err := prepareSpec(ro.specURL)
	if err != nil {
		return nil, fmt.Errorf("run resend: %w", err)
	}

	moduleOpts := []testcontainers.ContainerCustomizer{
		microcks.WithMainArtifact(specPath),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("8080/tcp"),
			wait.ForHTTP("/api/health").WithPort("8080/tcp").WithStatusCodeMatcher(func(status int) bool {
				return status == http.StatusOK
			}),
		),
	}
	moduleOpts = append(moduleOpts, microcksOpts...)

	container, err := microcks.Run(ctx, img, moduleOpts...)
	var c *Container
	if container != nil {
		c = &Container{
			MicrocksContainer: container,
			serviceName:       name,
			serviceVersion:    version,
		}
	}

	if err != nil {
		return c, fmt.Errorf("run resend: %w", err)
	}

	return c, nil
}

// ServiceName returns the API service name extracted from the OpenAPI spec (e.g. "Resend").
func (c *Container) ServiceName() string {
	return c.serviceName
}

// ServiceVersion returns the API service version extracted from the OpenAPI spec (e.g. "1.5.0").
func (c *Container) ServiceVersion() string {
	return c.serviceVersion
}

// BaseURL returns the mock endpoint base URL for the Resend REST API.
// This is the URL you should configure as the Resend API base URL in your client.
func (c *Container) BaseURL(ctx context.Context) (string, error) {
	return c.RestMockEndpoint(ctx, c.serviceName, c.serviceVersion)
}

// prepareSpec downloads the OpenAPI spec from specURL, enriches it with
// Microcks-compatible response examples, and writes it to a temporary file.
// If the download fails, it falls back to the embedded spec.
func prepareSpec(specURL string) (path, name, version string, err error) {
	data, err := downloadSpec(specURL)
	if err != nil {
		// Fallback to embedded spec.
		data, err = embeddedSpec.ReadFile("testdata/resend.yaml")
		if err != nil {
			return "", "", "", fmt.Errorf("read embedded spec: %w", err)
		}
	}

	var spec map[string]any
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return "", "", "", fmt.Errorf("parse spec: %w", err)
	}

	// Extract service name and version from the spec.
	info, _ := spec["info"].(map[string]any)
	name, _ = info["title"].(string)
	version, _ = info["version"].(string)
	if name == "" || version == "" {
		return "", "", "", fmt.Errorf("spec missing info.title or info.version")
	}

	// Enrich the spec with Microcks-compatible response examples.
	enrichSpec(spec)

	enriched, err := yaml.Marshal(spec)
	if err != nil {
		return "", "", "", fmt.Errorf("marshal enriched spec: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "resend-openapi-*")
	if err != nil {
		return "", "", "", fmt.Errorf("create temp dir: %w", err)
	}

	specPath := filepath.Join(tmpDir, "resend.yaml")
	if err := os.WriteFile(specPath, enriched, 0o644); err != nil {
		return "", "", "", fmt.Errorf("write spec file: %w", err)
	}

	return specPath, name, version, nil
}

// downloadSpec fetches the OpenAPI spec from the given URL.
func downloadSpec(specURL string) ([]byte, error) {
	resp, err := http.Get(specURL)
	if err != nil {
		return nil, fmt.Errorf("download spec: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download spec: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read spec body: %w", err)
	}
	return data, nil
}

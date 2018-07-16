package tracing

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	jaegercfg "github.com/uber/jaeger-client-go/config"
)

// InstallJaeger registers Jaeger as the OpenTracing implementation.
func InstallJaeger(serviceName string, cfg *jaegercfg.Configuration) io.Closer {
	closer, err := cfg.InitGlobalTracer(serviceName)
	if err != nil {
		fmt.Printf("Could not initialize jaeger tracer: %s\n", err.Error())
		os.Exit(1)
	}
	return closer
}

// NewFromEnv is a convenience function to allow tracing configuration
// via environment variables
// Tracing is disabled unless one of the following environment variables is used to configure jaeger:
// - JAEGER_AGENT_HOST
// - JAEGER_SAMPLER_MANAGER_HOST_PORT
func NewFromEnv(serviceName string) io.Closer {
	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		fmt.Printf("Could not load jaeger tracer configuration: %s\n", err.Error())
		os.Exit(1)
	}

	if cfg.Sampler.SamplingServerURL == "" && cfg.Reporter.LocalAgentHostPort == "" {
		fmt.Printf("Jaeger tracer disabled: No trace report agent or config server specified\n")
		return ioutil.NopCloser(nil)
	}

	return InstallJaeger(serviceName, cfg)
}

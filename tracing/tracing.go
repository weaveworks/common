package tracing

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"

	jaegercfg "github.com/uber/jaeger-client-go/config"
)

// New registers Jaeger as the OpenTracing implementation.
// If jaegerAgentHost is an empty string, tracing is disabled.
// Values are retrieved from environment variables, and are
//  configurable per Cortex component
func New(serviceName string) io.Closer {
	jaegerAgentHost := os.Getenv("JAEGER_AGENT_HOST")
	jaegerSamplerType := os.Getenv("JAEGER_SAMPLER_TYPE")
	jaegerSamplerParam, _ := strconv.ParseFloat(os.Getenv("JAEGER_SAMPLER_PARAM"), 64)

	if jaegerAgentHost != "" {
		if jaegerSamplerType == "" || jaegerSamplerParam == 0 {
			jaegerSamplerType = "ratelimiting"
			jaegerSamplerParam = 10.0
		}
		cfg := jaegercfg.Configuration{
			Sampler: &jaegercfg.SamplerConfig{
				SamplingServerURL: fmt.Sprintf("http://%s:5778/sampling", jaegerAgentHost),
				Type:              jaegerSamplerType,
				Param:             jaegerSamplerParam,
			},
			Reporter: &jaegercfg.ReporterConfig{
				LocalAgentHostPort: fmt.Sprintf("%s:6831", jaegerAgentHost),
			},
		}

		closer, err := cfg.InitGlobalTracer(serviceName)
		if err != nil {
			fmt.Printf("Could not initialize jaeger tracer: %s\n", err.Error())
			os.Exit(1)
		}
		return closer
	}
	return ioutil.NopCloser(nil)
}

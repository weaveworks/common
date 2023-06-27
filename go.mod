module github.com/weaveworks/common

go 1.14

require (
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5
	github.com/aws/aws-sdk-go v1.27.0
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/felixge/httpsnoop v1.0.3
	github.com/go-kit/log v0.2.1
	github.com/gogo/googleapis v1.1.0
	github.com/gogo/protobuf v1.3.2
	github.com/gogo/status v1.0.3
	github.com/golang/protobuf v1.5.3
	github.com/gorilla/mux v1.7.3
	github.com/mattn/go-colorable v0.0.9 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b
	github.com/opentracing-contrib/go-grpc v0.0.0-20180928155321-4b5a12d3ff02
	github.com/opentracing-contrib/go-stdlib v0.0.0-20190519235532-cf7a6c988dc9
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_golang v1.15.1
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/exporter-toolkit v0.8.2
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/sercand/kuberesolver/v4 v4.0.0
	github.com/sirupsen/logrus v1.6.0
	github.com/soheilhy/cmux v0.1.5
	github.com/stretchr/testify v1.8.3
	github.com/uber/jaeger-client-go v2.28.0+incompatible
	github.com/uber/jaeger-lib v2.2.0+incompatible
	github.com/weaveworks/promrus v1.2.0
	go.opentelemetry.io/contrib/samplers/jaegerremote v0.9.0
	go.opentelemetry.io/otel v1.15.0
	go.opentelemetry.io/otel/bridge/opentracing v1.15.0
	go.opentelemetry.io/otel/exporters/jaeger v1.15.0
	go.opentelemetry.io/otel/sdk v1.15.0
	go.opentelemetry.io/otel/trace v1.15.0
	go.uber.org/atomic v1.5.1 // indirect
	golang.org/x/net v0.8.0
	golang.org/x/sys v0.8.0 // indirect
	golang.org/x/tools v0.7.0
	google.golang.org/grpc v1.55.0
	gopkg.in/yaml.v2 v2.4.0
)

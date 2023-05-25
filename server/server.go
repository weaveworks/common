package server

import (
	"crypto/tls"
	"flag"
	"fmt"
	"math"
	"net"
	"net/http"
	_ "net/http/pprof" // anonymous import to get the pprof handler registered
	"strings"
	"time"

	"github.com/gorilla/mux"
	otgrpc "github.com/opentracing-contrib/go-grpc"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/soheilhy/cmux"
	"golang.org/x/net/context"
	"golang.org/x/net/netutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"github.com/weaveworks/common/httpgrpc"
	httpgrpc_server "github.com/weaveworks/common/httpgrpc/server"
	"github.com/weaveworks/common/instrument"
	"github.com/weaveworks/common/logging"
	"github.com/weaveworks/common/middleware"
	"github.com/weaveworks/common/signals"
)

// Listen on the named network
const (
	// DefaultNetwork  the host resolves to multiple IP addresses,
	// Dial will try each IP address in order until one succeeds
	DefaultNetwork = "tcp"
	// NetworkTCPV4 for IPV4 only
	NetworkTCPV4 = "tcp4"
)

// SignalHandler used by Server.
type SignalHandler interface {
	// Starts the signals handler. This method is blocking, and returns only after signal is received,
	// or "Stop" is called.
	Loop()

	// Stop blocked "Loop" method.
	Stop()
}

// TLSConfig contains TLS parameters for Config.
type TLSConfig struct {
	TLSCertPath string `yaml:"cert_file"`
	TLSKeyPath  string `yaml:"key_file"`
	ClientAuth  string `yaml:"client_auth_type"`
	ClientCAs   string `yaml:"client_ca_file"`
}

// Config for a Server
type Config struct {
	MetricsNamespace string `yaml:"-"`
	// Set to > 1 to add native histograms to requestDuration.
	// See documentation for NativeHistogramBucketFactor in
	// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#HistogramOpts
	// for details. A generally useful value is 1.1.
	MetricsNativeHistogramFactor float64 `yaml:"-"`

	HTTPListenNetwork string `yaml:"http_listen_network"`
	HTTPListenAddress string `yaml:"http_listen_address"`
	HTTPListenPort    int    `yaml:"http_listen_port"`
	HTTPConnLimit     int    `yaml:"http_listen_conn_limit"`
	GRPCListenNetwork string `yaml:"grpc_listen_network"`
	GRPCListenAddress string `yaml:"grpc_listen_address"`
	GRPCListenPort    int    `yaml:"grpc_listen_port"`
	GRPCConnLimit     int    `yaml:"grpc_listen_conn_limit"`

	CipherSuites  string    `yaml:"tls_cipher_suites"`
	MinVersion    string    `yaml:"tls_min_version"`
	HTTPTLSConfig TLSConfig `yaml:"http_tls_config"`
	GRPCTLSConfig TLSConfig `yaml:"grpc_tls_config"`

	RegisterInstrumentation  bool `yaml:"register_instrumentation"`
	ExcludeRequestInLog      bool `yaml:"-"`
	DisableRequestSuccessLog bool `yaml:"-"`

	ServerGracefulShutdownTimeout time.Duration `yaml:"graceful_shutdown_timeout"`
	HTTPServerReadTimeout         time.Duration `yaml:"http_server_read_timeout"`
	HTTPServerWriteTimeout        time.Duration `yaml:"http_server_write_timeout"`
	HTTPServerIdleTimeout         time.Duration `yaml:"http_server_idle_timeout"`

	GRPCOptions                   []grpc.ServerOption            `yaml:"-"`
	GRPCMiddleware                []grpc.UnaryServerInterceptor  `yaml:"-"`
	GRPCStreamMiddleware          []grpc.StreamServerInterceptor `yaml:"-"`
	HTTPMiddleware                []middleware.Interface         `yaml:"-"`
	Router                        *mux.Router                    `yaml:"-"`
	DoNotAddDefaultHTTPMiddleware bool                           `yaml:"-"`
	RouteHTTPToGRPC               bool                           `yaml:"-"`

	GPRCServerMaxRecvMsgSize           int           `yaml:"grpc_server_max_recv_msg_size"`
	GRPCServerMaxSendMsgSize           int           `yaml:"grpc_server_max_send_msg_size"`
	GPRCServerMaxConcurrentStreams     uint          `yaml:"grpc_server_max_concurrent_streams"`
	GRPCServerMaxConnectionIdle        time.Duration `yaml:"grpc_server_max_connection_idle"`
	GRPCServerMaxConnectionAge         time.Duration `yaml:"grpc_server_max_connection_age"`
	GRPCServerMaxConnectionAgeGrace    time.Duration `yaml:"grpc_server_max_connection_age_grace"`
	GRPCServerTime                     time.Duration `yaml:"grpc_server_keepalive_time"`
	GRPCServerTimeout                  time.Duration `yaml:"grpc_server_keepalive_timeout"`
	GRPCServerMinTimeBetweenPings      time.Duration `yaml:"grpc_server_min_time_between_pings"`
	GRPCServerPingWithoutStreamAllowed bool          `yaml:"grpc_server_ping_without_stream_allowed"`

	LogFormat                    logging.Format    `yaml:"log_format"`
	LogLevel                     logging.Level     `yaml:"log_level"`
	Log                          logging.Interface `yaml:"-"`
	LogSourceIPs                 bool              `yaml:"log_source_ips_enabled"`
	LogSourceIPsHeader           string            `yaml:"log_source_ips_header"`
	LogSourceIPsRegex            string            `yaml:"log_source_ips_regex"`
	LogRequestHeaders            bool              `yaml:"log_request_headers"`
	LogRequestAtInfoLevel        bool              `yaml:"log_request_at_info_level_enabled"`
	LogRequestExcludeHeadersList string            `yaml:"log_request_exclude_headers_list"`

	// If not set, default signal handler is used.
	SignalHandler SignalHandler `yaml:"-"`

	// If not set, default Prometheus registry is used.
	Registerer prometheus.Registerer `yaml:"-"`
	Gatherer   prometheus.Gatherer   `yaml:"-"`

	PathPrefix string `yaml:"http_path_prefix"`
}

var infinty = time.Duration(math.MaxInt64)

// RegisterFlags adds the flags required to config this to the given FlagSet
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	f.StringVar(&cfg.HTTPListenAddress, "server.http-listen-address", "", "HTTP server listen address.")
	f.StringVar(&cfg.HTTPListenNetwork, "server.http-listen-network", DefaultNetwork, "HTTP server listen network, default tcp")
	f.StringVar(&cfg.CipherSuites, "server.tls-cipher-suites", "", "Comma-separated list of cipher suites to use. If blank, the default Go cipher suites is used.")
	f.StringVar(&cfg.MinVersion, "server.tls-min-version", "", "Minimum TLS version to use. Allowed values: VersionTLS10, VersionTLS11, VersionTLS12, VersionTLS13. If blank, the Go TLS minimum version is used.")
	f.StringVar(&cfg.HTTPTLSConfig.TLSCertPath, "server.http-tls-cert-path", "", "HTTP server cert path.")
	f.StringVar(&cfg.HTTPTLSConfig.TLSKeyPath, "server.http-tls-key-path", "", "HTTP server key path.")
	f.StringVar(&cfg.HTTPTLSConfig.ClientAuth, "server.http-tls-client-auth", "", "HTTP TLS Client Auth type.")
	f.StringVar(&cfg.HTTPTLSConfig.ClientCAs, "server.http-tls-ca-path", "", "HTTP TLS Client CA path.")
	f.StringVar(&cfg.GRPCTLSConfig.TLSCertPath, "server.grpc-tls-cert-path", "", "GRPC TLS server cert path.")
	f.StringVar(&cfg.GRPCTLSConfig.TLSKeyPath, "server.grpc-tls-key-path", "", "GRPC TLS server key path.")
	f.StringVar(&cfg.GRPCTLSConfig.ClientAuth, "server.grpc-tls-client-auth", "", "GRPC TLS Client Auth type.")
	f.StringVar(&cfg.GRPCTLSConfig.ClientCAs, "server.grpc-tls-ca-path", "", "GRPC TLS Client CA path.")
	f.IntVar(&cfg.HTTPListenPort, "server.http-listen-port", 80, "HTTP server listen port. When set to -1, the HTTP listener is disabled.")
	f.IntVar(&cfg.HTTPConnLimit, "server.http-conn-limit", 0, "Maximum number of simultaneous http connections, <=0 to disable")
	f.StringVar(&cfg.GRPCListenNetwork, "server.grpc-listen-network", DefaultNetwork, "gRPC server listen network")
	f.StringVar(&cfg.GRPCListenAddress, "server.grpc-listen-address", "", "gRPC server listen address.")
	f.IntVar(&cfg.GRPCListenPort, "server.grpc-listen-port", 9095, "gRPC server listen port. When set to -1, the gRPC listener is disabled.")
	f.IntVar(&cfg.GRPCConnLimit, "server.grpc-conn-limit", 0, "Maximum number of simultaneous grpc connections, <=0 to disable")
	f.BoolVar(&cfg.RegisterInstrumentation, "server.register-instrumentation", true, "Register the intrumentation handlers (/metrics etc).")
	f.DurationVar(&cfg.ServerGracefulShutdownTimeout, "server.graceful-shutdown-timeout", 30*time.Second, "Timeout for graceful shutdowns")
	f.DurationVar(&cfg.HTTPServerReadTimeout, "server.http-read-timeout", 30*time.Second, "Read timeout for HTTP server")
	f.DurationVar(&cfg.HTTPServerWriteTimeout, "server.http-write-timeout", 30*time.Second, "Write timeout for HTTP server")
	f.DurationVar(&cfg.HTTPServerIdleTimeout, "server.http-idle-timeout", 120*time.Second, "Idle timeout for HTTP server")
	f.IntVar(&cfg.GPRCServerMaxRecvMsgSize, "server.grpc-max-recv-msg-size-bytes", 4*1024*1024, "Limit on the size of a gRPC message this server can receive (bytes).")
	f.IntVar(&cfg.GRPCServerMaxSendMsgSize, "server.grpc-max-send-msg-size-bytes", 4*1024*1024, "Limit on the size of a gRPC message this server can send (bytes).")
	f.UintVar(&cfg.GPRCServerMaxConcurrentStreams, "server.grpc-max-concurrent-streams", 100, "Limit on the number of concurrent streams for gRPC calls (0 = unlimited)")
	f.DurationVar(&cfg.GRPCServerMaxConnectionIdle, "server.grpc.keepalive.max-connection-idle", infinty, "The duration after which an idle connection should be closed. Default: infinity")
	f.DurationVar(&cfg.GRPCServerMaxConnectionAge, "server.grpc.keepalive.max-connection-age", infinty, "The duration for the maximum amount of time a connection may exist before it will be closed. Default: infinity")
	f.DurationVar(&cfg.GRPCServerMaxConnectionAgeGrace, "server.grpc.keepalive.max-connection-age-grace", infinty, "An additive period after max-connection-age after which the connection will be forcibly closed. Default: infinity")
	f.DurationVar(&cfg.GRPCServerTime, "server.grpc.keepalive.time", time.Hour*2, "Duration after which a keepalive probe is sent in case of no activity over the connection., Default: 2h")
	f.DurationVar(&cfg.GRPCServerTimeout, "server.grpc.keepalive.timeout", time.Second*20, "After having pinged for keepalive check, the duration after which an idle connection should be closed, Default: 20s")
	f.DurationVar(&cfg.GRPCServerMinTimeBetweenPings, "server.grpc.keepalive.min-time-between-pings", 5*time.Minute, "Minimum amount of time a client should wait before sending a keepalive ping. If client sends keepalive ping more often, server will send GOAWAY and close the connection.")
	f.BoolVar(&cfg.GRPCServerPingWithoutStreamAllowed, "server.grpc.keepalive.ping-without-stream-allowed", false, "If true, server allows keepalive pings even when there are no active streams(RPCs). If false, and client sends ping when there are no active streams, server will send GOAWAY and close the connection.")
	f.StringVar(&cfg.PathPrefix, "server.path-prefix", "", "Base path to serve all API routes from (e.g. /v1/)")
	cfg.LogFormat.RegisterFlags(f)
	cfg.LogLevel.RegisterFlags(f)
	f.BoolVar(&cfg.LogSourceIPs, "server.log-source-ips-enabled", false, "Optionally log the source IPs.")
	f.StringVar(&cfg.LogSourceIPsHeader, "server.log-source-ips-header", "", "Header field storing the source IPs. Only used if server.log-source-ips-enabled is true. If not set the default Forwarded, X-Real-IP and X-Forwarded-For headers are used")
	f.StringVar(&cfg.LogSourceIPsRegex, "server.log-source-ips-regex", "", "Regex for matching the source IPs. Only used if server.log-source-ips-enabled is true. If not set the default Forwarded, X-Real-IP and X-Forwarded-For headers are used")
	f.BoolVar(&cfg.LogRequestHeaders, "server.log-request-headers", false, "Optionally log request headers.")
	f.StringVar(&cfg.LogRequestExcludeHeadersList, "server.log-request-headers-exclude-list", "", "Comma separated list of headers to exclude from loggin. Only used if server.log-request-headers is true.")
	f.BoolVar(&cfg.LogRequestAtInfoLevel, "server.log-request-at-info-level-enabled", false, "Optionally log requests at info level instead of debug level. Applies to request headers as well if server.log-request-headers is enabled.")
}

func (cfg *Config) GRPCEnabled() bool { return cfg.GRPCListenPort != -1 }

func (cfg *Config) HTTPEnabled() bool { return cfg.HTTPListenPort != -1 }

// Server wraps an HTTP and gRPC server, and some common initialization.
//
// Servers will be automatically instrumented for Prometheus metrics.
type Server struct {
	cfg          Config
	handler      SignalHandler
	grpcListener net.Listener
	httpListener net.Listener

	// These fields are used to support grpc over the http server
	//  if RouteHTTPToGRPC is set. the fields are kept here
	//  so they can be initialized in New() and started in Run()
	grpchttpmux        cmux.CMux
	grpcOnHTTPListener net.Listener
	GRPCOnHTTPServer   *grpc.Server

	HTTP       *mux.Router
	HTTPServer *http.Server
	GRPC       *grpc.Server
	Log        logging.Interface
	Registerer prometheus.Registerer
	Gatherer   prometheus.Gatherer
}

// New makes a new Server
func New(cfg Config) (*Server, error) {
	if !cfg.GRPCEnabled() && !cfg.HTTPEnabled() {
		return nil, fmt.Errorf("both gRPC and HTTP ports are disabled, at least one must be enabled")
	}
	if cfg.RouteHTTPToGRPC && (!cfg.HTTPEnabled() || !cfg.GRPCEnabled()) {
		return nil, fmt.Errorf("both gRPC and HTTP ports must be enabled to route HTTP to gRPC")
	}

	// If user doesn't supply a logging implementation, by default instantiate
	// logrus.
	log := cfg.Log
	if log == nil {
		log = logging.NewLogrus(cfg.LogLevel)
	}

	// If user doesn't supply a registerer/gatherer, use Prometheus' by default.
	reg := cfg.Registerer
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	gatherer := cfg.Gatherer
	if gatherer == nil {
		gatherer = prometheus.DefaultGatherer
	}

	tcpConnections := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: cfg.MetricsNamespace,
		Name:      "tcp_connections",
		Help:      "Current number of accepted TCP connections.",
	}, []string{"protocol"})
	reg.MustRegister(tcpConnections)

	tcpConnectionsLimit := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: cfg.MetricsNamespace,
		Name:      "tcp_connections_limit",
		Help:      "The max number of TCP connections that can be accepted (0 means no limit).",
	}, []string{"protocol"})
	reg.MustRegister(tcpConnectionsLimit)

	// Prometheus histograms for requests.
	requestDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace:                       cfg.MetricsNamespace,
		Name:                            "request_duration_seconds",
		Help:                            "Time (in seconds) spent serving HTTP requests.",
		Buckets:                         instrument.DefBuckets,
		NativeHistogramBucketFactor:     cfg.MetricsNativeHistogramFactor,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: time.Hour,
	}, []string{"method", "route", "status_code", "ws"})
	reg.MustRegister(requestDuration)

	receivedMessageSize := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: cfg.MetricsNamespace,
		Name:      "request_message_bytes",
		Help:      "Size (in bytes) of messages received in the request.",
		Buckets:   middleware.BodySizeBuckets,
	}, []string{"method", "route"})
	reg.MustRegister(receivedMessageSize)

	sentMessageSize := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: cfg.MetricsNamespace,
		Name:      "response_message_bytes",
		Help:      "Size (in bytes) of messages sent in response.",
		Buckets:   middleware.BodySizeBuckets,
	}, []string{"method", "route"})
	reg.MustRegister(sentMessageSize)

	inflightRequests := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: cfg.MetricsNamespace,
		Name:      "inflight_requests",
		Help:      "Current number of inflight requests.",
	}, []string{"method", "route"})
	reg.MustRegister(inflightRequests)

	cipherSuites, err := stringToCipherSuites(cfg.CipherSuites)
	if err != nil {
		return nil, err
	}
	minVersion, err := stringToTLSVersion(cfg.MinVersion)
	if err != nil {
		return nil, err
	}

	server := &Server{
		cfg:        cfg,
		Log:        log,
		Registerer: reg,
		Gatherer:   gatherer,
	}

	if cfg.HTTPEnabled() {
		err := server.setupHTTPServer(
			cipherSuites,
			minVersion,
			tcpConnections,
			tcpConnectionsLimit,
			requestDuration,
			receivedMessageSize,
			sentMessageSize,
			inflightRequests,
		)
		if err != nil {
			return nil, err
		}
	}

	if cfg.GRPCEnabled() {
		err := server.setupGRPCServer(
			cipherSuites,
			minVersion,
			tcpConnections,
			tcpConnectionsLimit,
			requestDuration,
			receivedMessageSize,
			sentMessageSize,
			inflightRequests,
		)
		if err != nil {
			return nil, err
		}
	}

	log.WithField("http", prettyPrintListener(server.httpListener)).
		WithField("grpc", prettyPrintListener(server.grpcListener)).
		Infof("server listening on addresses")

	server.handler = cfg.SignalHandler
	if server.handler == nil {
		server.handler = signals.NewHandler(log)
	}

	return server, nil
}

// RegisterInstrumentation on the given router.
func RegisterInstrumentation(router *mux.Router) {
	RegisterInstrumentationWithGatherer(router, prometheus.DefaultGatherer)
}

// RegisterInstrumentationWithGatherer on the given router.
func RegisterInstrumentationWithGatherer(router *mux.Router, gatherer prometheus.Gatherer) {
	router.Handle("/metrics", promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))
	router.PathPrefix("/debug/pprof").Handler(http.DefaultServeMux)
}

// Run the server; blocks until SIGTERM (if signal handling is enabled), an error is received, or Stop() is called.
func (s *Server) Run() error {
	errChan := make(chan error, 1)
	grpcEnabled, httpEnabled := s.GRPC != nil, s.HTTPServer != nil

	// Wait for a signal
	go func() {
		s.handler.Loop()
		select {
		case errChan <- nil:
		default:
		}
	}()

	if httpEnabled {
		go func() {
			var err error
			if s.HTTPServer.TLSConfig == nil {
				err = s.HTTPServer.Serve(s.httpListener)
			} else {
				err = s.HTTPServer.ServeTLS(s.httpListener, s.cfg.HTTPTLSConfig.TLSCertPath, s.cfg.HTTPTLSConfig.TLSKeyPath)
			}
			if err == http.ErrServerClosed {
				err = nil
			}

			select {
			case errChan <- err:
			default:
			}
		}()
	}

	if grpcEnabled && httpEnabled {
		// Setup gRPC server for HTTP over gRPC, ensure we don't double-count the middleware
		httpgrpc.RegisterHTTPServer(s.GRPC, httpgrpc_server.NewServer(s.HTTP))
	}

	if grpcEnabled {
		go func() {
			err := s.GRPC.Serve(s.grpcListener)
			handleGRPCError(err, errChan)
		}()
	}

	// grpchttpmux will only be set if grpchttpmux RouteHTTPToGRPC is set and both HTTP and GRPC are enabled.
	if s.grpchttpmux != nil {
		go func() {
			err := s.grpchttpmux.Serve()
			handleGRPCError(err, errChan)
		}()
		go func() {
			err := s.GRPCOnHTTPServer.Serve(s.grpcOnHTTPListener)
			handleGRPCError(err, errChan)
		}()
	}

	return <-errChan
}

// handleGRPCError consolidates GRPC Server error handling by sending
// any error to errChan except for grpc.ErrServerStopped which is ignored.
func handleGRPCError(err error, errChan chan error) {
	if err == grpc.ErrServerStopped {
		err = nil
	}

	select {
	case errChan <- err:
	default:
	}
}

// HTTPListenAddr exposes `net.Addr` that `Server` is listening to for HTTP connections.
func (s *Server) HTTPListenAddr() net.Addr {
	if s.httpListener == nil {
		return nil
	}
	return s.httpListener.Addr()
}

// GRPCListenAddr exposes `net.Addr` that `Server` is listening to for GRPC connections.
func (s *Server) GRPCListenAddr() net.Addr {
	if s.grpcListener == nil {
		return nil
	}
	return s.grpcListener.Addr()
}

// Stop unblocks Run().
func (s *Server) Stop() {
	s.handler.Stop()
}

// Shutdown the server, gracefully.  Should be defered after New().
func (s *Server) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ServerGracefulShutdownTimeout)
	defer cancel() // releases resources if httpServer.Shutdown completes before timeout elapses

	if s.HTTPServer != nil {
		_ = s.HTTPServer.Shutdown(ctx)
	}
	if s.GRPC != nil {
		s.GRPC.GracefulStop()
	}
}

func (s *Server) setupHTTPServer(
	cipherSuites []web.Cipher,
	minVersion web.TLSVersion,
	tcpConnections *prometheus.GaugeVec,
	tcpConnectionsLimit *prometheus.GaugeVec,
	requestDuration *prometheus.HistogramVec,
	receivedMessageSize *prometheus.HistogramVec,
	sentMessageSize *prometheus.HistogramVec,
	inflightRequests *prometheus.GaugeVec,
) error {
	network := s.cfg.HTTPListenNetwork
	if network == "" {
		network = DefaultNetwork
	}
	// Setup listeners first, so we can fail early if the port is in use.
	httpListener, err := net.Listen(network, fmt.Sprintf("%s:%d", s.cfg.HTTPListenAddress, s.cfg.HTTPListenPort))
	if err != nil {
		return err
	}
	httpListener = middleware.CountingListener(httpListener, tcpConnections.WithLabelValues("http"))

	tcpConnectionsLimit.WithLabelValues("http").Set(float64(s.cfg.HTTPConnLimit))
	if s.cfg.HTTPConnLimit > 0 {
		httpListener = netutil.LimitListener(httpListener, s.cfg.HTTPConnLimit)
	}

	var grpcOnHTTPListener net.Listener
	var grpchttpmux cmux.CMux
	if s.cfg.RouteHTTPToGRPC {
		grpchttpmux = cmux.New(httpListener)

		httpListener = grpchttpmux.Match(cmux.HTTP1Fast())
		grpcOnHTTPListener = grpchttpmux.Match(cmux.HTTP2())
	}

	// Setup TLS if configured.
	httpTLSConfig, err := getHTTPTLSConfig(s.cfg, cipherSuites, minVersion)
	if err != nil {
		return err
	}

	// Setup HTTP server
	var router *mux.Router
	if s.cfg.Router != nil {
		router = s.cfg.Router
	} else {
		router = mux.NewRouter()
	}
	if s.cfg.PathPrefix != "" {
		// Expect metrics and pprof handlers to be prefixed with server's path prefix.
		// e.g. /loki/metrics or /loki/debug/pprof
		router = router.PathPrefix(s.cfg.PathPrefix).Subrouter()
	}
	if s.cfg.RegisterInstrumentation {
		RegisterInstrumentationWithGatherer(router, s.Gatherer)
	}

	var sourceIPs *middleware.SourceIPExtractor
	if s.cfg.LogSourceIPs {
		sourceIPs, err = middleware.NewSourceIPs(s.cfg.LogSourceIPsHeader, s.cfg.LogSourceIPsRegex)
		if err != nil {
			return fmt.Errorf("error setting up source IP extraction: %v", err)
		}
	}

	defaultLogMiddleware := middleware.NewLogMiddleware(
		s.Log,
		s.cfg.LogRequestHeaders,
		s.cfg.LogRequestAtInfoLevel,
		sourceIPs,
		strings.Split(s.cfg.LogRequestExcludeHeadersList, ","),
	)
	defaultLogMiddleware.DisableRequestSuccessLog = s.cfg.DisableRequestSuccessLog

	defaultHTTPMiddleware := []middleware.Interface{
		middleware.Tracer{
			RouteMatcher: router,
			SourceIPs:    sourceIPs,
		},
		defaultLogMiddleware,
		middleware.Instrument{
			RouteMatcher:     router,
			Duration:         requestDuration,
			RequestBodySize:  receivedMessageSize,
			ResponseBodySize: sentMessageSize,
			InflightRequests: inflightRequests,
		},
	}
	var httpMiddleware []middleware.Interface
	if s.cfg.DoNotAddDefaultHTTPMiddleware {
		httpMiddleware = s.cfg.HTTPMiddleware
	} else {
		httpMiddleware = append(defaultHTTPMiddleware, s.cfg.HTTPMiddleware...)
	}

	httpServer := &http.Server{
		ReadTimeout:  s.cfg.HTTPServerReadTimeout,
		WriteTimeout: s.cfg.HTTPServerWriteTimeout,
		IdleTimeout:  s.cfg.HTTPServerIdleTimeout,
		Handler:      middleware.Merge(httpMiddleware...).Wrap(router),
	}
	if httpTLSConfig != nil {
		httpServer.TLSConfig = httpTLSConfig
	}

	s.httpListener = httpListener
	s.grpcOnHTTPListener = grpcOnHTTPListener
	s.grpchttpmux = grpchttpmux
	s.HTTP = router
	s.HTTPServer = httpServer
	return nil
}

func (s *Server) setupGRPCServer(
	cipherSuites []web.Cipher,
	minVersion web.TLSVersion,
	tcpConnections *prometheus.GaugeVec,
	tcpConnectionsLimit *prometheus.GaugeVec,
	requestDuration *prometheus.HistogramVec,
	receivedMessageSize *prometheus.HistogramVec,
	sentMessageSize *prometheus.HistogramVec,
	inflightRequests *prometheus.GaugeVec,
) error {
	network := s.cfg.GRPCListenNetwork
	if network == "" {
		network = DefaultNetwork
	}

	grpcListener, err := net.Listen(network, fmt.Sprintf("%s:%d", s.cfg.GRPCListenAddress, s.cfg.GRPCListenPort))
	if err != nil {
		return err
	}
	grpcListener = middleware.CountingListener(grpcListener, tcpConnections.WithLabelValues("grpc"))

	tcpConnectionsLimit.WithLabelValues("grpc").Set(float64(s.cfg.GRPCConnLimit))
	if s.cfg.GRPCConnLimit > 0 {
		grpcListener = netutil.LimitListener(grpcListener, s.cfg.GRPCConnLimit)
	}

	// Setup TLS if configured.
	grpcTLSConfig, err := getGRPCTLSConfig(s.cfg, cipherSuites, minVersion)
	if err != nil {
		return err
	}

	// Setup gRPC server
	serverLog := middleware.GRPCServerLog{
		Log:                      s.Log,
		WithRequest:              !s.cfg.ExcludeRequestInLog,
		DisableRequestSuccessLog: s.cfg.DisableRequestSuccessLog,
	}
	grpcMiddleware := []grpc.UnaryServerInterceptor{
		serverLog.UnaryServerInterceptor,
		otgrpc.OpenTracingServerInterceptor(opentracing.GlobalTracer()),
		middleware.UnaryServerInstrumentInterceptor(requestDuration),
	}
	grpcMiddleware = append(grpcMiddleware, s.cfg.GRPCMiddleware...)

	grpcStreamMiddleware := []grpc.StreamServerInterceptor{
		serverLog.StreamServerInterceptor,
		otgrpc.OpenTracingStreamServerInterceptor(opentracing.GlobalTracer()),
		middleware.StreamServerInstrumentInterceptor(requestDuration),
	}
	grpcStreamMiddleware = append(grpcStreamMiddleware, s.cfg.GRPCStreamMiddleware...)

	grpcKeepAliveOptions := keepalive.ServerParameters{
		MaxConnectionIdle:     s.cfg.GRPCServerMaxConnectionIdle,
		MaxConnectionAge:      s.cfg.GRPCServerMaxConnectionAge,
		MaxConnectionAgeGrace: s.cfg.GRPCServerMaxConnectionAgeGrace,
		Time:                  s.cfg.GRPCServerTime,
		Timeout:               s.cfg.GRPCServerTimeout,
	}

	grpcKeepAliveEnforcementPolicy := keepalive.EnforcementPolicy{
		MinTime:             s.cfg.GRPCServerMinTimeBetweenPings,
		PermitWithoutStream: s.cfg.GRPCServerPingWithoutStreamAllowed,
	}

	grpcOptions := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(grpcMiddleware...),
		grpc.ChainStreamInterceptor(grpcStreamMiddleware...),
		grpc.KeepaliveParams(grpcKeepAliveOptions),
		grpc.KeepaliveEnforcementPolicy(grpcKeepAliveEnforcementPolicy),
		grpc.MaxRecvMsgSize(s.cfg.GPRCServerMaxRecvMsgSize),
		grpc.MaxSendMsgSize(s.cfg.GRPCServerMaxSendMsgSize),
		grpc.MaxConcurrentStreams(uint32(s.cfg.GPRCServerMaxConcurrentStreams)),
		grpc.StatsHandler(middleware.NewStatsHandler(receivedMessageSize, sentMessageSize, inflightRequests)),
	}
	grpcOptions = append(grpcOptions, s.cfg.GRPCOptions...)
	if grpcTLSConfig != nil {
		grpcCreds := credentials.NewTLS(grpcTLSConfig)
		grpcOptions = append(grpcOptions, grpc.Creds(grpcCreds))
	}
	grpcServer := grpc.NewServer(grpcOptions...)
	grpcOnHttpServer := grpc.NewServer(grpcOptions...)

	s.grpcListener = grpcListener
	s.GRPC = grpcServer
	s.GRPCOnHTTPServer = grpcOnHttpServer
	return nil
}

func getHTTPTLSConfig(cfg Config, cipherSuites []web.Cipher, minVersion web.TLSVersion) (*tls.Config, error) {
	var (
		tlsConfig *tls.Config
		err       error
	)
	if len(cfg.HTTPTLSConfig.TLSCertPath) > 0 && len(cfg.HTTPTLSConfig.TLSKeyPath) > 0 {
		// Note: ConfigToTLSConfig from prometheus/exporter-toolkit is awaiting security review.
		tlsConfig, err = web.ConfigToTLSConfig(&web.TLSConfig{
			TLSCertPath:  cfg.HTTPTLSConfig.TLSCertPath,
			TLSKeyPath:   cfg.HTTPTLSConfig.TLSKeyPath,
			ClientAuth:   cfg.HTTPTLSConfig.ClientAuth,
			ClientCAs:    cfg.HTTPTLSConfig.ClientCAs,
			CipherSuites: cipherSuites,
			MinVersion:   minVersion,
		})
		if err != nil {
			return nil, fmt.Errorf("error generating http tls config: %v", err)
		}
	}
	return tlsConfig, nil
}

func getGRPCTLSConfig(cfg Config, cipherSuites []web.Cipher, minVersion web.TLSVersion) (*tls.Config, error) {
	var (
		tlsConfig *tls.Config
		err       error
	)
	if len(cfg.GRPCTLSConfig.TLSCertPath) > 0 && len(cfg.GRPCTLSConfig.TLSKeyPath) > 0 {
		// Note: ConfigToTLSConfig from prometheus/exporter-toolkit is awaiting security review.
		tlsConfig, err = web.ConfigToTLSConfig(&web.TLSConfig{
			TLSCertPath:  cfg.GRPCTLSConfig.TLSCertPath,
			TLSKeyPath:   cfg.GRPCTLSConfig.TLSKeyPath,
			ClientAuth:   cfg.GRPCTLSConfig.ClientAuth,
			ClientCAs:    cfg.GRPCTLSConfig.ClientCAs,
			CipherSuites: cipherSuites,
			MinVersion:   minVersion,
		})
		if err != nil {
			return nil, fmt.Errorf("error generating grpc tls config: %v", err)
		}
	}
	return tlsConfig, nil
}

func prettyPrintListener(l net.Listener) string {
	if l == nil {
		return "disabled"
	}
	return l.Addr().String()
}

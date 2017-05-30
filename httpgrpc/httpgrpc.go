package httpgrpc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/mwitkow/go-grpc-middleware"
	"github.com/opentracing/opentracing-go"
	"github.com/sercand/kuberesolver"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/weaveworks/common/httpgrpc/types"
	"github.com/weaveworks/common/middleware"
)

// Server implements HTTPServer.  HTTPServer is a generated interface that gRPC
// servers must implement.
type Server struct {
	handler http.Handler
}

// NewServer makes a new Server.
func NewServer(handler http.Handler) *Server {
	return &Server{
		handler: handler,
	}
}

// Handle implements HTTPServer.
func (s Server) Handle(ctx context.Context, r *types.HTTPRequest) (*types.HTTPResponse, error) {
	req, err := http.NewRequest(r.Method, r.Url, ioutil.NopCloser(bytes.NewReader(r.Body)))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	toHeader(r.Headers, req.Header)
	req.RequestURI = r.Url
	recorder := httptest.NewRecorder()
	s.handler.ServeHTTP(recorder, req)
	resp := &types.HTTPResponse{
		Code:    int32(recorder.Code),
		Headers: fromHeader(recorder.Header()),
		Body:    recorder.Body.Bytes(),
	}
	if recorder.Code/100 == 5 {
		return nil, types.ErrorFromHTTPResponse(resp)
	}
	return resp, err
}

// Client is a http.Handler that forwards the request over gRPC.
type Client struct {
	mtx       sync.RWMutex
	service   string
	namespace string
	port      string
	client    types.HTTPClient
	conn      *grpc.ClientConn
}

// ParseURL deals with direct:// style URLs, as well as kubernetes:// urls.
// For backwards compatibility it treats URLs without schems as kubernetes://.
func ParseURL(unparsed string) (string, []grpc.DialOption, error) {
	parsed, err := url.Parse(unparsed)
	if err != nil {
		return "", nil, err
	}

	switch parsed.Scheme {
	case "direct":
		return parsed.Host, nil, err

	case "kubernetes", "":
		host, port, err := net.SplitHostPort(parsed.Host)
		if err != nil {
			return "", nil, err
		}
		parts := strings.SplitN(host, ".", 2)
		service, namespace := parts[0], "default"
		if len(parts) == 2 {
			namespace = parts[1]
		}
		balancer := kuberesolver.NewWithNamespace(namespace)
		address := fmt.Sprintf("kubernetes://%s:%s", service, port)
		dialOptions := []grpc.DialOption{balancer.DialOption()}
		return address, dialOptions, nil

	default:
		return "", nil, fmt.Errorf("unrecognised scheme: %s", parsed.Scheme)
	}
}

// NewClient makes a new Client, given a kubernetes service address.
func NewClient(address string) (*Client, error) {
	address, dialOptions, err := ParseURL(address)
	if err != nil {
		return nil, err
	}

	dialOptions = append(
		dialOptions,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			otgrpc.OpenTracingClientInterceptor(opentracing.GlobalTracer()),
			middleware.ClientUserHeaderInterceptor,
		)),
	)

	conn, err := grpc.Dial(address, dialOptions...)
	if err != nil {
		return nil, err
	}

	return &Client{
		client: types.NewHTTPClient(conn),
		conn:   conn,
	}, nil
}

// ServeHTTP implements http.Handler
func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req := &types.HTTPRequest{
		Method:  r.Method,
		Url:     r.RequestURI,
		Body:    body,
		Headers: fromHeader(r.Header),
	}

	resp, err := c.client.Handle(r.Context(), req)
	if err != nil {
		// Some errors will actually contain a valid resp, just need to unpack it
		var ok bool
		resp, ok = types.HTTPResponseFromError(err)

		if !ok {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	toHeader(resp.Headers, w.Header())
	w.WriteHeader(int(resp.Code))
	if _, err := w.Write(resp.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func toHeader(hs []*types.Header, header http.Header) {
	for _, h := range hs {
		header[h.Key] = h.Values
	}
}

func fromHeader(hs http.Header) []*types.Header {
	result := make([]*types.Header, 0, len(hs))
	for k, vs := range hs {
		result = append(result, &types.Header{
			Key:    k,
			Values: vs,
		})
	}
	return result
}

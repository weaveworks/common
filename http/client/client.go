package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/weaveworks/common/instrument"
)

// Requester executes an HTTP request.
type Requester interface {
	Do(req *http.Request) (*http.Response, error)
}

// TimedClient instruments a request. It implements Requester.
type TimedClient struct {
	client    Requester
	collector instrument.Collector
}

// CtxTimedOperationNameKey specifies the operation name location within the context
// for instrumentation.
const CtxTimedOperationNameKey = "op"

// NewTimedClient creates a Requester that instruments requests on `client`.
func NewTimedClient(client Requester, collector instrument.Collector) Requester {
	return &TimedClient{
		client:    client,
		collector: collector,
	}
}

// Do executes the request.
func (c TimedClient) Do(r *http.Request) (*http.Response, error) {
	operation := r.Context().Value(CtxTimedOperationNameKey).(string)
	if operation == "" {
		operation = r.URL.Path
	}
	return TimeRequest(r.Context(), operation, c.collector, c.client, r)
}

// TimeRequest performs an HTTP client request and records the duration in a histogram.
func TimeRequest(ctx context.Context, operation string, coll instrument.Collector, client Requester, request *http.Request) (*http.Response, error) {
	var response *http.Response
	doRequest := func(_ context.Context) error {
		var err error
		response, err = client.Do(request)
		return err
	}
	toStatusCode := func(err error) string {
		if err == nil {
			return strconv.Itoa(response.StatusCode)
		}
		return "error"
	}
	err := instrument.CollectedRequest(ctx, fmt.Sprintf("%s %s", request.Method, operation),
		coll, toStatusCode, doRequest)
	return response, err
}

// TimeRequestHistogram performs an HTTP client request and records the duration in a histogram.
// Deprecated: try to use TimeRequest() to avoid creation of a collector on every request
func TimeRequestHistogram(ctx context.Context, operation string, metric *prometheus.HistogramVec, client Requester, request *http.Request) (*http.Response, error) {
	coll := instrument.NewHistogramCollector(metric)
	return TimeRequest(ctx, operation, coll, client, request)
}

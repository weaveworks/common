package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/weaveworks/common/instrument"
	oldcontext "golang.org/x/net/context"
)

// TimeRequestHistogram performs an HTTP client request and records the duration in a histogram
func TimeRequestHistogram(ctx context.Context, operation string, metric *prometheus.HistogramVec, client *http.Client, request *http.Request) (*http.Response, error) {
	var response *http.Response
	doRequest := func(_ oldcontext.Context) error {
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
	err := instrument.TimeRequestHistogramStatus(ctx, fmt.Sprintf("%s %s", request.Method, operation), metric, toStatusCode, doRequest)
	return response, err
}

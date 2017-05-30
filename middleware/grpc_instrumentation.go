package middleware

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// ServerInstrumentInterceptor instruments gRPC requests for errors and latency.
func ServerInstrumentInterceptor(hist *prometheus.HistogramVec) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		begin := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(begin).Seconds()
		respStatus := "success"
		if err != nil {
			errInfo, ok := status.FromError(err)
			if ok {
				respStatus = strconv.Itoa(int(errInfo.Code()))
			} else {
				respStatus = "error"
			}
		}
		hist.WithLabelValues(gRPC, info.FullMethod, respStatus, "false").Observe(duration)
		return resp, err
	}
}

// ErrorToStatus handler to convert error objects to http-response errors
type ErrorToStatus func(error) (code int32, message string, err error)

// ServerErrorToStatusInterceptor converts error objects to http-response-like error objects
func ServerErrorToStatusInterceptor(converter ErrorToStatus) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			code, message, convertError := converter(err)
			if convertError == nil {
				err = status.ErrorProto(&spb.Status{
					Code:    code,
					Message: message,
				})
			}
		}
		return resp, err
	}
}

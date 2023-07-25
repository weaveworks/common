package middleware

import (
	"errors"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	grpcUtils "github.com/weaveworks/common/grpc"
	"github.com/weaveworks/common/logging"
	"github.com/weaveworks/common/user"
)

const (
	gRPC     = "gRPC"
	errorKey = "err"
)

// If an error implements Observe(), it will get called and GRPCServerLog will do nothing.
type Observer interface {
	Observe(ctx context.Context, _ logging.Interface, method string, duration time.Duration)
}

// GRPCServerLog logs grpc requests, errors, and latency.
type GRPCServerLog struct {
	Log logging.Interface
	// WithRequest will log the entire request rather than just the error
	WithRequest              bool
	DisableRequestSuccessLog bool
}

// UnaryServerInterceptor returns an interceptor that logs gRPC requests
func (s GRPCServerLog) UnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	begin := time.Now()
	resp, err := handler(ctx, req)
	if err == nil && s.DisableRequestSuccessLog {
		return resp, nil
	}
	var observer Observer
	if errors.As(err, &observer) {
		observer.Observe(ctx, s.Log, info.FullMethod, time.Since(begin))
		return resp, err
	}

	entry := user.LogWith(ctx, s.Log).WithFields(logging.Fields{"method": info.FullMethod, "duration": time.Since(begin)})
	if err != nil {
		if s.WithRequest {
			entry = entry.WithField("request", req)
		}
		if grpcUtils.IsCanceled(err) {
			entry.WithField(errorKey, err).Debugln(gRPC)
		} else {
			entry.WithField(errorKey, err).Warnln(gRPC)
		}
	} else {
		entry.Debugf("%s (success)", gRPC)
	}
	return resp, err
}

// StreamServerInterceptor returns an interceptor that logs gRPC requests
func (s GRPCServerLog) StreamServerInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	begin := time.Now()
	err := handler(srv, ss)
	if err == nil && s.DisableRequestSuccessLog {
		return nil
	}

	entry := user.LogWith(ss.Context(), s.Log).WithFields(logging.Fields{"method": info.FullMethod, "duration": time.Since(begin)})
	if err != nil {
		if grpcUtils.IsCanceled(err) {
			entry.WithField(errorKey, err).Debugln(gRPC)
		} else {
			entry.WithField(errorKey, err).Warnln(gRPC)
		}
	} else {
		entry.Debugf("%s (success)", gRPC)
	}
	return err
}

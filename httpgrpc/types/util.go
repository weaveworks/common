package types

import (
	"fmt"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/prometheus/common/log"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/status"
)

// ErrorFromHTTPResponse converts an HTTP response into a grpc error
func ErrorFromHTTPResponse(resp *HTTPResponse) error {
	a, err := ptypes.MarshalAny(resp)
	if err != nil {
		return err
	}

	return status.ErrorProto(&spb.Status{
		Code:    resp.Code,
		Message: string(resp.Body),
		Details: []*any.Any{a},
	})
}

// HTTPResponseFromError converts a grpc error into an HTTP response
func HTTPResponseFromError(err error) (*HTTPResponse, bool) {
	s, ok := status.FromError(err)
	if !ok {
		fmt.Println("not status")
		return nil, false
	}

	status := s.Proto()
	if len(status.Details) != 1 {
		return nil, false
	}

	var resp HTTPResponse
	if err := ptypes.UnmarshalAny(status.Details[0], &resp); err != nil {
		log.Errorf("Got error containing non-response: %v", err)
		return nil, false
	}

	return &resp, true
}

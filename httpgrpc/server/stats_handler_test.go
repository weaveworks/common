package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/stats"

	"github.com/weaveworks/common/httpgrpc"
)

func TestStatsHandler_PutBodyBuffer(t *testing.T) {
	const bodyCapacity = 3200

	var putRespBody []byte
	sh := statsHandler{
		putFn: func(b []byte) {
			putRespBody = b
		},
	}

	sh.HandleRPC(context.Background(), &stats.OutPayload{
		Payload: &httpgrpc.HTTPResponse{
			Body: make([]byte, 0, bodyCapacity),
		},
	})

	require.NotNil(t, putRespBody)
	require.Equal(t, bodyCapacity, cap(putRespBody))
}

func TestStatsHandler_DoNotPutLargeBodyBuffer(t *testing.T) {
	var putRespBody []byte
	sh := statsHandler{
		putFn: func(b []byte) {
			putRespBody = b
		},
	}

	sh.HandleRPC(context.Background(), &stats.OutPayload{
		Payload: &httpgrpc.HTTPResponse{
			Body: make([]byte, 0, maxInPoolBufferCapacity+1),
		},
	})

	require.Nil(t, putRespBody)
}

func TestStatsHandler_ForwardNext(t *testing.T) {
	next := &mockStatsHandler{}

	st := NewStatsHandler(next)

	st.HandleRPC(context.Background(), &stats.OutPayload{})
	st.TagRPC(context.Background(), &stats.RPCTagInfo{})
	st.TagConn(context.Background(), &stats.ConnTagInfo{})
	st.HandleConn(context.Background(), &stats.ConnBegin{})

	require.True(t, next.handleRPCInvoked)
	require.True(t, next.tagRPCInvoked)
	require.True(t, next.tagConnInvoked)
	require.True(t, next.handleConnInvoked)
}

type mockStatsHandler struct {
	handleRPCInvoked  bool
	tagRPCInvoked     bool
	tagConnInvoked    bool
	handleConnInvoked bool
}

func (m *mockStatsHandler) HandleRPC(_ context.Context, _ stats.RPCStats) {
	m.handleRPCInvoked = true
}

func (m *mockStatsHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	m.tagRPCInvoked = true
	return ctx
}

func (m *mockStatsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	m.tagConnInvoked = true
	return ctx
}

func (m *mockStatsHandler) HandleConn(_ context.Context, _ stats.ConnStats) {
	m.handleConnInvoked = true
}

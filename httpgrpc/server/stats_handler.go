package server

import (
	"bytes"
	"context"

	"google.golang.org/grpc/stats"

	"github.com/weaveworks/common/httpgrpc"
)

const (
	maxInPoolBufferCapacity = 1024 * 1024
)

type statsHandler struct {
	next  stats.Handler
	putFn func([]byte)
}

// NewStatsHandler creates a new stats.Handler that's specific to httpgrpc server.
//
// Rather than processing stats, the real purpose of this handler is to act as an interceptor that returns
// httpgrpc.HTTPResponse body buffers to the pool for future reuse, once they've been serialized and sent over the wire.
//
// The handler is also a pass-through for other stats.Handler.
func NewStatsHandler(next stats.Handler) stats.Handler {
	return statsHandler{
		next:  next,
		putFn: putBodyBuffer,
	}
}

func (sh statsHandler) HandleRPC(ctx context.Context, st stats.RPCStats) {
	if sh.next != nil {
		sh.next.HandleRPC(ctx, st)
	}
	outStats, ok := st.(*stats.OutPayload)
	if !ok {
		return
	}
	resp, ok := outStats.Payload.(*httpgrpc.HTTPResponse)
	if !ok {
		return
	}
	// At this point, response object has already been written to the wire,
	// so it's safe to return its buffer back to the pool.
	if cap(resp.Body) > maxInPoolBufferCapacity {
		return
	}
	b := resp.Body[:0]
	resp.Body = nil
	sh.putFn(b)
}

func (sh statsHandler) TagRPC(ctx context.Context, st *stats.RPCTagInfo) context.Context {
	if sh.next == nil {
		return ctx
	}
	return sh.next.TagRPC(ctx, st)
}

func (sh statsHandler) TagConn(ctx context.Context, st *stats.ConnTagInfo) context.Context {
	if sh.next == nil {
		return ctx
	}
	return sh.next.TagConn(ctx, st)
}

func (sh statsHandler) HandleConn(ctx context.Context, st stats.ConnStats) {
	if sh.next == nil {
		return
	}
	sh.next.HandleConn(ctx, st)
}

func putBodyBuffer(b []byte) {
	respBufferPool.Put(bytes.NewBuffer(b))
}

package middleware

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
)

const (
	maxResponseBodyInLogs = 4096 // At most 4k bytes from response bodies in our logs.
)

// badResponseLoggingWriter writes the body of "bad" responses (i.e. 5xx
// responses) to a buffer.
type badResponseLoggingWriter struct {
	rw            http.ResponseWriter
	buffer        bytes.Buffer
	logBody       bool
	bodyBytesLeft int
	statusCode    int
}

// newBadResponseLoggingWriter makes a new badResponseLoggingWriter.
func newBadResponseLoggingWriter(rw http.ResponseWriter) *badResponseLoggingWriter {
	return &badResponseLoggingWriter{
		rw:            rw,
		logBody:       false,
		bodyBytesLeft: maxResponseBodyInLogs,
	}
}

// Header returns the header map that will be sent by WriteHeader.
// Implements ResponseWriter.
func (b *badResponseLoggingWriter) Header() http.Header {
	return b.rw.Header()
}

// Write writes HTTP response data.
func (b *badResponseLoggingWriter) Write(data []byte) (int, error) {
	n, err := b.rw.Write(data)
	if b.logBody {
		b.captureResponseBody(data)
	}
	return n, err
}

// WriteHeader writes the HTTP response header.
func (b *badResponseLoggingWriter) WriteHeader(statusCode int) {
	b.statusCode = statusCode
	if statusCode >= 500 {
		b.logBody = true
	}
	b.rw.WriteHeader(statusCode)
}

// Hijack hijacks the first response writer that is a Hijacker.
func (b *badResponseLoggingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := b.rw.(http.Hijacker)
	if ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("badResponseLoggingWriter: can't cast underlying response writer to Hijacker")
}

func (b *badResponseLoggingWriter) dumpResponseBody() []byte {
	return b.buffer.Bytes()
}

func (b *badResponseLoggingWriter) captureResponseBody(data []byte) {
	if len(data) > b.bodyBytesLeft {
		b.buffer.Write(data[:b.bodyBytesLeft])
		b.buffer.WriteString("...")
		b.bodyBytesLeft = 0
		b.logBody = false
	} else {
		b.buffer.Write(data)
		b.bodyBytesLeft -= len(data)
	}
}

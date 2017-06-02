package middleware

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
)

const (
	maxResponseBodyInLogs = 4096 // At most 4k bytes from response bodies in our logs.
)

// MultiResponseWriter writes a single response to multiple http.ResponseWriters,
// similar to Unix's `tee` command.
type MultiResponseWriter struct {
	rws     []http.ResponseWriter
	header  http.Header
	written bool
}

// NewMultiResponseWriter makes a new response writer.
func NewMultiResponseWriter(rws ...http.ResponseWriter) *MultiResponseWriter {
	return &MultiResponseWriter{
		rws:     rws,
		header:  http.Header{},
		written: false,
	}
}

// Header returns the header map that will be sent by WriteHeader.
// Implements ResponseWriter.
//
// Note that updates to the headers will not be reflected in child
// ResponseWriters until 'Write' or 'WriteHeaders' are called.
func (m *MultiResponseWriter) Header() http.Header {
	return m.header
}

func (m *MultiResponseWriter) mirrorHeaders() {
	for _, rw := range m.rws {
		theirs := rw.Header()
		for key := range theirs {
			theirs.Del(key)
		}
		for key, value := range m.header {
			theirs[key] = value
		}
	}
}

func (m *MultiResponseWriter) Write(data []byte) (int, error) {
	// The contract of 'Write' has it set headers and call WriteHeader under
	// certain circmustances. We ignore that here, as we assume that all the
	// underlying implementations do that for us.
	m.mirrorHeaders()
	for _, rw := range m.rws {
		n, err := rw.Write(data)
		if err != nil {
			return n, err
		}
		if n < len(data) {
			return n, io.ErrShortWrite
		}
	}
	return len(data), nil
}

// WriteHeader writes the HTTP response header.
func (m *MultiResponseWriter) WriteHeader(statusCode int) {
	m.mirrorHeaders()
	for _, rw := range m.rws {
		rw.WriteHeader(statusCode)
	}
}

// Hijack hijacks the first response writer that is a Hijacker.
func (m *MultiResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	for _, rw := range m.rws {
		hj, ok := rw.(http.Hijacker)
		if ok {
			return hj.Hijack()
		}
	}
	return nil, nil, fmt.Errorf("MultiResponseWriter: can't cast any responses to Hijacker")
}

// badResponseLogger is an http.ResponseWriter that logs response headers and
// some of the body when the response is erroneous (i.e. a 5xx).
//
// Using this means holding an extra copy of the request and response headers
// in memory.
type badResponseLogger struct {
	recorder      *httptest.ResponseRecorder
	statusCode    int
	logBody       bool
	bodyBytesLeft int
}

// newBadResponseLogger makes a new badResponseLogger.
func newBadResponseLogger() *badResponseLogger {
	return &badResponseLogger{
		recorder:      httptest.NewRecorder(),
		logBody:       false,
		bodyBytesLeft: maxResponseBodyInLogs,
		statusCode:    http.StatusOK,
	}
}

func (b *badResponseLogger) dumpResponse() ([]byte, error) {
	return httputil.DumpResponse(b.recorder.Result(), true)
}

// Header implements http.ResponseWriter.
func (b *badResponseLogger) Header() http.Header {
	return b.recorder.Header()
}

// WriteHeader implements http.ResponseWriter. It will immediately log the
// response headers if `statusCode` is a 5XX.
func (b *badResponseLogger) WriteHeader(statusCode int) {
	b.statusCode = statusCode
	b.recorder.WriteHeader(statusCode)
	if 100 <= statusCode && statusCode < 500 {
		return
	}
	b.logBody = true
}

// Write implements http.ResponseWriter. It will log the body up to
// `maxResponseBodyInLogs` if the response is a 5XX.
func (b *badResponseLogger) Write(data []byte) (int, error) {
	// If we haven't written the headers yet, then Write is supposed to call
	// WriteHeader with http.StatusOK. Since we don't want to do anything in
	// that case (this struct is for *bad* responses only), we don't need to
	// track whether or not we've called WriteHeader.
	if !b.logBody {
		// We don't need to log anything, so lie and say that we wrote
		// everything, making this effectively a no-op.
		return len(data), nil
	}

	if len(data) > b.bodyBytesLeft {
		b.recorder.Write(data[:b.bodyBytesLeft])
		b.recorder.WriteString("…")
		b.bodyBytesLeft = 0
	} else {
		b.recorder.Write(data)
		b.bodyBytesLeft -= len(data)
	}
	// As far as any caller is concerned, we've written the whole response.
	// There has been no error, nor a short write. Everything is fine.
	return len(data), nil
}

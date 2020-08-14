package middleware

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/weaveworks/common/logging"
	"github.com/weaveworks/common/user"
)

// Log middleware logs http requests
type Log struct {
	Log                logging.Interface
	HighVolumeErrorLog logging.Interface
	LogRequestHeaders  bool // LogRequestHeaders true -> dump http headers at debug log level
}

// logWithRequest information from the request and context as fields.
func (l Log) logWithRequest(r *http.Request) logging.Interface {
	return l.logWithRequestAndLog(r, l.Log)
}

func (l Log) logWithRequestAndLog(r *http.Request, logger logging.Interface) logging.Interface {
	traceID, ok := ExtractTraceID(r.Context())
	if ok {
		logger = logger.WithField("traceID", traceID)
	}

	return user.LogWith(r.Context(), logger)
}

// Wrap implements Middleware
func (l Log) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		begin := time.Now()
		uri := r.RequestURI // capture the URI before running next, as it may get rewritten
		// Log headers before running 'next' in case other interceptors change the data.
		headers, err := dumpRequest(r)
		if err != nil {
			headers = nil
			l.logWithRequest(r).Errorf("Could not dump request headers: %v", err)
		}
		var buf bytes.Buffer
		wrapped := newBadResponseLoggingWriter(w, &buf)
		next.ServeHTTP(wrapped, r)

		statusCode, writeErr := wrapped.statusCode, wrapped.writeError

		if writeErr != nil {
			if errors.Is(writeErr, context.Canceled) {
				l.logWithRequest(r).Debugf("%s %s %s, request cancelled: %s ws: %v; %s", r.Method, uri, time.Since(begin), writeErr, IsWSHandshakeRequest(r), headers)
			} else {
				l.logWithRequest(r).Warnf("%s %s %s, error: %s ws: %v; %s", r.Method, uri, time.Since(begin), writeErr, IsWSHandshakeRequest(r), headers)
			}

			return
		}
		if 100 <= statusCode && statusCode < 500 {
			l.logWithRequest(r).Debugf("%s %s (%d) %s", r.Method, uri, statusCode, time.Since(begin))
			if l.LogRequestHeaders && headers != nil {
				l.logWithRequest(r).Debugf("ws: %v; %s", IsWSHandshakeRequest(r), string(headers))
			}
		} else if statusCode == http.StatusBadGateway || statusCode == http.StatusServiceUnavailable {
			l.logHighVolumeError(r, "%s %s (%d) %s", r.Method, uri, statusCode, time.Since(begin))
			if l.LogRequestHeaders && headers != nil {
				l.logHighVolumeError(r, "ws: %v; %s", IsWSHandshakeRequest(r), string(headers))
			}
		} else {
			l.logWithRequest(r).Warnf("%s %s (%d) %s Response: %q ws: %v; %s",
				r.Method, uri, statusCode, time.Since(begin), buf.Bytes(), IsWSHandshakeRequest(r), headers)
		}
	})
}

func (l Log) logHighVolumeError(r *http.Request, format string, args ...interface{}) {
	if l.HighVolumeErrorLog != nil {
		l.logWithRequestAndLog(r, l.HighVolumeErrorLog).Warnf(format, args...)
	} else {
		l.logWithRequest(r).Debugf(format, args...)
	}
}

// Logging middleware logs each HTTP request method, path, response code and
// duration for all HTTP requests.
var Logging = Log{
	Log: logging.Global(),
}

func dumpRequest(req *http.Request) ([]byte, error) {
	var b bytes.Buffer

	// Exclude some headers for security, or just that we don't need them when debugging
	err := req.Header.WriteSubset(&b, map[string]bool{
		"Cookie":        true,
		"X-Csrf-Token":  true,
		"Authorization": true,
	})
	if err != nil {
		return nil, err
	}

	ret := bytes.Replace(b.Bytes(), []byte("\r\n"), []byte("; "), -1)
	return ret, nil
}

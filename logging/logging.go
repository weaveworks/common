package logging

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"golang.org/x/net/context"

	log "github.com/sirupsen/logrus"
	"github.com/weaveworks/common/user"
	"github.com/weaveworks/promrus"
)

// Setup configures logging output to stderr, sets the log level and sets the formatter.
func Setup(logLevel string) error {
	log.SetOutput(os.Stderr)
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		return fmt.Errorf("Error parsing log level: %v", err)
	}
	log.SetLevel(level)
	log.SetFormatter(&textFormatter{})
	hook, err := promrus.NewPrometheusHook() // Expose number of log messages as Prometheus metrics.
	if err != nil {
		return err
	}
	log.AddHook(hook)
	return nil
}

type textFormatter struct{}

// Based off logrus.TextFormatter, which behaves completely
// differently when you don't want colored output
func (f *textFormatter) Format(entry *log.Entry) ([]byte, error) {
	b := &bytes.Buffer{}

	levelText := strings.ToUpper(entry.Level.String())[0:4]
	timeStamp := entry.Time.Format("2006/01/02 15:04:05.000000")
	fmt.Fprintf(b, "%s: %s %s", levelText, timeStamp, entry.Message)
	for k, v := range entry.Data {
		fmt.Fprintf(b, " %s=%v", k, v)
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

// With returns a log entry with common Weaveworks logging information.
//
// e.g.
//     logger := logging.With(ctx)
//     logger.Errorf("Some error")
func With(ctx context.Context) *log.Entry {
	return log.WithFields(user.LogFields(ctx))
}

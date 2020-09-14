package logging

import (
	"testing"

	"gotest.tools/assert"
)

type counterLogger struct {
	count int
}

func (c *counterLogger) Debugf(format string, args ...interface{}) { c.count++ }
func (c *counterLogger) Debugln(args ...interface{})               { c.count++ }
func (c *counterLogger) Infof(format string, args ...interface{})  { c.count++ }
func (c *counterLogger) Infoln(args ...interface{})                { c.count++ }
func (c *counterLogger) Warnf(format string, args ...interface{})  { c.count++ }
func (c *counterLogger) Warnln(args ...interface{})                { c.count++ }
func (c *counterLogger) Errorf(format string, args ...interface{}) { c.count++ }
func (c *counterLogger) Errorln(args ...interface{})               { c.count++ }
func (c *counterLogger) WithField(key string, value interface{}) Interface {
	return c
}
func (c *counterLogger) WithFields(Fields) Interface {
	return c
}

func TestRateLimitedLoggerLogs(t *testing.T) {
	c := &counterLogger{}
	r := NewRateLimitedLogger(c, 1)

	r.Errorln("asdf")
	assert.Equal(t, 1, c.count)
}

func TestRateLimitedLoggerLimits(t *testing.T) {
	c := &counterLogger{}
	r := NewRateLimitedLogger(c, 1)

	r.Errorln("asdf")
	r.Infoln("asdf")
	assert.Equal(t, 1, c.count)
}

func TestRateLimitedLoggerWith(t *testing.T) {
	c := &counterLogger{}
	r := NewRateLimitedLogger(c, 1)
	r2 := r.WithField("key", "value")

	r2.Errorln("asdf")
	r2.Warnln("asdf")
	assert.Equal(t, 1, c.count)
}

package logging

// Copy-pasted from prometheus/common/promlog.
// Copyright 2017 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import (
	"flag"

	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Level is a settable identifier for the minimum level a log entry
// must be have.
type Level struct {
	s      string
	logrus logrus.Level
	gokit  level.Option
}

// RegisterFlags adds the log level flag to the provided flagset.
func (l *Level) RegisterFlags(f *flag.FlagSet) {
	l.Set("info")
	f.Var(l, "log.level", "Only log messages with the given severity or above. Valid levels: [debug, info, warn, error]")
}

func (l *Level) String() string {
	return l.s
}

// Set updates the value of the allowed level.
func (l *Level) Set(s string) error {
	switch s {
	case "debug":
		l.logrus = logrus.DebugLevel
		l.gokit = level.AllowDebug()
	case "info":
		l.logrus = logrus.InfoLevel
		l.gokit = level.AllowInfo()
	case "warn":
		l.logrus = logrus.WarnLevel
		l.gokit = level.AllowWarn()
	case "error":
		l.logrus = logrus.ErrorLevel
		l.gokit = level.AllowError()
	default:
		return errors.Errorf("unrecognized log level %q", s)
	}

	l.s = s
	return nil
}

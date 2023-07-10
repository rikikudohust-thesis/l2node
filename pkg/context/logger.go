package context

import (
	"encoding/json"
	"fmt"

	"cloud.google.com/go/logging"
	raven "github.com/getsentry/raven-go"
	"github.com/sirupsen/logrus"
)

func (c Context) Debugln(args ...interface{}) {
	c.withID().Debugln(args...)
	c.sdLog(logging.Debug, fmt.Sprintln(args...))
}

func (c Context) Debugf(format string, args ...interface{}) {
	c.withID().Debugf(format, args...)
	c.sdLog(logging.Debug, fmt.Sprintf(format, args...))
}

func (c Context) Infoln(args ...interface{}) {
	c.withID().Infoln(args...)
	c.sdLog(logging.Info, fmt.Sprintln(args...))
}

func (c Context) Infof(format string, args ...interface{}) {
	c.withID().Infof(format, args...)
	c.sdLog(logging.Info, fmt.Sprintf(format, args...))
}

func (c Context) Warnln(args ...interface{}) {
	c.withID().Warnln(args...)
	c.sdLog(logging.Warning, fmt.Sprintln(args...))
}

func (c Context) Warnf(format string, args ...interface{}) {
	c.withID().Warnf(format, args...)
	c.sdLog(logging.Warning, fmt.Sprintf(format, args...))
}

func (c Context) Errorln(args ...interface{}) {
	c.withID().Errorln(args...)
	c.sdLog(logging.Error, fmt.Sprintln(args...))
	raven.CaptureError(fmt.Errorf("%s", fmt.Sprintln(args...)), nil)
}

func (c Context) Errorf(format string, args ...interface{}) {
	c.withID().Errorf(format, args...)
	c.sdLog(logging.Error, fmt.Sprintf(format, args...))
	raven.CaptureError(fmt.Errorf(format, args...), nil)
}

func (c Context) InfoFields(fields map[string]interface{}) {
	c.sdLogFields(logging.Info, fields)
}

func (c Context) InfoWithBasicFields(fields map[string]interface{}) {
	basicFields := c.basicFields()
	for k, v := range basicFields {
		fields[k] = v
	}
	c.sdLogFields(logging.Info, fields)
}

func (c Context) sdLogFields(level logging.Severity, fields map[string]interface{}) {
	if config.sdEventLogger == nil {
		return
	}

	fields["prefix"] = c.prefix

	config.sdEventLogger.Log(logging.Entry{
		Severity: level,
		Payload:  fields,
	})
}

func (c Context) sdLog(level logging.Severity, msg string) {
	if config.sdLogger == nil {
		return
	}

	fields := c.basicFields()
	fields["msg"] = msg

	config.sdLogger.Log(logging.Entry{
		Severity: level,
		Payload:  fields,
	})
}

func (c Context) withID() *logrus.Entry {
	field := logrus.Fields{"prefix": c.prefix}
	if c.reqIDEnabled {
		field["requestID"] = c.RequestInfo.RequestID
	}

	return config.stdoutLogger.WithFields(field)
}

func (c Context) basicFields() map[string]interface{} {
	jsonEvent, err := json.Marshal(c.RequestInfo)
	if err != nil {
		return nil
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(jsonEvent, &fields); err != nil {
		return nil
	}

	fields["prefix"] = c.prefix

	return fields
}

package utils

import (
	"fmt"
	"testing"

	log "github.com/sirupsen/logrus"
)

// Logger .
type Logger interface {
	Debug(...interface{})
	Info(...interface{})
	Warn(...interface{})
	Error(...interface{})

	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
}

// LoggerFactory .
type LoggerFactory interface {
	Logger(label string) Logger
}

type standardLogger struct{}

// NewStandardLogger .
func NewStandardLogger() Logger {
	return standardLogger{}
}

func (logger standardLogger) Debug(args ...interface{}) {
	log.Debug(args...)
}

func (logger standardLogger) Info(args ...interface{}) {
	log.Info(args...)
}

func (logger standardLogger) Error(args ...interface{}) {
	log.Error(args...)
}

func (logger standardLogger) Warn(args ...interface{}) {
	log.Warn(args...)
}

func (logger standardLogger) Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

func (logger standardLogger) Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

func (logger standardLogger) Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

func (logger standardLogger) Warnf(format string, args ...interface{}) {
	log.Warnf(format, args...)
}

type testLogger struct {
	test *testing.T
}

// NewTestLogger .
func NewTestLogger(test *testing.T) Logger {
	return testLogger{
		test: test,
	}
}

func (logger testLogger) Debug(args ...interface{}) {
	logger.test.Log(args...)
}

func (logger testLogger) Info(args ...interface{}) {
	logger.test.Log(args...)
}

func (logger testLogger) Warn(args ...interface{}) {
	logger.test.Log(args...)
}

func (logger testLogger) Error(args ...interface{}) {
	logger.test.Log(args...)
}

func (logger testLogger) Debugf(format string, args ...interface{}) {
	logger.test.Log(fmt.Sprintf(format, args...))
}

func (logger testLogger) Infof(format string, args ...interface{}) {
	logger.test.Log(fmt.Sprintf(format, args...))
}

func (logger testLogger) Warnf(format string, args ...interface{}) {
	logger.test.Log(fmt.Sprintf(format, args...))
}

func (logger testLogger) Errorf(format string, args ...interface{}) {
	logger.test.Log(fmt.Sprintf(format, args...))
}

type methodLogger struct {
	logger     Logger
	objectName string
	methodName string
}

// MethodLogger .
func MethodLogger(logger Logger, objectName string, methodName string) Logger {
	return methodLogger{
		logger:     logger,
		objectName: objectName,
		methodName: methodName,
	}
}

func (logger methodLogger) Debug(args ...interface{}) {
	logger.call(logger.logger.Debug, args...)
}

func (logger methodLogger) Info(args ...interface{}) {
	logger.call(log.Info, args...)
}

func (logger methodLogger) Error(args ...interface{}) {
	logger.call(log.Error, args...)
}

func (logger methodLogger) Warn(args ...interface{}) {
	logger.call(log.Warn, args...)
}

func (logger methodLogger) Debugf(format string, args ...interface{}) {
	logger.callf(log.Debugf, format, args...)
}

func (logger methodLogger) Infof(format string, args ...interface{}) {
	logger.callf(log.Infof, format, args...)
}

func (logger methodLogger) Errorf(format string, args ...interface{}) {
	logger.callf(log.Errorf, format, args...)
}

func (logger methodLogger) Warnf(format string, args ...interface{}) {
	logger.callf(log.Warnf, format, args...)
}

func (logger methodLogger) call(logFunc func(...interface{}), args ...interface{}) {
	logFunc("[%v::%v] %v", logger.objectName, logger.methodName, fmt.Sprint(args...))
}

func (logger methodLogger) callf(logFunc func(string, ...interface{}), format string, args ...interface{}) {
	logFunc("[%v::%v] %v", logger.objectName, logger.methodName, fmt.Sprintf(format, args...))
}

// ObjectLogger .
type ObjectLogger struct {
	Log        Logger
	ObjectName string
}

// NewObjectLogger .
func NewObjectLogger(objectName string) LoggerFactory {
	return ObjectLogger{
		Log:        NewStandardLogger(),
		ObjectName: objectName,
	}
}

// Logger .
func (obj ObjectLogger) Logger(methodName string) Logger {
	return MethodLogger(obj.Log, obj.ObjectName, methodName)
}

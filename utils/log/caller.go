package log

import (
	"context"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
)

func WithCaller() Entry {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return entryLogger{}
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return entryLogger{}
	}
	name := fn.Name()
	return callerEntryLogger{entrySupplier: func() *log.Entry {
		return log.WithField("func", name)
	}}
}

func WithSourceCode() Entry {
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		return entryLogger{}
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return entryLogger{}
	}
	name := fn.Name()
	return callerEntryLogger{entrySupplier: func() *log.Entry {
		return log.WithField("func", name).WithField("file", file).WithField("line", line)
	}}
}

type entryLogger struct{}

func (l entryLogger) WithError(err error) *log.Entry {
	return WithError(err)
}

func (l entryLogger) WithField(field string, value string) *log.Entry {
	return WithField(field, value)
}

func (l entryLogger) WithFields(fields log.Fields) *log.Entry {
	return WithFields(fields)
}

func (l entryLogger) WithContext(c context.Context) *log.Entry {
	return WithContext(c)
}

func (l entryLogger) WithTime(t time.Time) *log.Entry {
	return WithTime(t)
}

func (l entryLogger) WithCurrentTime() *log.Entry {
	return WithCurrentTime()
}

func (l entryLogger) Trace(args ...interface{}) {
	Trace(args...)
}

func (l entryLogger) Debug(args ...interface{}) {
	Debug(args...)
}

func (l entryLogger) Print(args ...interface{}) {
	Print(args...)
}

func (l entryLogger) Info(args ...interface{}) {
	Info(args...)
}

func (l entryLogger) Warn(args ...interface{}) {
	Warn(args...)
}

func (l entryLogger) Warning(args ...interface{}) {
	Warning(args...)
}

func (l entryLogger) Error(args ...interface{}) {
	Error(args...)
}

func (l entryLogger) Panic(args ...interface{}) {
	Panic(args...)
}

func (l entryLogger) Fatal(args ...interface{}) {
	Fatal(args...)
}

func (l entryLogger) Tracef(format string, args ...interface{}) {
	Tracef(format, args...)
}

func (l entryLogger) Debugf(format string, args ...interface{}) {
	Debugf(format, args...)
}

func (l entryLogger) Printf(format string, args ...interface{}) {
	Printf(format, args...)
}

func (l entryLogger) Infof(format string, args ...interface{}) {
	Infof(format, args...)
}

func (l entryLogger) Warnf(format string, args ...interface{}) {
	Warnf(format, args...)
}

func (l entryLogger) Warningf(format string, args ...interface{}) {
	Warningf(format, args...)
}

func (l entryLogger) Errorf(format string, args ...interface{}) {
	Errorf(format, args...)
}

func (l entryLogger) Panicf(format string, args ...interface{}) {
	Panicf(format, args...)
}

func (l entryLogger) Fatalf(format string, args ...interface{}) {
	Fatalf(format, args...)
}

func (l entryLogger) Traceln(args ...interface{}) {
	Traceln(args...)
}

func (l entryLogger) Debugln(args ...interface{}) {
	Debugln(args...)
}

func (l entryLogger) Println(args ...interface{}) {
	Println(args...)
}

func (l entryLogger) Infoln(args ...interface{}) {
	Infoln(args...)
}

func (l entryLogger) Warnln(args ...interface{}) {
	Warnln(args...)
}

func (l entryLogger) Warningln(args ...interface{}) {
	Warningln(args...)
}

func (l entryLogger) Errorln(args ...interface{}) {
	Errorln(args...)
}

func (l entryLogger) Panicln(args ...interface{}) {
	Panicln(args...)
}

func (l entryLogger) Fatalln(args ...interface{}) {
	Fatalln(args...)
}

type callerEntryLogger struct {
	entrySupplier func() *log.Entry
}

func (l callerEntryLogger) WithError(err error) *log.Entry {
	return l.entrySupplier().WithError(err)
}

func (l callerEntryLogger) WithField(field string, value string) *log.Entry {
	return l.entrySupplier().WithField(field, value)
}

func (l callerEntryLogger) WithFields(fields log.Fields) *log.Entry {
	return l.entrySupplier().WithFields(fields)
}

func (l callerEntryLogger) WithContext(c context.Context) *log.Entry {
	return l.entrySupplier().WithContext(c)
}

func (l callerEntryLogger) WithTime(t time.Time) *log.Entry {
	return l.entrySupplier().WithTime(t)
}

func (l callerEntryLogger) WithCurrentTime() *log.Entry {
	return l.entrySupplier().WithTime(time.Now())
}

func (l callerEntryLogger) Trace(args ...interface{}) {
	l.entrySupplier().Trace(args...)
}

func (l callerEntryLogger) Debug(args ...interface{}) {
	l.entrySupplier().Debug(args...)
}

func (l callerEntryLogger) Print(args ...interface{}) {
	l.entrySupplier().Print(args...)
}

func (l callerEntryLogger) Info(args ...interface{}) {
	l.entrySupplier().Info(args...)
}

func (l callerEntryLogger) Warn(args ...interface{}) {
	l.entrySupplier().Warn(args...)
}

func (l callerEntryLogger) Warning(args ...interface{}) {
	l.entrySupplier().Warning(args...)
}

func (l callerEntryLogger) Error(args ...interface{}) {
	l.entrySupplier().Error(args...)
}

func (l callerEntryLogger) Panic(args ...interface{}) {
	l.entrySupplier().Panic(args...)
}

func (l callerEntryLogger) Fatal(args ...interface{}) {
	l.entrySupplier().Fatal(args...)
}

func (l callerEntryLogger) Tracef(format string, args ...interface{}) {
	l.entrySupplier().Tracef(format, args...)
}

func (l callerEntryLogger) Debugf(format string, args ...interface{}) {
	l.entrySupplier().Debugf(format, args...)
}

func (l callerEntryLogger) Printf(format string, args ...interface{}) {
	l.entrySupplier().Printf(format, args...)
}

func (l callerEntryLogger) Infof(format string, args ...interface{}) {
	l.entrySupplier().Infof(format, args...)
}

func (l callerEntryLogger) Warnf(format string, args ...interface{}) {
	l.entrySupplier().Warnf(format, args...)
}

func (l callerEntryLogger) Warningf(format string, args ...interface{}) {
	l.entrySupplier().Warningf(format, args...)
}

func (l callerEntryLogger) Errorf(format string, args ...interface{}) {
	l.entrySupplier().Errorf(format, args...)
}

func (l callerEntryLogger) Panicf(format string, args ...interface{}) {
	l.entrySupplier().Panicf(format, args...)
}

func (l callerEntryLogger) Fatalf(format string, args ...interface{}) {
	l.entrySupplier().Fatalf(format, args...)
}

func (l callerEntryLogger) Traceln(args ...interface{}) {
	l.entrySupplier().Traceln(args...)
}

func (l callerEntryLogger) Debugln(args ...interface{}) {
	l.entrySupplier().Debugln(args...)
}

func (l callerEntryLogger) Println(args ...interface{}) {
	l.entrySupplier().Println(args...)
}

func (l callerEntryLogger) Infoln(args ...interface{}) {
	l.entrySupplier().Infoln(args...)
}

func (l callerEntryLogger) Warnln(args ...interface{}) {
	l.entrySupplier().Warnln(args...)
}

func (l callerEntryLogger) Warningln(args ...interface{}) {
	l.entrySupplier().Warningln(args...)
}

func (l callerEntryLogger) Errorln(args ...interface{}) {
	l.entrySupplier().Errorln(args...)
}

func (l callerEntryLogger) Panicln(args ...interface{}) {
	l.entrySupplier().Panicln(args...)
}

func (l callerEntryLogger) Fatalln(args ...interface{}) {
	l.entrySupplier().Fatalln(args...)
}

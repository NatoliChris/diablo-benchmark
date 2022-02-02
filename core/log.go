package core


import (
	"fmt"
	"io"
	"time"
)


type LogLevel uint8


const (
	LOG_SILENT LogLevel = 0
	LOG_FATAL  LogLevel = 1
	LOG_ERROR  LogLevel = 2
	LOG_WARN   LogLevel = 3
	LOG_INFO   LogLevel = 4
	LOG_DEBUG  LogLevel = 5
	LOG_TRACE  LogLevel = 6
)


type Logger interface {
	// Log a message with a printf format for different log levels.
	//
	Fatalf(string, ...interface{})
	Errorf(string, ...interface{})
	Warnf(string, ...interface{})
	Infof(string, ...interface{})
	Debugf(string, ...interface{})
	Tracef(string, ...interface{})

	// Return a new logger with the given `name` appended to this logger
	// current name.
	//
	Extend(string) Logger
}


var globalLogger Logger = &noLogger{}


func SetLogger(logger Logger) {
	globalLogger = logger
}

func Fatalf(format string, args ...interface{}) {
	globalLogger.Fatalf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	globalLogger.Errorf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	globalLogger.Warnf(format, args...)
}

func Infof(format string, args ...interface{}) {
	globalLogger.Infof(format, args...)
}

func Debugf(format string, args ...interface{}) {
	globalLogger.Debugf(format, args...)
}

func Tracef(format string, args ...interface{}) {
	globalLogger.Tracef(format, args...)
}

func ExtendLogger(name string) Logger {
	return globalLogger.Extend(name)
}


type noLogger struct {
}

func (this *noLogger) Fatalf(string, ...interface{}) {}
func (this *noLogger) Errorf(string, ...interface{}) {}
func (this *noLogger) Warnf(string, ...interface{}) {}
func (this *noLogger) Infof(string, ...interface{}) {}
func (this *noLogger) Debugf(string, ...interface{}) {}
func (this *noLogger) Tracef(string, ...interface{}) {}
func (this *noLogger) Extend(string) Logger { return this }


type printLogger struct {
	stream  io.Writer
	name    string
	level   LogLevel
}

func NewPrintLogger(stream io.Writer, name string, level LogLevel) Logger {
	return &printLogger{ stream, name, level }
}

func (this *printLogger) log(level, format string, args ...interface{}) {
	var year, month, day, hour, min, sec, ms int
	var now time.Time = time.Now()
	var str string

	year = now.Year()
	month = int(now.Month())
	day = now.Day()
	hour = now.Hour()
	min = now.Minute()
	sec = now.Second()
	ms = now.Nanosecond() / 1000000

	str = fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d.%03d " +
		"%s %s: ", year, month, day, hour, min, sec, ms, level,
		this.name)
	str += fmt.Sprintf(format, args...)
	str += fmt.Sprintf("\n")

	io.WriteString(this.stream, str)
}

func (this *printLogger) Fatalf(format string, args ...interface{}) {
	if this.level >= LOG_FATAL {
		this.log("FATAL", format, args...)
	}
}

func (this *printLogger) Errorf(format string, args ...interface{}) {
	if this.level >= LOG_ERROR {
		this.log("ERROR", format, args...)
	}
}

func (this *printLogger) Warnf(format string, args ...interface{}) {
	if this.level >= LOG_WARN {
		this.log("WARN ", format, args...)
	}
}

func (this *printLogger) Infof(format string, args ...interface{}) {
	if this.level >= LOG_INFO {
		this.log("INFO ", format, args...)
	}
}

func (this *printLogger) Debugf(format string, args ...interface{}) {
	if this.level >= LOG_DEBUG {
		this.log("DEBUG", format, args...)
	}
}

func (this *printLogger) Tracef(format string, args ...interface{}) {
	if this.level >= LOG_TRACE {
		this.log("TRACE", format, args...)
	}
}

func (this *printLogger) Extend(name string) Logger {
	var childName string

	if len(this.name) == 0 {
		childName = name
	} else {
		childName = this.name + "." + name
	}

	return NewPrintLogger(this.stream, childName, this.level)
}

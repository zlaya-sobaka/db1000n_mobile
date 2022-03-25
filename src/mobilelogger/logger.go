package mobilelogger

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2/widget"
)

// A global variable so that log functions can be directly accessed
var log Logger
var counter int = 1
var currentLog string

//Fields Type to pass when we want to call WithFields for structured logging
type Fields map[string]interface{}

type LogLevel string

const (
	// Debug has verbose message.
	Debug LogLevel = "debug"
	// Info is default log level.
	Info LogLevel = "info"
	// Error is for logging errors.
	Error LogLevel = "error"
	// Fatal is for logging fatal messages. The system shutdown after logging the message.
	Fatal LogLevel = "fatal"
)

//Logger is our contract for the logger
type Logger interface {
	Debugf(format string, args ...interface{})

	Infof(format string, args ...interface{})

	Errorf(format string, args ...interface{})

	Fatalf(format string, args ...interface{})
}

// Configuration stores the config for the logger
// For some loggers there can only be one level across writers, for such the level of Console is picked by default
type Configuration struct {
	ConsoleLevel LogLevel
}

// NewLogger returns an instance of logger
func NewLogger(textgrid *widget.TextGrid) {
	logger := newZeroLogger(textgrid)
	log = logger
}

func Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

func Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

type MobileLogLogger struct {
	textgrid *widget.TextGrid
}

func newZeroLogger(textgrid *widget.TextGrid) Logger {
	return &MobileLogLogger{
		textgrid: textgrid,
	}
}

func (l *MobileLogLogger) Debugf(format string, args ...interface{}) {
	currentLog = l.textgrid.Text()
	if len(strings.Split(currentLog, "\n")) > 100 {
		currentLog = ""
	}
	currentLog = "\n " + fmt.Sprintf(format, args...)

	l.textgrid.SetText(currentLog)
}

func (l *MobileLogLogger) Infof(format string, args ...interface{}) {
	currentLog = l.textgrid.Text()
	if len(strings.Split(currentLog, "\n")) > 100 {
		currentLog = ""
	}
	currentLog = "\n " + fmt.Sprintf(format, args...)

	l.textgrid.SetText(currentLog)
}

func (l *MobileLogLogger) Errorf(format string, args ...interface{}) {
	currentLog = l.textgrid.Text()
	if len(strings.Split(currentLog, "\n")) > 100 {
		currentLog = ""
	}
	currentLog = "\n " + fmt.Sprintf(format, args...)

	l.textgrid.SetText(currentLog)
}

func (l *MobileLogLogger) Fatalf(format string, args ...interface{}) {
	currentLog = l.textgrid.Text()
	if len(strings.Split(currentLog, "\n")) > 100 {
		currentLog = ""
	}
	currentLog = "\n " + fmt.Sprintf(format, args...)

	l.textgrid.SetText(currentLog)
}

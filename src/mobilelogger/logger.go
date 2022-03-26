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
var lastLogs []string

//Logger is our contract for the logger
type Logger interface {
	Infof(format string, args ...interface{})
}

// NewLogger returns an instance of logger
func NewLogger(textgrid *widget.Label) {
	logger := newZeroLogger(textgrid)
	log = logger
}

func Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

type MobileLogLogger struct {
	list *widget.Label
}

func newZeroLogger(list *widget.Label) Logger {
	return &MobileLogLogger{
		list: list,
	}
}

func (l *MobileLogLogger) Infof(format string, args ...interface{}) {
	newLog := fmt.Sprintf(format, args...)
	if len(lastLogs) >= 25 {
		for i := range lastLogs {
			if i != len(lastLogs)-1 {
				lastLogs[i] = lastLogs[i+1]
			} else {
				lastLogs[i] = strings.TrimSpace(newLog)
			}
		}
	} else {
		lastLogs = append(lastLogs, strings.TrimSpace(newLog))
	}
	currLog := ""
	for _, v := range lastLogs {
		currLog += fmt.Sprintf("%s \n", v)
	}

	l.list.SetText(currLog)
	l.list.Refresh()
}

package examples

import (
	"encoding/json"
	"log"

	golog "github.com/fclairamb/go-log"
)

type Logger struct {
	Verbose bool
}

func (l Logger) log(level, event string, keyvals ...interface{}) {
	k, _ := json.Marshal(keyvals)

	log.Println(level, event, string(k))
}

func (l Logger) Trace(event string, keyvals ...interface{}) {
	if l.Verbose {
		l.log("TRACE", event, keyvals)
	}
}

func (l Logger) Debug(event string, keyvals ...interface{}) {
	if l.Verbose {
		l.log("DEBUG", event, keyvals)
	}
}

func (l Logger) Info(event string, keyvals ...interface{}) {
	l.log("INFO", event, keyvals)
}

func (l Logger) Warn(event string, keyvals ...interface{}) {
	l.log("WARN", event, keyvals)
}

func (l Logger) Error(event string, keyvals ...interface{}) {
	l.log("ERROR", event, keyvals)
}

func (l Logger) With(keyvals ...interface{}) golog.Logger {
	return l
}

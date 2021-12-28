package logging

import (
	"log"

	golog "github.com/fclairamb/go-log"
)

type JSONLogger struct{}

func NewJSONLogger() *JSONLogger {
	return &JSONLogger{}
}

func (l JSONLogger) Trace(event string, keyvals ...interface{}) {
	log.Println("TRACE", event, keyvals)
}

func (l JSONLogger) Debug(event string, keyvals ...interface{}) {
	log.Println("DEBUG", event, keyvals)
}

func (l JSONLogger) Info(event string, keyvals ...interface{}) {
	log.Println("INFO", event, keyvals)
}

func (l JSONLogger) Warn(event string, keyvals ...interface{}) {
	log.Println("WARN", event, keyvals)
}

func (l JSONLogger) Error(event string, keyvals ...interface{}) {
	log.Println("ERROR", event, keyvals)
}

func (l JSONLogger) With(keyvals ...interface{}) golog.Logger {
	return l
}

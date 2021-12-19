package ftp

import (
	"log"

	golog "github.com/fclairamb/go-log"
)

type Logger struct{}

func (l Logger) Debug(event string, keyvals ...interface{}) {
	log.Println(event, keyvals)
}

func (l Logger) Info(event string, keyvals ...interface{}) {
	log.Println(event, keyvals)
}

func (l Logger) Warn(event string, keyvals ...interface{}) {
	log.Println(event, keyvals)
}

func (l Logger) Error(event string, keyvals ...interface{}) {
	log.Println(event, keyvals)
}

func (l Logger) With(keyvals ...interface{}) golog.Logger {
	return l
}

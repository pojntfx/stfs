package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	golog "github.com/fclairamb/go-log"
)

type logMessage struct {
	Time  int64       `json:"time"`
	Level string      `json:"level"`
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

func printJSON(level string, event string, keyvals interface{}) {
	line, _ := json.Marshal(&logMessage{
		Time:  time.Now().Unix(),
		Level: level,
		Event: event,
		Data:  keyvals,
	}) // We are ignoring JSON marshalling erorrs

	fmt.Fprintln(os.Stderr, string(line))
}

type JSONLogger struct{}

func NewJSONLogger() *JSONLogger {
	return &JSONLogger{}
}

func (l JSONLogger) Trace(event string, keyvals ...interface{}) {
	printJSON("TRACE", event, keyvals)
}

func (l JSONLogger) Debug(event string, keyvals ...interface{}) {
	printJSON("DEBUG", event, keyvals)
}

func (l JSONLogger) Info(event string, keyvals ...interface{}) {
	printJSON("INFO", event, keyvals)
}

func (l JSONLogger) Warn(event string, keyvals ...interface{}) {
	printJSON("WARN", event, keyvals)
}

func (l JSONLogger) Error(event string, keyvals ...interface{}) {
	printJSON("ERROR", event, keyvals)
}

func (l JSONLogger) With(keyvals ...interface{}) golog.Logger {
	return l
}

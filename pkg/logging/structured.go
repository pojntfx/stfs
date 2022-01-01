package logging

import (
	golog "github.com/fclairamb/go-log"
)

type StructuredLogger interface {
	golog.Logger

	Trace(event string, keyvals ...interface{})
}

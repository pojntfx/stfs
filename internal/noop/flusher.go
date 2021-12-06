package noop

import "io"

type Flusher interface {
	io.WriteCloser

	Flush() error
}

type NoOpFlusher struct {
	io.WriteCloser
}

func (NoOpFlusher) Flush() error { return nil }

func AddFlush(w io.WriteCloser) NoOpFlusher {
	return NoOpFlusher{w}
}

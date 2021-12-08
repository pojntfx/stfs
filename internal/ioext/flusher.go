package ioext

import "io"

type Flusher interface {
	io.WriteCloser

	Flush() error
}

type NopFlusher struct {
	io.WriteCloser
}

func (NopFlusher) Flush() error { return nil }

func AddFlush(w io.WriteCloser) NopFlusher {
	return NopFlusher{w}
}

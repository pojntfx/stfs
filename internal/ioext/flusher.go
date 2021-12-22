package ioext

import "io"

type FlusherWriter interface {
	io.WriteCloser

	Flush() error
}

type NopFlusherWriter struct {
	io.WriteCloser
}

func (NopFlusherWriter) Flush() error { return nil }

func AddFlushNop(w io.WriteCloser) NopFlusherWriter {
	return NopFlusherWriter{w}
}

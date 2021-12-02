package noop

import "io"

type NoOpCloser struct {
	io.Writer
}

func (NoOpCloser) Close() error { return nil }

func AddClose(w io.Writer) NoOpCloser {
	return NoOpCloser{w}
}

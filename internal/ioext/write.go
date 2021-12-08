package ioext

import "io"

type NopCloser struct {
	io.Writer
}

func (NopCloser) Close() error { return nil }

func AddClose(w io.Writer) NopCloser {
	return NopCloser{w}
}

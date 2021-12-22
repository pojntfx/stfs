package ioext

import "io"

type NopCloser struct {
	io.Writer
}

func (NopCloser) Close() error { return nil }

func AddCloseNopToWriter(w io.Writer) NopCloser {
	return NopCloser{w}
}

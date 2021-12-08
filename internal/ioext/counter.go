package ioext

import "io"

type CounterReader struct {
	Reader io.Reader

	BytesRead int
}

func (r *CounterReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)

	r.BytesRead += n

	return n, err
}

type CounterWriter struct {
	Writer io.Writer

	BytesRead int
}

func (w *CounterWriter) Write(p []byte) (n int, err error) {
	n, err = w.Writer.Write(p)

	w.BytesRead += n

	return n, err
}

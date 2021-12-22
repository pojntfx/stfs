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

type CounterReadCloser struct {
	Reader io.ReadCloser

	BytesRead int
}

func (r *CounterReadCloser) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)

	r.BytesRead += n

	return n, err
}

func (r *CounterReadCloser) Close() error {
	return r.Reader.Close()
}

type CounterReadSeekCloser struct {
	Reader io.ReadSeekCloser

	BytesRead int
}

func (r *CounterReadSeekCloser) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)

	r.BytesRead += n

	return n, err
}

func (r *CounterReadSeekCloser) Close() error {
	return r.Reader.Close()
}

func (r *CounterReadSeekCloser) Seek(offset int64, whence int) (int64, error) {
	return r.Reader.Seek(offset, whence)
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

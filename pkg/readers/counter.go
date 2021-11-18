package readers

import "io"

type Counter struct {
	Reader io.Reader

	BytesRead int
}

func (r *Counter) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)

	r.BytesRead += n

	return n, err
}

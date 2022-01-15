//go:build windows

package fs

type Stat struct {
	Uid uint32
	Gid uint32
}

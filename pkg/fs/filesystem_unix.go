//go:build !(windows || wasm)

package fs

import "syscall"

const (
	O_ACCMODE = syscall.O_ACCMODE
)

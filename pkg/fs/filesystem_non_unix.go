//go:build windows || wasm

package fs

const (
	O_ACCMODE = 0x3 // It is safe to hard-code this bit as the bits are not being set from the OS
)

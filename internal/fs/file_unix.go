//go:build !windows

package fs

import "syscall"

type Stat syscall.Stat_t

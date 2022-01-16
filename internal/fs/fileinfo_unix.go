//go:build linux

package fs

import "syscall"

type Stat syscall.Stat_t

func NewStat(
	uid uint32,
	gid uint32,
	mtim int64,
	atim int64,
	ctim int64,
) *Stat {
	return &Stat{
		Uid:  uid,
		Gid:  gid,
		Mtim: syscall.NsecToTimespec(mtim),
		Atim: syscall.NsecToTimespec(atim),
		Ctim: syscall.NsecToTimespec(ctim),
	}
}

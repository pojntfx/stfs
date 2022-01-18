//go:build !linux

package fs

// From the Go stdlib for linux/amd64
type Timespec struct {
	Sec  int64
	Nsec int64
}

func (ts *Timespec) Unix() (sec int64, nsec int64) {
	return int64(ts.Sec), int64(ts.Nsec)
}

func (ts *Timespec) Nano() int64 {
	return int64(ts.Sec)*1e9 + int64(ts.Nsec)
}

func NsecToTimespec(nsec int64) Timespec {
	sec := nsec / 1e9
	nsec = nsec % 1e9
	if nsec < 0 {
		nsec += 1e9
		sec--
	}

	return Timespec{Sec: sec, Nsec: nsec}
}

type Stat struct {
	Uid  uint32
	Gid  uint32
	Mtim Timespec
	Atim Timespec
	Ctim Timespec
}

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
		Mtim: NsecToTimespec(mtim),
		Atim: NsecToTimespec(atim),
		Ctim: NsecToTimespec(ctim),
	}
}

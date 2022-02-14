package fs

import (
	"archive/tar"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/pojntfx/stfs/examples"
)

var (
	now = time.Now()
)

type newFileInfoArgs struct {
	name       string
	size       int64
	mode       fs.FileMode
	modTime    time.Time
	accessTime time.Time
	changeTime time.Time
	gid        int
	uid        int
	isDir      bool
}

var newFileInfoTests = []struct {
	name string
	args newFileInfoArgs
	want *FileInfo
}{
	{
		"Can set file info attributes",
		newFileInfoArgs{
			"test.txt",
			100,
			os.ModePerm,
			now,
			now,
			now,
			1000,
			1000,
			false,
		},
		&FileInfo{
			name:       "test.txt",
			size:       100,
			mode:       os.ModePerm,
			modTime:    now,
			accessTime: now,
			changeTime: now,
			gid:        1000,
			uid:        1000,
			isDir:      false,
		},
	},
	{
		"Can set directory info attributes",
		newFileInfoArgs{
			"test",
			1024,
			os.ModePerm,
			now,
			now,
			now,
			100,
			100,
			true,
		},
		&FileInfo{
			name:       "test",
			size:       1024,
			mode:       os.ModePerm,
			modTime:    now,
			accessTime: now,
			changeTime: now,
			gid:        100,
			uid:        100,
			isDir:      true,
		},
	},
}

func TestNewFileInfo(t *testing.T) {
	jsonLogger := &examples.Logger{
		Verbose: verbose,
	}

	for _, tt := range newFileInfoTests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewFileInfo(tt.args.name, tt.args.size, tt.args.mode, tt.args.modTime, tt.args.accessTime, tt.args.changeTime, tt.args.gid, tt.args.uid, tt.args.isDir, jsonLogger)

			if got.Name() != tt.want.name {
				t.Errorf("FileInfo.Name() = %v, want %v", got.Name(), tt.want.Name())
			}

			if got.Size() != tt.want.size {
				t.Errorf("FileInfo.Size() = %v, want %v", got.Size(), tt.want.Size())
			}

			if got.Mode() != tt.want.mode {
				t.Errorf("FileInfo.Mode() = %v, want %v", got.Mode(), tt.want.Mode())
			}

			if got.ModTime() != tt.want.modTime {
				t.Errorf("FileInfo.ModTime() = %v, want %v", got.ModTime(), tt.want.modTime)
			}

			if got.IsDir() != tt.want.isDir {
				t.Errorf("FileInfo.IsDir() = %v, want %v", got.IsDir(), tt.want.isDir)
			}

			gotSys, ok := got.Sys().(*Stat)
			if !ok {
				t.Errorf("FileInfo.Sys() !ok")
			}

			if gotSys.Atim.Nano() != tt.want.accessTime.UnixNano() {
				t.Errorf("FileInfo.Atim.Nano() = %v, want %v", gotSys.Atim.Nano(), tt.want.accessTime.UnixNano())
			}

			if gotSys.Ctim.Nano() != tt.want.changeTime.UnixNano() {
				t.Errorf("FileInfo.Ctim.Nano() = %v, want %v", gotSys.Ctim.Nano(), tt.want.changeTime.UnixNano())
			}

			if gotSys.Gid != uint32(tt.want.gid) {
				t.Errorf("FileInfo.Gid = %v, want %v", gotSys.Gid, tt.want.gid)
			}

			if gotSys.Uid != uint32(tt.want.uid) {
				t.Errorf("FileInfo.Uid = %v, want %v", gotSys.Uid, tt.want.uid)
			}
		})
	}
}

type newFileInfoFromTarHeaderArgs struct {
	hdr *tar.Header
}

var newFileInfoFromTarHeaderTests = []struct {
	name string
	args newFileInfoFromTarHeaderArgs
	want *FileInfo
}{
	{
		"Can set file info attributes",
		newFileInfoFromTarHeaderArgs{
			hdr: &tar.Header{
				Typeflag:   tar.TypeReg,
				Name:       "test.txt",
				Linkname:   "",
				Size:       100,
				Mode:       int64(os.ModePerm),
				Uid:        1000,
				Gid:        1000,
				Uname:      "pojntfx",
				Gname:      "pojntfx",
				ModTime:    now,
				AccessTime: now,
				ChangeTime: now,
				Devmajor:   0,
				Devminor:   0,
				Xattrs:     map[string]string{},
				PAXRecords: map[string]string{},
				Format:     tar.FormatPAX,
			},
		},
		&FileInfo{
			name:       "test.txt",
			size:       100,
			mode:       os.ModePerm,
			modTime:    now,
			accessTime: now,
			changeTime: now,
			gid:        1000,
			uid:        1000,
			isDir:      false,
		},
	},
	{
		"Can set directory info attributes",
		newFileInfoFromTarHeaderArgs{
			hdr: &tar.Header{
				Typeflag:   tar.TypeDir,
				Name:       "test",
				Linkname:   "",
				Size:       1024,
				Mode:       int64(os.ModePerm),
				Uid:        100,
				Gid:        100,
				Uname:      "pojntfx",
				Gname:      "pojntfx",
				ModTime:    now,
				AccessTime: now,
				ChangeTime: now,
				Devmajor:   0,
				Devminor:   0,
				Xattrs:     map[string]string{},
				PAXRecords: map[string]string{},
				Format:     tar.FormatPAX,
			},
		},
		&FileInfo{
			name:       "test",
			size:       1024,
			mode:       os.ModePerm,
			modTime:    now,
			accessTime: now,
			changeTime: now,
			gid:        100,
			uid:        100,
			isDir:      true,
		},
	},
}

func TestNewFileInfoFromTarHeader(t *testing.T) {
	jsonLogger := &examples.Logger{
		Verbose: verbose,
	}

	for _, tt := range newFileInfoFromTarHeaderTests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewFileInfoFromTarHeader(tt.args.hdr, jsonLogger)

			if got.Name() != tt.want.name {
				t.Errorf("FileInfo.Name() = %v, want %v", got.Name(), tt.want.Name())
			}

			if got.Size() != tt.want.size {
				t.Errorf("FileInfo.Size() = %v, want %v", got.Size(), tt.want.Size())
			}

			if got.Mode().Perm() != tt.want.mode {
				t.Errorf("FileInfo.Mode() = %v, want %v", got.Mode().Perm(), tt.want.mode)
			}

			if got.ModTime() != tt.want.modTime {
				t.Errorf("FileInfo.ModTime() = %v, want %v", got.ModTime(), tt.want.modTime)
			}

			if got.IsDir() != tt.want.isDir {
				t.Errorf("FileInfo.IsDir() = %v, want %v", got.IsDir(), tt.want.isDir)
			}

			gotSys, ok := got.Sys().(*Stat)
			if !ok {
				t.Errorf("FileInfo.Sys() !ok")
			}

			if gotSys.Atim.Nano() != tt.want.accessTime.UnixNano() {
				t.Errorf("FileInfo.Atim.Nano() = %v, want %v", gotSys.Atim.Nano(), tt.want.accessTime.UnixNano())
			}

			if gotSys.Ctim.Nano() != tt.want.changeTime.UnixNano() {
				t.Errorf("FileInfo.Ctim.Nano() = %v, want %v", gotSys.Ctim.Nano(), tt.want.changeTime.UnixNano())
			}

			if gotSys.Gid != uint32(tt.want.gid) {
				t.Errorf("FileInfo.Gid = %v, want %v", gotSys.Gid, tt.want.gid)
			}

			if gotSys.Uid != uint32(tt.want.uid) {
				t.Errorf("FileInfo.Uid = %v, want %v", gotSys.Uid, tt.want.uid)
			}
		})
	}
}

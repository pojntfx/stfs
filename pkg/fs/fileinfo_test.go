package fs

import (
	"archive/tar"
	"io/fs"
	"os"
	"reflect"
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

type fileInfoFields struct {
	FileInfo   os.FileInfo
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

var nameTests = []struct {
	name   string
	fields fileInfoFields
	want   string
}{
	{
		"Can get name for file",
		fileInfoFields{
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
		"test.txt",
	},
	{
		"Can get name for directory",
		fileInfoFields{
			name:       "test",
			size:       100,
			mode:       os.ModePerm,
			modTime:    now,
			accessTime: now,
			changeTime: now,
			gid:        1000,
			uid:        1000,
			isDir:      true,
		},
		"test",
	},
}

func TestFileInfo_Name(t *testing.T) {
	jsonLogger := &examples.Logger{
		Verbose: verbose,
	}

	for _, tt := range nameTests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FileInfo{
				FileInfo:   tt.fields.FileInfo,
				name:       tt.fields.name,
				size:       tt.fields.size,
				mode:       tt.fields.mode,
				modTime:    tt.fields.modTime,
				accessTime: tt.fields.accessTime,
				changeTime: tt.fields.changeTime,
				gid:        tt.fields.gid,
				uid:        tt.fields.uid,
				isDir:      tt.fields.isDir,
				log:        jsonLogger,
			}
			if got := f.Name(); got != tt.want {
				t.Errorf("FileInfo.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

var sizeTests = []struct {
	name   string
	fields fileInfoFields
	want   int64
}{
	{
		"Can get size for file",
		fileInfoFields{
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
		100,
	},
	{
		"Can get size for directory",
		fileInfoFields{
			name:       "test",
			size:       100,
			mode:       os.ModePerm,
			modTime:    now,
			accessTime: now,
			changeTime: now,
			gid:        1000,
			uid:        1000,
			isDir:      true,
		},
		100,
	},
}

func TestFileInfo_Size(t *testing.T) {
	jsonLogger := &examples.Logger{
		Verbose: verbose,
	}

	for _, tt := range sizeTests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FileInfo{
				FileInfo:   tt.fields.FileInfo,
				name:       tt.fields.name,
				size:       tt.fields.size,
				mode:       tt.fields.mode,
				modTime:    tt.fields.modTime,
				accessTime: tt.fields.accessTime,
				changeTime: tt.fields.changeTime,
				gid:        tt.fields.gid,
				uid:        tt.fields.uid,
				isDir:      tt.fields.isDir,
				log:        jsonLogger,
			}
			if got := f.Size(); got != tt.want {
				t.Errorf("FileInfo.Size() = %v, want %v", got, tt.want)
			}
		})
	}
}

var modeTests = []struct {
	name   string
	fields fileInfoFields
	want   os.FileMode
}{
	{
		"Can get mode for file",
		fileInfoFields{
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
		os.ModePerm.Perm(),
	},
	{
		"Can get mode for directory",
		fileInfoFields{
			name:       "test",
			size:       100,
			mode:       os.ModePerm,
			modTime:    now,
			accessTime: now,
			changeTime: now,
			gid:        1000,
			uid:        1000,
			isDir:      true,
		},
		os.ModePerm.Perm(),
	},
}

func TestFileInfo_Mode(t *testing.T) {
	jsonLogger := &examples.Logger{
		Verbose: verbose,
	}

	for _, tt := range modeTests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FileInfo{
				FileInfo:   tt.fields.FileInfo,
				name:       tt.fields.name,
				size:       tt.fields.size,
				mode:       tt.fields.mode,
				modTime:    tt.fields.modTime,
				accessTime: tt.fields.accessTime,
				changeTime: tt.fields.changeTime,
				gid:        tt.fields.gid,
				uid:        tt.fields.uid,
				isDir:      tt.fields.isDir,
				log:        jsonLogger,
			}
			if got := f.Mode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FileInfo.Mode() = %v, want %v", got, tt.want)
			}
		})
	}
}

var modTimeTests = []struct {
	name   string
	fields fileInfoFields
	want   time.Time
}{
	{
		"Can get modTime for file",
		fileInfoFields{
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
		now,
	},
	{
		"Can get modTime for directory",
		fileInfoFields{
			name:       "test",
			size:       100,
			mode:       os.ModePerm,
			modTime:    now,
			accessTime: now,
			changeTime: now,
			gid:        1000,
			uid:        1000,
			isDir:      true,
		},
		now,
	},
}

func TestFileInfo_ModTime(t *testing.T) {
	jsonLogger := &examples.Logger{
		Verbose: verbose,
	}

	for _, tt := range modTimeTests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FileInfo{
				FileInfo:   tt.fields.FileInfo,
				name:       tt.fields.name,
				size:       tt.fields.size,
				mode:       tt.fields.mode,
				modTime:    tt.fields.modTime,
				accessTime: tt.fields.accessTime,
				changeTime: tt.fields.changeTime,
				gid:        tt.fields.gid,
				uid:        tt.fields.uid,
				isDir:      tt.fields.isDir,
				log:        jsonLogger,
			}
			if got := f.ModTime(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FileInfo.ModTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

var isDirTests = []struct {
	name   string
	fields fileInfoFields
	want   bool
}{
	{
		"Can get isDir for file",
		fileInfoFields{
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
		false,
	},
	{
		"Can get isDir for directory",
		fileInfoFields{
			name:       "test",
			size:       100,
			mode:       os.ModePerm,
			modTime:    now,
			accessTime: now,
			changeTime: now,
			gid:        1000,
			uid:        1000,
			isDir:      true,
		},
		true,
	},
}

func TestFileInfo_IsDir(t *testing.T) {
	jsonLogger := &examples.Logger{
		Verbose: verbose,
	}

	for _, tt := range isDirTests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FileInfo{
				FileInfo:   tt.fields.FileInfo,
				name:       tt.fields.name,
				size:       tt.fields.size,
				mode:       tt.fields.mode,
				modTime:    tt.fields.modTime,
				accessTime: tt.fields.accessTime,
				changeTime: tt.fields.changeTime,
				gid:        tt.fields.gid,
				uid:        tt.fields.uid,
				isDir:      tt.fields.isDir,
				log:        jsonLogger,
			}
			if got := f.IsDir(); got != tt.want {
				t.Errorf("FileInfo.IsDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

var sysTests = []struct {
	name   string
	fields fileInfoFields
	want   *FileInfo
}{
	{
		"Can set file info attributes",
		fileInfoFields{
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
		fileInfoFields{
			name:       "test",
			size:       100,
			mode:       os.ModePerm,
			modTime:    now,
			accessTime: now,
			changeTime: now,
			gid:        100,
			uid:        100,
			isDir:      true,
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

func TestFileInfo_Sys(t *testing.T) {
	jsonLogger := &examples.Logger{
		Verbose: verbose,
	}

	for _, tt := range sysTests {
		t.Run(tt.name, func(t *testing.T) {
			got := &FileInfo{
				FileInfo:   tt.fields.FileInfo,
				name:       tt.fields.name,
				size:       tt.fields.size,
				mode:       tt.fields.mode,
				modTime:    tt.fields.modTime,
				accessTime: tt.fields.accessTime,
				changeTime: tt.fields.changeTime,
				gid:        tt.fields.gid,
				uid:        tt.fields.uid,
				isDir:      tt.fields.isDir,
				log:        jsonLogger,
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

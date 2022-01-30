package fs

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
)

type deterministicReader struct {
	maxIndex int
	index    int
}

func newDeterministicReader(maxIndex int) *deterministicReader {
	return &deterministicReader{
		maxIndex: maxIndex,
	}
}

func (w *deterministicReader) Read(p []byte) (int, error) {
	if w.index > w.maxIndex {
		return -1, io.EOF
	}

	buf := make([]byte, len(p))

	for i := 0; i < len(buf); i++ {
		buf[i] = byte(w.index)
	}

	w.index++

	return copy(p, buf), nil
}

var fileNameTests = []struct {
	name      string
	open      string
	prepare   func(symFs) error
	check     func(string) error
	withCache bool
	withOsFs  bool
}{
	{
		"Can get correct file name for /test.txt",
		"/test.txt",
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(got string) error {
			want := "/test.txt"

			if got != want {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can get correct file name for /mydir/test.txt",
		"/mydir/test.txt",
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(got string) error {
			want := "/mydir/test.txt"

			if got != want {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can get correct file name for /",
		"/",
		func(f symFs) error { return nil },
		func(got string) error {
			want := ""

			if got != want {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can get correct file name for ''",
		"",
		func(f symFs) error { return nil },
		func(got string) error {
			want := ""

			if got != want {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can get correct file name for symlink to /test.txt",
		"/existingsymlink",
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(got string) error {
			want := "/existingsymlink"

			if got != want {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can get correct file name for symlink to /mydir",
		"/existingsymlink",
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/mydir", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(got string) error {
			want := "/existingsymlink"

			if got != want {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can get correct file name for symlink to /mydir/nesteddir",
		"/existingsymlink",
		func(f symFs) error {
			if err := f.MkdirAll("/mydir/nesteddir", os.ModePerm); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/mydir/nesteddir", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(got string) error {
			want := "/existingsymlink"

			if got != want {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can get correct file name for symlink to root",
		"/existingsymlink",
		func(f symFs) error {
			if err := f.SymlinkIfPossible("/", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(got string) error {
			want := "/existingsymlink"

			if got != want {
				return fmt.Errorf("invalid name, got %v, want %v", got, want)
			}

			return nil
		},
		true,
		true,
	},
}

func TestFile_Name(t *testing.T) {
	for _, tt := range fileNameTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			file, err := symFs.Open(tt.open)
			if err != nil {
				t.Errorf("%v open() error = %v", symFs.Name(), err)

				return
			}

			got := file.Name()

			if err := tt.check(got); err != nil {
				t.Errorf("%v check() error = %v", symFs.Name(), err)

				return
			}
		})
	}
}

var fileStatTests = []struct {
	name      string
	open      string
	wantErr   bool
	prepare   func(symFs) error
	check     func(os.FileInfo) error
	withCache bool
	withOsFs  bool
}{
	{
		"Can stat /",
		"/",
		false,
		func(f symFs) error { return nil },
		func(f os.FileInfo) error {
			dir, _ := path.Split(f.Name())
			if !(dir == "/" || dir == "") {
				return fmt.Errorf("invalid dir part of path %v, should be ''", dir)

			}

			if !f.IsDir() {
				return fmt.Errorf("%v is not a directory", dir)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can stat /test.txt",
		"/test.txt",
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo) error {
			wantName := "test.txt"
			gotName := f.Name()

			if wantName != gotName {
				return fmt.Errorf("invalid name, got %v, want %v", gotName, wantName)
			}

			wantPerm := fmt.Sprintf("%o", 0666)
			gotPerm := fmt.Sprintf("%o", f.Mode())

			if wantPerm != gotPerm {
				return fmt.Errorf("invalid perm, got %v, want %v", gotPerm, wantPerm)
			}

			return nil
		},
		true,
		false, // HACK: OsFs uses umask, which yields unexpected permission bits (see https://github.com/golang/go/issues/38282)
	},
	{
		"Can stat system-specific properties of /test.txt",
		"/test.txt",
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if err := f.Chtimes("/test.txt", time.Date(2021, 12, 23, 0, 0, 0, 0, time.UTC), time.Date(2022, 01, 14, 0, 0, 0, 0, time.UTC)); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo) error {
			wantName := "test.txt"
			gotName := f.Name()

			if wantName != gotName {
				return fmt.Errorf("invalid name, got %v, want %v", gotName, wantName)
			}

			wantPerm := fmt.Sprintf("%o", 0666)
			gotPerm := fmt.Sprintf("%o", f.Mode())

			if wantPerm != gotPerm {
				return fmt.Errorf("invalid perm, got %v, want %v", gotPerm, wantPerm)
			}

			wantAtime := time.Date(2021, 12, 23, 0, 0, 0, 0, time.UTC)
			wantMtime := time.Date(2022, 01, 14, 0, 0, 0, 0, time.UTC)

			gotSys, ok := f.Sys().(*Stat)
			if !ok {
				return errors.New("could not get fs.Stat from FileInfo.Sys()")
			}

			gotAtime := time.Unix(0, gotSys.Atim.Nano())
			gotMtime := f.ModTime()

			if !wantAtime.Equal(gotAtime) {
				return fmt.Errorf("invalid Atime, got %v, want %v", gotAtime, wantAtime)
			}

			if !wantMtime.Equal(gotMtime) {
				return fmt.Errorf("invalid Mtime, got %v, want %v", gotMtime, wantMtime)
			}

			return nil
		},
		true,
		false, // HACK: OsFs uses umask, which yields unexpected permission bits (see https://github.com/golang/go/issues/38282)
	},
	{
		"Can stat /mydir/test.txt",
		"/mydir/test.txt",
		false,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo) error {
			wantName := "test.txt"
			gotName := f.Name()

			if wantName != gotName {
				return fmt.Errorf("invalid name, got %v, want %v", gotName, wantName)
			}

			wantPerm := fmt.Sprintf("%o", 0666)
			gotPerm := fmt.Sprintf("%o", f.Mode())

			if wantPerm != gotPerm {
				return fmt.Errorf("invalid perm, got %v, want %v", gotPerm, wantPerm)
			}

			return nil
		},
		true,
		false, // HACK: OsFs uses umask, which yields unexpected permission bits (see https://github.com/golang/go/issues/38282)
	},
	{
		"Can stat system-specific properties of /mydir/test.txt",
		"/mydir/test.txt",
		false,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/test.txt"); err != nil {
				return err
			}

			if err := f.Chtimes("/mydir/test.txt", time.Date(2021, 12, 23, 0, 0, 0, 0, time.UTC), time.Date(2022, 01, 14, 0, 0, 0, 0, time.UTC)); err != nil {
				return err
			}

			return nil
		},
		func(f os.FileInfo) error {
			wantName := "test.txt"
			gotName := f.Name()

			if wantName != gotName {
				return fmt.Errorf("invalid name, got %v, want %v", gotName, wantName)
			}

			wantPerm := fmt.Sprintf("%o", 0666)
			gotPerm := fmt.Sprintf("%o", f.Mode())

			if wantPerm != gotPerm {
				return fmt.Errorf("invalid perm, got %v, want %v", gotPerm, wantPerm)
			}

			wantAtime := time.Date(2021, 12, 23, 0, 0, 0, 0, time.UTC)
			wantMtime := time.Date(2022, 01, 14, 0, 0, 0, 0, time.UTC)

			gotSys, ok := f.Sys().(*Stat)
			if !ok {
				return errors.New("could not get fs.Stat from FileInfo.Sys()")
			}

			gotAtime := time.Unix(0, gotSys.Atim.Nano())
			gotMtime := f.ModTime()

			if !wantAtime.Equal(gotAtime) {
				return fmt.Errorf("invalid Atime, got %v, want %v", gotAtime, wantAtime)
			}

			if !wantMtime.Equal(gotMtime) {
				return fmt.Errorf("invalid Mtime, got %v, want %v", gotMtime, wantMtime)
			}

			return nil
		},
		true,
		false, // HACK: OsFs uses umask, which yields unexpected permission bits (see https://github.com/golang/go/issues/38282)
	},
	{
		"Can stat symlink to /test.txt",
		"/existingsymlink",
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f os.FileInfo) error {
			wantName := "existingsymlink"
			gotName := f.Name()

			if wantName != gotName {
				return fmt.Errorf("invalid name, got %v, want %v", gotName, wantName)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can stat symlink to /mydir",
		"/existingsymlink",
		false,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/mydir", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f os.FileInfo) error {
			wantName := "existingsymlink"
			gotName := f.Name()

			if wantName != gotName {
				return fmt.Errorf("invalid name, got %v, want %v", gotName, wantName)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can stat symlink to /mydir/nesteddir",
		"/existingsymlink",
		false,
		func(f symFs) error {
			if err := f.MkdirAll("/mydir/nesteddir", os.ModePerm); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/mydir/nesteddir", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f os.FileInfo) error {
			wantName := "existingsymlink"
			gotName := f.Name()

			if wantName != gotName {
				return fmt.Errorf("invalid name, got %v, want %v", gotName, wantName)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can stat symlink to symlink to root",
		"/existingsymlink",
		false,
		func(f symFs) error {
			if err := f.SymlinkIfPossible("/", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f os.FileInfo) error {
			wantName := "existingsymlink"
			gotName := f.Name()

			if wantName != gotName {
				return fmt.Errorf("invalid name, got %v, want %v", gotName, wantName)
			}

			return nil
		},
		true,
		true,
	},
}

func TestFile_Stat(t *testing.T) {
	for _, tt := range fileStatTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			file, err := symFs.Open(tt.open)
			if err != nil {
				t.Errorf("%v open() error = %v", symFs.Name(), err)

				return
			}

			got, err := file.Stat()
			if (err != nil) != tt.wantErr {
				t.Errorf("%v.File.Stat() error = %v, wantErr %v", fs.fs.Name(), err, tt.wantErr)

				return
			}

			if err := tt.check(got); err != nil {
				t.Errorf("%v check() error = %v", fs.fs.Name(), err)

				return
			}
		})
	}
}

type readdirArgs struct {
	count int
}

var readdirTests = []struct {
	name      string
	open      string
	args      readdirArgs
	wantErr   bool
	prepare   func(symFs) error
	check     func([]os.FileInfo) error
	withCache bool
	withOsFs  bool
}{
	{
		"Can readdir all in / if there are no children",
		"/",
		readdirArgs{-1},
		false,
		func(f symFs) error { return nil },
		func(f []os.FileInfo) error {
			if len(f) > 0 {
				return errors.New("found unexpected children in empty directory")
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdir all in '' if there are no children",
		"",
		readdirArgs{-1},
		false,
		func(f symFs) error { return nil },
		func(f []os.FileInfo) error {
			if len(f) > 0 {
				return errors.New("found unexpected children in empty directory")
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdir all in / if there is one child",
		"/",
		readdirArgs{-1},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f []os.FileInfo) error {
			wantNames := []string{"test.txt"}

			for i, info := range f {
				wantName := wantNames[i]
				gotName := info.Name()

				if wantName != gotName {
					return fmt.Errorf("invalid name, got %v, want %v", gotName, wantName)
				}
			}

			wantLength := len(wantNames)
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdir all with -1 in / if there are multiple children",
		"/",
		readdirArgs{-1},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/asdf.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f []os.FileInfo) error {
			wantNames := map[string]struct{}{
				"test.txt": {},
				"asdf.txt": {},
				"mydir":    {},
			}

			for _, info := range f {
				if _, ok := wantNames[info.Name()]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info.Name())
				}
			}

			wantLength := len(wantNames)
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdir all with -1 in / if there are multiple children and non-broken symlinks",
		"/",
		readdirArgs{-1},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/asdf.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f []os.FileInfo) error {
			wantNames := map[string]struct{}{
				"test.txt":        {},
				"asdf.txt":        {},
				"mydir":           {},
				"existingsymlink": {},
			}

			for _, info := range f {
				if _, ok := wantNames[info.Name()]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info.Name())
				}
			}

			wantLength := len(wantNames)
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdir all with -1 in / if there are multiple children and broken symlinks",
		"/",
		readdirArgs{-1},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/asdf.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test2.txt", "/brokensymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f []os.FileInfo) error {
			wantNames := map[string]struct{}{
				"test.txt":      {},
				"asdf.txt":      {},
				"mydir":         {},
				"brokensymlink": {},
			}

			for _, info := range f {
				if _, ok := wantNames[info.Name()]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info.Name())
				}
			}

			wantLength := len(wantNames)
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdir all with 0 in / if there are multiple children",
		"/",
		readdirArgs{0},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/asdf.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f []os.FileInfo) error {
			wantNames := map[string]struct{}{
				"test.txt": {},
				"asdf.txt": {},
				"mydir":    {},
			}

			for _, info := range f {
				if _, ok := wantNames[info.Name()]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info.Name())
				}
			}

			wantLength := len(wantNames)
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdir 2 in / if there are multiple children",
		"/",
		readdirArgs{2},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/asdf.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f []os.FileInfo) error {
			wantNames := map[string]struct{}{
				"test.txt": {},
				"asdf.txt": {},
				"mydir":    {},
			}

			for _, info := range f {
				if _, ok := wantNames[info.Name()]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info.Name())
				}
			}

			wantLength := 2
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdir 2 in /mydir if there are multiple children",
		"/mydir",
		readdirArgs{2},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/asdf.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/hmm.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f []os.FileInfo) error {
			wantNames := map[string]struct{}{
				"asdf.txt": {},
				"hmm.txt":  {},
			}

			for _, info := range f {
				if _, ok := wantNames[info.Name()]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info.Name())
				}
			}

			wantLength := 2
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdir 3 in /mydir/nested if there are multiple children",
		"/mydir/nested",
		readdirArgs{3},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if err := f.MkdirAll("/mydir/nested", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/asdf.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/hmm.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/hmm2.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f []os.FileInfo) error {
			wantNames := map[string]struct{}{
				"asdf.txt": {},
				"hmm.txt":  {},
				"hmm2.txt": {},
			}

			for _, info := range f {
				if _, ok := wantNames[info.Name()]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info.Name())
				}
			}

			wantLength := len(wantNames)
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdir 5 in /mydir/nested if there are multiple children and non-broken symlinks",
		"/mydir/nested",
		readdirArgs{5},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if err := f.MkdirAll("/mydir/nested", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/asdf.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/hmm.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/hmm2.txt"); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/mydir/nested/existingsymlink"); err != nil {
				return nil
			}

			if err := f.SymlinkIfPossible("/mydir/nested/hmm.txt", "/mydir/nested/existingsymlink2"); err != nil {
				return nil
			}

			return nil
		},
		func(f []os.FileInfo) error {
			wantNames := map[string]struct{}{
				"asdf.txt":         {},
				"hmm.txt":          {},
				"hmm2.txt":         {},
				"existingsymlink":  {},
				"existingsymlink2": {},
			}

			for _, info := range f {
				if _, ok := wantNames[info.Name()]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info.Name())
				}
			}

			wantLength := len(wantNames)
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdir 5 in /mydir/nested if there are multiple children and broken symlinks",
		"/mydir/nested",
		readdirArgs{5},
		false,
		func(f symFs) error {
			if err := f.MkdirAll("/mydir/nested", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/asdf.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/hmm.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/hmm2.txt"); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/mydir/nested/existingsymlink"); err != nil {
				return nil
			}

			if err := f.SymlinkIfPossible("/mydir/nested/hmm.txt", "/mydir/nested/existingsymlink2"); err != nil {
				return nil
			}

			return nil
		},
		func(f []os.FileInfo) error {
			wantNames := map[string]struct{}{
				"asdf.txt":         {},
				"hmm.txt":          {},
				"hmm2.txt":         {},
				"existingsymlink":  {},
				"existingsymlink2": {},
			}

			for _, info := range f {
				if _, ok := wantNames[info.Name()]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info.Name())
				}
			}

			wantLength := len(wantNames)
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
}

func TestFile_Readdir(t *testing.T) {
	for _, tt := range readdirTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			file, err := symFs.Open(tt.open)
			if err != nil {
				t.Errorf("%v open() error = %v", symFs.Name(), err)

				return
			}

			got, err := file.Readdir(tt.args.count)
			if (err != nil) != tt.wantErr {
				t.Errorf("%v.File.Readdir() error = %v, wantErr %v", fs.fs.Name(), err, tt.wantErr)

				return
			}

			if err := tt.check(got); err != nil {
				t.Errorf("%v check() error = %v", fs.fs.Name(), err)

				return
			}
		})
	}
}

type readdirnamesArgs struct {
	count int
}

var readdirnamesTests = []struct {
	name      string
	open      string
	args      readdirnamesArgs
	wantErr   bool
	prepare   func(symFs) error
	check     func([]string) error
	withCache bool
	withOsFs  bool
}{
	{
		"Can readdirnames all in / if there are no children",
		"/",
		readdirnamesArgs{-1},
		false,
		func(f symFs) error { return nil },
		func(f []string) error {
			if len(f) > 0 {
				return errors.New("found unexpected children in empty directory")
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdirnames all in '' if there are no children",
		"",
		readdirnamesArgs{-1},
		false,
		func(f symFs) error { return nil },
		func(f []string) error {
			if len(f) > 0 {
				return errors.New("found unexpected children in empty directory")
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdirnames all in / if there is one child",
		"/",
		readdirnamesArgs{-1},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f []string) error {
			wantNames := []string{"test.txt"}

			for i, info := range f {
				wantName := wantNames[i]
				gotName := info

				if wantName != gotName {
					return fmt.Errorf("invalid name, got %v, want %v", gotName, wantName)
				}
			}

			wantLength := len(wantNames)
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdirnames all with -1 in / if there are multiple children",
		"/",
		readdirnamesArgs{-1},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/asdf.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f []string) error {
			wantNames := map[string]struct{}{
				"test.txt": {},
				"asdf.txt": {},
				"mydir":    {},
			}

			for _, info := range f {
				if _, ok := wantNames[info]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info)
				}
			}

			wantLength := len(wantNames)
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdirnames all with -1 in / if there are multiple children and non-broken symlinks",
		"/",
		readdirnamesArgs{-1},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/asdf.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f []string) error {
			wantNames := map[string]struct{}{
				"test.txt":        {},
				"asdf.txt":        {},
				"mydir":           {},
				"existingsymlink": {},
			}

			for _, info := range f {
				if _, ok := wantNames[info]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info)
				}
			}

			wantLength := len(wantNames)
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdirnames all with -1 in / if there are multiple children and broken symlinks",
		"/",
		readdirnamesArgs{-1},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/asdf.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test2.txt", "/brokensymlink"); err != nil {
				return nil
			}

			return nil
		},
		func(f []string) error {
			wantNames := map[string]struct{}{
				"test.txt":      {},
				"asdf.txt":      {},
				"mydir":         {},
				"brokensymlink": {},
			}

			for _, info := range f {
				if _, ok := wantNames[info]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info)
				}
			}

			wantLength := len(wantNames)
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdirnames all with 0 in / if there are multiple children",
		"/",
		readdirnamesArgs{0},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/asdf.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f []string) error {
			wantNames := map[string]struct{}{
				"test.txt": {},
				"asdf.txt": {},
				"mydir":    {},
			}

			for _, info := range f {
				if _, ok := wantNames[info]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info)
				}
			}

			wantLength := len(wantNames)
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdirnames 2 in / if there are multiple children",
		"/",
		readdirnamesArgs{2},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/asdf.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f []string) error {
			wantNames := map[string]struct{}{
				"test.txt": {},
				"asdf.txt": {},
				"mydir":    {},
			}

			for _, info := range f {
				if _, ok := wantNames[info]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info)
				}
			}

			wantLength := 2
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdirnames 2 in /mydir if there are multiple children",
		"/mydir",
		readdirnamesArgs{2},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/asdf.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/hmm.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f []string) error {
			wantNames := map[string]struct{}{
				"asdf.txt": {},
				"hmm.txt":  {},
			}

			for _, info := range f {
				if _, ok := wantNames[info]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info)
				}
			}

			wantLength := 2
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdirnames 3 in /mydir/nested if there are multiple children",
		"/mydir/nested",
		readdirnamesArgs{3},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if err := f.MkdirAll("/mydir/nested", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/asdf.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/hmm.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/hmm2.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f []string) error {
			wantNames := map[string]struct{}{
				"asdf.txt": {},
				"hmm.txt":  {},
				"hmm2.txt": {},
			}

			for _, info := range f {
				if _, ok := wantNames[info]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info)
				}
			}

			wantLength := len(f)
			gotLength := 3
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdirnames 5 in /mydir/nested if there are multiple children and non-broken symlinks",
		"/mydir/nested",
		readdirnamesArgs{5},
		false,
		func(f symFs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			if err := f.MkdirAll("/mydir/nested", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/asdf.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/hmm.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/hmm2.txt"); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/mydir/nested/existingsymlink"); err != nil {
				return nil
			}

			if err := f.SymlinkIfPossible("/mydir/nested/hmm.txt", "/mydir/nested/existingsymlink2"); err != nil {
				return nil
			}

			return nil
		},
		func(f []string) error {
			wantNames := map[string]struct{}{
				"asdf.txt":         {},
				"hmm.txt":          {},
				"hmm2.txt":         {},
				"existingsymlink":  {},
				"existingsymlink2": {},
			}

			for _, info := range f {
				if _, ok := wantNames[info]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info)
				}
			}

			wantLength := len(wantNames)
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can readdirnames 5 in /mydir/nested if there are multiple children and broken symlinks",
		"/mydir/nested",
		readdirnamesArgs{5},
		false,
		func(f symFs) error {
			if err := f.MkdirAll("/mydir/nested", os.ModePerm); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/asdf.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/hmm.txt"); err != nil {
				return err
			}

			if _, err := f.Create("/mydir/nested/hmm2.txt"); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/mydir/nested/existingsymlink"); err != nil {
				return nil
			}

			if err := f.SymlinkIfPossible("/mydir/nested/hmm.txt", "/mydir/nested/existingsymlink2"); err != nil {
				return nil
			}

			return nil
		},
		func(f []string) error {
			wantNames := map[string]struct{}{
				"asdf.txt":         {},
				"hmm.txt":          {},
				"hmm2.txt":         {},
				"existingsymlink":  {},
				"existingsymlink2": {},
			}

			for _, info := range f {
				if _, ok := wantNames[info]; !ok {
					return fmt.Errorf("could not find file or directory with name %v", info)
				}
			}

			wantLength := len(wantNames)
			gotLength := len(f)
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", gotLength, wantLength)
			}

			return nil
		},
		true,
		true,
	},
}

func TestFile_Readdirnames(t *testing.T) {
	for _, tt := range readdirnamesTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			file, err := symFs.Open(tt.open)
			if err != nil {
				t.Errorf("%v open() error = %v", symFs.Name(), err)

				return
			}

			got, err := file.Readdirnames(tt.args.count)
			if (err != nil) != tt.wantErr {
				t.Errorf("%v.File.Readdirnames() error = %v, wantErr %v", fs.fs.Name(), err, tt.wantErr)

				return
			}

			if err := tt.check(got); err != nil {
				t.Errorf("%v check() error = %v", fs.fs.Name(), err)

				return
			}
		})
	}
}

var readTests = []struct {
	name           string
	open           string
	wantErr        bool
	prepare        func(afero.Fs) error
	check          func(afero.File) error
	withCache      bool
	withOsFs       bool
	large          bool
	followSymlinks bool
}{
	{
		"Can read / into empty buffer",
		"/",
		false,
		func(f afero.Fs) error { return nil },
		func(f afero.File) error {
			wantContent := []byte{}
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.Read(gotContent)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid read length, got %v, want %v", gotLength, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid read content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can read /mydir into empty buffer",
		"/mydir",
		false,
		func(f afero.Fs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			wantContent := []byte{}
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.Read(gotContent)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid read length, got %v, want %v", gotLength, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid read content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can not read / into non-empty buffer",
		"/",
		true,
		func(f afero.Fs) error { return nil },
		func(f afero.File) error {
			gotContent := make([]byte, 10)

			if _, err := f.Read(gotContent); err != io.EOF {
				return err
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can not read /mydir into non-empty buffer",
		"/mydir",
		true,
		func(f afero.Fs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			gotContent := make([]byte, 10)

			if _, err := f.Read(gotContent); err != io.EOF {
				return err
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can read /test.txt if it exists and is empty",
		"/test.txt",
		false,
		func(f afero.Fs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			wantContent := []byte{}
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.Read(gotContent)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid read length, got %v, want %v", gotLength, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid read content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can read /test.txt if it exists and contains small amount of data",
		"/test.txt",
		false,
		func(f afero.Fs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			if _, err := file.Write([]byte("Hello, world")); err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File) error {
			wantContent := []byte("Hello, world")
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.Read(gotContent)
			if err != io.EOF {
				return err
			}

			if gotLength != wantLength {
				return fmt.Errorf("invalid read length, got %v, want %v", gotLength, wantLength)
			}

			if string(gotContent) != string(wantContent) {
				return fmt.Errorf("invalid read content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can read /test.txt if it exists and contains 30 MB amount of data",
		"/test.txt",
		false,
		func(f afero.Fs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			r := newDeterministicReader(1000)

			if _, err := io.Copy(file, r); err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File) error {
			wantHash := "HTUi7GuNreHASha4hhl1xwuYk03pyTJ0IJbFLv04UdccT9m_NA2oBFTrnMxJhEu3VMGxDYk_04Th9C0zOj5MyA=="
			wantLength := int64(32800768)

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, f)
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid read length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid read hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can read /test.txt if it exists and contains 300 MB of data",
		"/test.txt",
		false,
		func(f afero.Fs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			r := newDeterministicReader(10000)

			if _, err := io.Copy(file, r); err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File) error {
			wantHash := "3NXGfwSdGiFZjd-sdIcx4xrUnsOPOb4LeDBYGZFVPoRyMqGdqTEHsTbk1Ow3Vn-wIdFqaO8Zj6eXhYvWBakkuQ=="
			wantLength := int64(327712768)

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, f)
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid read length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid read hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		true,
		false,
	},
	{
		"Can read /test.txt sequentially if it exists and contains 30 MB amount of data",
		"/test.txt",
		false,
		func(f afero.Fs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			r := newDeterministicReader(1000)

			if _, err := io.Copy(file, r); err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File) error {
			firstChunk := make([]byte, 32800768/2)
			secondChunk := make([]byte, 32800768/2)

			if _, err := f.Read(firstChunk); err != nil {
				return err
			}

			if _, err := f.Read(secondChunk); err != nil {
				return err
			}

			wantHash := "HTUi7GuNreHASha4hhl1xwuYk03pyTJ0IJbFLv04UdccT9m_NA2oBFTrnMxJhEu3VMGxDYk_04Th9C0zOj5MyA=="
			wantLength := int64(32800768)

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, bytes.NewBuffer(append(firstChunk, secondChunk...)))
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid read length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid read hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can not read /brokensymlink into non-empty buffer",
		"/brokensymlink",
		true,
		func(f afero.Fs) error {
			symFs, ok := f.(symFs)
			if !ok {
				return nil
			}

			if err := symFs.SymlinkIfPossible("/mydir", "/brokensymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			gotContent := make([]byte, 10)

			if _, err := f.Read(gotContent); err != io.EOF {
				return err
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can read /existingsymlink into non-empty buffer without readlink",
		"/existingsymlink",
		false,
		func(f afero.Fs) error {
			symFs, ok := f.(symFs)
			if !ok {
				return nil
			}

			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			r := newDeterministicReader(1000)

			if _, err := io.Copy(file, r); err != nil {
				return err
			}

			if err := file.Close(); err != nil {
				return err
			}

			if err := symFs.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			wantHash := "HTUi7GuNreHASha4hhl1xwuYk03pyTJ0IJbFLv04UdccT9m_NA2oBFTrnMxJhEu3VMGxDYk_04Th9C0zOj5MyA=="
			wantLength := int64(32800768)

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, f)
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid read length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid read hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
}

func TestFile_Read(t *testing.T) {
	for _, tt := range readTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			if tt.large && testing.Short() {
				return
			}

			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			open := tt.open
			if tt.followSymlinks {
				var err error
				open, err = symFs.ReadlinkIfPossible(tt.open)
				if err != nil {
					t.Errorf("%v readling() error = %v", symFs.Name(), err)

					return
				}
			}

			file, err := symFs.Open(open)
			if err != nil && tt.wantErr {
				return
			}

			if err != nil {
				t.Errorf("%v open() error = %v", symFs.Name(), err)

				return
			}

			if err := tt.check(file); (err != nil) != tt.wantErr {
				t.Errorf("%v check() error = %v", fs.fs.Name(), err)

				return
			}
		})
	}
}

var readAtTests = []struct {
	name           string
	open           string
	wantErr        bool
	prepare        func(afero.Fs) error
	check          func(afero.File) error
	withCache      bool
	withOsFs       bool
	large          bool
	followSymlinks bool
}{
	{
		"Can readAt / into empty buffer",
		"/",
		false,
		func(f afero.Fs) error { return nil },
		func(f afero.File) error {
			wantContent := []byte{}
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.ReadAt(gotContent, 0)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid readAt length, got %v, want %v", gotLength, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid readAt content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can readAt /mydir into empty buffer",
		"/mydir",
		false,
		func(f afero.Fs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			wantContent := []byte{}
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.ReadAt(gotContent, 0)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid readAt length, got %v, want %v", gotLength, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid readAt content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can not readAt / into non-empty buffer",
		"/",
		true,
		func(f afero.Fs) error { return nil },
		func(f afero.File) error {
			gotContent := make([]byte, 10)

			if _, err := f.ReadAt(gotContent, 0); err != io.EOF {
				return err
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can not readAt /mydir into non-empty buffer",
		"/mydir",
		true,
		func(f afero.Fs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			gotContent := make([]byte, 10)

			if _, err := f.ReadAt(gotContent, 0); err != io.EOF {
				return err
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can readAt /test.txt if it exists and is empty",
		"/test.txt",
		false,
		func(f afero.Fs) error {
			if _, err := f.Create("/test.txt"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			wantContent := []byte{}
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.ReadAt(gotContent, 0)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid readAt length, got %v, want %v", gotLength, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid readAt content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can readAt /test.txt if it exists and contains small amount of data",
		"/test.txt",
		false,
		func(f afero.Fs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			if _, err := file.Write([]byte("Hello, world")); err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File) error {
			wantContent := []byte("Hello, world")
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.ReadAt(gotContent, 0)
			if err != io.EOF {
				return err
			}

			if gotLength != wantLength {
				return fmt.Errorf("invalid readAt length, got %v, want %v", gotLength, wantLength)
			}

			if string(gotContent) != string(wantContent) {
				return fmt.Errorf("invalid readAt content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can readAt /test.txt if it exists and contains 30 MB amount of data",
		"/test.txt",
		false,
		func(f afero.Fs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			r := newDeterministicReader(1000)

			if _, err := io.Copy(file, r); err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File) error {
			wantHash := "HTUi7GuNreHASha4hhl1xwuYk03pyTJ0IJbFLv04UdccT9m_NA2oBFTrnMxJhEu3VMGxDYk_04Th9C0zOj5MyA=="
			wantLength := int64(32800768)

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, f)
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid readAt length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid readAt hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can readAt /test.txt if it exists and contains 300 MB of data",
		"/test.txt",
		false,
		func(f afero.Fs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			r := newDeterministicReader(10000)

			if _, err := io.Copy(file, r); err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File) error {
			wantHash := "3NXGfwSdGiFZjd-sdIcx4xrUnsOPOb4LeDBYGZFVPoRyMqGdqTEHsTbk1Ow3Vn-wIdFqaO8Zj6eXhYvWBakkuQ=="
			wantLength := int64(327712768)

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, f)
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid readAt length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid readAt hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		true,
		false,
	},
	{
		"Can readAt /test.txt sequentially if it exists and contains 30 MB amount of data",
		"/test.txt",
		false,
		func(f afero.Fs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			r := newDeterministicReader(1000)

			if _, err := io.Copy(file, r); err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File) error {
			firstChunk := make([]byte, 32800768/2)
			secondChunk := make([]byte, 32800768/2)

			if _, err := f.ReadAt(firstChunk, 0); err != nil {
				return err
			}

			if _, err := f.ReadAt(secondChunk, 32800768/2); err != nil {
				return err
			}

			wantHash := "HTUi7GuNreHASha4hhl1xwuYk03pyTJ0IJbFLv04UdccT9m_NA2oBFTrnMxJhEu3VMGxDYk_04Th9C0zOj5MyA=="
			wantLength := int64(32800768)

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, bytes.NewBuffer(append(firstChunk, secondChunk...)))
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid readAt length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid readAt hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can not readAt /brokensymlink into non-empty buffer",
		"/brokensymlink",
		true,
		func(f afero.Fs) error {
			symFs, ok := f.(symFs)
			if !ok {
				return nil
			}

			if err := symFs.SymlinkIfPossible("/mydir", "/brokensymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			gotContent := make([]byte, 10)

			if _, err := f.ReadAt(gotContent, 0); err != io.EOF {
				return err
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can readAt /existingsymlink into non-empty buffer without readAtlink",
		"/existingsymlink",
		false,
		func(f afero.Fs) error {
			symFs, ok := f.(symFs)
			if !ok {
				return nil
			}

			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			r := newDeterministicReader(1000)

			if _, err := io.Copy(file, r); err != nil {
				return err
			}

			if err := file.Close(); err != nil {
				return err
			}

			if err := symFs.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			wantHash := "HTUi7GuNreHASha4hhl1xwuYk03pyTJ0IJbFLv04UdccT9m_NA2oBFTrnMxJhEu3VMGxDYk_04Th9C0zOj5MyA=="
			wantLength := int64(32800768)

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, f)
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid readAt length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid readAt hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can readAt /test.txt sequentially, but not in order if it exists and contains 30 MB amount of data",
		"/test.txt",
		false,
		func(f afero.Fs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			r := newDeterministicReader(1000)

			if _, err := io.Copy(file, r); err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File) error {
			firstChunk := make([]byte, 32800768/2)
			secondChunk := make([]byte, 32800768/2)

			if _, err := f.ReadAt(secondChunk, 32800768/2); err != nil {
				return err
			}

			if _, err := f.ReadAt(firstChunk, 0); err != nil {
				return err
			}

			wantHash := "HTUi7GuNreHASha4hhl1xwuYk03pyTJ0IJbFLv04UdccT9m_NA2oBFTrnMxJhEu3VMGxDYk_04Th9C0zOj5MyA=="
			wantLength := int64(32800768)

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, bytes.NewBuffer(append(firstChunk, secondChunk...)))
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid readAt length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid readAt hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can readAt /test.txt sequentially if it exists and contains small amount of data",
		"/test.txt",
		false,
		func(f afero.Fs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			if _, err := file.Write([]byte("Hello, world!")); err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File) error {
			firstChunk := make([]byte, 6)
			secondChunk := make([]byte, 7)

			if _, err := f.ReadAt(firstChunk, 0); err != nil {
				return err
			}

			if _, err := f.ReadAt(secondChunk, 6); err != io.EOF {
				return err
			}

			wantContent := []byte("Hello, world!")
			gotContent := append([]byte{}, append(firstChunk, secondChunk...)...)

			wantLength := len(wantContent)
			gotLength := len(gotContent)

			if gotLength != wantLength {
				return fmt.Errorf("invalid readAt length, got %v, want %v", gotLength, wantLength)
			}

			if string(gotContent) != string(wantContent) {
				return fmt.Errorf("invalid readAt content, got %v, want %v", string(gotContent), string(wantContent))
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
	{
		"Can readAt /test.txt sequentially, but not in order if it exists and contains small amount of data",
		"/test.txt",
		false,
		func(f afero.Fs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			if _, err := file.Write([]byte("Hello, world!")); err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File) error {
			firstChunk := make([]byte, 6)
			secondChunk := make([]byte, 7)

			if _, err := f.ReadAt(secondChunk, 6); err != io.EOF {
				return err
			}

			if _, err := f.ReadAt(firstChunk, 0); err != nil {
				return err
			}

			wantContent := []byte("Hello, world!")
			gotContent := append([]byte{}, append(firstChunk, secondChunk...)...)

			wantLength := len(wantContent)
			gotLength := len(gotContent)

			if gotLength != wantLength {
				return fmt.Errorf("invalid readAt length, got %v, want %v", gotLength, wantLength)
			}

			if string(gotContent) != string(wantContent) {
				return fmt.Errorf("invalid readAt content, got %v, want %v", string(gotContent), string(wantContent))
			}

			return nil
		},
		true,
		true,
		false,
		false,
	},
}

func TestFile_ReadAt(t *testing.T) {
	for _, tt := range readAtTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			if tt.large && testing.Short() {
				return
			}

			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			open := tt.open
			if tt.followSymlinks {
				var err error
				open, err = symFs.ReadlinkIfPossible(tt.open)
				if err != nil {
					t.Errorf("%v readAt() error = %v", symFs.Name(), err)

					return
				}
			}

			file, err := symFs.Open(open)
			if err != nil && tt.wantErr {
				return
			}

			if err != nil {
				t.Errorf("%v open() error = %v", symFs.Name(), err)

				return
			}

			if err := tt.check(file); (err != nil) != tt.wantErr {
				t.Errorf("%v check() error = %v", fs.fs.Name(), err)

				return
			}
		})
	}
}

type seekArgs struct {
	offset int64
	whence int
}

var seekTests = []struct {
	name      string
	open      string
	args      seekArgs
	wantErr   bool
	prepare   func(symFs) error
	check     func(afero.File, int64) error
	withCache bool
	withOsFs  bool
}{
	{
		"Can seek on /",
		"/",
		seekArgs{1000, io.SeekStart},
		false,
		func(f symFs) error { return nil },
		func(f afero.File, i int64) error { return nil },
		true,
		true,
	},
	{
		"Can seek on /mydir",
		"/mydir",
		seekArgs{1000, io.SeekStart},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File, i int64) error { return nil },
		true,
		true,
	},
	{
		"Can seek on /test.txt within limits",
		"/test.txt",
		seekArgs{6, io.SeekStart},
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			if _, err := file.WriteString("Hello, world!"); err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File, i int64) error {
			wantCount := int64(6)
			gotCount := i

			if wantCount != gotCount {
				return fmt.Errorf("invalid count, got %v, want %v", gotCount, wantCount)
			}

			wantContent := " world!"
			gotContent := make([]byte, 7)

			if _, err := f.Read(gotContent); err != nil {
				return err
			}

			if string(gotContent) != string(wantContent) {
				return fmt.Errorf("invalid read content, got %v, want %v", string(gotContent), string(wantContent))
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can seek on /test.txt outside limits",
		"/test.txt",
		seekArgs{1000, io.SeekStart},
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			if _, err := file.WriteString("Hello, world!"); err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File, i int64) error {
			wantCount := int64(1000)
			gotCount := i

			if wantCount != gotCount {
				return fmt.Errorf("invalid count, got %v, want %v", gotCount, wantCount)
			}

			gotContent := make([]byte, 1)

			if _, err := f.Read(gotContent); !strings.Contains(err.Error(), "EOF") {
				return err
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can seek on empty /test.txt outside limits",
		"/test.txt",
		seekArgs{1000, io.SeekStart},
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File, i int64) error {
			wantCount := int64(1000)
			gotCount := i

			if wantCount != gotCount {
				return fmt.Errorf("invalid count, got %v, want %v", gotCount, wantCount)
			}

			gotContent := make([]byte, 1)

			if _, err := f.Read(gotContent); !strings.Contains(err.Error(), "EOF") {
				return err
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can seek on symlink to /",
		"/existingsymlink",
		seekArgs{1000, io.SeekStart},
		false,
		func(f symFs) error {
			if err := f.SymlinkIfPossible("/", "/existingsymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File, i int64) error { return nil },
		true,
		true,
	},
	{
		"Can seek on symlink to /mydir",
		"/existingsymlink",
		seekArgs{1000, io.SeekStart},
		false,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/mydir", "/existingsymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File, i int64) error { return nil },
		true,
		true,
	},
	{
		"Can seek on symlink to /test.txt within limits",
		"/existingsymlink",
		seekArgs{6, io.SeekStart},
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			if _, err := file.WriteString("Hello, world!"); err != nil {
				return err
			}

			if err := file.Close(); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File, i int64) error {
			wantCount := int64(6)
			gotCount := i

			if wantCount != gotCount {
				return fmt.Errorf("invalid count, got %v, want %v", gotCount, wantCount)
			}

			wantContent := " world!"
			gotContent := make([]byte, 7)

			if _, err := f.Read(gotContent); err != nil {
				return err
			}

			if string(gotContent) != string(wantContent) {
				return fmt.Errorf("invalid read content, got %v, want %v", string(gotContent), string(wantContent))
			}

			return nil
		},
		true,
		true,
	},
	{
		"Can not seek on broken symlink",
		"/brokensymlink",
		seekArgs{1000, io.SeekStart},
		true,
		func(f symFs) error {
			if err := f.SymlinkIfPossible("/mydir", "/brokensymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File, i int64) error { return nil },
		true,
		true,
	},
}

func TestFile_Seek(t *testing.T) {
	for _, tt := range seekTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			file, err := symFs.Open(tt.open)
			if err != nil && tt.wantErr {
				return
			}

			if err != nil {
				t.Errorf("%v open() error = %v", symFs.Name(), err)

				return
			}

			n, err := file.Seek(tt.args.offset, tt.args.whence)
			if (err != nil) != tt.wantErr {
				t.Errorf("%v.Seek() error = %v", symFs.Name(), err)

				return
			}

			if err := tt.check(file, n); (err != nil) != tt.wantErr {
				t.Errorf("%v check() error = %v", fs.fs.Name(), err)

				return
			}
		})
	}
}

type writeArgs struct {
	p io.Reader
}

var writeTests = []struct {
	name      string
	open      string
	args      func() writeArgs
	wantErr   bool
	prepare   func(symFs) error
	check     func(afero.File, int) error
	withCache bool
	withOsFs  bool
	large     bool
}{
	{
		"Can not write to /",
		"/",
		func() writeArgs {
			return writeArgs{strings.NewReader("Hello, world!")}
		},
		true,
		func(f symFs) error { return nil },
		func(f afero.File, i int) error { return nil },
		true,
		true,
		false,
	},
	{
		"Can not write to /mydir",
		"/mydir",
		func() writeArgs {
			return writeArgs{strings.NewReader("Hello, world!")}
		},
		true,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File, i int) error { return nil },
		true,
		true,
		false,
	},
	{
		"Can write empty string to /test.txt",
		"/test.txt",
		func() writeArgs {
			return writeArgs{strings.NewReader("")}
		},
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File, i int) error {
			wantContent := []byte{}
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.Read(gotContent)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid write length, got %v, want %v", gotLength, wantLength)
			}

			if wantLength != i {
				return fmt.Errorf("invalid write length n, got %v, want %v", i, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid write content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
	},
	{
		"Can write small amount of data to /test.txt if seeking afterwards",
		"/test.txt",
		func() writeArgs {
			return writeArgs{strings.NewReader("Hello, world!")}
		},
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File, i int) error {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}

			wantContent := []byte("Hello, world")
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.Read(gotContent)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid write length, got %v, want %v", gotLength, wantLength)
			}

			if wantLength != i {
				return fmt.Errorf("invalid write length n, got %v, want %v", i, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid write content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
	},
	{
		"Can write small amount of data to /test.txt if not seeking afterwards",
		"/test.txt",
		func() writeArgs {
			return writeArgs{strings.NewReader("Hello, world!")}
		},
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File, i int) error {
			wantContent := []byte("")
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.Read(gotContent)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid write length, got %v, want %v", gotLength, wantLength)
			}

			if wantLength != i {
				return fmt.Errorf("invalid write length n, got %v, want %v", i, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid write content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
	},
	{
		"Can write 30 MB amount of data to /test.txt",
		"/test.txt",
		func() writeArgs {
			return writeArgs{newDeterministicReader(1000)}
		},
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File, i int) error {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}

			wantHash := "HTUi7GuNreHASha4hhl1xwuYk03pyTJ0IJbFLv04UdccT9m_NA2oBFTrnMxJhEu3VMGxDYk_04Th9C0zOj5MyA=="
			wantLength := int64(32800768)

			if wantLength != int64(i) {
				return fmt.Errorf("invalid write length n, got %v, want %v", i, wantLength)
			}

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, f)
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid read length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid read hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		false,
	},
	{
		"Can write 300 MB amount of data to /test.txt",
		"/test.txt",
		func() writeArgs {
			return writeArgs{newDeterministicReader(10000)}
		},
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File, i int) error {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}

			wantHash := "3NXGfwSdGiFZjd-sdIcx4xrUnsOPOb4LeDBYGZFVPoRyMqGdqTEHsTbk1Ow3Vn-wIdFqaO8Zj6eXhYvWBakkuQ=="
			wantLength := int64(327712768)

			if wantLength != int64(i) {
				return fmt.Errorf("invalid write length n, got %v, want %v", i, wantLength)
			}

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, f)
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid read length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid read hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		true,
	},
	{
		"Can not write to symlink to /",
		"/existingsymlink",
		func() writeArgs {
			return writeArgs{strings.NewReader("Hello, world!")}
		},
		true,
		func(f symFs) error {
			if err := f.SymlinkIfPossible("/", "/existingsymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File, i int) error { return nil },
		true,
		true,
		false,
	},
	{
		"Can not write to symlink to /mydir",
		"/existingsymlink",
		func() writeArgs {
			return writeArgs{strings.NewReader("Hello, world!")}
		},
		true,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/mydir", "/existingsymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File, i int) error { return nil },
		true,
		true,
		false,
	},
	{
		"Can write empty string to symlink to /test.txt",
		"/existingsymlink",
		func() writeArgs {
			return writeArgs{strings.NewReader("")}
		},
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			if err := file.Close(); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File, i int) error {
			wantContent := []byte{}
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.Read(gotContent)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid write length, got %v, want %v", gotLength, wantLength)
			}

			if wantLength != i {
				return fmt.Errorf("invalid write length n, got %v, want %v", i, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid write content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
	},
	{
		"Can write small amount of data to symlink to /test.txt if seeking afterwards",
		"/existingsymlink",
		func() writeArgs {
			return writeArgs{strings.NewReader("Hello, world!")}
		},
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			if err := file.Close(); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File, i int) error {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}

			wantContent := []byte("Hello, world")
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.Read(gotContent)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid write length, got %v, want %v", gotLength, wantLength)
			}

			if wantLength != i {
				return fmt.Errorf("invalid write length n, got %v, want %v", i, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid write content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
	},
	{
		"Can write small amount of data to symlink to /test.txt if not seeking afterwards",
		"/existingsymlink",
		func() writeArgs {
			return writeArgs{strings.NewReader("Hello, world!")}
		},
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			if err := file.Close(); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File, i int) error {
			wantContent := []byte("")
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.Read(gotContent)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid write length, got %v, want %v", gotLength, wantLength)
			}

			if wantLength != i {
				return fmt.Errorf("invalid write length n, got %v, want %v", i, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid write content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
	},
	{
		"Can write 30 MB amount of data to symlink to /test.txt",
		"/existingsymlink",
		func() writeArgs {
			return writeArgs{newDeterministicReader(1000)}
		},
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			if err := file.Close(); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File, i int) error {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}

			wantHash := "HTUi7GuNreHASha4hhl1xwuYk03pyTJ0IJbFLv04UdccT9m_NA2oBFTrnMxJhEu3VMGxDYk_04Th9C0zOj5MyA=="
			wantLength := int64(32800768)

			if wantLength != int64(i) {
				return fmt.Errorf("invalid write length n, got %v, want %v", i, wantLength)
			}

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, f)
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid read length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid read hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		false,
	},
	{
		"Can write 300 MB amount of data to symlink to /test.txt",
		"/existingsymlink",
		func() writeArgs {
			return writeArgs{newDeterministicReader(10000)}
		},
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			if err := file.Close(); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File, i int) error {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}

			wantHash := "3NXGfwSdGiFZjd-sdIcx4xrUnsOPOb4LeDBYGZFVPoRyMqGdqTEHsTbk1Ow3Vn-wIdFqaO8Zj6eXhYvWBakkuQ=="
			wantLength := int64(327712768)

			if wantLength != int64(i) {
				return fmt.Errorf("invalid write length n, got %v, want %v", i, wantLength)
			}

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, f)
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid read length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid read hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		true,
	},
}

func TestFile_Write(t *testing.T) {
	for _, tt := range writeTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			if tt.large && testing.Short() {
				return
			}

			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			file, err := symFs.OpenFile(tt.open, os.O_RDWR, os.ModePerm)
			if err != nil && tt.wantErr {
				return
			}

			if err != nil {
				t.Errorf("%v open() error = %v", symFs.Name(), err)

				return
			}

			n, err := io.Copy(file, tt.args().p)
			if (err != nil) != tt.wantErr {
				t.Errorf("%v.Write() error = %v", symFs.Name(), err)

				return
			}

			if err == nil {
				if err := tt.check(file, int(n)); (err != nil) != tt.wantErr {
					t.Errorf("%v check() error = %v", fs.fs.Name(), err)

					return
				}
			}
		})
	}
}

var writeAtTests = []struct {
	name      string
	open      string
	wantErr   bool
	prepare   func(symFs) error
	apply     func(afero.File) error
	check     func(afero.File) error
	withCache bool
	withOsFs  bool
	large     bool
}{
	{
		"Can not write to /",
		"/",
		true,
		func(f symFs) error { return nil },
		func(f afero.File) error {
			if _, err := f.WriteAt([]byte("Hello, world!"), 0); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error { return nil },
		true,
		true,
		false,
	},
	{
		"Can not write to /mydir",
		"/mydir",
		true,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			if _, err := f.WriteAt([]byte("Hello, world!"), 0); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error { return nil },
		true,
		true,
		false,
	},
	{
		"Can write empty string to /test.txt",
		"/test.txt",
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File) error {
			if _, err := f.WriteAt([]byte(""), 0); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			wantContent := []byte{}
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.Read(gotContent)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid write length, got %v, want %v", gotLength, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid write content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
	},
	{
		"Can write small amount of data to /test.txt if seeking afterwards",
		"/test.txt",
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File) error {
			if _, err := f.WriteAt([]byte("Hello, world!"), 0); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}

			wantContent := []byte("Hello, world")
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.Read(gotContent)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid write length, got %v, want %v", gotLength, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid write content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
	},
	{
		"Can write small amount of data to /test.txt if not seeking afterwards",
		"/test.txt",
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File) error {
			if _, err := f.WriteAt([]byte(""), 0); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}

			wantContent := []byte("")
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.Read(gotContent)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid write length, got %v, want %v", gotLength, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid write content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
	},
	{
		"Can write 30 MB amount of data at once to start of /test.txt",
		"/test.txt",
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File) error {
			buf := make([]byte, 32800768)

			r := newDeterministicReader(1000)
			if _, err := r.Read(buf); err != nil {
				return err
			}

			if _, err := f.WriteAt(buf, 0); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}

			wantHash := "sZ4wO0XI9RAwmM2bFMyhTVcRMJZ9kcWIehLlvF10NKvn7-GakaTo7oee-qTVArFoLtZDwyAev7-zugiw81U31g=="
			wantLength := int64(32800768)

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, f)
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid read length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid read hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		false,
	},
	{
		"Can write 30 MB amount of data in chunks to start of /test.txt",
		"/test.txt",
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File) error {
			r := newDeterministicReader(30000)

			curr := int64(0)
			chunkSize := 1024
			for {
				buf := make([]byte, chunkSize)

				if _, err := r.Read(buf); err != nil {
					if err == io.EOF {
						break
					}

					return err
				}

				n, err := f.WriteAt(buf, curr)
				if err != nil {
					return err
				}

				curr += int64(n)
			}

			return nil
		},
		func(f afero.File) error {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}

			wantHash := "rKXm2Rt1HzSEyLcA7ifXl0BJke75vxNZVoa0v9Z_ltDWfvURoB1z1w9muLf73ajUa6k5Q-7vGOIA0jEpetz0hg=="
			wantLength := int64(30721024)

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, f)
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid read length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid read hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		false,
	},
	{
		"Can write 30 MB amount of data in chunks to start of non-empty /test.txt",
		"/test.txt",
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			r := newDeterministicReader(3000)

			chunkSize := 1024
			for {
				buf := make([]byte, chunkSize)

				if _, err := r.Read(buf); err != nil {
					if err == io.EOF {
						break
					}

					return err
				}

				if _, err := file.Write(buf); err != nil {
					return err
				}
			}

			return file.Close()
		},
		func(f afero.File) error {
			r := newDeterministicReader(30000)

			curr := int64(0)
			chunkSize := 1024
			for {
				buf := make([]byte, chunkSize)

				if _, err := r.Read(buf); err != nil {
					if err == io.EOF {
						break
					}

					return err
				}

				n, err := f.WriteAt(buf, curr)
				if err != nil {
					return err
				}

				curr += int64(n)
			}

			return nil
		},
		func(f afero.File) error {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}

			wantHash := "rKXm2Rt1HzSEyLcA7ifXl0BJke75vxNZVoa0v9Z_ltDWfvURoB1z1w9muLf73ajUa6k5Q-7vGOIA0jEpetz0hg=="
			wantLength := int64(30721024)

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, f)
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid read length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid read hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		false,
	},
	{
		"Can write 30 MB amount of data in chunks to end of non-empty /test.txt",
		"/test.txt",
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			r := newDeterministicReader(30000)

			chunkSize := 1024
			for {
				buf := make([]byte, chunkSize)

				if _, err := r.Read(buf); err != nil {
					if err == io.EOF {
						break
					}

					return err
				}

				if _, err := file.Write(buf); err != nil {
					return err
				}
			}

			return file.Close()
		},
		func(f afero.File) error {
			r := newDeterministicReader(30000)

			curr := int64(30721024)
			chunkSize := 1024
			for {
				buf := make([]byte, chunkSize)

				if _, err := r.Read(buf); err != nil {
					if err == io.EOF {
						break
					}

					return err
				}

				n, err := f.WriteAt(buf, curr)
				if err != nil {
					return err
				}

				curr += int64(n)
			}

			return nil
		},
		func(f afero.File) error {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}

			wantHash := "729-ysgoUbnxEh45qqheOgZr3nwialhQGnNEIRewc3dLZIgqGqxDipUndZVH6xzmC_j_LwAOHCeVobYNuOcAIg=="
			wantLength := int64(61442048)

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, f)
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid read length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid read hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		false,
	},
	{
		"Can write 300 MB amount of data in chunks to start of /test.txt",
		"/test.txt",
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			return file.Close()
		},
		func(f afero.File) error {
			r := newDeterministicReader(300000)

			curr := int64(0)
			chunkSize := 1024
			for {
				buf := make([]byte, chunkSize)

				if _, err := r.Read(buf); err != nil {
					if err == io.EOF {
						break
					}

					return err
				}

				n, err := f.WriteAt(buf, curr)
				if err != nil {
					return err
				}

				curr += int64(n)
			}

			return nil
		},
		func(f afero.File) error {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}

			wantHash := "CugMuX6FzQQ8rkSD2etvKlqjrghwxROsmRxdNmry4OR0SEGmUsHrRTXjSsmxrUZUDTxOdhz7gEBktbiFE0A30Q=="
			wantLength := int64(307201024)

			hasher := sha512.New()
			gotLength, err := io.Copy(hasher, f)
			if err != nil {
				return err
			}
			gotHash := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

			if gotLength != wantLength {
				return fmt.Errorf("invalid read length, got %v, want %v", gotLength, wantLength)
			}

			if gotHash != wantHash {
				return fmt.Errorf("invalid read hash, got %v, want %v", gotHash, wantHash)
			}

			return nil
		},
		true,
		true,
		true,
	},
	{
		"Can not write to symlink to /",
		"/existingsymlink",
		true,
		func(f symFs) error {
			if err := f.SymlinkIfPossible("/", "/existingsymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			if _, err := f.WriteAt([]byte("Hello, world!"), 0); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error { return nil },
		true,
		true,
		false,
	},
	{
		"Can not write to symlink to /mydir",
		"/existingsymlink",
		true,
		func(f symFs) error {
			if err := f.Mkdir("/mydir", os.ModePerm); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/mydir", "/existingsymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			if _, err := f.WriteAt([]byte("Hello, world!"), 0); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error { return nil },
		true,
		true,
		false,
	},
	{
		"Can write empty string to symlink to /test.txt",
		"/existingsymlink",
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			if err := file.Close(); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			if _, err := f.WriteAt([]byte(""), 0); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			wantContent := []byte{}
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.Read(gotContent)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid write length, got %v, want %v", gotLength, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid write content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
	},
	{
		"Can write small amount of data to symlink to /test.txt if seeking afterwards",
		"/existingsymlink",
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			if err := file.Close(); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			if _, err := f.WriteAt([]byte("Hello, world!"), 0); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}

			wantContent := []byte("Hello, world")
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.Read(gotContent)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid write length, got %v, want %v", gotLength, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid write content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
	},
	{
		"Can write small amount of data to symlink to /test.txt if not seeking afterwards",
		"/existingsymlink",
		false,
		func(f symFs) error {
			file, err := f.Create("/test.txt")
			if err != nil {
				return err
			}

			if err := file.Close(); err != nil {
				return err
			}

			if err := f.SymlinkIfPossible("/test.txt", "/existingsymlink"); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			if _, err := f.WriteAt([]byte(""), 0); err != nil {
				return err
			}

			return nil
		},
		func(f afero.File) error {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}

			wantContent := []byte("")
			gotContent := make([]byte, len(wantContent))

			wantLength := len(wantContent)
			gotLength, err := f.Read(gotContent)
			if err != io.EOF {
				return err
			}

			if wantLength != gotLength {
				return fmt.Errorf("invalid write length, got %v, want %v", gotLength, wantLength)
			}

			if string(wantContent) != string(gotContent) {
				return fmt.Errorf("invalid write content, got %v, want %v", gotContent, wantContent)
			}

			return nil
		},
		true,
		true,
		false,
	},
}

func TestFile_WriteAt(t *testing.T) {
	for _, tt := range writeAtTests {
		tt := tt

		runTestForAllFss(t, tt.name, true, tt.withCache, tt.withOsFs, func(t *testing.T, fs fsConfig) {
			if tt.large && testing.Short() {
				return
			}

			symFs, ok := fs.fs.(symFs)
			if !ok {
				return
			}

			if err := tt.prepare(symFs); err != nil {
				t.Errorf("%v prepare() error = %v", symFs.Name(), err)

				return
			}

			file, err := symFs.OpenFile(tt.open, os.O_RDWR, os.ModePerm)
			if err != nil && tt.wantErr {
				return
			}

			if err != nil {
				t.Errorf("%v open() error = %v", symFs.Name(), err)

				return
			}

			err = tt.apply(file)
			if (err != nil) != tt.wantErr {
				t.Errorf("%v.Write() error = %v", symFs.Name(), err)

				return
			}

			if err == nil {
				if err := tt.check(file); (err != nil) != tt.wantErr {
					t.Errorf("%v check() error = %v", fs.fs.Name(), err)

					return
				}
			}
		})
	}
}

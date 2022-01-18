package fs

import (
	"errors"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/spf13/afero"
)

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

type fileStatArgs struct {
	name string
}

var fileStatTests = []struct {
	name      string
	open      string
	wantErr   bool
	prepare   func(afero.Fs) error
	check     func(os.FileInfo) error
	withCache bool
	withOsFs  bool
}{
	{
		"Can stat /",
		"/",
		false,
		func(f afero.Fs) error { return nil },
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
		func(f afero.Fs) error {
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
		func(f afero.Fs) error {
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
		func(f afero.Fs) error {
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
		func(f afero.Fs) error {
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
}

func TestSTFS_FileStat(t *testing.T) {
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
	prepare   func(afero.Fs) error
	check     func([]os.FileInfo) error
	withCache bool
	withOsFs  bool
}{
	{
		"Can readdir all in / if there are no children",
		"/",
		readdirArgs{-1},
		false,
		func(f afero.Fs) error { return nil },
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
		func(f afero.Fs) error { return nil },
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
		func(f afero.Fs) error {
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

			wantLength := len(f)
			gotLength := 1
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", wantLength, gotLength)
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
		func(f afero.Fs) error {
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
				name, ok := wantNames[info.Name()]
				if !ok {
					return fmt.Errorf("could not find file or directory with name %v", name)
				}
			}

			wantLength := len(f)
			gotLength := 3
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", wantLength, gotLength)
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
		func(f afero.Fs) error {
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
				name, ok := wantNames[info.Name()]
				if !ok {
					return fmt.Errorf("could not find file or directory with name %v", name)
				}
			}

			wantLength := len(f)
			gotLength := 3
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", wantLength, gotLength)
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
		func(f afero.Fs) error {
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
				name, ok := wantNames[info.Name()]
				if !ok {
					return fmt.Errorf("could not find file or directory with name %v", name)
				}
			}

			wantLength := len(f)
			gotLength := 2
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", wantLength, gotLength)
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
		func(f afero.Fs) error {
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
				name, ok := wantNames[info.Name()]
				if !ok {
					return fmt.Errorf("could not find file or directory with name %v", name)
				}
			}

			wantLength := len(f)
			gotLength := 2
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", wantLength, gotLength)
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
		func(f afero.Fs) error {
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
				name, ok := wantNames[info.Name()]
				if !ok {
					return fmt.Errorf("could not find file or directory with name %v", name)
				}
			}

			wantLength := len(f)
			gotLength := 3
			if wantLength != gotLength {
				return fmt.Errorf("invalid amount of children, got %v, want %v", wantLength, gotLength)
			}

			return nil
		},
		true,
		true,
	},
}

func TestSTFS_Readdir(t *testing.T) {
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

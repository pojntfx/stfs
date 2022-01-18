package fs

import (
	"fmt"
	"os"
	"testing"
)

var filenameTests = []struct {
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
	for _, tt := range filenameTests {
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

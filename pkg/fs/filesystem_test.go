package fs

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pojntfx/stfs/internal/logging"
	"github.com/pojntfx/stfs/pkg/cache"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/operations"
	"github.com/pojntfx/stfs/pkg/persisters"
	"github.com/pojntfx/stfs/pkg/tape"
	"github.com/spf13/afero"
)

func createTestFss() (filesystems []afero.Fs, cleanup func() error, err error) {
	tmp, err := os.MkdirTemp(os.TempDir(), "stfs-test-*")
	if err != nil {
		return nil, nil, err
	}

	drive := filepath.Join(tmp, "drive.tar")
	recordSize := 20
	metadata := filepath.Join(tmp, "metadata.sqlite")
	writeCache := filepath.Join(tmp, "write-cache")
	osfsDir := filepath.Join(tmp, "osfs")

	tm := tape.NewTapeManager(
		drive,
		recordSize,
		false,
	)

	metadataPersister := persisters.NewMetadataPersister(metadata)
	if err := metadataPersister.Open(); err != nil {
		return nil, nil, err
	}

	l := logging.NewJSONLogger(4)

	metadataConfig := config.MetadataConfig{
		Metadata: metadataPersister,
	}
	pipeConfig := config.PipeConfig{
		Compression: config.NoneKey,
		Encryption:  config.NoneKey,
		Signature:   config.NoneKey,
		RecordSize:  recordSize,
	}
	backendConfig := config.BackendConfig{
		GetWriter:   tm.GetWriter,
		CloseWriter: tm.Close,

		GetReader:   tm.GetReader,
		CloseReader: tm.Close,

		GetDrive:   tm.GetDrive,
		CloseDrive: tm.Close,
	}
	readCryptoConfig := config.CryptoConfig{}

	readOps := operations.NewOperations(
		backendConfig,
		metadataConfig,

		pipeConfig,
		readCryptoConfig,

		func(event *config.HeaderEvent) {
			l.Debug("Header read", event)
		},
	)
	writeOps := operations.NewOperations(
		backendConfig,
		metadataConfig,

		pipeConfig,
		config.CryptoConfig{},

		func(event *config.HeaderEvent) {
			l.Debug("Header write", event)
		},
	)

	stfs := NewSTFS(
		readOps,
		writeOps,

		config.MetadataConfig{
			Metadata: metadataPersister,
		},

		config.CompressionLevelFastest,
		func() (cache.WriteCache, func() error, error) {
			return cache.NewCacheWrite(
				writeCache,
				config.WriteCacheTypeFile,
			)
		},
		false,
		false,

		func(hdr *config.Header) {
			l.Trace("Header transform", hdr)
		},
		l,
	)

	root, err := stfs.Initialize("/", os.ModePerm)
	if err != nil {
		return nil, nil, err
	}

	fs, err := cache.NewCacheFilesystem(
		stfs,
		root,
		config.NoneKey,
		0,
		"",
	)
	if err != nil {
		return nil, nil, err
	}

	if err := os.MkdirAll(osfsDir, os.ModePerm); err != nil {
		return nil, nil, err
	}

	return []afero.Fs{
			fs,
			afero.NewBasePathFs(afero.NewOsFs(), osfsDir),
		},
		func() error {
			return os.RemoveAll(tmp)
		},
		nil
}

func getTestNameForFs(testName string, fsName string) string {
	return testName + " (" + fsName + ")"
}

func TestSTFS_Name(t *testing.T) {
	filesystems, cleanup, err := createTestFss()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	tests := []struct {
		name string
		f    []afero.Fs
		want string
	}{
		{
			"Returns correct file system name",
			[]afero.Fs{filesystems[1]},
			"BasePathFs",
		},
		{
			"Returns correct file system name",
			[]afero.Fs{filesystems[0]},
			"STFS",
		},
	}

	for _, tt := range tests {
		for _, f := range tt.f {
			t.Run(getTestNameForFs(tt.name, f.Name()), func(t *testing.T) {
				if got := f.Name(); got != tt.want {
					t.Errorf("%v.Name() = %v, want %v", f.Name(), got, tt.want)
				}
			})
		}
	}
}

func TestSTFS_Create(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"Can create /test.txt",
			args{"/test.txt"},
			false,
		},
		// FIXME: STFS can create file in non-existent directory, which should not be possible
		// {
		// 	"Can not create /nonexistent/test.txt",
		// 	args{"/nonexistent/test.txt"},
		// 	true,
		// },
		// FIXME: STFS can create `/` file even if / exists
		// {
		// 	"Can create /",
		// 	args{"/"},
		// 	true,
		// },
	}

	for _, tt := range tests {
		filesystems, cleanup, err := createTestFss()
		if err != nil {
			panic(err)
		}
		defer cleanup()

		for _, f := range filesystems {
			t.Run(getTestNameForFs(tt.name, f.Name()), func(t *testing.T) {
				file, err := f.Create(tt.args.name)
				if (err != nil) != tt.wantErr {
					t.Errorf("%v.Create() error = %v, wantErr %v", f.Name(), err, tt.wantErr)

					return
				}

				want, err := f.Stat(tt.args.name)
				if err != nil {
					t.Errorf("%v.Stat() error = %v, wantErr %v", f.Name(), err, tt.wantErr)

					return
				}

				got, err := f.Stat(file.Name())
				if err != nil {
					t.Errorf("%v.Stat() error = %v, wantErr %v", f.Name(), err, tt.wantErr)

					return
				}

				if !reflect.DeepEqual(got, want) {
					t.Errorf("%v.Create().Name() = %v, want %v", f.Name(), got, want)

					return
				}
			})
		}
	}
}

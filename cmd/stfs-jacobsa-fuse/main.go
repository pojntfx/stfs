package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
)

type FileSystem struct {
	fuseutil.NotImplementedFileSystem
}

func (f *FileSystem) GetInodeAttributes(ctx context.Context, op *fuseops.GetInodeAttributesOp) error {
	op.Attributes = fuseops.InodeAttributes{
		Nlink: 1,
		Mode:  0555 | os.ModeDir,
	}

	log.Println(op.Inode)

	return nil
}

func main() {
	mountpoint := flag.String("mountpoint", ".", "Directory to mount the FUSE in")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")

	flag.Parse()

	srv, err := fuse.Mount(
		*mountpoint,
		fuseutil.NewFileSystemServer(&FileSystem{}),
		&fuse.MountConfig{
			DebugLogger: func() *log.Logger {
				if *verbose {
					return log.Default()
				}

				return nil
			}(),
		},
	)
	if err != nil {
		panic(err)
	}

	if err := srv.Join(context.Background()); err != nil {
		panic(err)
	}
}

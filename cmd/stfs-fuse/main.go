package main

import (
	"context"
	"flag"
	"log"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type Root struct {
	fs.Inode
}

func (r *Root) OnAdd(ctx context.Context) {
	r.AddChild(
		"hello_world.txt",
		r.NewPersistentInode(
			ctx,
			&fs.MemRegularFile{
				Data: []byte("Hello, world!"),
				Attr: fuse.Attr{
					Mode: 0644,
				},
			},
			fs.StableAttr{},
		),
		false,
	)
}

func main() {
	mountpoint := flag.String("mountpoint", ".", "Directory to mount the FUSE in")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")

	flag.Parse()

	server, err := fs.Mount(
		*mountpoint,
		&Root{},
		&fs.Options{
			Logger: func() *log.Logger {
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
	defer server.Unmount()

	server.Wait()
}

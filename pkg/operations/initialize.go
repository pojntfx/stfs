package operations

import (
	"archive/tar"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pojntfx/stfs/pkg/config"
)

func (o *Operations) Initialize(
	name string,
	perm os.FileMode,
	compressionLevel string,
) error {
	o.diskOperationLock.Lock()
	defer o.diskOperationLock.Unlock()

	usr, err := user.Current()
	if err != nil {
		return err
	}

	gid, err := strconv.Atoi(usr.Gid)
	if err != nil {
		// Some OSes like i.e. Windows don't support numeric GIDs, so use 0 instead
		gid = 0
	}

	uid, err := strconv.Atoi(usr.Uid)
	if err != nil {
		// Some OSes like i.e. Windows don't support numeric UIDs, so use 0 instead
		uid = 0
	}

	groups, err := usr.GroupIds()
	if err != nil {
		return err
	}

	gname := ""
	if len(groups) >= 1 {
		gname = groups[0]
	}

	typeflag := tar.TypeDir

	hdr := &tar.Header{
		Typeflag: byte(typeflag),

		Name: name,

		Mode:  int64(perm),
		Uid:   uid,
		Gid:   gid,
		Uname: usr.Username,
		Gname: gname,

		ModTime: time.Now(),
	}

	done := false
	if _, err := o.archive(
		func() (config.FileConfig, error) {
			// Exit after the first write
			if done {
				return config.FileConfig{}, io.EOF
			}
			done = true

			return config.FileConfig{
				GetFile: nil, // Not required as we never replace
				Info:    hdr.FileInfo(),
				Path:    filepath.ToSlash(name),
				Link:    filepath.ToSlash(hdr.Linkname),
			}, nil
		},
		compressionLevel,
		true,
		true,
	); err != nil {
		return err
	}

	return nil
}

package converters

import (
	"archive/tar"
	"encoding/json"

	models "github.com/pojntfx/stfs/pkg/db/sqlite/models/metadata"
)

func DBHeaderToTarHeader(dbhdr *models.Header) (*tar.Header, error) {
	paxRecords := map[string]string{}
	if err := json.Unmarshal([]byte(dbhdr.Paxrecords), &paxRecords); err != nil {
		return nil, err
	}

	hdr := &tar.Header{
		Typeflag:   byte(dbhdr.Typeflag),
		Name:       dbhdr.Name,
		Linkname:   dbhdr.Linkname,
		Size:       dbhdr.Size,
		Mode:       dbhdr.Mode,
		Uid:        int(dbhdr.UID),
		Gid:        int(dbhdr.Gid),
		Uname:      dbhdr.Uname,
		Gname:      dbhdr.Gname,
		ModTime:    dbhdr.Modtime,
		AccessTime: dbhdr.Accesstime,
		ChangeTime: dbhdr.Changetime,
		Devmajor:   dbhdr.Devmajor,
		Devminor:   dbhdr.Devminor,
		PAXRecords: paxRecords,
		Format:     tar.Format(dbhdr.Format),
	}

	return hdr, nil
}

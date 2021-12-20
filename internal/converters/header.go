package converters

import (
	"archive/tar"
	"encoding/json"

	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
)

func DBHeaderToTarHeader(dbhdr *models.Header) (*tar.Header, error) {
	paxRecords := map[string]string{}
	if err := json.Unmarshal([]byte(dbhdr.Paxrecords), &paxRecords); err != nil {
		return nil, err
	}
	if paxRecords == nil {
		paxRecords = map[string]string{}
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

func TarHeaderToDBHeader(record, lastKnownRecord, block, lastKnownBlock int64, tarhdr *tar.Header) (*models.Header, error) {
	paxRecords, err := json.Marshal(tarhdr.PAXRecords)
	if err != nil {
		return nil, err
	}

	hdr := models.Header{
		Record:          record,
		Lastknownrecord: lastKnownRecord,
		Block:           block,
		Lastknownblock:  lastKnownBlock,
		Typeflag:        int64(tarhdr.Typeflag),
		Name:            tarhdr.Name,
		Linkname:        tarhdr.Linkname,
		Size:            tarhdr.Size,
		Mode:            tarhdr.Mode,
		UID:             int64(tarhdr.Uid),
		Gid:             int64(tarhdr.Gid),
		Uname:           tarhdr.Uname,
		Gname:           tarhdr.Gname,
		Modtime:         tarhdr.ModTime,
		Accesstime:      tarhdr.AccessTime,
		Changetime:      tarhdr.ChangeTime,
		Devmajor:        tarhdr.Devmajor,
		Devminor:        tarhdr.Devminor,
		Paxrecords:      string(paxRecords),
		Format:          int64(tarhdr.Format),
	}

	return &hdr, nil
}

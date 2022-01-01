package converters

import (
	"archive/tar"
	"encoding/json"

	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/pkg/config"
)

func ConfigHeaderToDBHeader(confighdr *config.Header) *models.Header {
	return &models.Header{
		Record:          confighdr.Record,
		Lastknownrecord: confighdr.Lastknownrecord,
		Block:           confighdr.Block,
		Lastknownblock:  confighdr.Lastknownblock,
		Typeflag:        confighdr.Typeflag,
		Name:            confighdr.Name,
		Linkname:        confighdr.Linkname,
		Size:            confighdr.Size,
		Mode:            confighdr.Mode,
		UID:             confighdr.UID,
		Gid:             confighdr.Gid,
		Uname:           confighdr.Uname,
		Gname:           confighdr.Gname,
		Modtime:         confighdr.Modtime,
		Accesstime:      confighdr.Accesstime,
		Changetime:      confighdr.Changetime,
		Devmajor:        confighdr.Devmajor,
		Devminor:        confighdr.Devminor,
		Paxrecords:      confighdr.Paxrecords,
		Format:          confighdr.Format,
		Deleted:         confighdr.Deleted,
	}
}

func DBHeaderToConfigHeader(dbhdr *models.Header) *config.Header {
	return &config.Header{
		Record:          dbhdr.Record,
		Lastknownrecord: dbhdr.Lastknownrecord,
		Block:           dbhdr.Block,
		Lastknownblock:  dbhdr.Lastknownblock,
		Typeflag:        dbhdr.Typeflag,
		Name:            dbhdr.Name,
		Linkname:        dbhdr.Linkname,
		Size:            dbhdr.Size,
		Mode:            dbhdr.Mode,
		UID:             dbhdr.UID,
		Gid:             dbhdr.Gid,
		Uname:           dbhdr.Uname,
		Gname:           dbhdr.Gname,
		Modtime:         dbhdr.Modtime,
		Accesstime:      dbhdr.Accesstime,
		Changetime:      dbhdr.Changetime,
		Devmajor:        dbhdr.Devmajor,
		Devminor:        dbhdr.Devminor,
		Paxrecords:      dbhdr.Paxrecords,
		Format:          dbhdr.Format,
		Deleted:         dbhdr.Deleted,
	}
}

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

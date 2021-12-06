package formatting

import (
	"archive/tar"
	"encoding/csv"
	"fmt"
	"os"
	"time"
)

var (
	TARHeaderCSV = []string{
		"record", "lastknownrecord", "block", "lastknownblock", "typeflag", "name", "linkname", "size", "mode", "uid", "gid", "uname", "gname", "modtime", "accesstime", "changetime", "devmajor", "devminor", "paxrecords", "format",
	}
)

func PrintCSV(input []string) error {
	w := csv.NewWriter(os.Stdout)

	return w.WriteAll([][]string{input})
}

func GetTARHeaderAsCSV(record, lastKnownRecord, block, lastKnownBlock int64, hdr *tar.Header) []string {
	return []string{
		fmt.Sprintf("%v", record), fmt.Sprintf("%v", lastKnownRecord), fmt.Sprintf("%v", block), fmt.Sprintf("%v", lastKnownBlock), fmt.Sprintf("%v", hdr.Typeflag), hdr.Name, hdr.Linkname, fmt.Sprintf("%v", hdr.Size), fmt.Sprintf("%v", hdr.Mode), fmt.Sprintf("%v", hdr.Uid), fmt.Sprintf("%v", hdr.Gid), fmt.Sprintf("%v", hdr.Uname), fmt.Sprintf("%v", hdr.Gname), hdr.ModTime.Format(time.RFC3339), hdr.AccessTime.Format(time.RFC3339), hdr.ChangeTime.Format(time.RFC3339), fmt.Sprintf("%v", hdr.Devmajor), fmt.Sprintf("%v", hdr.Devminor), fmt.Sprintf("%v", hdr.PAXRecords), fmt.Sprintf("%v", hdr.Format),
	}
}

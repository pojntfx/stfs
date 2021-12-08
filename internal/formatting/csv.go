package formatting

import (
	"encoding/csv"
	"os"
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

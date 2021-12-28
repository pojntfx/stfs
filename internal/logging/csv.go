package logging

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/pkg/config"
)

var (
	tarHeaderCSV = []string{
		"record", "lastknownrecord", "block", "lastknownblock", "typeflag", "name", "linkname", "size", "mode", "uid", "gid", "uname", "gname", "modtime", "accesstime", "changetime", "devmajor", "devminor", "paxrecords", "format",
	}
	tarHeaderEventCSV = append([]string{"type", "indexed"}, tarHeaderCSV...)
)

func headerToCSV(hdr *models.Header) []string {
	return []string{
		fmt.Sprintf("%v", hdr.Record), fmt.Sprintf("%v", hdr.Lastknownrecord), fmt.Sprintf("%v", hdr.Block), fmt.Sprintf("%v", hdr.Lastknownblock), fmt.Sprintf("%v", hdr.Typeflag), hdr.Name, hdr.Linkname, fmt.Sprintf("%v", hdr.Size), fmt.Sprintf("%v", hdr.Mode), fmt.Sprintf("%v", hdr.UID), fmt.Sprintf("%v", hdr.Gid), fmt.Sprintf("%v", hdr.Uname), fmt.Sprintf("%v", hdr.Gname), hdr.Modtime.Format(time.RFC3339), hdr.Accesstime.Format(time.RFC3339), hdr.Changetime.Format(time.RFC3339), fmt.Sprintf("%v", hdr.Devmajor), fmt.Sprintf("%v", hdr.Devminor), fmt.Sprintf("%v", hdr.Paxrecords), fmt.Sprintf("%v", hdr.Format),
	}
}

func headerEventToCSV(event *config.HeaderEvent) []string {
	return append([]string{event.Type, fmt.Sprintf("%v", event.Indexed)}, headerToCSV(event.Header)...)
}

type CSVLogger struct {
	n int
}

func NewCSVLogger() *CSVLogger {
	return &CSVLogger{}
}

func (l *CSVLogger) PrintHeader(hdr *models.Header) {
	w := csv.NewWriter(os.Stdout)

	if l.n <= 0 {
		_ = w.Write(tarHeaderCSV) // Errors are ignored for compatibility with traditional logging APIs
	}

	_ = w.Write(headerToCSV(hdr)) // Errors are ignored for compatibility with traditional logging APIs

	w.Flush()

	l.n++
}

func (l *CSVLogger) PrintHeaderEvent(event *config.HeaderEvent) {
	w := csv.NewWriter(os.Stdout)

	if l.n <= 0 {
		_ = w.Write(tarHeaderEventCSV) // Errors are ignored for compatibility with traditional logging APIs
	}

	_ = w.Write(headerEventToCSV(event)) // Errors are ignored for compatibility with traditional logging APIs

	w.Flush()

	l.n++
}

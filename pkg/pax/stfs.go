package pax

import "errors"

const (
	STFSRecordVersion  = "STFS.Version"
	STFSRecordVersion1 = "1"

	STFSRecordAction       = "STFS.Action"
	STFSRecordActionCreate = "CREATE"
	STFSRecordActionDelete = "DELETE"

	STFSReplacesContent      = "STFS.ReplacesContent"
	STFSReplacesContentTrue  = "true"
	STFSReplacesContentFalse = "false"

	STFSReplacesName = "STFS.ReplacesName"
)

var (
	ErrUnsupportedVersion = errors.New("unsupported STFS version")
	ErrUnsupportedAction  = errors.New("unsupported STFS action")
)

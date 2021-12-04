package pax

import "errors"

const (
	STFSPrefix = "STFS."

	STFSRecordVersion  = STFSPrefix + "Version"
	STFSRecordVersion1 = "1"

	STFSRecordAction       = STFSPrefix + "Action"
	STFSRecordActionCreate = "CREATE"
	STFSRecordActionDelete = "DELETE"
	STFSRecordActionUpdate = "UPDATE"

	STFSRecordReplacesContent      = STFSPrefix + "ReplacesContent"
	STFSRecordReplacesContentTrue  = "true"
	STFSRecordReplacesContentFalse = "false"

	STFSRecordReplacesName = STFSPrefix + "ReplacesName"

	STFSRecordUncompressedSize = STFSPrefix + "UncompressedSize"

	STFSRecordSignature = STFSPrefix + "Signature"

	STFSEmbeddedHeader = STFSPrefix + "EmbeddedHeader"
)

var (
	ErrUnsupportedVersion = errors.New("unsupported STFS version")
	ErrUnsupportedAction  = errors.New("unsupported STFS action")
)

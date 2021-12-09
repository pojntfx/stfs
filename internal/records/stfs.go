package records

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

	STFSRecordEmbeddedHeader = STFSPrefix + "EmbeddedHeader"
)

-- +migrate Up
create table headers (
	-- Typeflag is the type of header entry.
	-- The zero value is automatically promoted to either TypeReg or TypeDir
	-- depending on the presence of a trailing slash in Name.
	typeflag integer not null,
	-- Name of file entry
	name text not null primary key,
	-- Target name of link (valid for TypeLink or TypeSymlink)
	linkname text not null,
	-- Logical file size in bytes
	size integer not null,
	-- Permission and mode bits
	mode integer not null,
	-- User ID of owner
	uid integer not null,
	-- Group ID of owner
	gid integer not null,
	-- User name of owner
	uname text not null,
	-- Group name of owner
	gname text not null,
	-- If the Format is unspecified, then Writer.WriteHeader rounds ModTime
	-- to the nearest second and ignores the AccessTime and ChangeTime fields.
	--
	-- To use AccessTime or ChangeTime, specify the Format as PAX or GNU.
	-- To use sub-second resolution, specify the Format as PAX.
	-- Modification time
	modtime date not null,
	-- Access time (requires either PAX or GNU support)
	accesstime date not null,
	-- Change time (requires either PAX or GNU support)
	changetime date not null,
	-- Major device number (valid for TypeChar or TypeBlock)
	devmajor integer not null,
	-- Minor device number (valid for TypeChar or TypeBlock)
	devminor integer not null,
	-- PAXRecords is a map of PAX extended header records.
	--
	-- User-defined records should have keys of the following form:
	--	VENDOR.keyword
	-- Where VENDOR is some namespace in all uppercase, and keyword may
	-- not contain the '=' character (e.g., "GOLANG.pkg.version").
	-- The key and value should be non-empty UTF-8 strings.
	--
	-- When Writer.WriteHeader is called, PAX records derived from the
	-- other fields in Header take precedence over PAXRecords.
	paxrecords text not null,
	-- Format specifies the format of the tar header.
	--
	-- This is set by Reader.Next as a best-effort guess at the format.
	-- Since the Reader liberally reads some non-compliant files,
	-- it is possible for this to be FormatUnknown.
	--
	-- If the format is unspecified when Writer.WriteHeader is called,
	-- then it uses the first format (in the order of USTAR, PAX, GNU)
	-- capable of encoding this Header (see Format).
	format text not null
);
-- +migrate Down
drop table headers;
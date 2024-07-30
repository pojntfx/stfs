# STFS

Simple Tape File System (STFS), a file system for tapes and tar files.

[![hydrun CI](https://github.com/pojntfx/stfs/actions/workflows/hydrun.yaml/badge.svg)](https://github.com/pojntfx/stfs/actions/workflows/hydrun.yaml)
![Go Version](https://img.shields.io/badge/go%20version-%3E=1.21-61CFDD.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/pojntfx/stfs.svg)](https://pkg.go.dev/github.com/pojntfx/stfs)
[![Matrix](https://img.shields.io/matrix/stfs:matrix.org)](https://matrix.to/#/#stfs:matrix.org?via=matrix.org)

## Overview

STFS is a filesystem that brings tapes and tar files into the 21st century.

It enables you to:

- **Use a tape or tar file like a regular disk**: STFS uses the robust `tar` format and tape technology to provide a fully features filesystem. This makes such storage solutions much for accessible and manageable, while also significantly increasing the performance of everyday operations such as listing and searching for files by using a SQLite-based on-disk index.
- **Archive data securely**: The integrated transparent, asymmetrical encryption and signature support makes it possible to use almost any tape as a regulations compliant storage medium, while still being able to take advantage of all the benefits of tapes like reduced cost and increased reliability.
- **Compress data efficiently**: By leveraging the embedded compression functionality, it is possible to store even more data on tapes without sacrificing the user experience.
- **Recover data in unexpected scenarios**: Even if sudden power drops happen, the drive fails, the index gets corrupted or STFS stops being maintained, your data is secure. Because it is based on open standards such as `tar`, SQLite, `zstandard`, PGP and others, it is possible to extract your data even if STFS's integrated recovery tools don't suffice.
- **Build your own storage solution**: In addition to its own, optimized APIs, STFS provides a [`afero.FS implementation`](https://github.com/spf13/afero). This makes embedding STFS and accessing files on a tape or in a tar file through Go easy.

## Installation

Static binaries are available on [GitHub releases](https://github.com/pojntfx/stfs/releases).

On Linux, you can install them like so:

```shell
$ curl -L -o /tmp/stfs "https://github.com/pojntfx/stfs/releases/latest/download/stfs.linux-$(uname -m)"
$ sudo install /tmp/stfs /usr/local/bin
```

On macOS, you can use the following:

```shell
$ curl -L -o /tmp/stfs "https://github.com/pojntfx/stfs/releases/latest/download/stfs.darwin-$(uname -m)"
$ sudo install /tmp/stfs /usr/local/bin
```

On Windows, the following should work (using PowerShell as administrator):

```shell
PS> Invoke-WebRequest https://github.com/pojntfx/stfs/releases/latest/download/stfs.windows-x86_64.exe -OutFile \Windows\System32\stfs.exe
```

Note that only the Linux version supports reading from tape drives; macOS and Windows are limited to operating on tar files.

You can find binaries for more operating systems and architectures on [GitHub releases](https://github.com/pojntfx/stfs/releases).

## Tutorial

> Please note that this is only a short overview and does not explain all configuration options. To get more info on available commands or options, use `--help`.

### 1. Generating Keys with `stfs keygen`

While not strictly required, it is recommended to generate keys to sign and encrypt your data on tape. There are multiple methods available; `PGP` and [age](https://github.com/FiloSottile/age) for encryption, and `PGP` as well as [Minisign](https://github.com/aead/minisign) for signatures. In most cases, using age for encryption and PGP for signatures is the best option. To generate the appropriate keys, run the following; make sure to save the keys in a secure location and use a secure password:

```shell
$ stfs keygen --encryption age --password mysecureencryptionpassword --identity ~/.stfs-age.priv --recipient ~/.stfs-age.pub
$ stfs keygen --signature pgp --password mysecuresignaturepassword --identity ~/.stfs-pgp.priv --recipient ~/.stfs-pgp.pub
```

For more information, see the [key generation reference](#key-generation).

### 2. Serving a Tape Read-Write with `stfs serve ftp`

The simplest way to read or write to/from the tape (or tar file) is to use the integrated FTP server. To speed up operations, caching mechanisms and compression are available. For the write cache (`--cache-write-type`) the following types are available:

- `memory`: A simple in-memory cache; should not be used in most cases due to potential RAM exhaustion when adding large files
- `file`: A on-disk cache; this is recommended in most cases, especially if a SSD is available

For the read cache (`--cache-filesystem-type`), which is especially useful when working with many small files, similar types are available (`memory` and `dir`). `dir` uses a overlay filesystem to cache files in the directory specified with `--cache-dir`.

To further speed up IO-limited read/write operations, multiple compression options are available to be selected with `--compression` and can be tuned with `--compression-level`:

- `zstandard`: A Meta-led replacement for `gzip` with very high speeds and a very good compression ratio; this is recommended for most users
- `gzip`/`parallelgzip`: The GNU format commonly used in combination with `tar`, i.e. for `.tar.gz`; reasonably fast and with a good compression ratio
- `bzip2`/`parallelbzip2`: A reliable compression format with good speeds and a better compression ratio than `gzip`.
- `lz4`: Very fast, but at the cost of a lower compression ratio
- `brotli`: A Google-led compression format with good adoption on the web platform; very high compression ratio, very slow speeds

To serve a tape (or tar file), run the following (adjust the options accordingly):

```shell
# Use `-d /dev/nst0` for your primary tape drive instead
$ stfs serve ftp \
    -d ~/Downloads/drive.tar \
    -m ~/Downloads/metadata.sqlite \
    -e age \
    --encryption-identity ~/.stfs-age.priv \
    --encryption-recipient ~/.stfs-age.pub \
    --encryption-password mysecureencryptionpassword \
    -s pgp \
    --signature-identity ~/.stfs-pgp.priv \
    --signature-recipient ~/.stfs-pgp.pub \
    --signature-password mysecuresignaturepassword \
    --compression zstandard
{"time":1652646259,"level":"INFO","event":"FTP server listening","data":[{"laddr":":1337"}]}
{"time":1652646259,"level":"INFO","event":"Listening...","data":["address",{"IP":"::","Port":1337,"Zone":""}]}
{"time":1652646259,"level":"INFO","event":"Starting...","data":null}
```

You can now point your file manager (GNOME files on Linux, Windows Explorer on Windows and Finder on macOS all have support for it, but macOS is read-only) to `ftp://localhost:1337` and read/write files from the tape (or tape file).

For more information, see the [servers reference](#servers).

### 3. Serving a Tape Read-Only with `stfs serve http`

If you want to serve a tape (or tar file) read-only, using the integrated HTTP server is the best option. It inherits all the same options from [Serving a Tape Read-Write with `stfs serve ftp`](#2-serving-a-tape-read-write-with-stfs-serve-ftp), minus the write cache due to it being read-only. To use it, run:

```shell
# Use `-d /dev/nst0` for your primary tape drive instead
$ stfs serve http \
    -d ~/Downloads/drive.tar \
    -m ~/Downloads/metadata.sqlite \
    -e age \
    --identity ~/.stfs-age.priv \
    --password mysecureencryptionpassword \
    -s pgp \
    --recipient ~/.stfs-pgp.pub \
    --compression zstandard
{"time":1652653259,"level":"INFO","event":"HTTP server listening","data":[{"laddr":":1337"}]}
```

You can now point your web browser to `http://localhost:1337` and read files from the tape (or tape file).

For more information, see the [servers reference](#servers).

### 4. Using Optimized Operations with `stfs operation`

While the file system API is convenient because of its similarity to most filesystems, it also can't be used without a write cache. While this isn't an issue for most applications, it requires you to have a disk that is at least as large as the largest file you want to add to the tape. To get around these limitations, STFS also provides a `tar`-like interface for interacting with the tape. Please note that these operations should be used carefully, as the usual checks (such as checking if a parent directory exists before adding files to it) don't apply.

First, initialize an empty tape:

```shell
# Use `-d /dev/nst0` for your primary tape drive instead
$ stfs operation initialize \
    -d ~/Downloads/drive.tar \
    -m ~/Downloads/metadata.sqlite \
    -e age \
    --recipient ~/.stfs-age.pub \
    -s pgp \
    --identity ~/.stfs-pgp.priv \
    --password mysecuresignaturepassword \
    --compression zstandard
type,indexed,record,lastknownrecord,block,lastknownblock,typeflag,name,linkname,size,mode,uid,gid,uname,gname,modtime,accesstime,changetime,devmajor,devminor,paxrecords,format
archive,false,-1,-1,-1,-1,53,/,,0,511,1000,1000,pojntfx,1000,2022-05-16T22:24:13+02:00,0001-01-01T00:00:00Z,0001-01-01T00:00:00Z,0,0,null,4
archive,true,0,-1,0,-1,53,/,,0,511,1000,1000,pojntfx,1000,2022-05-16T22:24:13+02:00,0001-01-01T00:00:00Z,0001-01-01T00:00:00Z,0,0,null,4
```

You can now add files to it:

```shell
# Use `-d /dev/nst0` for your primary tape drive instead
$ stfs operation archive \
    -d ~/Downloads/drive.tar \
    -m ~/Downloads/metadata.sqlite \
    -e age \
    --recipient ~/.stfs-age.pub \
    -s pgp \
    --identity ~/.stfs-pgp.priv \
    --password mysecuresignaturepassword \
    --compression zstandard \
    .
# ...
archive,true,1480,-1,9,-1,48,pkg/tape/write.go,,1544,420,1000,1000,pojntfx,pojntfx,2022-05-15T23:41:54+02:00,2022-05-15T23:41:54+02:00,2022-05-15T23:41:54+02:00,0,0,"{""STFS.Signature"":""wnUEABYIACcFAmKCsyUJkGA0c/4XcV5qFqEEjWKRLHhppJ6S+ZJlYDRz/hdxXmoAAF5mAP95DKo/r136fL/SKuBwmxoMNfGZ+v61bwk/xcOBQk5vrwEAs07QV2RF6h/FME+/nXxjZrbBWmFWg8pC4IGdScnJbQ4="",""STFS.UncompressedSize"":""1544""}",4
archive,true,1480,-1,17,-1,53,pkg/utility,,0,509,1000,1000,pojntfx,pojntfx,2022-04-18T20:19:52+02:00,2022-05-15T23:36:33+02:00,2022-04-23T16:08:59+02:00,0,0,null,4
# ...
```

Full CRUD support is implemented, so you can `delete`, `move`, `restore` and `update` files like this as well. For example, to restore `pkg/tape/write.go`, run the following:

```shell
# Use `-d /dev/nst0` for your primary tape drive instead
$ stfs operation restore \
    -d ~/Downloads/drive.tar \
    -m ~/Downloads/metadata.sqlite \
    -e age \
    --identity ~/.stfs-age.priv \
    --password mysecureencryptionpassword \
    -s pgp \
    --recipient ~/.stfs-pgp.pub \
    --compression zstandard \
    --flatten \
    --from pkg/tape/write.go \
    --to write.go
type,indexed,record,lastknownrecord,block,lastknownblock,typeflag,name,linkname,size,mode,uid,gid,uname,gname,modtime,accesstime,changetime,devmajor,devminor,paxrecords,format
restore,true,1480,1480,9,9,48,/pkg/tape/write.go,,1544,420,1000,1000,pojntfx,pojntfx,2022-05-15T23:41:54+02:00,2022-05-15T23:41:54+02:00,2022-05-15T23:41:54+02:00,0,0,"{""STFS.Signature"":""wnUEABYIACcFAmKCsyUJkGA0c/4XcV5qFqEEjWKRLHhppJ6S+ZJlYDRz/hdxXmoAAF5mAP95DKo/r136fL/SKuBwmxoMNfGZ+v61bwk/xcOBQk5vrwEAs07QV2RF6h/FME+/nXxjZrbBWmFWg8pC4IGdScnJbQ4="",""STFS.UncompressedSize"":""1544""}",4
$ file write.go
write.go: ASCII text
```

For more information, see the [operations reference](#operations).

### 5. Managing the Index with `stfs inventory`

For similar reasons as described in [Using Optimized Operations with `stfs operation`](#4-using-optimized-operations-with-stfs-operation), it can make sense to take advantage of the index in order to quickly find a file or directory. For example, to list all files in the `pkg` directory, run the following:

```shell
$ stfs inventory list \
    -m ~/Downloads/metadata.sqlite \
    --name pkg
record,lastknownrecord,block,lastknownblock,typeflag,name,linkname,size,mode,uid,gid,uname,gname,modtime,accesstime,changetime,devmajor,devminor,paxrecords,format
1454,1454,14,14,53,/pkg/cache,,0,493,1000,1000,pojntfx,pojntfx,2022-05-15T23:41:54+02:00,2022-05-15T23:41:54+02:00,2022-05-15T23:41:54+02:00,0,0,null,4
# ...
```

It is also possible to search for a file by regex with `stfs inventory find`:

```shell
$ stfs inventory find \
    -m ~/Downloads/metadata.sqlite \
    --expression '(.*).yaml'
record,lastknownrecord,block,lastknownblock,typeflag,name,linkname,size,mode,uid,gid,uname,gname,modtime,accesstime,changetime,devmajor,devminor,paxrecords,format
118,118,11,11,48,/.github/workflows/hydrun.yaml,,3000,420,1000,1000,pojntfx,pojntfx,2022-05-15T23:41:54+02:00,2022-05-15T23:41:54+02:00,2022-05-15T23:41:54+02:00,0,0,"{""STFS.Signature"":""wnUEABYIACcFAmKCsyMJkGA0c/4XcV5qFqEEjWKRLHhppJ6S+ZJlYDRz/hdxXmoAAF9nAP98fVspytLtKjTATbtH7hoVaK7tyVGKVDabY4OkOnYiygD/QcTtdWR48Eq5pIT0bR2M9u168aXTbWoWX8JVXcm7uwg="",""STFS.UncompressedSize"":""3000""}",4
# ...
```

It is also possible to get information on a single file or directory using `stfs inventory stat`. For more information, see the [inventory reference](#inventory).

### 6. Recovering Data with `stfs recovery`

In case of unfinished write operations, sudden power losses or other forms of data corruption, the integrated recovery tools can help. For example, to query a tape starting from a specific record and block, use `stfs query`:

```shell
# Use `-d /dev/nst0` for your primary tape drive instead
$ stfs recovery query \
    -d ~/Downloads/drive.tar \
    -e age \
    --identity ~/.stfs-age.priv \
    --password mysecureencryptionpassword \
    -s pgp \
    --recipient ~/.stfs-pgp.pub \
    --compression zstandard \
    --record 118 \
    --block 11
118,-1,11,-1,48,.github/workflows/hydrun.yaml.zst.age,,1272,420,1000,1000,pojntfx,pojntfx,2022-05-15T23:41:54+02:00,2022-05-15T23:41:54+02:00,2022-05-15T23:41:54+02:00,0,0,"{""STFS.Signature"":""wnUEABYIACcFAmKCsyMJkGA0c/4XcV5qFqEEjWKRLHhppJ6S+ZJlYDRz/hdxXmoAAF9nAP98fVspytLtKjTATbtH7hoVaK7tyVGKVDabY4OkOnYiygD/QcTtdWR48Eq5pIT0bR2M9u168aXTbWoWX8JVXcm7uwg="",""STFS.UncompressedSize"":""3000""}",4
119,-1,0,-1,48,.gitignore.zst.age,,220,436,1000,1000,pojntfx,pojntfx,2022-04-18T20:19:52+02:00,2022-05-15T23:38:41+02:00,2022-04-23T16:08:59+02:00,0,0,"{""STFS.Signature"":""wnUEABYIACcFAmKCsyMJkGA0c/4XcV5qFqEEjWKRLHhppJ6S+ZJlYDRz/hdxXmoAAGAmAP9G6z9HSr5puQjDMRpYZ11Jge95wG2g3LSetF+ts4CG7wEA38qbJx92BbQN4tWmm5G3dXg+PAnGKONAkc0IU9dmtgA="",""STFS.UncompressedSize"":""4""}",4
# ...
```

If you know the record and block of a file (which you can get from the index with the `inventory` commands), you can also recover it directly:

```shell
# Use `-d /dev/nst0` for your primary tape drive instead
$ stfs recovery fetch \
    -d ~/Downloads/drive.tar \
    -e age \
    --identity ~/.stfs-age.priv \
    --password mysecureencryptionpassword \
    -s pgp \
    --recipient ~/.stfs-pgp.pub \
    --compression zstandard \
    --record 118 \
    --block 11 \
    --to hydrun.yaml
record,lastknownrecord,block,lastknownblock,typeflag,name,linkname,size,mode,uid,gid,uname,gname,modtime,accesstime,changetime,devmajor,devminor,paxrecords,format
118,-1,11,-1,48,.github/workflows/hydrun.yaml.zst.age,,1272,420,1000,1000,pojntfx,pojntfx,2022-05-15T23:41:54+02:00,2022-05-15T23:41:54+02:00,2022-05-15T23:41:54+02:00,0,0,"{""STFS.Signature"":""wnUEABYIACcFAmKCsyMJkGA0c/4XcV5qFqEEjWKRLHhppJ6S+ZJlYDRz/hdxXmoAAF9nAP98fVspytLtKjTATbtH7hoVaK7tyVGKVDabY4OkOnYiygD/QcTtdWR48Eq5pIT0bR2M9u168aXTbWoWX8JVXcm7uwg="",""STFS.UncompressedSize"":""3000""}",4
$ file hydrun.yaml
hydrun.yaml: ASCII text
```

It is also possible to restore a broken index from scratch with `stfs recovery index`. For more information, see the [recovery reference](#recovery).

### 7. Managing the Drive with `stfs drive`

STFS can also manage the physical tape drive directly without having to rely on external tools. For example, to eject a tape from the drive, use `stfs drive eject`:

```shell
$ stfs drive eject \
    -d /dev/nst0
```

It is also possible to get the current tape position with `stfs drive tell`. For more information, see the [drive management reference](#drive-management).

### 8. Embedding STFS with `fs.STFS`

STFS at its core provides quite a few public APIs, but the easiest way to embed it is to use it's provided [`afero.FS implementation`](https://github.com/spf13/afero). This makes it possible to easily swap out the filesystem implementation with a native one, layer caching implementations and decouple your storage layer.

Using this API is fairly straightforward:

```go
// ...
stfs := fs.NewSTFS(
	readOps,
	writeOps,

	config.MetadataConfig{
		Metadata: metadataPersister,
	},

	config.CompressionLevelFastestKey,
	func() (cache.WriteCache, func() error, error) {
		return cache.NewCacheWrite(
			*writeCacheFlag,
			config.WriteCacheTypeFile,
		)
	},
	false,
	false,

	func(hdr *config.Header) {
		l.Trace("Header transform", hdr)
	},
	l,
)

root, err := stfs.Initialize("/", os.ModePerm)
if err != nil {
	panic(err)
}

fs, err := cache.NewCacheFilesystem(
	stfs,
	root,
	config.NoneKey,
	0,
	"",
)
if err != nil {
	panic(err)
}
```

You can now use the Afero APIs to interact with the filesystem; if you've worked with Go's `fs` package before, they should be very familiar:

```go
log.Println("stat /")

stat, err := fs.Stat("/")
if err != nil {
	panic(err)
}

log.Println("Result of stat /:", stat)

log.Println("create /test.txt")

file, err := fs.Create("/test.txt")
if err != nil {
	panic(err)
}

log.Println("Result of create /test.txt:", file)

log.Println("writeString /test.txt")

n, err := file.WriteString("Hello, world!")
if err != nil {
	panic(err)
}

log.Println("Result of writeString /test.txt:", n)

if err := file.Close(); err != nil {
	panic(err)
}

// ...
```

Note that STFS also implements `afero.Symlinker`, so symlinks are available as well.

For more information, check out the [Go API](https://pkg.go.dev/github.com/pojntfx/stfs) and take a look at the provided [examples](./examples), utilities, services and tests in the package for examples.

🚀 **That's it!** We hope you enjoy using STFS.

## Reference

### Command Line Arguments

```shell
$ stfs --help
Simple Tape File System (STFS), a file system for tapes and tar files.

Find more information at:
https://github.com/pojntfx/stfs

Usage:
  stfs [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  drive       Manage tape drives
  help        Help about any command
  inventory   Get contents and metadata of tape or tar file from the index
  keygen      Generate a encryption or signature key
  operation   Perform operations on tape or tar file and the index
  recovery    Recover tapes or tar files
  serve       Serve tape or tar file and the index

Flags:
  -c, --compression string   Compression format to use (default , available are [ gzip parallelgzip lz4 zstandard brotli bzip2 parallelbzip2])
  -d, --drive string         Tape or tar file to use (default "/dev/nst0")
  -e, --encryption string    Encryption format to use (default , available are [ age pgp])
  -h, --help                 help for stfs
  -m, --metadata string      Metadata database to use (default "/home/pojntfx/.local/share/stfs/var/lib/stfs/metadata.sqlite")
  -s, --signature string     Signature format to use (default , available are [ minisign pgp])
  -v, --verbose int          Verbosity level (default 2, available are [0 1 2 3 4]) (default 2)

Use "stfs [command] --help" for more information about a command.
```

<details>
  <summary>Expand subcommand reference</summary>

#### Drive Management

```shell
$ stfs drive --help
Manage tape drives

Usage:
  stfs drive [command]

Aliases:
  drive, dri, d

Available Commands:
  eject       Eject tape from drive
  tell        Get the current record on the tape

Flags:
  -h, --help   help for drive

Global Flags:
  -c, --compression string   Compression format to use (default , available are [ gzip parallelgzip lz4 zstandard brotli bzip2 parallelbzip2])
  -d, --drive string         Tape or tar file to use (default "/dev/nst0")
  -e, --encryption string    Encryption format to use (default , available are [ age pgp])
  -m, --metadata string      Metadata database to use (default "/home/pojntfx/.local/share/stfs/var/lib/stfs/metadata.sqlite")
  -s, --signature string     Signature format to use (default , available are [ minisign pgp])
  -v, --verbose int          Verbosity level (default 2, available are [0 1 2 3 4]) (default 2)

Use "stfs drive [command] --help" for more information about a command.
```

#### Inventory

```shell
$ stfs inventory --help
Get contents and metadata of tape or tar file from the index

Usage:
  stfs inventory [command]

Aliases:
  inventory, inv, i

Available Commands:
  find        Find a file or directory on tape or tar file by matching against a regex
  list        List the contents of a directory on tape or tar file
  stat        Get information on a file or directory on tape or tar file

Flags:
  -h, --help   help for inventory

Global Flags:
  -c, --compression string   Compression format to use (default , available are [ gzip parallelgzip lz4 zstandard brotli bzip2 parallelbzip2])
  -d, --drive string         Tape or tar file to use (default "/dev/nst0")
  -e, --encryption string    Encryption format to use (default , available are [ age pgp])
  -m, --metadata string      Metadata database to use (default "/home/pojntfx/.local/share/stfs/var/lib/stfs/metadata.sqlite")
  -s, --signature string     Signature format to use (default , available are [ minisign pgp])
  -v, --verbose int          Verbosity level (default 2, available are [0 1 2 3 4]) (default 2)

Use "stfs inventory [command] --help" for more information about a command.
```

#### Key Generation

```shell
$ stfs keygen --help
Generate a encryption or signature key

Usage:
  stfs keygen [flags]

Aliases:
  keygen, key, k

Flags:
  -h, --help               help for keygen
  -i, --identity string    Path to write the private key to
  -p, --password string    Password to protect the private key with
  -r, --recipient string   Path to write the public key to

Global Flags:
  -c, --compression string   Compression format to use (default , available are [ gzip parallelgzip lz4 zstandard brotli bzip2 parallelbzip2])
  -d, --drive string         Tape or tar file to use (default "/dev/nst0")
  -e, --encryption string    Encryption format to use (default , available are [ age pgp])
  -m, --metadata string      Metadata database to use (default "/home/pojntfx/.local/share/stfs/var/lib/stfs/metadata.sqlite")
  -s, --signature string     Signature format to use (default , available are [ minisign pgp])
  -v, --verbose int          Verbosity level (default 2, available are [0 1 2 3 4]) (default 2)
```

#### Operations

```shell
$ stfs operation --help
Perform operations on tape or tar file and the index

Usage:
  stfs operation [command]

Aliases:
  operation, ope, o, op

Available Commands:
  archive     Archive a file or directory to tape or tar file
  delete      Delete a file or directory from tape or tar file
  initialize  Truncate and initalize a file or directory
  move        Move a file or directory on tape or tar file
  restore     Restore a file or directory from tape or tar file
  update      Update a file or directory's content and metadata on tape or tar file

Flags:
  -h, --help   help for operation

Global Flags:
  -c, --compression string   Compression format to use (default , available are [ gzip parallelgzip lz4 zstandard brotli bzip2 parallelbzip2])
  -d, --drive string         Tape or tar file to use (default "/dev/nst0")
  -e, --encryption string    Encryption format to use (default , available are [ age pgp])
  -m, --metadata string      Metadata database to use (default "/home/pojntfx/.local/share/stfs/var/lib/stfs/metadata.sqlite")
  -s, --signature string     Signature format to use (default , available are [ minisign pgp])
  -v, --verbose int          Verbosity level (default 2, available are [0 1 2 3 4]) (default 2)

Use "stfs operation [command] --help" for more information about a command.
```

#### Recovery

```shell
$ stfs recovery --help
Recover tapes or tar files

Usage:
  stfs recovery [command]

Aliases:
  recovery, rec, r

Available Commands:
  fetch       Fetch a file or directory from tape or tar file by record and block without the index
  index       Index contents of tape or tar file
  query       Query contents of tape or tar file without the index

Flags:
  -h, --help   help for recovery

Global Flags:
  -c, --compression string   Compression format to use (default , available are [ gzip parallelgzip lz4 zstandard brotli bzip2 parallelbzip2])
  -d, --drive string         Tape or tar file to use (default "/dev/nst0")
  -e, --encryption string    Encryption format to use (default , available are [ age pgp])
  -m, --metadata string      Metadata database to use (default "/home/pojntfx/.local/share/stfs/var/lib/stfs/metadata.sqlite")
  -s, --signature string     Signature format to use (default , available are [ minisign pgp])
  -v, --verbose int          Verbosity level (default 2, available are [0 1 2 3 4]) (default 2)

Use "stfs recovery [command] --help" for more information about a command.
```

#### Servers

```shell
stfs serve --help
Serve tape or tar file and the index

Usage:
  stfs serve [command]

Aliases:
  serve, ser, s, srv

Available Commands:
  ftp         Serve tape or tar file and the index over FTP (read-write)
  http        Serve tape or tar file and the index over HTTP (read-only)

Flags:
  -h, --help   help for serve

Global Flags:
  -c, --compression string   Compression format to use (default , available are [ gzip parallelgzip lz4 zstandard brotli bzip2 parallelbzip2])
  -d, --drive string         Tape or tar file to use (default "/dev/nst0")
  -e, --encryption string    Encryption format to use (default , available are [ age pgp])
  -m, --metadata string      Metadata database to use (default "/home/pojntfx/.local/share/stfs/var/lib/stfs/metadata.sqlite")
  -s, --signature string     Signature format to use (default , available are [ minisign pgp])
  -v, --verbose int          Verbosity level (default 2, available are [0 1 2 3 4]) (default 2)

Use "stfs serve [command] --help" for more information about a command.
```

</details>

### Environment Variables

All command line arguments described above can also be set using environment variables; for example, to set `--drive` to `/tmp/drive.tar` with an environment variable, use `STFS_DRIVE=/tmp/drive.tar`.

## Acknowledgements

- [aead.dev/minisign](https://github.com/aead/minisign) provides the Minisign signature implementation.
- [filippo.io/age](https://github.com/FiloSottile/age) provides the Age encryption implementation.
- [ProtonMail/gopenpgp](github.com/ProtonMail/gopenpgp) provides the PGP signature and encryption implementation.
- [fclairamb/ftpserverlib](github.com/fclairamb/ftpserverlib) provides the FTP server implementation.
- [spf13/afero](https://github.com/spf13/afero) provides the file system abstraction layer implementation.

## Contributing

To contribute, please use the [GitHub flow](https://guides.github.com/introduction/flow/) and follow our [Code of Conduct](./CODE_OF_CONDUCT.md).

To build and start a development version of STFS locally, run the following:

```shell
$ git clone https://github.com/pojntfx/stfs.git
$ cd stfs
$ make depend
$ make && sudo make install
$ stfs serve ftp -d /tmp/drive.tar -m /tmp/dev.sqlite # Now point your file explorer to `ftp://localhost:1337`
```

Have any questions or need help? Chat with us [on Matrix](https://matrix.to/#/#stfs:matrix.org?via=matrix.org)!

## License

STFS (c) 2024 Felicitas Pojtinger and contributors

SPDX-License-Identifier: AGPL-3.0

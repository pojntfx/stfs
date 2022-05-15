# STFS

Simple Tape File System (STFS), a file system for tapes and tar files.

⚠️ STFS has not yet been audited! While we try to make it as secure as possible, it has not yet undergone a formal security audit by a third party. Please keep this in mind if you use it for security-critical applications. ⚠️

[![hydrun CI](https://github.com/pojntfx/stfs/actions/workflows/hydrun.yaml/badge.svg)](https://github.com/pojntfx/stfs/actions/workflows/hydrun.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/pojntfx/stfs.svg)](https://pkg.go.dev/github.com/pojntfx/stfs)
[![Matrix](https://img.shields.io/matrix/stfs:matrix.org)](https://matrix.to/#/#stfs:matrix.org?via=matrix.org)
[![Binary Downloads](https://img.shields.io/github/downloads/pojntfx/stfs/total?label=binary%20downloads)](https://github.com/pojntfx/stfs/releases)

## Overview

🚧 This project is a work-in-progress! Instructions will be added as soon as it is usable. 🚧

## Installation

Static binaries are available on [GitHub releases](https://github.com/pojntfx/stfs/releases).

On Linux, you can install them like so:

```shell
$ curl -L -o /tmp/stfs "https://github.com/pojntfx/stfs/releases/latest/download/stfs.linux-$(uname -m)"
$ sudo install /tmp/stfs /usr/local/bin
$ sudo setcap cap_net_admin+ep /usr/local/bin/stfs # This allows rootless execution
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

You can find binaries for more operating systems and architectures on [GitHub releases](https://github.com/pojntfx/stfs/releases).

## Usage

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

### 5. Managing the Index with `stfs inventory`

### 6. Recovering Data with `stfs recovery`

### 7. Managing the Drive with `stfs drive`

### 8. Embedding STFS with `fs.STFS`

🚀 **That's it!** We hope you enjoy using stfs.

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
- All the rest of the authors who worked on the dependencies used! Thanks a lot!

## Contributing

To contribute, please use the [GitHub flow](https://guides.github.com/introduction/flow/) and follow our [Code of Conduct](./CODE_OF_CONDUCT.md).

To build and start a development version of stfs locally, run the following:

```shell
$ git clone https://github.com/pojntfx/stfs.git
$ cd stfs
$ make depend
$ make && sudo make install
$ stfs serve ftp -d /tmp/drive.tar -m /tmp/dev.sqlite # Now point Nautilus to `ftp://localhost:1337`
```

Have any questions or need help? Chat with us [on Matrix](https://matrix.to/#/#stfs:matrix.org?via=matrix.org)!

## License

STFS (c) 2022 Felicitas Pojtinger and contributors

SPDX-License-Identifier: AGPL-3.0

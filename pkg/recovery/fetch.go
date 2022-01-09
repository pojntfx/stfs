package recovery

import (
	"archive/tar"
	"bufio"
	"io"
	"io/fs"
	"path"
	"path/filepath"

	"github.com/pojntfx/stfs/internal/converters"
	"github.com/pojntfx/stfs/internal/mtio"
	"github.com/pojntfx/stfs/internal/records"
	"github.com/pojntfx/stfs/pkg/compression"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/encryption"
	"github.com/pojntfx/stfs/pkg/signature"
)

func Fetch(
	reader config.DriveReaderConfig,
	drive config.DriveConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	getDst func(path string, mode fs.FileMode) (io.WriteCloser, error),
	mkdirAll func(path string, mode fs.FileMode) error,

	record int,
	block int,
	to string,
	preview bool,

	onHeader func(hdr *config.Header),
) error {
	to = filepath.ToSlash(to)

	var tr *tar.Reader
	if reader.DriveIsRegular {
		// Seek to record and block
		if _, err := reader.Drive.Seek(int64((pipes.RecordSize*mtio.BlockSize*record)+block*mtio.BlockSize), io.SeekStart); err != nil {
			return err
		}

		tr = tar.NewReader(reader.Drive)
	} else {
		// Seek to record
		if err := mtio.SeekToRecordOnTape(drive.Drive.Fd(), int32(record)); err != nil {
			return err
		}

		// Seek to block
		br := bufio.NewReaderSize(drive.Drive, mtio.BlockSize*pipes.RecordSize)
		if _, err := br.Read(make([]byte, block*mtio.BlockSize)); err != nil {
			return err
		}

		tr = tar.NewReader(br)
	}

	hdr, err := tr.Next()
	if err != nil {
		return err
	}

	if err := encryption.DecryptHeader(hdr, pipes.Encryption, crypto.Identity); err != nil {
		return err
	}

	if err := signature.VerifyHeader(hdr, drive.DriveIsRegular, pipes.Signature, crypto.Recipient); err != nil {
		return err
	}

	if onHeader != nil {
		dbhdr, err := converters.TarHeaderToDBHeader(int64(record), -1, int64(block), -1, hdr)
		if err != nil {
			return err
		}

		onHeader(converters.DBHeaderToConfigHeader(dbhdr))
	}

	if !preview {
		if to == "" {
			to = path.Base(hdr.Name)
		}

		if hdr.Typeflag == tar.TypeDir {
			return mkdirAll(to, hdr.FileInfo().Mode())
		}

		dstFile, err := getDst(to, hdr.FileInfo().Mode())
		if err != nil {
			return err
		}

		// Don't decompress non-regular files
		if !hdr.FileInfo().Mode().IsRegular() {
			if _, err := io.Copy(dstFile, tr); err != nil {
				return err
			}

			return nil
		}

		decryptor, err := encryption.Decrypt(tr, pipes.Encryption, crypto.Identity)
		if err != nil {
			return err
		}

		decompressor, err := compression.Decompress(decryptor, pipes.Compression)
		if err != nil {
			return err
		}

		sig := ""
		if hdr.PAXRecords != nil {
			if s, ok := hdr.PAXRecords[records.STFSRecordSignature]; ok {
				sig = s
			}
		}

		verifier, verify, err := signature.Verify(decompressor, drive.DriveIsRegular, pipes.Signature, crypto.Recipient, sig)
		if err != nil {
			return err
		}

		if _, err := io.Copy(dstFile, verifier); err != nil {
			return err
		}

		if err := verify(); err != nil {
			return err
		}

		if err := decryptor.Close(); err != nil {
			return err
		}

		if err := decompressor.Close(); err != nil {
			return err
		}

		if err := dstFile.Close(); err != nil {
			return err
		}
	}

	return nil
}

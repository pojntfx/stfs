package recovery

import (
	"archive/tar"
	"bufio"
	"io"
	"os"
	"path/filepath"

	"github.com/pojntfx/stfs/internal/compression"
	"github.com/pojntfx/stfs/internal/converters"
	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/internal/encryption"
	"github.com/pojntfx/stfs/internal/mtio"
	"github.com/pojntfx/stfs/internal/records"
	"github.com/pojntfx/stfs/internal/signature"
	"github.com/pojntfx/stfs/pkg/config"
)

func Fetch(
	reader config.DriveReaderConfig,
	drive config.DriveConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	record int,
	block int,
	to string,
	preview bool,

	onHeader func(hdr *models.Header),
) error {
	var tr *tar.Reader
	if reader.DriveIsRegular {
		// Seek to record and block
		if _, err := reader.Drive.Seek(int64((recordSize*mtio.BlockSize*record)+block*mtio.BlockSize), io.SeekStart); err != nil {
			return err
		}

		tr = tar.NewReader(reader.Drive)
	} else {
		// Seek to record
		if err := mtio.SeekToRecordOnTape(drive.Drive, int32(record)); err != nil {
			return err
		}

		// Seek to block
		br := bufio.NewReaderSize(drive.Drive, mtio.BlockSize*recordSize)
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

		onHeader(dbhdr)
	}

	if !preview {
		if to == "" {
			to = filepath.Base(hdr.Name)
		}

		if hdr.Typeflag == tar.TypeDir {
			return os.MkdirAll(to, hdr.FileInfo().Mode())
		}

		dstFile, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE, hdr.FileInfo().Mode())
		if err != nil {
			return err
		}

		if err := dstFile.Truncate(0); err != nil {
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

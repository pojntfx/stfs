package recovery

import (
	"archive/tar"
	"bufio"
	"io"
	"os"
	"path/filepath"

	"github.com/pojntfx/stfs/internal/compression"
	"github.com/pojntfx/stfs/internal/converters"
	"github.com/pojntfx/stfs/internal/encryption"
	"github.com/pojntfx/stfs/internal/formatting"
	"github.com/pojntfx/stfs/internal/mtio"
	"github.com/pojntfx/stfs/internal/records"
	"github.com/pojntfx/stfs/internal/signature"
	"github.com/pojntfx/stfs/internal/tape"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/hardware"
)

func Fetch(
	state hardware.DriveConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	record int,
	block int,
	to string,
	preview bool,

	showHeader bool,
) error {
	f, isRegular, err := tape.OpenTapeReadOnly(state.Drive)
	if err != nil {
		return err
	}
	defer f.Close()

	var tr *tar.Reader
	if isRegular {
		// Seek to record and block
		if _, err := f.Seek(int64((recordSize*mtio.BlockSize*record)+block*mtio.BlockSize), io.SeekStart); err != nil {
			return err
		}

		tr = tar.NewReader(f)
	} else {
		// Seek to record
		if err := mtio.SeekToRecordOnTape(f, int32(record)); err != nil {
			return err
		}

		// Seek to block
		br := bufio.NewReaderSize(f, mtio.BlockSize*recordSize)
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

	if err := signature.VerifyHeader(hdr, isRegular, pipes.Signature, crypto.Recipient); err != nil {
		return err
	}

	if showHeader {
		if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
			return err
		}

		if err := formatting.PrintCSV(converters.TARHeaderToCSV(int64(record), -1, int64(block), -1, hdr)); err != nil {
			return err
		}
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

		verifier, verify, err := signature.Verify(decompressor, isRegular, pipes.Signature, crypto.Recipient, sig)
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

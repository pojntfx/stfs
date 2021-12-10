package operations

import (
	"archive/tar"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pojntfx/stfs/internal/compression"
	"github.com/pojntfx/stfs/internal/converters"
	models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/internal/encryption"
	"github.com/pojntfx/stfs/internal/ioext"
	"github.com/pojntfx/stfs/internal/mtio"
	"github.com/pojntfx/stfs/internal/records"
	"github.com/pojntfx/stfs/internal/signature"
	"github.com/pojntfx/stfs/internal/statext"
	"github.com/pojntfx/stfs/internal/suffix"
	"github.com/pojntfx/stfs/internal/tape"
	"github.com/pojntfx/stfs/pkg/config"
)

func Update(
	state config.StateConfig,
	pipes config.PipeConfig,
	crypto config.CryptoConfig,

	recordSize int,
	from string,
	overwrite bool,
	compressionLevel string,

	onHeader func(hdr *models.Header),
) ([]*tar.Header, error) {
	dirty := false
	tw, isRegular, cleanup, err := tape.OpenTapeWriteOnly(state.Drive, recordSize, false)
	if err != nil {
		return []*tar.Header{}, err
	}
	defer cleanup(&dirty)

	headers := []*tar.Header{}
	return headers, filepath.Walk(from, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		link := ""
		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			if link, err = os.Readlink(path); err != nil {
				return err
			}
		}

		hdr, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}

		if err := statext.EnhanceHeader(path, hdr); err != nil {
			return err
		}

		hdr.Name = path
		hdr.Format = tar.FormatPAX
		if hdr.PAXRecords == nil {
			hdr.PAXRecords = map[string]string{}
		}
		hdr.PAXRecords[records.STFSRecordVersion] = records.STFSRecordVersion1
		hdr.PAXRecords[records.STFSRecordAction] = records.STFSRecordActionUpdate

		if info.Mode().IsRegular() && overwrite {
			// Get the compressed size for the header
			fileSizeCounter := &ioext.CounterWriter{
				Writer: io.Discard,
			}

			encryptor, err := encryption.Encrypt(fileSizeCounter, pipes.Encryption, crypto.Recipient)
			if err != nil {
				return err
			}

			compressor, err := compression.Compress(
				encryptor,
				pipes.Compression,
				compressionLevel,
				isRegular,
				recordSize,
			)
			if err != nil {
				return err
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}

			signer, sign, err := signature.Sign(file, isRegular, pipes.Signature, crypto.Identity)
			if err != nil {
				return err
			}

			if isRegular {
				if _, err := io.Copy(compressor, signer); err != nil {
					return err
				}
			} else {
				buf := make([]byte, mtio.BlockSize*recordSize)
				if _, err := io.CopyBuffer(compressor, signer, buf); err != nil {
					return err
				}
			}

			if err := file.Close(); err != nil {
				return err
			}

			if err := compressor.Flush(); err != nil {
				return err
			}

			if err := compressor.Close(); err != nil {
				return err
			}

			if err := encryptor.Close(); err != nil {
				return err
			}

			if hdr.PAXRecords == nil {
				hdr.PAXRecords = map[string]string{}
			}
			hdr.PAXRecords[records.STFSRecordUncompressedSize] = strconv.Itoa(int(hdr.Size))
			signature, err := sign()
			if err != nil {
				return err
			}

			if signature != "" {
				hdr.PAXRecords[records.STFSRecordSignature] = signature
			}
			hdr.Size = int64(fileSizeCounter.BytesRead)

			hdr.Name, err = suffix.AddSuffix(hdr.Name, pipes.Compression, pipes.Encryption)
			if err != nil {
				return err
			}
		}

		if overwrite {
			hdr.PAXRecords[records.STFSRecordReplacesContent] = records.STFSRecordReplacesContentTrue

			if onHeader != nil {
				dbhdr, err := converters.TarHeaderToDBHeader(-1, -1, -1, -1, hdr)
				if err != nil {
					return err
				}

				onHeader(dbhdr)
			}

			hdrToAppend := *hdr
			headers = append(headers, &hdrToAppend)

			if err := signature.SignHeader(hdr, isRegular, pipes.Signature, crypto.Identity); err != nil {
				return err
			}

			if err := encryption.EncryptHeader(hdr, pipes.Encryption, crypto.Recipient); err != nil {
				return err
			}

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			// Compress and write the file
			encryptor, err := encryption.Encrypt(tw, pipes.Encryption, crypto.Recipient)
			if err != nil {
				return err
			}

			compressor, err := compression.Compress(
				encryptor,
				pipes.Compression,
				compressionLevel,
				isRegular,
				recordSize,
			)
			if err != nil {
				return err
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}

			if isRegular {
				if _, err := io.Copy(compressor, file); err != nil {
					return err
				}
			} else {
				buf := make([]byte, mtio.BlockSize*recordSize)
				if _, err := io.CopyBuffer(compressor, file, buf); err != nil {
					return err
				}
			}

			if err := file.Close(); err != nil {
				return err
			}

			if err := compressor.Flush(); err != nil {
				return err
			}

			if err := compressor.Close(); err != nil {
				return err
			}

			if err := encryptor.Close(); err != nil {
				return err
			}
		} else {
			hdr.Size = 0 // Don't try to seek after the record

			if onHeader != nil {
				dbhdr, err := converters.TarHeaderToDBHeader(-1, -1, -1, -1, hdr)
				if err != nil {
					return err
				}

				onHeader(dbhdr)
			}

			hdrToAppend := *hdr
			headers = append(headers, &hdrToAppend)

			if err := signature.SignHeader(hdr, isRegular, pipes.Signature, crypto.Identity); err != nil {
				return err
			}

			if err := encryption.EncryptHeader(hdr, pipes.Encryption, crypto.Recipient); err != nil {
				return err
			}

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
		}

		dirty = true

		return nil
	})
}

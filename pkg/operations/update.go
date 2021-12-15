package operations

import (
	"archive/tar"
	"context"
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
	"github.com/pojntfx/stfs/internal/suffix"
	"github.com/pojntfx/stfs/internal/tarext"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/recovery"
)

func (o *Operations) Update(
	from string,
	compressionLevel string,
	overwrite bool,
) ([]*tar.Header, error) {
	o.diskOperationLock.Lock()
	defer o.diskOperationLock.Unlock()

	writer, err := o.backend.GetWriter()
	if err != nil {
		return []*tar.Header{}, err
	}

	dirty := false
	tw, cleanup, err := tarext.NewTapeWriter(writer.Drive, writer.DriveIsRegular, o.pipes.RecordSize)
	if err != nil {
		return []*tar.Header{}, err
	}

	lastIndexedRecord, lastIndexedBlock, err := o.metadata.Metadata.GetLastIndexedRecordAndBlock(context.Background(), o.pipes.RecordSize)
	if err != nil {
		return []*tar.Header{}, err
	}

	hdrs := []*tar.Header{}
	if err := filepath.Walk(from, func(path string, info fs.FileInfo, err error) error {
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

			encryptor, err := encryption.Encrypt(fileSizeCounter, o.pipes.Encryption, o.crypto.Recipient)
			if err != nil {
				return err
			}

			compressor, err := compression.Compress(
				encryptor,
				o.pipes.Compression,
				compressionLevel,
				writer.DriveIsRegular,
				o.pipes.RecordSize,
			)
			if err != nil {
				return err
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}

			signer, sign, err := signature.Sign(file, writer.DriveIsRegular, o.pipes.Signature, o.crypto.Identity)
			if err != nil {
				return err
			}

			if writer.DriveIsRegular {
				if _, err := io.Copy(compressor, signer); err != nil {
					return err
				}
			} else {
				buf := make([]byte, mtio.BlockSize*o.pipes.RecordSize)
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

			hdr.Name, err = suffix.AddSuffix(hdr.Name, o.pipes.Compression, o.pipes.Encryption)
			if err != nil {
				return err
			}
		}

		if overwrite {
			hdr.PAXRecords[records.STFSRecordReplacesContent] = records.STFSRecordReplacesContentTrue

			if o.onHeader != nil {
				dbhdr, err := converters.TarHeaderToDBHeader(-1, -1, -1, -1, hdr)
				if err != nil {
					return err
				}

				o.onHeader(&config.HeaderEvent{
					Type:    config.HeaderEventTypeUpdate,
					Indexed: false,
					Header:  dbhdr,
				})
			}

			hdrToAppend := *hdr
			hdrs = append(hdrs, &hdrToAppend)

			if err := signature.SignHeader(hdr, writer.DriveIsRegular, o.pipes.Signature, o.crypto.Identity); err != nil {
				return err
			}

			if err := encryption.EncryptHeader(hdr, o.pipes.Encryption, o.crypto.Recipient); err != nil {
				return err
			}

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			// Compress and write the file
			encryptor, err := encryption.Encrypt(tw, o.pipes.Encryption, o.crypto.Recipient)
			if err != nil {
				return err
			}

			compressor, err := compression.Compress(
				encryptor,
				o.pipes.Compression,
				compressionLevel,
				writer.DriveIsRegular,
				o.pipes.RecordSize,
			)
			if err != nil {
				return err
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}

			if writer.DriveIsRegular {
				if _, err := io.Copy(compressor, file); err != nil {
					return err
				}
			} else {
				buf := make([]byte, mtio.BlockSize*o.pipes.RecordSize)
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

			if o.onHeader != nil {
				dbhdr, err := converters.TarHeaderToDBHeader(-1, -1, -1, -1, hdr)
				if err != nil {
					return err
				}

				o.onHeader(&config.HeaderEvent{
					Type:    config.HeaderEventTypeUpdate,
					Indexed: false,
					Header:  dbhdr,
				})
			}

			hdrToAppend := *hdr
			hdrs = append(hdrs, &hdrToAppend)

			if err := signature.SignHeader(hdr, writer.DriveIsRegular, o.pipes.Signature, o.crypto.Identity); err != nil {
				return err
			}

			if err := encryption.EncryptHeader(hdr, o.pipes.Encryption, o.crypto.Recipient); err != nil {
				return err
			}

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
		}

		dirty = true

		return nil
	}); err != nil {
		return []*tar.Header{}, err
	}

	if err := cleanup(&dirty); err != nil {
		return []*tar.Header{}, err
	}

	if err := o.backend.CloseWriter(); err != nil {
		return []*tar.Header{}, err
	}

	reader, err := o.backend.GetReader()
	if err != nil {
		return []*tar.Header{}, err
	}
	defer o.backend.CloseReader()

	drive, err := o.backend.GetDrive()
	if err != nil {
		return []*tar.Header{}, err
	}
	defer o.backend.CloseDrive()

	return hdrs, recovery.Index(
		reader,
		drive,
		o.metadata,
		o.pipes,
		o.crypto,

		o.pipes.RecordSize,
		int(lastIndexedRecord),
		int(lastIndexedBlock),
		false,
		1, // Ignore the first header, which is the last header which we already indexed

		func(hdr *tar.Header, i int) error {
			if len(hdrs) <= i {
				return config.ErrTarHeaderMissing
			}

			*hdr = *hdrs[i]

			return nil
		},
		func(hdr *tar.Header, isRegular bool) error {
			return nil // We sign above, no need to verify
		},

		func(hdr *models.Header) {
			o.onHeader(&config.HeaderEvent{
				Type:    config.HeaderEventTypeUpdate,
				Indexed: true,
				Header:  hdr,
			})
		},
	)
}

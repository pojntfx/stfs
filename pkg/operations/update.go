package operations

import (
	"archive/tar"
	"context"
	"io"
	"strconv"
	"strings"

	"github.com/pojntfx/stfs/internal/converters"
	"github.com/pojntfx/stfs/internal/ioext"
	"github.com/pojntfx/stfs/internal/records"
	"github.com/pojntfx/stfs/internal/suffix"
	"github.com/pojntfx/stfs/internal/tarext"
	"github.com/pojntfx/stfs/pkg/compression"
	"github.com/pojntfx/stfs/pkg/config"
	"github.com/pojntfx/stfs/pkg/encryption"
	"github.com/pojntfx/stfs/pkg/recovery"
	"github.com/pojntfx/stfs/pkg/signature"
)

func (o *Operations) Update(
	getSrc func() (config.FileConfig, error),
	compressionLevel string,
	replace bool,
	skipSizeCheck bool,
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
	for {
		file, err := getSrc()
		if err == io.EOF {
			break
		}

		if err != nil {
			return []*tar.Header{}, err
		}

		hdr, err := tar.FileInfoHeader(file.Info, file.Link)
		if err != nil {
			// Skip sockets
			if strings.Contains(err.Error(), errSocketsNotSupported.Error()) {
				continue
			}

			return []*tar.Header{}, err
		}

		hdr.Name = file.Path
		hdr.Format = tar.FormatPAX
		if hdr.PAXRecords == nil {
			hdr.PAXRecords = map[string]string{}
		}
		hdr.PAXRecords[records.STFSRecordVersion] = records.STFSRecordVersion1
		hdr.PAXRecords[records.STFSRecordAction] = records.STFSRecordActionUpdate

		var f io.ReadSeekCloser
		if file.Info.Mode().IsRegular() && replace && (file.Info.Size() > 0 || skipSizeCheck) {
			// Get the compressed size for the header
			fileSizeCounter := &ioext.CounterWriter{
				Writer: io.Discard,
			}

			encryptor, err := encryption.Encrypt(fileSizeCounter, o.pipes.Encryption, o.crypto.Recipient)
			if err != nil {
				return []*tar.Header{}, err
			}

			compressor, err := compression.Compress(
				encryptor,
				o.pipes.Compression,
				compressionLevel,
				writer.DriveIsRegular,
				o.pipes.RecordSize,
			)
			if err != nil {
				return []*tar.Header{}, err
			}

			f, err = file.GetFile()
			if err != nil {
				return []*tar.Header{}, err
			}

			signer, sign, err := signature.Sign(f, writer.DriveIsRegular, o.pipes.Signature, o.crypto.Identity)
			if err != nil {
				return []*tar.Header{}, err
			}

			if writer.DriveIsRegular {
				if _, err := io.Copy(compressor, signer); err != nil {
					return []*tar.Header{}, err
				}
			} else {
				buf := make([]byte, config.MagneticTapeBlockSize*o.pipes.RecordSize)
				if _, err := io.CopyBuffer(compressor, signer, buf); err != nil {
					return []*tar.Header{}, err
				}
			}

			if err := compressor.Flush(); err != nil {
				return []*tar.Header{}, err
			}

			if err := compressor.Close(); err != nil {
				return []*tar.Header{}, err
			}

			if err := encryptor.Close(); err != nil {
				return []*tar.Header{}, err
			}

			if hdr.PAXRecords == nil {
				hdr.PAXRecords = map[string]string{}
			}
			hdr.PAXRecords[records.STFSRecordUncompressedSize] = strconv.Itoa(int(hdr.Size))
			signature, err := sign()
			if err != nil {
				return []*tar.Header{}, err
			}

			if signature != "" {
				hdr.PAXRecords[records.STFSRecordSignature] = signature
			}
			hdr.Size = int64(fileSizeCounter.BytesRead)

			hdr.Name, err = suffix.AddSuffix(hdr.Name, o.pipes.Compression, o.pipes.Encryption)
			if err != nil {
				return []*tar.Header{}, err
			}
		}

		if replace {
			hdr.PAXRecords[records.STFSRecordReplacesContent] = records.STFSRecordReplacesContentTrue

			if o.onHeader != nil {
				dbhdr, err := converters.TarHeaderToDBHeader(-1, -1, -1, -1, hdr)
				if err != nil {
					return []*tar.Header{}, err
				}

				o.onHeader(&config.HeaderEvent{
					Type:    config.HeaderEventTypeUpdate,
					Indexed: false,
					Header:  converters.DBHeaderToConfigHeader(dbhdr),
				})
			}

			hdrToAppend := *hdr
			hdrs = append(hdrs, &hdrToAppend)

			if err := signature.SignHeader(hdr, writer.DriveIsRegular, o.pipes.Signature, o.crypto.Identity); err != nil {
				return []*tar.Header{}, err
			}

			if err := encryption.EncryptHeader(hdr, o.pipes.Encryption, o.crypto.Recipient); err != nil {
				return []*tar.Header{}, err
			}

			if err := tw.WriteHeader(hdr); err != nil {
				return []*tar.Header{}, err
			}

			dirty = true

			if !file.Info.Mode().IsRegular() || (!skipSizeCheck && file.Info.Size() <= 0) {
				if f != nil {
					if err := f.Close(); err != nil {
						return []*tar.Header{}, err
					}
				}

				continue
			}

			// Compress and write the file
			encryptor, err := encryption.Encrypt(tw, o.pipes.Encryption, o.crypto.Recipient)
			if err != nil {
				return []*tar.Header{}, err
			}

			compressor, err := compression.Compress(
				encryptor,
				o.pipes.Compression,
				compressionLevel,
				writer.DriveIsRegular,
				o.pipes.RecordSize,
			)
			if err != nil {
				return []*tar.Header{}, err
			}

			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return []*tar.Header{}, err
			}

			if writer.DriveIsRegular {
				if _, err := io.Copy(compressor, f); err != nil {
					return []*tar.Header{}, err
				}
			} else {
				buf := make([]byte, config.MagneticTapeBlockSize*o.pipes.RecordSize)
				if _, err := io.CopyBuffer(compressor, f, buf); err != nil {
					return []*tar.Header{}, err
				}
			}

			if err := compressor.Flush(); err != nil {
				return []*tar.Header{}, err
			}

			if err := compressor.Close(); err != nil {
				return []*tar.Header{}, err
			}

			if err := encryptor.Close(); err != nil {
				return []*tar.Header{}, err
			}

			if err := f.Close(); err != nil {
				return []*tar.Header{}, err
			}
		} else {
			hdr.Size = 0 // Don't try to seek after the record

			if o.onHeader != nil {
				dbhdr, err := converters.TarHeaderToDBHeader(-1, -1, -1, -1, hdr)
				if err != nil {
					return []*tar.Header{}, err
				}

				o.onHeader(&config.HeaderEvent{
					Type:    config.HeaderEventTypeUpdate,
					Indexed: false,
					Header:  converters.DBHeaderToConfigHeader(dbhdr),
				})
			}

			hdrToAppend := *hdr
			hdrs = append(hdrs, &hdrToAppend)

			if err := signature.SignHeader(hdr, writer.DriveIsRegular, o.pipes.Signature, o.crypto.Identity); err != nil {
				return []*tar.Header{}, err
			}

			if err := encryption.EncryptHeader(hdr, o.pipes.Encryption, o.crypto.Recipient); err != nil {
				return []*tar.Header{}, err
			}

			if err := tw.WriteHeader(hdr); err != nil {
				return []*tar.Header{}, err
			}
		}

		dirty = true
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

	return hdrs, recovery.Index(
		reader,
		o.backend.MagneticTapeIO,
		o.metadata,
		o.pipes,
		o.crypto,

		int(lastIndexedRecord),
		int(lastIndexedBlock),
		false,
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

		func(hdr *config.Header) {
			o.onHeader(&config.HeaderEvent{
				Type:    config.HeaderEventTypeUpdate,
				Indexed: true,
				Header:  hdr,
			})
		},
	)
}

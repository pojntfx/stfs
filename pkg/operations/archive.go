package operations

import (
	"archive/tar"
	"context"
	"errors"
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

var (
	errSocketsNotSupported = errors.New("archive/tar: sockets not supported")
)

func (o *Operations) Archive(
	getSrc func() (config.FileConfig, error),
	compressionLevel string,
	overwrite bool,
	initializing bool,
) ([]*tar.Header, error) {
	o.diskOperationLock.Lock()
	defer o.diskOperationLock.Unlock()

	return o.archive(getSrc, compressionLevel, overwrite, initializing)
}

func (o *Operations) archive(
	getSrc func() (config.FileConfig, error),
	compressionLevel string,
	overwrite bool,
	initializing bool,
) ([]*tar.Header, error) {

	writer, err := o.backend.GetWriter()
	if err != nil {
		return []*tar.Header{}, err
	}

	dirty := false
	tw, cleanup, err := tarext.NewTapeWriter(writer.Drive, writer.DriveIsRegular, o.pipes.RecordSize)
	if err != nil {
		return []*tar.Header{}, err
	}

	lastIndexedRecord := int64(0)
	lastIndexedBlock := int64(0)
	if !overwrite {
		lastIndexedRecord, lastIndexedBlock, err = o.metadata.Metadata.GetLastIndexedRecordAndBlock(context.Background(), o.pipes.RecordSize)
		if err != nil {
			return []*tar.Header{}, err
		}
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

		var f io.ReadSeekCloser
		if file.Info.Mode().IsRegular() && file.Info.Size() > 0 {
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

		if o.onHeader != nil {
			dbhdr, err := converters.TarHeaderToDBHeader(-1, -1, -1, -1, hdr)
			if err != nil {
				return []*tar.Header{}, err
			}

			o.onHeader(&config.HeaderEvent{
				Type:    config.HeaderEventTypeArchive,
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

		if !file.Info.Mode().IsRegular() || file.Info.Size() <= 0 {
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
	}

	if err := cleanup(&dirty); err != nil {
		return []*tar.Header{}, err
	}

	index := 1 // Ignore the first header, which is the last header which we already indexed
	if overwrite {
		index = 0 // If we are starting fresh, index from start
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
		overwrite,
		initializing,
		index,

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
				Type:    config.HeaderEventTypeArchive,
				Indexed: true,
				Header:  hdr,
			})
		},
	)
}

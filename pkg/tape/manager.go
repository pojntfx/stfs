package tape

import (
	"io"
	"os"
	"sync"

	"github.com/pojntfx/stfs/pkg/config"
)

type TapeManager struct {
	drive      string
	mt         config.MagneticTapeIO
	recordSize int
	overwrite  bool

	physicalLock sync.Mutex

	readerLock      sync.Mutex
	reader          *os.File
	readerIsRegular bool

	closer func() error

	overwrote bool
}

func NewTapeManager(
	drive string,
	mt config.MagneticTapeIO,
	recordSize int,
	overwrite bool,
) *TapeManager {
	return &TapeManager{
		drive:      drive,
		mt:         mt,
		recordSize: recordSize,
		overwrite:  overwrite,
	}
}

func (m *TapeManager) GetWriter() (config.DriveWriterConfig, error) {
	m.physicalLock.Lock()

	overwrite := m.overwrite
	if m.overwrote {
		overwrite = false
	}
	m.overwrote = true

	writer, writerIsRegular, err := OpenTapeWriteOnly(
		m.drive,
		m.mt,
		m.recordSize,
		overwrite,
	)
	if err != nil {
		return config.DriveWriterConfig{}, err
	}

	m.closer = writer.Close

	return config.DriveWriterConfig{
		Drive:          writer,
		DriveIsRegular: writerIsRegular,
	}, nil
}

func (m *TapeManager) GetReader() (config.DriveReaderConfig, error) {
	if err := m.openOrReuseReader(); err != nil {
		return config.DriveReaderConfig{}, err
	}

	return config.DriveReaderConfig{
		Drive:          m.reader,
		DriveIsRegular: m.readerIsRegular,
	}, nil
}

func (m *TapeManager) Close() error {
	if m.closer != nil {
		if err := m.closer(); err != nil {
			return err
		}
	}

	m.physicalLock.Unlock()

	return nil
}

func (m *TapeManager) openOrReuseReader() error {
	m.readerLock.Lock()
	defer m.readerLock.Unlock()

	reopen := false
	if m.reader == nil {
		reopen = true
	} else if _, err := m.reader.Seek(0, io.SeekCurrent); err != nil {
		// File is closed
		reopen = true
	}

	if reopen {
		m.physicalLock.Lock()

		r, rr, err := OpenTapeReadOnly(m.drive)
		if err != nil {
			return err
		}

		m.reader = r
		m.readerIsRegular = rr

		m.closer = r.Close
	}

	return nil
}

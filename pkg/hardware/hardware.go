package hardware

type DriveConfig struct {
	Drive string
}

func Eject(
	state DriveConfig,
) error

func Tell(
	state DriveConfig,
) (int, error)

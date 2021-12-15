package config

import models "github.com/pojntfx/stfs/internal/db/sqlite/models/metadata"

type HeaderEvent struct {
	Type    int
	Indexed bool
	Header  *models.Header
}

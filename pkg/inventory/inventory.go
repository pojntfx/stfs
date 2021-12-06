package inventory

import "archive/tar"

type MetadataConfig struct {
	Metadata string
}

func Find(
	state MetadataConfig,

	expression string,
) ([]*tar.Header, error)

func List(
	state MetadataConfig,

	name string,
) ([]*tar.Header, error)

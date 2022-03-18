package docker

import "context"

type ImageDiskSource struct {
	client ImageDiskLoader
	file   string
}

func NewDiskSource(client ImageDiskLoader, file string) *ImageDiskSource {
	return &ImageDiskSource{
		client: client,
		file:   file,
	}
}

func (s *ImageDiskSource) Load(ctx context.Context, images ...string) error {
	return s.client.LoadFromFile(ctx, s.file)
}

type ImageDiskDestination struct {
	client ImageDiskWriter
	file   string
}

func NewDiskDestination(client ImageDiskWriter, file string) *ImageDiskDestination {
	return &ImageDiskDestination{
		client: client,
		file:   file,
	}
}

func (s *ImageDiskDestination) Write(ctx context.Context, images ...string) error {
	return s.client.SaveToFile(ctx, s.file, images...)
}

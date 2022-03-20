package docker

import (
	"context"

	"github.com/aws/eks-anywhere/pkg/logger"
)

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
	logger.Info("Loading images from disk")
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
	logger.Info("Writing images to disk")
	return s.client.SaveToFile(ctx, s.file, images...)
}

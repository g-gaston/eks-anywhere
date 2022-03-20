package docker

import (
	"context"
	"fmt"

	"github.com/aws/eks-anywhere/pkg/types"
)

type ImageDiskLoader interface {
	LoadFromFile(ctx context.Context, filepath string) error
}

type ImageDiskWriter interface {
	SaveToFile(ctx context.Context, filepath string, images ...string) error
}

type ImagePusher interface {
	PushImage(ctx context.Context, image string, endpoint string) error
}

type ImageTaggerPusher interface {
	ImagePusher
	TagImage(ctx context.Context, image string, endpoint string) error
}

type ImagePuller interface {
	PullImage(ctx context.Context, image string) error
}

type DockerClient interface {
	ImageDiskLoader
	ImageDiskWriter
	ImagePuller
	ImagePusher
}

type ImageSource interface {
	Load(ctx context.Context, images ...string) error
}

type ImageDestination interface {
	Write(ctx context.Context, images ...string) error
}

type ImageMover struct {
	source      ImageSource
	destination ImageDestination
}

func NewImageMover(source ImageSource, destination ImageDestination) *ImageMover {
	return &ImageMover{
		source:      source,
		destination: destination,
	}
}

func (m *ImageMover) Move(ctx context.Context, images ...string) error {
	uniqueImages := removesDuplicates(images)

	if err := m.source.Load(ctx, uniqueImages...); err != nil {
		return fmt.Errorf("loading docker image mover source: %v", err)
	}

	if err := m.destination.Write(ctx, uniqueImages...); err != nil {
		return fmt.Errorf("writing images to destination with image mover: %v", err)
	}

	return nil
}

func removesDuplicates(images []string) []string {
	return types.SliceToLookup(images).ToSlice()
}

package docker

import (
	"context"
	"fmt"
)

type DockerClient interface {
	PullImage(ctx context.Context, image string) error
	PushImage(ctx context.Context, image string, endpoint string) error
	LoadFromFile(ctx context.Context, filepath string) error
	SaveToFile(ctx context.Context, filepath string, images ...string) error
}

type (
	ImageSourceLoader      func(ctx context.Context, client DockerClient, images ...string) error
	ImageDestinationLoader func(ctx context.Context, client DockerClient, images ...string) error
)

type ImageMover struct {
	client             DockerClient
	loadSource         ImageSourceLoader
	writeToDestination ImageDestinationLoader
}

func NewImageMover(client DockerClient) *ImageMover {
	return &ImageMover{
		client: client,
	}
}

func (m *ImageMover) From(source ImageSourceLoader) {
	m.loadSource = source
}

func (m *ImageMover) To(destination ImageDestinationLoader) {
	m.writeToDestination = destination
}

func (m *ImageMover) Move(ctx context.Context, images ...string) error {
	if err := m.loadSource(ctx, m.client, images...); err != nil {
		return fmt.Errorf("loading docker image mover source: %v", err)
	}

	if err := m.writeToDestination(ctx, m.client, images...); err != nil {
		return fmt.Errorf("writing images to destination with image mover: %v", err)
	}

	return nil
}

package docker

import (
	"context"
)

type ImageRegistryDestination struct {
	client   ImagePusher
	endpoint string
}

func NewRegistryDestination(client ImagePusher, registryEndpoint string) *ImageRegistryDestination {
	return &ImageRegistryDestination{
		client:   client,
		endpoint: registryEndpoint,
	}
}

func (d *ImageRegistryDestination) Write(ctx context.Context, images ...string) error {
	for _, i := range images {
		if err := d.client.PushImage(ctx, i, d.endpoint); err != nil {
			return err
		}
	}

	return nil
}

type ImageOriginalRegistrySource struct {
	client ImagePuller
}

func NewOriginalRegistrySource(client ImagePuller) *ImageOriginalRegistrySource {
	return &ImageOriginalRegistrySource{
		client: client,
	}
}

func (s *ImageOriginalRegistrySource) Load(ctx context.Context, images ...string) error {
	for _, i := range images {
		if err := s.client.PullImage(ctx, i); err != nil {
			return err
		}
	}

	return nil
}

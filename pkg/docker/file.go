package docker

import "context"

func NewDiskSource(file string) ImageSourceLoader {
	return func(ctx context.Context, client DockerClient, _ ...string) error {
		return client.LoadFromFile(ctx, file)
	}
}

func NewDiskDestination(file string) ImageDestinationLoader {
	return func(ctx context.Context, client DockerClient, images ...string) error {
		return client.SaveToFile(ctx, file, images...)
	}
}

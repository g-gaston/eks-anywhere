package docker

import "context"

func NewRegistryDestination(registryEndpoint string) ImageDestinationLoader {
	return func(ctx context.Context, client DockerClient, images ...string) error {
		for _, i := range images {
			if err := client.PushImage(ctx, i, registryEndpoint); err != nil {
				return err
			}
		}

		return nil
	}
}

func NewOriginalRegistrySource() ImageSourceLoader {
	return func(ctx context.Context, client DockerClient, images ...string) error {
		for _, i := range images {
			if err := client.PullImage(ctx, i); err != nil {
				return err
			}
		}

		return nil
	}
}

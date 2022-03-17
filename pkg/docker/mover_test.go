package docker_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"

	"github.com/aws/eks-anywhere/pkg/docker"
	"github.com/aws/eks-anywhere/pkg/docker/mocks"
)

func TestImageMoverMove(t *testing.T) {
	g := NewWithT(t)
	ctrl := gomock.NewController(t)
	client := mocks.NewMockDockerClient(ctrl)
	ctx := context.Background()

	images := []string{"image1:1", "image2:2"}

	m := docker.NewImageMover(client)
	m.From(dummySource)
	m.To(dummyDst)

	g.Expect(m.Move(ctx, images...)).To(Succeed())
}

func TestImageMoverMoveErrorSource(t *testing.T) {
	g := NewWithT(t)
	ctrl := gomock.NewController(t)
	client := mocks.NewMockDockerClient(ctrl)
	ctx := context.Background()

	images := []string{"image1:1", "image2:2"}
	errorMsg := "fake error"
	sourceError := func(_ context.Context, _ docker.DockerClient, _ ...string) error {
		return errors.New(errorMsg)
	}

	m := docker.NewImageMover(client)
	m.From(sourceError)
	m.To(dummyDst)

	g.Expect(m.Move(ctx, images...)).To(MatchError("loading docker image mover source: fake error"))
}

func TestImageMoverMoveErrorDestination(t *testing.T) {
	g := NewWithT(t)
	ctrl := gomock.NewController(t)
	client := mocks.NewMockDockerClient(ctrl)
	ctx := context.Background()

	images := []string{"image1:1", "image2:2"}
	errorMsg := "fake error"
	dstError := func(_ context.Context, _ docker.DockerClient, _ ...string) error {
		return errors.New(errorMsg)
	}

	m := docker.NewImageMover(client)
	m.From(dummySource)
	m.To(dstError)

	g.Expect(m.Move(ctx, images...)).To(MatchError("writing images to destination with image mover: fake error"))
}

func dummySource(_ context.Context, _ docker.DockerClient, _ ...string) error {
	return nil
}

func dummyDst(_ context.Context, _ docker.DockerClient, _ ...string) error {
	return nil
}

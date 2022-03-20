/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"

	"github.com/aws/eks-anywhere/pkg/bundles"
	"github.com/aws/eks-anywhere/pkg/dependencies"
	"github.com/aws/eks-anywhere/pkg/docker"
	"github.com/aws/eks-anywhere/pkg/files"
	"github.com/aws/eks-anywhere/pkg/manifests"
	"github.com/aws/eks-anywhere/pkg/version"
)

// imagesCmd represents the images command
var downloadImagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Download all eks-a images to disk",
	Long: `Creates a tarball containing all necessary images
to create an eks-a cluster for any of the supported
Kubernetes versions.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		return downloadImagesCommand.Call(ctx)
	},
}

func init() {
	downloadCmd.AddCommand(downloadImagesCmd)

	downloadImagesCmd.Flags().StringVarP(&downloadImagesCommand.OutputFile, "output", "o", "", "Output tarball containing all downloaded images")
	if err := downloadImagesCmd.MarkFlagRequired("output"); err != nil {
		log.Fatalf("Failed making output flag as required: %v", err)
	}
}

var downloadImagesCommand = DownloadImagesCommand{}

type DownloadImagesCommand struct {
	OutputFile string
}

func (c DownloadImagesCommand) Call(ctx context.Context) error {
	reader := files.NewReader()
	manifestReader := manifests.NewReader(reader)

	bundle, err := manifestReader.ReadBundlesForVersion(version.Get().GitVersion)
	if err != nil {
		return err
	}

	images, err := bundles.ReadImages(reader, bundle)
	if err != nil {
		return err
	}
	imageTags := make([]string, 0, len(images))
	for _, i := range images {
		imageTags = append(imageTags, i.VersionedImage())
	}

	toolsImage := bundle.Spec.VersionsBundles[0].Eksa.CliTools.VersionedImage()
	deps, err := dependencies.NewFactory().
		WithExecutableImage(toolsImage).
		WithDocker().
		Build(ctx)
	if err != nil {
		return err
	}
	defer deps.Close(ctx)

	mover := docker.NewImageMover(
		docker.NewOriginalRegistrySource(deps.DockerClient),
		docker.NewDiskDestination(deps.DockerClient, c.OutputFile),
	)

	return mover.Move(ctx, imageTags...)
}

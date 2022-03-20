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
var importImagesCmd = &cobra.Command{
	Use:   "images",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		return importImagesCommand.Call(ctx)
	},
}

func init() {
	importCmd.AddCommand(importImagesCmd)

	importImagesCmd.Flags().StringVarP(&importImagesCommand.InputFile, "input", "i", "", "Input tarball containing all images to import")
	if err := importImagesCmd.MarkFlagRequired("input"); err != nil {
		log.Fatalf("Failed making flag input as required: %v", err)
	}
	importImagesCmd.Flags().StringVarP(&importImagesCommand.RegistryEndpoint, "registry", "r", "", "Registry where to import images")
	if err := importImagesCmd.MarkFlagRequired("registry"); err != nil {
		log.Fatalf("Failed making flag registry as required: %v", err)
	}
}

var importImagesCommand = ImportImagesCommand{}

type ImportImagesCommand struct {
	InputFile, RegistryEndpoint string
}

func (c ImportImagesCommand) Call(ctx context.Context) error {
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
		docker.NewDiskSource(deps.DockerClient, c.InputFile),
		docker.NewRegistryDestination(deps.DockerClient, c.RegistryEndpoint),
	)

	return mover.Move(ctx, imageTags...)
}

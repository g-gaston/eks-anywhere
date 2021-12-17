package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/aws/eks-anywhere/pkg/dependencies"
	"github.com/aws/eks-anywhere/pkg/features"
	"github.com/aws/eks-anywhere/pkg/logger"
)

var testClusterCmd = &cobra.Command{
	Use:          "verify-setup -f <cluster-config-file> [flags]",
	Short:        "Test setup",
	Long:         "This command is used to test",
	PreRunE:      preRunCreateCluster,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cc.validate(cmd.Context()); err != nil {
			return err
		}
		if err := cc.test(cmd); err != nil {
			return fmt.Errorf("failed to create cluster: %v", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(testClusterCmd)
	testClusterCmd.Flags().StringVarP(&cc.fileName, "filename", "f", "", "Filename that contains EKS-A cluster configuration")
	if features.IsActive(features.TinkerbellProvider()) {
		testClusterCmd.Flags().StringVarP(&cc.hardwareFileName, "hardwarefile", "w", "", "Filename that contains datacenter hardware information")
	}
	testClusterCmd.Flags().BoolVar(&cc.forceClean, "force-cleanup", false, "Force deletion of previously created bootstrap cluster")
	testClusterCmd.Flags().BoolVar(&cc.skipIpCheck, "skip-ip-check", false, "Skip check for whether cluster control plane ip is in use")
	testClusterCmd.Flags().StringVar(&cc.bundlesOverride, "bundles-override", "", "Override default Bundles manifest (not recommended)")
	testClusterCmd.Flags().StringVar(&cc.managementKubeconfig, "kubeconfig", "", "Management cluster kubeconfig file")
	err := testClusterCmd.MarkFlagRequired("filename")
	if err != nil {
		log.Fatalf("Error marking flag as required: %v", err)
	}
}

func (cc *createClusterOptions) test(cmd *cobra.Command) error {
	ctx := cmd.Context()

	clusterSpec, err := newClusterSpec(cc.clusterOptions)
	if err != nil {
		return err
	}

	// Add logs here to verify the bundled components
	logger.Info("Eks-a components manifest", "file", clusterSpec.VersionsBundle.Eksa.Components.URI)
	logger.Info("Etcdadm bootstrap controller image", "image", clusterSpec.VersionsBundle.ExternalEtcdBootstrap.Controller.URI)
	logger.Info("Etcdadm controller image", "image", clusterSpec.VersionsBundle.ExternalEtcdController.Controller.URI)
	logger.Info("Bottlerocket bootstrap image", "image", clusterSpec.VersionsBundle.BottleRocketBootstrap.Bootstrap.URI)

	deps, err := dependencies.ForSpec(ctx, clusterSpec).WithExecutableMountDirs(cc.mountDirs()...).
		WithBootstrapper().
		WithClusterManager(clusterSpec.Cluster).
		WithProvider(cc.fileName, clusterSpec.Cluster, cc.skipIpCheck, cc.hardwareFileName).
		WithFluxAddonClient(ctx, clusterSpec.Cluster, clusterSpec.GitOpsConfig).
		WithWriter().
		Build(ctx)
	if err != nil {
		return err
	}
	defer cleanup(ctx, deps, &err)

	return err
}

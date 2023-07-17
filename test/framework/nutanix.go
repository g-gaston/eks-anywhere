package framework

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aws/eks-anywhere/internal/pkg/api"
	"github.com/aws/eks-anywhere/internal/pkg/nutanix"
	"github.com/aws/eks-anywhere/internal/test/cleanup"
	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/constants"
	releasev1 "github.com/aws/eks-anywhere/release/api/v1alpha1"
	clusterf "github.com/aws/eks-anywhere/test/framework/cluster"
)

const (
	nutanixEndpoint                 = "T_NUTANIX_ENDPOINT"
	nutanixPort                     = "T_NUTANIX_PORT"
	nutanixAdditionalTrustBundle    = "T_NUTANIX_ADDITIONAL_TRUST_BUNDLE"
	nutanixInsecure                 = "T_NUTANIX_INSECURE"
	nutanixMachineBootType          = "T_NUTANIX_MACHINE_BOOT_TYPE"
	nutanixMachineMemorySize        = "T_NUTANIX_MACHINE_MEMORY_SIZE"
	nutanixSystemDiskSize           = "T_NUTANIX_SYSTEMDISK_SIZE"
	nutanixMachineVCPUsPerSocket    = "T_NUTANIX_MACHINE_VCPU_PER_SOCKET"
	nutanixMachineVCPUSocket        = "T_NUTANIX_MACHINE_VCPU_SOCKET"
	nutanixPrismElementClusterName  = "T_NUTANIX_PRISM_ELEMENT_CLUSTER_NAME"
	nutanixSSHAuthorizedKey         = "T_NUTANIX_SSH_AUTHORIZED_KEY"
	nutanixSubnetName               = "T_NUTANIX_SUBNET_NAME"
	nutanixControlPlaneEndpointIP   = "T_NUTANIX_CONTROL_PLANE_ENDPOINT_IP"
	nutanixControlPlaneCidrVar      = "T_NUTANIX_CONTROL_PLANE_CIDR"
	nutanixPodCidrVar               = "T_NUTANIX_POD_CIDR"
	nutanixServiceCidrVar           = "T_NUTANIX_SERVICE_CIDR"
	nutanixTemplateNameUbuntu123Var = "T_NUTANIX_TEMPLATE_NAME_UBUNTU_1_23"
	nutanixTemplateNameUbuntu124Var = "T_NUTANIX_TEMPLATE_NAME_UBUNTU_1_24"
	nutanixTemplateNameUbuntu125Var = "T_NUTANIX_TEMPLATE_NAME_UBUNTU_1_25"
	nutanixTemplateNameUbuntu126Var = "T_NUTANIX_TEMPLATE_NAME_UBUNTU_1_26"
	nutanixTemplateNameUbuntu127Var = "T_NUTANIX_TEMPLATE_NAME_UBUNTU_1_27"
)

var requiredNutanixEnvVars = []string{
	constants.EksaNutanixUsernameKey,
	constants.EksaNutanixPasswordKey,
	nutanixEndpoint,
	nutanixPort,
	nutanixAdditionalTrustBundle,
	nutanixMachineBootType,
	nutanixMachineMemorySize,
	nutanixSystemDiskSize,
	nutanixMachineVCPUsPerSocket,
	nutanixMachineVCPUSocket,
	nutanixPrismElementClusterName,
	nutanixSSHAuthorizedKey,
	nutanixSubnetName,
	nutanixPodCidrVar,
	nutanixServiceCidrVar,
	nutanixTemplateNameUbuntu123Var,
	nutanixTemplateNameUbuntu124Var,
	nutanixTemplateNameUbuntu125Var,
	nutanixTemplateNameUbuntu126Var,
	nutanixTemplateNameUbuntu127Var,
	nutanixInsecure,
}

type Nutanix struct {
	t                      *testing.T
	fillers                []api.NutanixFiller
	clusterFillers         []api.ClusterFiller
	client                 nutanix.PrismClient
	controlPlaneEndpointIP string
	cpCidr                 string
	podCidr                string
	serviceCidr            string
	devRelease             *releasev1.EksARelease
	templatesRegistry      *templateRegistry
}

type NutanixOpt func(*Nutanix)

func NewNutanix(t *testing.T, opts ...NutanixOpt) *Nutanix {
	checkRequiredEnvVars(t, requiredNutanixEnvVars)

	nutanixProvider := &Nutanix{
		t: t,
		fillers: []api.NutanixFiller{
			api.WithNutanixStringFromEnvVar(nutanixEndpoint, api.WithNutanixEndpoint),
			api.WithNutanixIntFromEnvVar(nutanixPort, api.WithNutanixPort),
			api.WithNutanixStringFromEnvVar(nutanixAdditionalTrustBundle, api.WithNutanixAdditionalTrustBundle),
			api.WithNutanixStringFromEnvVar(nutanixMachineMemorySize, api.WithNutanixMachineMemorySize),
			api.WithNutanixStringFromEnvVar(nutanixSystemDiskSize, api.WithNutanixMachineSystemDiskSize),
			api.WithNutanixInt32FromEnvVar(nutanixMachineVCPUsPerSocket, api.WithNutanixMachineVCPUsPerSocket),
			api.WithNutanixInt32FromEnvVar(nutanixMachineVCPUSocket, api.WithNutanixMachineVCPUSocket),
			api.WithNutanixStringFromEnvVar(nutanixSSHAuthorizedKey, api.WithNutanixSSHAuthorizedKey),
			api.WithNutanixBoolFromEnvVar(nutanixInsecure, api.WithNutanixInsecure),
			// Assumption: generated clusterconfig by nutanix provider sets name as id type by default.
			// for uuid specific id type, we will set it thru each specific test so that current CI
			// works as is with name id type for following resources
			api.WithNutanixStringFromEnvVar(nutanixPrismElementClusterName, api.WithNutanixPrismElementClusterName),
			api.WithNutanixStringFromEnvVar(nutanixSubnetName, api.WithNutanixSubnetName),
		},
	}

	nutanixProvider.controlPlaneEndpointIP = os.Getenv(nutanixControlPlaneEndpointIP)
	nutanixProvider.cpCidr = os.Getenv(nutanixControlPlaneCidrVar)
	nutanixProvider.podCidr = os.Getenv(nutanixPodCidrVar)
	nutanixProvider.serviceCidr = os.Getenv(nutanixServiceCidrVar)
	client, err := nutanix.NewPrismClient(os.Getenv(nutanixEndpoint), os.Getenv(nutanixPort), true)
	if err != nil {
		t.Fatalf("Failed to initialize Nutanix Prism Client: %v", err)
	}
	nutanixProvider.client = client
	nutanixProvider.templatesRegistry = &templateRegistry{cache: map[string]string{}, generator: nutanixProvider}

	for _, opt := range opts {
		opt(nutanixProvider)
	}

	return nutanixProvider
}

// RequiredNutanixEnvVars returns a list of environment variables needed for Nutanix tests.
func RequiredNutanixEnvVars() []string {
	return requiredNutanixEnvVars
}

func (n *Nutanix) Name() string {
	return "nutanix"
}

func (n *Nutanix) Setup() {}

// UpdateKubeConfig customizes generated kubeconfig for the provider.
func (n *Nutanix) UpdateKubeConfig(content *[]byte, clusterName string) error {
	return nil
}

// CleanupVMs satisfies the test framework Provider.
func (n *Nutanix) CleanupVMs(clustername string) error {
	return cleanup.NutanixTestResourcesCleanup(context.Background(), clustername, os.Getenv(nutanixEndpoint), os.Getenv(nutanixPort), true, true)
}

// ClusterConfigUpdates satisfies the test framework Provider.
func (n *Nutanix) ClusterConfigUpdates() []api.ClusterConfigFiller {
	f := make([]api.ClusterFiller, 0, len(n.clusterFillers)+3)
	f = append(f, n.clusterFillers...)
	if n.controlPlaneEndpointIP != "" {
		f = append(f, api.WithControlPlaneEndpointIP(n.controlPlaneEndpointIP))
	} else {
		clusterIP, err := GetIP(n.cpCidr, ClusterIPPoolEnvVar)
		if err != nil {
			n.t.Fatalf("failed to get cluster ip for test environment: %v", err)
		}
		f = append(f, api.WithControlPlaneEndpointIP(clusterIP))
	}

	if n.podCidr != "" {
		f = append(f, api.WithPodCidr(n.podCidr))
	}

	if n.serviceCidr != "" {
		f = append(f, api.WithServiceCidr(n.serviceCidr))
	}

	return []api.ClusterConfigFiller{api.ClusterToConfigFiller(f...), api.NutanixToConfigFiller(n.fillers...)}
}

func (n *Nutanix) WithProviderUpgrade(fillers ...api.NutanixFiller) ClusterE2ETestOpt {
	return func(e *ClusterE2ETest) {
		e.UpdateClusterConfig(api.NutanixToConfigFiller(fillers...))
	}
}

// WithKubeVersionAndOS returns a cluster config filler that sets the cluster kube version and the right template for all
// nutanix machine configs.
func (n *Nutanix) WithKubeVersionAndOS(osFamily anywherev1.OSFamily, kubeVersion anywherev1.KubernetesVersion, osVersion ...string) api.ClusterConfigFiller {
	// TODO: Update tests to use this
	panic("Not implemented for Nutanix yet")
}

// WithNewWorkerNodeGroup returns an api.ClusterFiller that adds a new workerNodeGroupConfiguration and
// a corresponding NutanixMachineConfig to the cluster config.
func (n *Nutanix) WithNewWorkerNodeGroup(name string, workerNodeGroup *WorkerNodeGroup) api.ClusterConfigFiller {
	// TODO: Implement for Nutanix provider
	panic("Not implemented for Nutanix yet")
}

// withNutanixKubeVersionAndOS returns a NutanixOpt that adds API fillers to use a Nutanix template for
// the specified OS family and version (default if not provided), corresponding to a particular
// Kubernetes version, in addition to configuring all machine configs to use this OS family.
func withNutanixKubeVersionAndOS(osFamily anywherev1.OSFamily, kubeVersion anywherev1.KubernetesVersion, osVersion ...string) NutanixOpt {
	return func(n *Nutanix) {
		n.fillers = append(n.fillers,
			n.templateForKubeVersionAndOS(osFamily, kubeVersion, osVersion...),
			api.WithOsFamilyForAllNutanixMachines(osFamily),
		)
	}
}

// WithUbuntu123Nutanix returns a NutanixOpt that adds API fillers to use a Ubuntu Nutanix template for k8s 1.23
// and the "ubuntu" osFamily in all machine configs.
func WithUbuntu123Nutanix(osVersion ...string) NutanixOpt {
	return withNutanixKubeVersionAndOS(anywherev1.Ubuntu, anywherev1.Kube123, osVersion...)
}

// WithUbuntu124Nutanix returns a NutanixOpt that adds API fillers to use a Ubuntu Nutanix template for k8s 1.24
// and the "ubuntu" osFamily in all machine configs.
func WithUbuntu124Nutanix(osVersion ...string) NutanixOpt {
	return withNutanixKubeVersionAndOS(anywherev1.Ubuntu, anywherev1.Kube124, osVersion...)
}

// WithUbuntu125Nutanix returns a NutanixOpt that adds API fillers to use a Ubuntu Nutanix template for k8s 1.25
// and the "ubuntu" osFamily in all machine configs.
func WithUbuntu125Nutanix(osVersion ...string) NutanixOpt {
	return withNutanixKubeVersionAndOS(anywherev1.Ubuntu, anywherev1.Kube125, osVersion...)
}

// WithUbuntu126Nutanix returns a NutanixOpt that adds API fillers to use a Ubuntu Nutanix template for k8s 1.26
// and the "ubuntu" osFamily in all machine configs.
func WithUbuntu126Nutanix(osVersion ...string) NutanixOpt {
	return withNutanixKubeVersionAndOS(anywherev1.Ubuntu, anywherev1.Kube126, osVersion...)
}

// WithUbuntu127Nutanix returns a NutanixOpt that adds API fillers to use a Ubuntu Nutanix template for k8s 1.27
// and the "ubuntu" osFamily in all machine configs.
func WithUbuntu127Nutanix(osVersion ...string) NutanixOpt {
	return withNutanixKubeVersionAndOS(anywherev1.Ubuntu, anywherev1.Kube127, osVersion...)
}

// withNutanixKubeVersionAndOSForUUID returns a NutanixOpt that adds API fillers to use a Nutanix template UUID
// corresponding to the provided OS family and Kubernetes version, in addition to configuring all machine configs
// to use this OS family.
func withNutanixKubeVersionAndOSForUUID(osFamily anywherev1.OSFamily, kubeVersion anywherev1.KubernetesVersion, osVersion ...string) NutanixOpt {
	return func(n *Nutanix) {
		name := n.templateForDevRelease(osFamily, kubeVersion, osVersion...)
		n.fillers = append(n.fillers, n.withNutanixUUID(name, osFamily)...)
	}
}

// WithUbuntu123NutanixUUID returns a NutanixOpt that adds API fillers to use a Ubuntu Nutanix template UUID for k8s 1.23
// and the "ubuntu" osFamily in all machine configs.
func WithUbuntu123NutanixUUID(osVersion ...string) NutanixOpt {
	return withNutanixKubeVersionAndOSForUUID(anywherev1.Ubuntu, anywherev1.Kube123, osVersion...)
}

// WithUbuntu124NutanixUUID returns a NutanixOpt that adds API fillers to use a Ubuntu Nutanix template UUID for k8s 1.24
// and the "ubuntu" osFamily in all machine configs.
func WithUbuntu124NutanixUUID(osVersion ...string) NutanixOpt {
	return withNutanixKubeVersionAndOSForUUID(anywherev1.Ubuntu, anywherev1.Kube124, osVersion...)
}

// WithUbuntu125NutanixUUID returns a NutanixOpt that adds API fillers to use a Ubuntu Nutanix template UUID for k8s 1.25
// and the "ubuntu" osFamily in all machine configs.
func WithUbuntu125NutanixUUID(osVersion ...string) NutanixOpt {
	return withNutanixKubeVersionAndOSForUUID(anywherev1.Ubuntu, anywherev1.Kube125, osVersion...)
}

// WithUbuntu126NutanixUUID returns a NutanixOpt that adds API fillers to use a Ubuntu Nutanix template UUID for k8s 1.26
// and the "ubuntu" osFamily in all machine configs.
func WithUbuntu126NutanixUUID(osVersion ...string) NutanixOpt {
	return withNutanixKubeVersionAndOSForUUID(anywherev1.Ubuntu, anywherev1.Kube126, osVersion...)
}

// WithUbuntu127NutanixUUID returns a NutanixOpt that adds API fillers to use a Ubuntu Nutanix template UUID for k8s 1.27
// and the "ubuntu" osFamily in all machine configs.
func WithUbuntu127NutanixUUID(osVersion ...string) NutanixOpt {
	return withNutanixKubeVersionAndOSForUUID(anywherev1.Ubuntu, anywherev1.Kube127, osVersion...)
}

func (n *Nutanix) withNutanixUUID(name string, osFamily anywherev1.OSFamily) []api.NutanixFiller {
	uuid, err := n.client.GetImageUUIDFromName(context.Background(), name)
	if err != nil {
		n.t.Fatalf("Failed to get UUID for image %s: %v", name, err)
	}
	return append([]api.NutanixFiller{},
		api.WithNutanixMachineTemplateImageUUID(*uuid),
		api.WithOsFamilyForAllNutanixMachines(osFamily),
	)
}

// WithPrismElementClusterUUID returns a NutanixOpt that adds API fillers to use a PE Cluster UUID.
func WithPrismElementClusterUUID() NutanixOpt {
	return func(n *Nutanix) {
		name := os.Getenv(nutanixPrismElementClusterName)
		uuid, err := n.client.GetClusterUUIDFromName(context.Background(), name)
		if err != nil {
			n.t.Fatalf("Failed to get UUID for image %s: %v", name, err)
		}
		n.fillers = append(n.fillers, api.WithNutanixPrismElementClusterUUID(*uuid))
	}
}

// WithNutanixSubnetUUID returns a NutanixOpt that adds API fillers to use a Subnet UUID.
func WithNutanixSubnetUUID() NutanixOpt {
	return func(n *Nutanix) {
		name := os.Getenv(nutanixSubnetName)
		uuid, err := n.client.GetSubnetUUIDFromName(context.Background(), name)
		if err != nil {
			n.t.Fatalf("Failed to get UUID for image %s: %v", name, err)
		}
		n.fillers = append(n.fillers, api.WithNutanixSubnetUUID(*uuid))
	}
}

// templateForKubeVersionAndOS returns a Nutanix filler for the given OS and Kubernetes version.
func (n *Nutanix) templateForKubeVersionAndOS(osFamily anywherev1.OSFamily, kubeVersion anywherev1.KubernetesVersion, osVersion ...string) api.NutanixFiller {
	return api.WithNutanixMachineTemplateImageName(n.templateForDevRelease(osFamily, kubeVersion, osVersion...))
}

// Ubuntu123Template returns a Nutanix filler for 1.23 Ubuntu.
func (n *Nutanix) Ubuntu123Template(osVersion ...string) api.NutanixFiller {
	return n.templateForKubeVersionAndOS(anywherev1.Ubuntu, anywherev1.Kube123, osVersion...)
}

// Ubuntu124Template returns NutanixFiller by reading the env var and setting machine config's
// image name parameter in the spec.
func (n *Nutanix) Ubuntu124Template(osVersion ...string) api.NutanixFiller {
	return n.templateForKubeVersionAndOS(anywherev1.Ubuntu, anywherev1.Kube124, osVersion...)
}

// Ubuntu125Template returns NutanixFiller by reading the env var and setting machine config's
// image name parameter in the spec.
func (n *Nutanix) Ubuntu125Template(osVersion ...string) api.NutanixFiller {
	return n.templateForKubeVersionAndOS(anywherev1.Ubuntu, anywherev1.Kube125, osVersion...)
}

// Ubuntu126Template returns NutanixFiller by reading the env var and setting machine config's
// image name parameter in the spec.
func (n *Nutanix) Ubuntu126Template(osVersion ...string) api.NutanixFiller {
	return n.templateForKubeVersionAndOS(anywherev1.Ubuntu, anywherev1.Kube126, osVersion...)
}

// Ubuntu127Template returns NutanixFiller by reading the env var and setting machine config's
// image name parameter in the spec.
func (n *Nutanix) Ubuntu127Template(osVersion ...string) api.NutanixFiller {
	return n.templateForKubeVersionAndOS(anywherev1.Ubuntu, anywherev1.Kube127, osVersion...)
}

// ClusterStateValidations returns a list of provider specific ClusterStateValidations.
func (n *Nutanix) ClusterStateValidations() []clusterf.StateValidation {
	return []clusterf.StateValidation{}
}

func (n *Nutanix) getDevRelease() *releasev1.EksARelease {
	n.t.Helper()
	if n.devRelease == nil {
		latestRelease, err := getLatestDevRelease()
		if err != nil {
			n.t.Fatal(err)
		}
		n.devRelease = latestRelease
	}

	return n.devRelease
}

func (n *Nutanix) templateForDevRelease(osFamily anywherev1.OSFamily, kubeVersion anywherev1.KubernetesVersion, osVersion ...string) string {
	n.t.Helper()
	return n.templatesRegistry.templateForRelease(n.t, osFamily, n.getDevRelease(), kubeVersion, osVersion...)
}

// envVarForTemplate looks for explicit configuration through an env var: "T_NUTANIX_TEMPLATE_{osFamily}_{eks-d version}"
// eg: T_NUTANIX_TEMPLATE_UBUNTU_KUBERNETES_1_23_EKS_22.
func (n *Nutanix) envVarForTemplate(osFamily, osVersion, eksDName string) string {
	// Use the old template env var naming scheme for backwards compatibility.
	templateEnvVar := fmt.Sprintf("T_NUTANIX_TEMPLATE_NAME_%s_%s", strings.ToUpper(osFamily), strings.ToUpper(strings.ReplaceAll(eksDName, "-", "_")))
	// If the OS version provided is not the default one, include it in the template env var name.
	if osVersion != defaultOSVersions[osFamily] {
		templateEnvVar = fmt.Sprintf("T_NUTANIX_TEMPLATE_NAME_%s_%s_%s", strings.ToUpper(osFamily), osVersion, strings.ToUpper(strings.ReplaceAll(eksDName, "-", "_")))
	}
	return templateEnvVar
}

// defaultNameForTemplate looks for a template: "{eks-d version}-{osFamily}"
// eg: kubernetes-1-23-eks-22-ubuntu.
func (n *Nutanix) defaultNameForTemplate(osFamily, osVersion, eksDName string) string {
	// Use the old template naming scheme for backwards compatibility.
	defaultTemplateName := fmt.Sprintf("%s-%s", strings.ToLower(eksDName), strings.ToLower(osFamily))
	// If the OS version provided is not the default one, include it in the env var name.
	if osVersion != defaultOSVersions[osFamily] {
		defaultTemplateName = fmt.Sprintf("%s-%s-%s", strings.ToLower(eksDName), strings.ToLower(osFamily), osVersion)
	}
	return defaultTemplateName
}

// defaultEnvVarForTemplate returns the value of the default template env vars: "T_NUTANIX_TEMPLATE_{osFamily}_{kubeVersion}"
// eg. T_NUTANIX_TEMPLATE_UBUNTU_1_23.
func (n *Nutanix) defaultEnvVarForTemplate(osFamily, osVersion string, kubeVersion anywherev1.KubernetesVersion) string {
	// Use the old template env var naming scheme for backwards compatibility.
	defaultTemplateEnvVar := fmt.Sprintf("T_NUTANIX_TEMPLATE_NAME_%s_%s", strings.ToUpper(osFamily), strings.ReplaceAll(string(kubeVersion), ".", "_"))
	// If the OS version provided is not the default one, include it in the template env var name.
	if osVersion != defaultOSVersions[osFamily] {
		defaultTemplateEnvVar = fmt.Sprintf("T_NUTANIX_TEMPLATE_NAME_%s_%s_%s", strings.ToUpper(osFamily), osVersion, strings.ReplaceAll(string(kubeVersion), ".", "_"))
	}
	return defaultTemplateEnvVar
}

// searchTemplate returns template name if the given template exists in Prism Central.
func (n *Nutanix) searchTemplate(ctx context.Context, template string) (string, error) {
	// TODO: implement search functionality for Nutanix templates
	return "", nil
}

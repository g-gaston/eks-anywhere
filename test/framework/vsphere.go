package framework

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"

	"github.com/aws/eks-anywhere/internal/pkg/api"
	"github.com/aws/eks-anywhere/internal/test/cleanup"
	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/executables"
	"github.com/aws/eks-anywhere/pkg/manifests/bundles"
	"github.com/aws/eks-anywhere/pkg/manifests/releases"
	anywheretypes "github.com/aws/eks-anywhere/pkg/types"
	releasev1 "github.com/aws/eks-anywhere/release/api/v1alpha1"
	clusterf "github.com/aws/eks-anywhere/test/framework/cluster"
)

const (
	vsphereDatacenterVar        = "T_VSPHERE_DATACENTER"
	vsphereDatastoreVar         = "T_VSPHERE_DATASTORE"
	vsphereFolderVar            = "T_VSPHERE_FOLDER"
	vsphereNetworkVar           = "T_VSPHERE_NETWORK"
	vspherePrivateNetworkVar    = "T_VSPHERE_PRIVATE_NETWORK"
	vsphereResourcePoolVar      = "T_VSPHERE_RESOURCE_POOL"
	vsphereServerVar            = "T_VSPHERE_SERVER"
	vsphereSshAuthorizedKeyVar  = "T_VSPHERE_SSH_AUTHORIZED_KEY"
	vsphereStoragePolicyNameVar = "T_VSPHERE_STORAGE_POLICY_NAME"
	vsphereTlsInsecureVar       = "T_VSPHERE_TLS_INSECURE"
	vsphereTlsThumbprintVar     = "T_VSPHERE_TLS_THUMBPRINT"
	vsphereUsernameVar          = "EKSA_VSPHERE_USERNAME"
	vspherePasswordVar          = "EKSA_VSPHERE_PASSWORD"
	cidrVar                     = "T_VSPHERE_CIDR"
	privateNetworkCidrVar       = "T_VSPHERE_PRIVATE_NETWORK_CIDR"
	govcUrlVar                  = "VSPHERE_SERVER"
	govcInsecureVar             = "GOVC_INSECURE"
	govcDatacenterVar           = "GOVC_DATACENTER"
	vsphereTemplateEnvVarPrefix = "T_VSPHERE_TEMPLATE_"
	vsphereTemplatesFolder      = "T_VSPHERE_TEMPLATE_FOLDER"
	vsphereTestTagEnvVar        = "T_VSPHERE_TAG"
)

var requiredEnvVars = []string{
	vsphereDatacenterVar,
	vsphereDatastoreVar,
	vsphereFolderVar,
	vsphereNetworkVar,
	vspherePrivateNetworkVar,
	vsphereResourcePoolVar,
	vsphereServerVar,
	vsphereSshAuthorizedKeyVar,
	vsphereTlsInsecureVar,
	vsphereTlsThumbprintVar,
	vsphereUsernameVar,
	vspherePasswordVar,
	cidrVar,
	privateNetworkCidrVar,
	govcUrlVar,
	govcInsecureVar,
	govcDatacenterVar,
	vsphereTestTagEnvVar,
}

type VSphere struct {
	t                 *testing.T
	testsConfig       vsphereConfig
	fillers           []api.VSphereFiller
	clusterFillers    []api.ClusterFiller
	cidr              string
	GovcClient        *executables.Govc
	devRelease        *releasev1.EksARelease
	templatesRegistry *templateRegistry
}

type vsphereConfig struct {
	Datacenter        string
	Datastore         string
	Folder            string
	Network           string
	ResourcePool      string
	Server            string
	SSHAuthorizedKey  string
	StoragePolicyName string
	TLSInsecure       bool
	TLSThumbprint     string
	TemplatesFolder   string
}

// VSphereOpt is construction option for the E2E vSphere provider.
type VSphereOpt func(*VSphere)

func NewVSphere(t *testing.T, opts ...VSphereOpt) *VSphere {
	checkRequiredEnvVars(t, requiredEnvVars)
	c := buildGovc(t)
	config, err := readVSphereConfig()
	if err != nil {
		t.Fatalf("Failed reading vSphere tests config: %v", err)
	}
	v := &VSphere{
		t:           t,
		GovcClient:  c,
		testsConfig: config,
		fillers: []api.VSphereFiller{
			api.WithDatacenter(config.Datacenter),
			api.WithDatastoreForAllMachines(config.Datastore),
			api.WithFolderForAllMachines(config.Folder),
			api.WithNetwork(config.Network),
			api.WithResourcePoolForAllMachines(config.ResourcePool),
			api.WithServer(config.Server),
			api.WithSSHAuthorizedKeyForAllMachines(config.SSHAuthorizedKey),
			api.WithStoragePolicyNameForAllMachines(config.StoragePolicyName),
			api.WithTLSInsecure(config.TLSInsecure),
			api.WithTLSThumbprint(config.TLSThumbprint),
		},
	}

	v.cidr = os.Getenv(cidrVar)
	v.templatesRegistry = &templateRegistry{cache: map[string]string{}, generator: v}
	for _, opt := range opts {
		opt(v)
	}

	return v
}

// withVSphereKubeVersionAndOS returns a VSphereOpt that adds API fillers to use a vSphere template for
// the specified OS family and version (default if not provided), corresponding to a particular
// Kubernetes version, in addition to configuring all machine configs to use this OS family.
func withVSphereKubeVersionAndOS(os OS, kubeVersion anywherev1.KubernetesVersion) VSphereOpt {
	return func(v *VSphere) {
		v.fillers = append(v.fillers,
			v.templateForKubeVersionAndOS(os, kubeVersion),
			api.WithOsFamilyForAllMachines(osFamiliesForOS[os]),
		)
	}
}

// WithRedHat123VSphere vsphere test with redhat 1.23.
func WithRedHat123VSphere() VSphereOpt {
	return withVSphereKubeVersionAndOS(RedHat8, anywherev1.Kube123)
}

// WithRedHat124VSphere vsphere test with redhat 1.24.
func WithRedHat124VSphere() VSphereOpt {
	return withVSphereKubeVersionAndOS(RedHat8, anywherev1.Kube124)
}

// WithRedHat125VSphere vsphere test with redhat 1.25.
func WithRedHat125VSphere() VSphereOpt {
	return withVSphereKubeVersionAndOS(RedHat8, anywherev1.Kube125)
}

// WithRedHat126VSphere vsphere test with redhat 1.26.
func WithRedHat126VSphere() VSphereOpt {
	return withVSphereKubeVersionAndOS(RedHat8, anywherev1.Kube126)
}

// WithRedHat127VSphere vsphere test with redhat 1.27.
func WithRedHat127VSphere() VSphereOpt {
	return withVSphereKubeVersionAndOS(RedHat8, anywherev1.Kube127)
}

// WithUbuntu127 returns a VSphereOpt that adds API fillers to use a Ubuntu vSphere template for k8s 1.27
// and the "ubuntu" osFamily in all machine configs.
func WithUbuntu127() VSphereOpt {
	return withVSphereKubeVersionAndOS(Ubuntu2004, anywherev1.Kube127)
}

// WithUbuntu126 returns a VSphereOpt that adds API fillers to use a Ubuntu vSphere template for k8s 1.26
// and the "ubuntu" osFamily in all machine configs.
func WithUbuntu126() VSphereOpt {
	return withVSphereKubeVersionAndOS(Ubuntu2004, anywherev1.Kube126)
}

// WithUbuntu125 returns a VSphereOpt that adds API fillers to use a Ubuntu vSphere template for k8s 1.25
// and the "ubuntu" osFamily in all machine configs.
func WithUbuntu125() VSphereOpt {
	return withVSphereKubeVersionAndOS(Ubuntu2004, anywherev1.Kube125)
}

// WithUbuntu124 returns a VSphereOpt that adds API fillers to use a Ubuntu vSphere template for k8s 1.24
// and the "ubuntu" osFamily in all machine configs.
func WithUbuntu124() VSphereOpt {
	return withVSphereKubeVersionAndOS(Ubuntu2004, anywherev1.Kube124)
}

// WithUbuntu123 returns a VSphereOpt that adds API fillers to use a Ubuntu vSphere template for k8s 1.23
// and the "ubuntu" osFamily in all machine configs.
func WithUbuntu123() VSphereOpt {
	return withVSphereKubeVersionAndOS(Ubuntu2004, anywherev1.Kube123)
}

func WithBottleRocket123() VSphereOpt {
	return withVSphereKubeVersionAndOS(Bottlerocket1, anywherev1.Kube123)
}

// WithBottleRocket124 returns br 124 var.
func WithBottleRocket124() VSphereOpt {
	return withVSphereKubeVersionAndOS(Bottlerocket1, anywherev1.Kube124)
}

// WithBottleRocket125 returns br 1.25 var.
func WithBottleRocket125() VSphereOpt {
	return withVSphereKubeVersionAndOS(Bottlerocket1, anywherev1.Kube125)
}

// WithBottleRocket126 returns br 1.26 var.
func WithBottleRocket126() VSphereOpt {
	return withVSphereKubeVersionAndOS(Bottlerocket1, anywherev1.Kube126)
}

// WithBottleRocket127 returns br 1.27 var.
func WithBottleRocket127() VSphereOpt {
	return withVSphereKubeVersionAndOS(Bottlerocket1, anywherev1.Kube127)
}

func WithPrivateNetwork() VSphereOpt {
	return func(v *VSphere) {
		v.fillers = append(v.fillers,
			api.WithVSphereStringFromEnvVar(vspherePrivateNetworkVar, api.WithNetwork),
		)
		v.cidr = os.Getenv(privateNetworkCidrVar)
	}
}

// WithLinkedCloneMode sets clone mode to LinkedClone for all the machine.
func WithLinkedCloneMode() VSphereOpt {
	return func(v *VSphere) {
		v.fillers = append(v.fillers,
			api.WithCloneModeForAllMachines(anywherev1.LinkedClone),
		)
	}
}

// WithFullCloneMode sets clone mode to FullClone for all the machine.
func WithFullCloneMode() VSphereOpt {
	return func(v *VSphere) {
		v.fillers = append(v.fillers,
			api.WithCloneModeForAllMachines(anywherev1.FullClone),
		)
	}
}

// WithDiskGiBForAllMachines sets diskGiB for all the machines.
func WithDiskGiBForAllMachines(value int) VSphereOpt {
	return func(v *VSphere) {
		v.fillers = append(v.fillers,
			api.WithDiskGiBForAllMachines(value),
		)
	}
}

// WithNTPServersForAllMachines sets NTP servers for all the machines.
func WithNTPServersForAllMachines() VSphereOpt {
	return func(v *VSphere) {
		checkRequiredEnvVars(v.t, RequiredNTPServersEnvVars())
		v.fillers = append(v.fillers,
			api.WithNTPServersForAllMachines(GetNTPServersFromEnv()),
		)
	}
}

// WithBottlerocketKubernetesSettingsForAllMachines sets Bottlerocket Kubernetes settings for all the machines.
func WithBottlerocketKubernetesSettingsForAllMachines() VSphereOpt {
	return func(v *VSphere) {
		checkRequiredEnvVars(v.t, RequiredBottlerocketKubernetesSettingsEnvVars())
		unsafeSysctls, clusterDNSIPS, maxPods, err := GetBottlerocketKubernetesSettingsFromEnv()
		if err != nil {
			v.t.Fatalf("failed to get bottlerocket kubernetes settings from env: %v", err)
		}
		config := &anywherev1.BottlerocketConfiguration{
			Kubernetes: &v1beta1.BottlerocketKubernetesSettings{
				AllowedUnsafeSysctls: unsafeSysctls,
				ClusterDNSIPs:        clusterDNSIPS,
				MaxPods:              maxPods,
			},
		}
		v.fillers = append(v.fillers,
			api.WithBottlerocketConfigurationForAllMachines(config),
		)
	}
}

// WithSSHAuthorizedKeyForAllMachines sets SSH authorized keys for all the machines.
func WithSSHAuthorizedKeyForAllMachines(sshKey string) VSphereOpt {
	return func(v *VSphere) {
		v.fillers = append(v.fillers, api.WithSSHAuthorizedKeyForAllMachines(sshKey))
	}
}

// WithVSphereTags with vsphere tags option.
func WithVSphereTags() VSphereOpt {
	return func(v *VSphere) {
		tags := []string{os.Getenv(vsphereTestTagEnvVar)}
		v.fillers = append(v.fillers,
			api.WithTagsForAllMachines(tags),
		)
	}
}

func WithVSphereWorkerNodeGroup(name string, workerNodeGroup *WorkerNodeGroup, fillers ...api.VSphereMachineConfigFiller) VSphereOpt {
	return func(v *VSphere) {
		v.fillers = append(v.fillers, vSphereMachineConfig(name, fillers...))

		v.clusterFillers = append(v.clusterFillers, buildVSphereWorkerNodeGroupClusterFiller(name, workerNodeGroup))
	}
}

// WithNewWorkerNodeGroup returns an api.ClusterFiller that adds a new workerNodeGroupConfiguration and
// a corresponding VSphereMachineConfig to the cluster config.
func (v *VSphere) WithNewWorkerNodeGroup(name string, workerNodeGroup *WorkerNodeGroup) api.ClusterConfigFiller {
	machineConfigFillers := []api.VSphereMachineConfigFiller{updateMachineSSHAuthorizedKey()}
	return api.JoinClusterConfigFillers(
		api.VSphereToConfigFiller(vSphereMachineConfig(name, machineConfigFillers...)),
		api.ClusterToConfigFiller(buildVSphereWorkerNodeGroupClusterFiller(name, workerNodeGroup)),
	)
}

// WithWorkerNodeGroupConfiguration returns an api.ClusterFiller that adds a new workerNodeGroupConfiguration item to the cluster config.
func (v *VSphere) WithWorkerNodeGroupConfiguration(name string, workerNodeGroup *WorkerNodeGroup) api.ClusterConfigFiller {
	return api.ClusterToConfigFiller(buildVSphereWorkerNodeGroupClusterFiller(name, workerNodeGroup))
}

// updateMachineSSHAuthorizedKey updates a vsphere machine configs SSHAuthorizedKey.
func updateMachineSSHAuthorizedKey() api.VSphereMachineConfigFiller {
	return api.WithStringFromEnvVar(vsphereSshAuthorizedKeyVar, api.WithSSHKey)
}

// WithVSphereFillers adds VSphereFiller to the provider default fillers.
func WithVSphereFillers(fillers ...api.VSphereFiller) VSphereOpt {
	return func(v *VSphere) {
		v.fillers = append(v.fillers, fillers...)
	}
}

// Name returns the provider name. It satisfies the test framework Provider.
func (v *VSphere) Name() string {
	return "vsphere"
}

// Setup does nothing. It satisfies the test framework Provider.
func (v *VSphere) Setup() {}

// UpdateKubeConfig customizes generated kubeconfig for the provider.
func (v *VSphere) UpdateKubeConfig(content *[]byte, clusterName string) error {
	return nil
}

// ClusterConfigUpdates satisfies the test framework Provider.
func (v *VSphere) ClusterConfigUpdates() []api.ClusterConfigFiller {
	clusterIP, err := GetIP(v.cidr, ClusterIPPoolEnvVar)
	if err != nil {
		v.t.Fatalf("failed to get cluster ip for test environment: %v", err)
	}

	f := make([]api.ClusterFiller, 0, len(v.clusterFillers)+1)
	f = append(f, v.clusterFillers...)
	f = append(f, api.WithControlPlaneEndpointIP(clusterIP))

	return []api.ClusterConfigFiller{api.ClusterToConfigFiller(f...), api.VSphereToConfigFiller(v.fillers...)}
}

// WithKubeVersionAndOS returns a cluster config filler that sets the cluster kube version and the right template for all
// vsphere machine configs.
func (v *VSphere) WithKubeVersionAndOS(kubeVersion anywherev1.KubernetesVersion, os OS) api.ClusterConfigFiller {
	return api.JoinClusterConfigFillers(
		api.ClusterToConfigFiller(api.WithKubernetesVersion(kubeVersion)),
		api.VSphereToConfigFiller(
			v.templateForKubeVersionAndOS(os, kubeVersion),
			api.WithOsFamilyForAllMachines(osFamiliesForOS[os]),
		),
	)
}

// WithUbuntu123 returns a cluster config filler that sets the kubernetes version of the cluster to 1.23
// as well as the right ubuntu template and osFamily for all VSphereMachineConfigs.
func (v *VSphere) WithUbuntu123() api.ClusterConfigFiller {
	return v.WithKubeVersionAndOS(Ubuntu2004, anywherev1.Kube123)
}

// WithUbuntu124 returns a cluster config filler that sets the kubernetes version of the cluster to 1.24
// as well as the right ubuntu template and osFamily for all VSphereMachineConfigs.
func (v *VSphere) WithUbuntu124() api.ClusterConfigFiller {
	return v.WithKubeVersionAndOS(Ubuntu2004, anywherev1.Kube124)
}

// WithUbuntu125 returns a cluster config filler that sets the kubernetes version of the cluster to 1.25
// as well as the right ubuntu template and osFamily for all VSphereMachineConfigs.
func (v *VSphere) WithUbuntu125() api.ClusterConfigFiller {
	return v.WithKubeVersionAndOS(Ubuntu2004, anywherev1.Kube125)
}

// WithUbuntu126 returns a cluster config filler that sets the kubernetes version of the cluster to 1.26
// as well as the right ubuntu template and osFamily for all VSphereMachineConfigs.
func (v *VSphere) WithUbuntu126() api.ClusterConfigFiller {
	return v.WithKubeVersionAndOS(Ubuntu2004, anywherev1.Kube126)
}

// WithBottleRocket123 returns a cluster config filler that sets the kubernetes version of the cluster to 1.23
// as well as the right botllerocket template and osFamily for all VSphereMachaineConfigs.
func (v *VSphere) WithBottleRocket123() api.ClusterConfigFiller {
	return v.WithKubeVersionAndOS(Bottlerocket1, anywherev1.Kube123)
}

// WithBottleRocket124 returns a cluster config filler that sets the kubernetes version of the cluster to 1.24
// as well as the right botllerocket template and osFamily for all VSphereMachaineConfigs.
func (v *VSphere) WithBottleRocket124() api.ClusterConfigFiller {
	return v.WithKubeVersionAndOS(Bottlerocket1, anywherev1.Kube123)
}

// CleanupVMs deletes all the VMs owned by the test EKS-A cluster. It satisfies the test framework Provider.
func (v *VSphere) CleanupVMs(clusterName string) error {
	return cleanup.CleanUpVsphereTestResources(context.Background(), clusterName)
}

func (v *VSphere) WithProviderUpgrade(fillers ...api.VSphereFiller) ClusterE2ETestOpt {
	return func(e *ClusterE2ETest) {
		e.UpdateClusterConfig(api.VSphereToConfigFiller(fillers...))
	}
}

func (v *VSphere) WithProviderUpgradeGit(fillers ...api.VSphereFiller) ClusterE2ETestOpt {
	return func(e *ClusterE2ETest) {
		e.UpdateClusterConfig(api.VSphereToConfigFiller(fillers...))
	}
}

// WithNewVSphereWorkerNodeGroup adds a new worker node group to the cluster config.
func (v *VSphere) WithNewVSphereWorkerNodeGroup(name string, workerNodeGroup *WorkerNodeGroup) ClusterE2ETestOpt {
	return func(e *ClusterE2ETest) {
		e.UpdateClusterConfig(
			api.ClusterToConfigFiller(buildVSphereWorkerNodeGroupClusterFiller(name, workerNodeGroup)),
		)
	}
}

// templateForKubeVersionAndOS returns a vSphere filler for the given OS and Kubernetes version.
func (v *VSphere) templateForKubeVersionAndOS(os OS, kubeVersion anywherev1.KubernetesVersion) api.VSphereFiller {
	return api.WithTemplateForAllMachines(v.templateForDevRelease(os, kubeVersion))
}

// Ubuntu123Template returns vsphere filler for 1.23 Ubuntu.
func (v *VSphere) Ubuntu123Template() api.VSphereFiller {
	return v.templateForKubeVersionAndOS(Ubuntu2004, anywherev1.Kube123)
}

// Ubuntu124Template returns vsphere filler for 1.24 Ubuntu.
func (v *VSphere) Ubuntu124Template() api.VSphereFiller {
	return v.templateForKubeVersionAndOS(Ubuntu2004, anywherev1.Kube124)
}

// Ubuntu125Template returns vsphere filler for 1.25 Ubuntu.
func (v *VSphere) Ubuntu125Template() api.VSphereFiller {
	return v.templateForKubeVersionAndOS(Ubuntu2004, anywherev1.Kube125)
}

// Ubuntu126Template returns vsphere filler for 1.26 Ubuntu.
func (v *VSphere) Ubuntu126Template() api.VSphereFiller {
	return v.templateForKubeVersionAndOS(Ubuntu2004, anywherev1.Kube126)
}

// Ubuntu127Template returns vsphere filler for 1.27 Ubuntu.
func (v *VSphere) Ubuntu127Template() api.VSphereFiller {
	return v.templateForKubeVersionAndOS(Ubuntu2004, anywherev1.Kube127)
}

// Bottlerocket123Template returns vsphere filler for 1.23 BR.
func (v *VSphere) Bottlerocket123Template() api.VSphereFiller {
	return v.templateForKubeVersionAndOS(Bottlerocket1, anywherev1.Kube123)
}

// Bottlerocket124Template returns vsphere filler for 1.24 BR.
func (v *VSphere) Bottlerocket124Template() api.VSphereFiller {
	return v.templateForKubeVersionAndOS(Bottlerocket1, anywherev1.Kube124)
}

// Bottlerocket125Template returns vsphere filler for 1.25 BR.
func (v *VSphere) Bottlerocket125Template() api.VSphereFiller {
	return v.templateForKubeVersionAndOS(Bottlerocket1, anywherev1.Kube125)
}

// Bottlerocket126Template returns vsphere filler for 1.26 BR.
func (v *VSphere) Bottlerocket126Template() api.VSphereFiller {
	return v.templateForKubeVersionAndOS(Bottlerocket1, anywherev1.Kube126)
}

// Bottlerocket127Template returns vsphere filler for 1.27 BR.
func (v *VSphere) Bottlerocket127Template() api.VSphereFiller {
	return v.templateForKubeVersionAndOS(Bottlerocket1, anywherev1.Kube127)
}

// Redhat127Template returns vsphere filler for 1.27 Redhat.
func (v *VSphere) Redhat127Template() api.VSphereFiller {
	return api.WithTemplateForAllMachines(v.templateForDevRelease(RedHat8, anywherev1.Kube127))
}

func (v *VSphere) getDevRelease() *releasev1.EksARelease {
	v.t.Helper()
	if v.devRelease == nil {
		latestRelease, err := getLatestDevRelease()
		if err != nil {
			v.t.Fatal(err)
		}
		v.devRelease = latestRelease
	}

	return v.devRelease
}

func (v *VSphere) templateForDevRelease(os OS, kubeVersion anywherev1.KubernetesVersion, osVersion ...string) string {
	v.t.Helper()
	return v.templatesRegistry.templateForRelease(v.t, os, v.getDevRelease(), kubeVersion)
}

func RequiredVsphereEnvVars() []string {
	return requiredEnvVars
}

// VSphereExtraEnvVarPrefixes returns prefixes for env vars that although not always required,
// might be necessary for certain tests.
func VSphereExtraEnvVarPrefixes() []string {
	return []string{
		vsphereTemplateEnvVarPrefix,
	}
}

func vSphereMachineConfig(name string, fillers ...api.VSphereMachineConfigFiller) api.VSphereFiller {
	f := make([]api.VSphereMachineConfigFiller, 0, len(fillers)+6)
	// Need to add these because at this point the default fillers that assign these
	// values to all machines have already ran
	f = append(f,
		api.WithVSphereMachineDefaultValues(),
		api.WithDatastore(os.Getenv(vsphereDatastoreVar)),
		api.WithFolder(os.Getenv(vsphereFolderVar)),
		api.WithResourcePool(os.Getenv(vsphereResourcePoolVar)),
		api.WithStoragePolicyName(os.Getenv(vsphereStoragePolicyNameVar)),
		api.WithSSHKey(os.Getenv(vsphereSshAuthorizedKeyVar)),
	)
	f = append(f, fillers...)

	return api.WithVSphereMachineConfig(name, f...)
}

func buildVSphereWorkerNodeGroupClusterFiller(machineConfigName string, workerNodeGroup *WorkerNodeGroup) api.ClusterFiller {
	// Set worker node group ref to vsphere machine config
	workerNodeGroup.MachineConfigKind = anywherev1.VSphereMachineConfigKind
	workerNodeGroup.MachineConfigName = machineConfigName
	return workerNodeGroup.ClusterFiller()
}

func WithUbuntuForRelease(release *releasev1.EksARelease, kubeVersion anywherev1.KubernetesVersion) VSphereOpt {
	return optionToSetTemplateForRelease(Ubuntu2004, release, kubeVersion)
}

func WithBottlerocketForRelease(release *releasev1.EksARelease, kubeVersion anywherev1.KubernetesVersion) VSphereOpt {
	return optionToSetTemplateForRelease(Bottlerocket1, release, kubeVersion)
}

// WithRedhatForRelease sets the redhat template for the given release.
func WithRedhatForRelease(release *releasev1.EksARelease, kubeVersion anywherev1.KubernetesVersion) VSphereOpt {
	return optionToSetTemplateForRelease(RedHat8, release, kubeVersion)
}

func (v *VSphere) WithBottleRocketForRelease(release *releasev1.EksARelease, kubeVersion anywherev1.KubernetesVersion) api.ClusterConfigFiller {
	return api.VSphereToConfigFiller(
		api.WithTemplateForAllMachines(v.templatesRegistry.templateForRelease(v.t, Bottlerocket1, release, kubeVersion)),
	)
}

func optionToSetTemplateForRelease(os OS, release *releasev1.EksARelease, kubeVersion anywherev1.KubernetesVersion) VSphereOpt {
	return func(v *VSphere) {
		v.fillers = append(v.fillers,
			api.WithTemplateForAllMachines(v.templatesRegistry.templateForRelease(v.t, os, release, kubeVersion)),
		)
	}
}

// envVarForTemplate looks for explicit configuration through an env var: "T_VSPHERE_TEMPLATE_{osFamily}_{eks-d version}"
// eg: T_VSPHERE_TEMPLATE_REDHAT_KUBERNETES_1_23_EKS_22.
func (v *VSphere) envVarForTemplate(os OS, eksDName string) string {
	templateEnvVar := fmt.Sprintf("T_VSPHERE_TEMPLATE_%s_%s", strings.ToUpper(strings.ReplaceAll(string(os), "-", "_")), strings.ToUpper(strings.ReplaceAll(eksDName, "-", "_")))
	return templateEnvVar
}

// defaultNameForTemplate looks for a template with the name path: "{folder}/{eks-d version}-{osFamily}"
// eg: /SDDC-Datacenter/vm/Templates/kubernetes-1-23-eks-22-redhat.
func (v *VSphere) defaultNameForTemplate(os OS, eksDName string) string {
	folder := v.testsConfig.TemplatesFolder
	if folder == "" {
		v.t.Log("vSphere templates folder is not configured.")
		return ""
	}
	// Use the old template naming scheme for backwards compatibility.
	defaultTemplateName := fmt.Sprintf("%s-%s", strings.ToLower(eksDName), strings.ToLower(string(os)))
	return filepath.Join(folder, defaultTemplateName)
}

// defaultEnvVarForTemplate returns the value of the default template env vars: "T_VSPHERE_TEMPLATE_{osFamily}_{kubeVersion}"
// eg. T_VSPHERE_TEMPLATE_REDHAT_1_23.
func (v *VSphere) defaultEnvVarForTemplate(os OS, kubeVersion anywherev1.KubernetesVersion) string {
	defaultTemplateEnvVar := fmt.Sprintf("T_VSPHERE_TEMPLATE_%s_%s", strings.ToUpper(strings.ReplaceAll(string(os), "-", "_")), strings.ReplaceAll(string(kubeVersion), ".", "_"))
	return defaultTemplateEnvVar
}

// searchTemplate returns template name if the given template exists in the datacenter.
func (v *VSphere) searchTemplate(ctx context.Context, template string) (string, error) {
	foundTemplate, err := v.GovcClient.SearchTemplate(context.Background(), v.testsConfig.Datacenter, template)
	if err != nil {
		return "", err
	}
	return foundTemplate, nil
}

func readVersionsBundles(t testing.TB, release *releasev1.EksARelease, kubeVersion anywherev1.KubernetesVersion) *releasev1.VersionsBundle {
	reader := newFileReader()
	b, err := releases.ReadBundlesForRelease(reader, release)
	if err != nil {
		t.Fatal(err)
	}

	return bundles.VersionsBundleForKubernetesVersion(b, string(kubeVersion))
}

func readVSphereConfig() (vsphereConfig, error) {
	return vsphereConfig{
		Datacenter:        os.Getenv(vsphereDatacenterVar),
		Datastore:         os.Getenv(vsphereDatastoreVar),
		Folder:            os.Getenv(vsphereFolderVar),
		Network:           os.Getenv(vsphereNetworkVar),
		ResourcePool:      os.Getenv(vsphereResourcePoolVar),
		Server:            os.Getenv(vsphereServerVar),
		SSHAuthorizedKey:  os.Getenv(vsphereSshAuthorizedKeyVar),
		StoragePolicyName: os.Getenv(vsphereStoragePolicyNameVar),
		TLSInsecure:       os.Getenv(vsphereTlsInsecureVar) == "true",
		TLSThumbprint:     os.Getenv(vsphereTlsThumbprintVar),
		TemplatesFolder:   os.Getenv(vsphereTemplatesFolder),
	}, nil
}

// ClusterStateValidations returns a list of provider specific validations.
func (v *VSphere) ClusterStateValidations() []clusterf.StateValidation {
	return []clusterf.StateValidation{}
}

// ValidateNodesDiskGiB validates DiskGiB for all the machines.
func (v *VSphere) ValidateNodesDiskGiB(machines map[string]anywheretypes.Machine, expectedDiskSize int) error {
	v.t.Log("===================== Disk Size Validation Task =====================")
	for _, m := range machines {
		v.t.Log("Verifying disk size for VM", "Virtual Machine", m.Metadata.Name)
		diskSize, err := v.GovcClient.GetVMDiskSizeInGB(context.Background(), m.Metadata.Name, v.testsConfig.Datacenter)
		if err != nil {
			v.t.Fatalf("validating disk size: %v", err)
		}

		v.t.Log("Disk Size in GiB", "Expected", expectedDiskSize, "Actual", diskSize)
		if diskSize != expectedDiskSize {
			v.t.Fatalf("diskGib for node %s did not match the expected disk size. Expected=%dGiB, Actual=%dGiB", m.Metadata.Name, expectedDiskSize, diskSize)
		}
	}
	return nil
}

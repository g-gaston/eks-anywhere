package cluster_test

import (
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aws/eks-anywhere/internal/test"
	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/cluster"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name                      string
		yamlManifest              []byte
		wantCluster               *anywherev1.Cluster
		wantVsphereDatacenter     *anywherev1.VSphereDatacenterConfig
		wantDockerDatacenter      *anywherev1.DockerDatacenterConfig
		wantVsphereMachineConfigs []*anywherev1.VSphereMachineConfig
		wantOIDCConfigs           []*anywherev1.OIDCConfig
		wantAWSIamConfigs         []*anywherev1.AWSIamConfig
		wantGitOpsConfig          *anywherev1.GitOpsConfig
		wantErr                   bool
	}{
		{
			name:         "cluster 1.19 valid",
			yamlManifest: []byte(test.ReadFile(t, "testdata/cluster_1_19.yaml")),
			wantCluster: &anywherev1.Cluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       anywherev1.ClusterKind,
					APIVersion: anywherev1.SchemeBuilder.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eksa-unit-test",
				},
				Spec: anywherev1.ClusterSpec{
					KubernetesVersion: "1.19",
					ControlPlaneConfiguration: anywherev1.ControlPlaneConfiguration{
						Count:    1,
						Endpoint: &anywherev1.Endpoint{Host: "myHostIp"},
						MachineGroupRef: &anywherev1.Ref{
							Kind: "VSphereMachineConfig",
							Name: "eksa-unit-test-cp",
						},
					},
					WorkerNodeGroupConfigurations: []anywherev1.WorkerNodeGroupConfiguration{
						{
							Count: 1,
							MachineGroupRef: &anywherev1.Ref{
								Kind: "VSphereMachineConfig",
								Name: "eksa-unit-test",
							},
						},
					},
					DatacenterRef: anywherev1.Ref{
						Kind: "VSphereDatacenterConfig",
						Name: "eksa-unit-test",
					},
					ClusterNetwork: anywherev1.ClusterNetwork{
						Pods: anywherev1.Pods{
							CidrBlocks: []string{"192.168.0.0/16"},
						},
						Services: anywherev1.Services{
							CidrBlocks: []string{"10.96.0.0/12"},
						},
						CNI: "cilium",
					},
				},
			},
			wantVsphereDatacenter: &anywherev1.VSphereDatacenterConfig{
				TypeMeta: metav1.TypeMeta{
					Kind:       anywherev1.VSphereDatacenterKind,
					APIVersion: anywherev1.SchemeBuilder.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "eksa-unit-test",
				},
				Spec: anywherev1.VSphereDatacenterConfigSpec{
					Datacenter: "myDatacenter",
					Network:    "myNetwork",
					Server:     "myServer",
					Thumbprint: "myTlsThumbprint",
					Insecure:   false,
				},
			},
			wantVsphereMachineConfigs: []*anywherev1.VSphereMachineConfig{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       anywherev1.VSphereMachineConfigKind,
						APIVersion: anywherev1.SchemeBuilder.GroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "eksa-unit-test-cp",
					},
					Spec: anywherev1.VSphereMachineConfigSpec{
						DiskGiB:   25,
						MemoryMiB: 8192,
						NumCPUs:   2,
						OSFamily:  anywherev1.Ubuntu,
						Users: []anywherev1.UserConfiguration{{
							Name:              "mySshUsername",
							SshAuthorizedKeys: []string{"mySshAuthorizedKey"},
						}},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       anywherev1.VSphereMachineConfigKind,
						APIVersion: anywherev1.SchemeBuilder.GroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "eksa-unit-test",
					},
					Spec: anywherev1.VSphereMachineConfigSpec{
						DiskGiB:   25,
						MemoryMiB: 8192,
						NumCPUs:   2,
						OSFamily:  anywherev1.Ubuntu,
						Users: []anywherev1.UserConfiguration{{
							Name:              "mySshUsername",
							SshAuthorizedKeys: []string{"mySshAuthorizedKey"},
						}},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			got, err := cluster.ParseConfig(tt.yamlManifest)

			g.Expect(err).To(Not(HaveOccurred()))
			
			g.Expect(got.Cluster()).To(Equal(tt.wantCluster))
			g.Expect(got.VSphereDatacenter()).To(Equal(tt.wantVsphereDatacenter))
			g.Expect(got.DockerDatacenter()).To(Equal(tt.wantDockerDatacenter))
			g.Expect(len(got.MachineConfigs())).To(Equal(len(tt.wantVsphereMachineConfigs)))
			for _, m := range tt.wantVsphereMachineConfigs {
				g.Expect(got.VsphereMachineConfig(m.Name)).To(Equal(m))
			}
			for _, o := range tt.wantOIDCConfigs {
				g.Expect(got.OIDCConfig(o.Name)).To(Equal(o))
			}
			for _, a := range tt.wantAWSIamConfigs {
				g.Expect(got.AWSIamConfig(a.Name)).To(Equal(a))
			}
			g.Expect(got.GitOpsConfig()).To(Equal(tt.wantGitOpsConfig))
		})
	}
}

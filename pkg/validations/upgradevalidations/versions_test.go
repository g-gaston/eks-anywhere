package upgradevalidations_test

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/aws/eks-anywhere/internal/test"
	"github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/validations"
	"github.com/aws/eks-anywhere/pkg/validations/upgradevalidations"
)

func TestValidateVersionSkew(t *testing.T) {
	tests := []struct {
		name                  string
		wantErr               error
		upgradeVersion        v1alpha1.KubernetesVersion
		serverVersionResponse string
	}{
		{
			name:                  "FailureTwoMinorVersions",
			wantErr:               errors.New("MINOR version difference between upgrade version (1.20) and server version (1.18) does not meet the supported version increment of +1"),
			upgradeVersion:        v1alpha1.Kube120,
			serverVersionResponse: "testdata/kubectl_version_server_118.json",
		},
		{
			name:                  "FailureMinusOneMinorVersion",
			wantErr:               errors.New("MINOR version downgrade is not supported. Difference between upgrade version (1.19) and server version (1.20) should meet the supported version increment of +1"),
			upgradeVersion:        v1alpha1.Kube119,
			serverVersionResponse: "testdata/kubectl_version_server_120.json",
		},
		{
			name:                  "SuccessSameVersion",
			wantErr:               nil,
			upgradeVersion:        v1alpha1.Kube119,
			serverVersionResponse: "testdata/kubectl_version_server_119.json",
		},
		{
			name:                  "SuccessOneMinorVersion",
			wantErr:               nil,
			upgradeVersion:        v1alpha1.Kube120,
			serverVersionResponse: "testdata/kubectl_version_server_119.json",
		},
		{
			name:                  "FailureParsingVersion",
			wantErr:               errors.New("parsing comparison version: could not parse \"test\" as version"),
			upgradeVersion:        "test",
			serverVersionResponse: "testdata/kubectl_version_server_119.json",
		},
		{
			name:                  "FailureParsingServerVersion",
			wantErr:               errors.New("parsing cluster version: could not parse \"test\" as version"),
			upgradeVersion:        v1alpha1.Kube119,
			serverVersionResponse: "testdata/kubectl_invalid_server.json",
		},
	}

	k, ctx, cluster, e := validations.NewKubectl(t)
	for _, tc := range tests {
		t.Run(tc.name, func(tt *testing.T) {
			fileContent := test.ReadFile(t, tc.serverVersionResponse)
			e.EXPECT().Execute(ctx, []string{"version", "-o", "json", "--kubeconfig", cluster.KubeconfigFile}).Return(*bytes.NewBufferString(fileContent), nil)
			err := upgradevalidations.ValidateServerVersionSkew(ctx, tc.upgradeVersion, cluster, k)
			if err != nil && !reflect.DeepEqual(err.Error(), tc.wantErr.Error()) {
				t.Errorf("%v got = %v, \nwant %v", tc.name, err, tc.wantErr)
			}
		})
	}
}

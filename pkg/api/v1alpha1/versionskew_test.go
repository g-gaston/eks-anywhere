package v1alpha1_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/version"

	"github.com/aws/eks-anywhere/pkg/api/v1alpha1"
)

func TestValidateVersionSkew(t *testing.T) {
	v122, _ := version.ParseGeneric(string(v1alpha1.Kube122))
	v123, _ := version.ParseGeneric(string(v1alpha1.Kube123))
	v124, _ := version.ParseGeneric(string(v1alpha1.Kube124))
	v21, _ := version.ParseGeneric(string("2.1"))

	tests := []struct {
		name       string
		oldVersion *version.Version
		newVersion *version.Version
		wantErr    error
	}{
		{
			name:       "No upgrade",
			oldVersion: v122,
			newVersion: v122,
			wantErr:    nil,
		},
		{
			name:       "Minor version increment success",
			oldVersion: v122,
			newVersion: v123,
			wantErr:    nil,
		},
		{
			name:       "Major version change failure",
			oldVersion: v122,
			newVersion: v21,
			wantErr:    errors.New("MAJOR version upgrades are not supported. Major version difference between upgrade version (2.1) and server version (1.22) is 1.000000"),
		},
		{
			name:       "Minor version invalid, failure",
			oldVersion: v122,
			newVersion: v124,
			wantErr:    errors.New("MINOR version difference between upgrade version (1.24) and server version (1.22) does not meet the supported version increment of +1"),
		},
		{
			name:       "Minor version downgrade, failure",
			oldVersion: v124,
			newVersion: v123,
			wantErr:    fmt.Errorf("MINOR version downgrade is not supported. Difference between upgrade version (1.23) and server version (1.24) should meet the supported version increment of +1"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v1alpha1.ValidateVersionSkew(tt.oldVersion, tt.newVersion)
			if err != nil && !reflect.DeepEqual(err.Error(), tt.wantErr.Error()) {
				t.Errorf("ValidateVersionSkew() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

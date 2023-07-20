package clusters_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/aws/eks-anywhere/internal/test"
	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/controller"
	"github.com/aws/eks-anywhere/pkg/controller/clusters"
)

func TestCleanupStatusAfterValidate(t *testing.T) {
	g := NewWithT(t)
	spec := test.NewClusterSpec(func(s *cluster.Spec) {
		s.Cluster.SetFailure(anywherev1.FailureReasonType("InvalidCluster"), "invalid cluster")
	})

	g.Expect(
		clusters.CleanupStatusAfterValidate(context.Background(), test.NewNullLogger(), spec),
	).To(Equal(controller.Result{}))
	g.Expect(spec.Cluster.Status.FailureMessage).To(BeNil())
	g.Expect(spec.Cluster.Status.FailureReason).To(BeNil())
}

func TestValidateManagementClusterNameSuccess(t *testing.T) {
	tt := newClusterValidatorTest(t)

	objs := []runtime.Object{tt.cluster, tt.managementCluster}
	cb := fake.NewClientBuilder()
	cl := cb.WithRuntimeObjects(objs...).Build()

	validator := clusters.NewClusterValidator(cl)
	tt.Expect(validator.ValidateManagementClusterName(context.Background(), tt.logger, tt.cluster)).To(BeNil())
}

func TestValidateManagementClusterNameMissing(t *testing.T) {
	tt := newClusterValidatorTest(t)

	tt.cluster.Spec.ManagementCluster.Name = "missing"
	objs := []runtime.Object{tt.cluster, tt.managementCluster}
	cb := fake.NewClientBuilder()
	cl := cb.WithRuntimeObjects(objs...).Build()

	validator := clusters.NewClusterValidator(cl)
	tt.Expect(validator.ValidateManagementClusterName(context.Background(), tt.logger, tt.cluster)).
		To(MatchError(errors.New("unable to retrieve management cluster missing: clusters.anywhere.eks.amazonaws.com \"missing\" not found")))
}

func TestValidateManagementClusterNameInvalid(t *testing.T) {
	tt := newClusterValidatorTest(t)

	tt.managementCluster.SetManagedBy("differentCluster")
	objs := []runtime.Object{tt.cluster, tt.managementCluster}
	cb := fake.NewClientBuilder()
	cl := cb.WithRuntimeObjects(objs...).Build()

	validator := clusters.NewClusterValidator(cl)
	tt.Expect(validator.ValidateManagementClusterName(context.Background(), tt.logger, tt.cluster)).
		To(MatchError(errors.New("my-management-cluster is not a valid management cluster")))
}

func TestValidateManagementEksaVersionInvalid(t *testing.T) {
	v1 := anywherev1.EksaVersion("v0.0.0")
	v2 := anywherev1.EksaVersion("v0.1.0")
	badVersion := anywherev1.EksaVersion("badvalue")
	tests := []struct {
		name              string
		wantErr           string
		managementVersion *anywherev1.EksaVersion
		workerVersion     *anywherev1.EksaVersion
	}{
		{
			name:              "management nil",
			wantErr:           "management cluster has nil EksaVersion",
			managementVersion: nil,
			workerVersion:     &v1,
		},
		{
			name:              "worker nil",
			wantErr:           "cluster has nil EksaVersion",
			managementVersion: &v1,
			workerVersion:     nil,
		},
		{
			name:              "management invalid version",
			wantErr:           "invalid major version in semver",
			managementVersion: &badVersion,
			workerVersion:     &v1,
		},
		{
			name:              "worker invalid version",
			wantErr:           "invalid major version in semver",
			managementVersion: &v1,
			workerVersion:     &badVersion,
		},
		{
			name:              "fail",
			wantErr:           "cannot upgrade workload cluster with version",
			managementVersion: &v1,
			workerVersion:     &v2,
		},
		{
			name:              "success",
			wantErr:           "cannot upgrade workload cluster with version",
			managementVersion: &v1,
			workerVersion:     &v1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tt := newClusterValidatorTest(t)
			tt.cluster.Spec.EksaVersion = tc.workerVersion
			tt.managementCluster.Spec.EksaVersion = tc.managementVersion
			objs := []runtime.Object{tt.cluster, tt.managementCluster}
			cb := fake.NewClientBuilder()
			cl := cb.WithRuntimeObjects(objs...).Build()

			validator := clusters.NewClusterValidator(cl)
			err := validator.ValidateManagementEksaVersion(context.Background(), tt.logger, tt.cluster)
			if err != nil {
				tt.Expect(err.Error()).To(ContainSubstring(tc.wantErr))
			}
		})
	}
}

type clusterValidatorTest struct {
	*WithT
	logger            logr.Logger
	cluster           *anywherev1.Cluster
	managementCluster *anywherev1.Cluster
}

func newClusterValidatorTest(t *testing.T) *clusterValidatorTest {
	version := anywherev1.EksaVersion("v0.0.0")
	logger := test.NewNullLogger()
	managementCluster := &anywherev1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-management-cluster",
			Namespace: "my-namespace",
		},
		Spec: anywherev1.ClusterSpec{
			EksaVersion: &version,
		},
	}

	cluster := &anywherev1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-cluster",
			Namespace: "my-namespace",
		},
		Spec: anywherev1.ClusterSpec{
			EksaVersion: &version,
		},
	}
	cluster.SetManagedBy("my-management-cluster")
	return &clusterValidatorTest{
		WithT:             NewWithT(t),
		logger:            logger,
		cluster:           cluster,
		managementCluster: managementCluster,
	}
}

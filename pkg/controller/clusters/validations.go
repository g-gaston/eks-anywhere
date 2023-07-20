package clusters

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/controller"
	"github.com/aws/eks-anywhere/pkg/semver"
	"github.com/aws/eks-anywhere/pkg/utils/ptr"
)

// CleanupStatusAfterValidate removes errors from the cluster status. Intended to be used as a reconciler phase
// after all validation phases have been executed.
func CleanupStatusAfterValidate(_ context.Context, _ logr.Logger, spec *cluster.Spec) (controller.Result, error) {
	spec.Cluster.ClearFailure()
	return controller.Result{}, nil
}

// ClusterValidator runs cluster level validations.
type ClusterValidator struct {
	client client.Client
}

// NewClusterValidator returns a validator that will run cluster level validations.
func NewClusterValidator(client client.Client) *ClusterValidator {
	return &ClusterValidator{
		client: client,
	}
}

// ValidateManagementClusterName checks if the management cluster specified in the workload cluster spec is valid.
func (v *ClusterValidator) ValidateManagementClusterName(ctx context.Context, log logr.Logger, cluster *anywherev1.Cluster) error {
	mgmtCluster := &anywherev1.Cluster{}
	mgmtClusterKey := client.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      cluster.Spec.ManagementCluster.Name,
	}
	if err := v.client.Get(ctx, mgmtClusterKey, mgmtCluster); err != nil {
		if apierrors.IsNotFound(err) {
			err := fmt.Errorf("unable to retrieve management cluster %v: %v", cluster.Spec.ManagementCluster.Name, err)
			log.Error(err, "Invalid cluster configuration")
			return err
		}
	}
	if mgmtCluster.IsManaged() {
		err := fmt.Errorf("%s is not a valid management cluster", mgmtCluster.Name)
		log.Error(err, "Invalid cluster configuration")
		return err
	}

	return nil
}

// ValidateManagementEksaVersion checks if the workload cluster's EksaVersion does not exceed its management cluster.
func (v *ClusterValidator) ValidateManagementEksaVersion(ctx context.Context, log logr.Logger, cluster *anywherev1.Cluster) error {
	mgmtCluster := &anywherev1.Cluster{}
	mgmtClusterKey := client.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      cluster.Spec.ManagementCluster.Name,
	}

	if err := v.client.Get(ctx, mgmtClusterKey, mgmtCluster); err != nil {
		if apierrors.IsNotFound(err) {
			err := fmt.Errorf("unable to retrieve management cluster %v: %v", cluster.Spec.ManagementCluster.Name, err)
			log.Error(err, "Invalid cluster configuration")
			return err
		}
	}

	if mgmtCluster.Spec.EksaVersion == nil {
		err := fmt.Errorf("management cluster has nil EksaVersion")
		log.Error(err, "cannot find management cluster's eksaVersion")
		return err
	}

	mVersion, err := semver.New(string(*mgmtCluster.Spec.EksaVersion))
	if err != nil {
		log.Error(err, "Management cluster has invalid EksaVersion")
		return err
	}

	if cluster.Spec.EksaVersion == nil {
		err := fmt.Errorf("cluster has nil EksaVersion")
		log.Error(err, "cannot find cluster's eksaVersion")
		return err
	}

	wVersion, err := semver.New(string(*cluster.Spec.EksaVersion))
	if err != nil {
		log.Error(err, "Workload cluster has invalid EksaVersion")
		return err
	}

	if wVersion.GreaterThan(mVersion) {
		err := fmt.Errorf("cannot upgrade workload cluster with version %v while management cluster is an older version %v", wVersion, mVersion)
		log.Error(err, "Invalid cluster configuration")
		return err
	}

	// reset failure message if old matches this validation
	oldFailure := cluster.Status.FailureMessage
	if oldFailure != nil && strings.Contains(*oldFailure, "cannot upgrade workload cluster with version") {
		cluster.Status.FailureMessage = ptr.String("")
	}
	return nil
}

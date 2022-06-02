package cilium

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/aws/eks-anywhere/controllers/controllers/reconciler"
	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/constants"
)

type Reconciler struct {
	generator YamlGenerator
}

type YamlGenerator interface {
	GenerateManifest(ctx context.Context, clusterSpec *cluster.Spec, providerNamespaces []string) ([]byte, error)
}

func NewReconciler(generator YamlGenerator) Reconciler {
	return Reconciler{
		generator: generator,
	}
}

func (r Reconciler) Reconcile(ctx context.Context, log logr.Logger, client client.Client, clusterSpec *cluster.Spec) (reconciler.Result, error) {
	needsUpgrade, err := ciliumNeedsUpgrade(ctx, log, client, clusterSpec)
	if err != nil {
		return reconciler.Result{}, err
	}

	if !needsUpgrade {
		log.Info("Cilium already updated")
	}

	log.Info("Deploying")

	// TODO: figure out better way to pass namespaces
	// TODO: rewrite this into a proper upgrade flow:
	// TODO: installing the preflights before hand, waiting for it to be ready, etc.
	log.Info("Generating manifest")
	ciliumSpec, err := r.generator.GenerateManifest(ctx, clusterSpec, []string{constants.CapvSystemNamespace})
	if err != nil {
		return reconciler.Result{}, err
	}
	log.Info("Reconciling Cilium")
	if err := reconciler.ReconcileYaml(ctx, client, ciliumSpec); err != nil {
		return reconciler.Result{}, err
	}

	return reconciler.Result{}, nil
}

func getCiliumDS(ctx context.Context, client client.Client) (*v1.DaemonSet, error) {
	ds := &v1.DaemonSet{}
	err := client.Get(ctx, types.NamespacedName{Name: "cilium", Namespace: "kube-system"}, ds)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return ds, nil
}

func getCiliumDeployment(ctx context.Context, client client.Client) (*v1.Deployment, error) {
	deployment := &v1.Deployment{}
	err := client.Get(ctx, types.NamespacedName{Name: "cilium-operator", Namespace: "kube-system"}, deployment)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return deployment, nil
}

func ciliumNeedsUpgrade(ctx context.Context, log logr.Logger, client client.Client, clusterSpec *cluster.Spec) (bool, error) {
	log.Info("Checking if Cilium DS needs upgrade")
	needsUpgrade, err := ciliumDSNeedsUpgrade(ctx, log, client, clusterSpec)
	if err != nil {
		return false, err
	}

	if needsUpgrade {
		log.Info("Cilium DS needs upgrade")
		return true, nil
	}

	log.Info("Checking if Cilium operator deployment needs upgrade")
	needsUpgrade, err = ciliumOperatorNeedsUpgrade(ctx, log, client, clusterSpec)
	if err != nil {
		return false, err
	}

	if needsUpgrade {
		log.Info("Cilium operator deployment needs upgrade")
		return true, nil
	}

	return false, nil
}

func ciliumDSNeedsUpgrade(ctx context.Context, log logr.Logger, client client.Client, clusterSpec *cluster.Spec) (bool, error) {
	ds, err := getCiliumDS(ctx, client)
	if err != nil {
		return false, err
	}

	if ds == nil {
		log.Info("Cilium DS doesn't exist")
		return true, nil
	}

	dsImage := clusterSpec.VersionsBundle.Cilium.Cilium.VersionedImage()
	containers := make([]corev1.Container, 0, len(ds.Spec.Template.Spec.Containers)+len(ds.Spec.Template.Spec.InitContainers))
	for _, c := range containers {
		if c.Image != dsImage {
			log.Info("Cilium DS container needs upgrade", "container", c.Name)
			return true, nil
		}
	}

	return false, nil
}

func ciliumOperatorNeedsUpgrade(ctx context.Context, log logr.Logger, client client.Client, clusterSpec *cluster.Spec) (bool, error) {
	operator, err := getCiliumDeployment(ctx, client)
	if err != nil {
		return false, err
	}

	if operator == nil {
		log.Info("Cilium operator deployment doesn't exist")
		return true, nil
	}

	operatorImage := clusterSpec.VersionsBundle.Cilium.Operator.VersionedImage()
	if len(operator.Spec.Template.Spec.Containers) == 0 {
		return false, errors.New("cilium-operator deployment doesn't have any containers")
	}

	if operator.Spec.Template.Spec.Containers[0].Image != operatorImage {
		return true, nil
	}

	return false, nil
}

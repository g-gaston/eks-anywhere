package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/aws/eks-anywhere/controllers/controllers/registry"
	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/constants"
)

const (
	defaultRequeueTime   = time.Minute
	clusterFinalizerName = "clusters.anywhere.eks.amazonaws.com/finalizer"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client              client.Client
	log                 logr.Logger
	providerReconcilers registry.ClusterReconcilerRegistry
}

func NewClusterReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme, providerReconcilers registry.ClusterReconcilerRegistry) *ClusterReconciler {
	return &ClusterReconciler{
		client:              client,
		log:                 log,
		providerReconcilers: providerReconcilers,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&anywherev1.Cluster{}).
		Watches(&source.Kind{Type: &clusterv1.Cluster{}}, handler.EnqueueRequestsFromMapFunc(r.capiClusterToCluster)).
		// Watches(&source.Kind{Type: &anywherev1.VSphereDatacenterConfig{}}, &handler.EnqueueRequestForObject{}).
		// Watches(&source.Kind{Type: &anywherev1.VSphereMachineConfig{}}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}

//+kubebuilder:rbac:groups=anywhere.eks.amazonaws.com,resources=clusters;vspheredatacenterconfigs;vspheremachineconfigs;dockerdatacenterconfigs;bundles;awsiamconfigs;snowmachineconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=anywhere.eks.amazonaws.com,resources=oidcconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=anywhere.eks.amazonaws.com,resources=clusters/status;vspheredatacenterconfigs/status;vspheremachineconfigs/status;dockerdatacenterconfigs/status;bundles/status;awsiamconfigs/status;snowmachineconfigs/status,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=anywhere.eks.amazonaws.com,resources=clusters/finalizers;vspheredatacenterconfigs/finalizers;vspheremachineconfigs/finalizers;dockerdatacenterconfigs/finalizers;bundles/finalizers;awsiamconfigs/finalizers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=test,resources=test,verbs=get;list;watch;create;update;patch;delete;kill
//+kubebuilder:rbac:groups=distro.eks.amazonaws.com,resources=releases,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=*,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=*,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=etcdcluster.cluster.x-k8s.io,resources=*,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bootstrap.cluster.x-k8s.io,resources=*,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=*,verbs=get;list;watch;create;update;patch;delete

func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := r.log.WithValues("cluster", req.NamespacedName)
	// Fetch the Cluster object
	cluster := &anywherev1.Cluster{}
	log.Info("Reconciling cluster", "name", req.NamespacedName)
	if err := r.client.Get(ctx, req.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(cluster, r.client)
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		// Always attempt to patch the object and status after each reconciliation.
		if err := patchHelper.Patch(ctx, cluster); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	if cluster.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(cluster, clusterFinalizerName) {
			controllerutil.AddFinalizer(cluster, clusterFinalizerName)
		}
	} else {
		return r.reconcileDelete(ctx, cluster)
	}

	// If the cluster is paused, return without any further processing.
	if cluster.IsReconcilePaused() {
		log.Info("Cluster reconciliation is paused")
		return ctrl.Result{}, nil
	}

	if cluster.IsSelfManaged() {
		log.Info("Ignoring self managed cluster")
		return ctrl.Result{}, nil
	}

	result, err := r.reconcile(ctx, cluster, log)
	if err != nil {
		failureMessage := err.Error()
		cluster.Status.FailureMessage = &failureMessage
		log.Error(err, "Failed to reconcile Cluster")
	}
	return result, err
}

func (r *ClusterReconciler) reconcile(ctx context.Context, cluster *anywherev1.Cluster, log logr.Logger) (ctrl.Result, error) {
	clusterProviderReconciler := r.providerReconcilers.Get(cluster.Spec.DatacenterRef.Kind)
	if clusterProviderReconciler == nil {
		return ctrl.Result{}, fmt.Errorf("no reconciler available for datacenter of type %s", cluster.Spec.DatacenterRef.Kind)
	}

	reconcileResult, err := clusterProviderReconciler.Reconcile(ctx, log, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}
	return reconcileResult.ToCtrlResult(), nil
}

func (r *ClusterReconciler) reconcileDelete(ctx context.Context, cluster *anywherev1.Cluster) (ctrl.Result, error) {
	capiCluster := &clusterv1.Cluster{}
	capiClusterName := types.NamespacedName{Namespace: constants.EksaSystemNamespace, Name: cluster.Name}
	r.log.Info("Deleting", "name", cluster.Name)
	err := r.client.Get(ctx, capiClusterName, capiCluster)

	switch {
	case err == nil:
		r.log.Info("Deleting CAPI cluster", "name", capiCluster.Name)
		if err := r.client.Delete(ctx, capiCluster); err != nil {
			r.log.Info("Error deleting CAPI cluster", "name", capiCluster.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: defaultRequeueTime}, nil
	case apierrors.IsNotFound(err):
		r.log.Info("Deleting EKS Anywhere cluster", "name", capiCluster.Name, "cluster.DeletionTimestamp", cluster.DeletionTimestamp, "finalizer", cluster.Finalizers)

		// TODO delete GitOps,Datacenter and MachineConfig objects
		controllerutil.RemoveFinalizer(cluster, clusterFinalizerName)
	default:
		return ctrl.Result{}, err

	}
	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) capiClusterToCluster(o client.Object) []ctrl.Request {
	capiCluster, ok := o.(*clusterv1.Cluster)
	if !ok {
		panic(fmt.Sprintf("Expected a CAPI Cluster but got a %T", o))
	}

	return r.objectWithClusterLabelNameToCluster(capiCluster)
}

func (r *ClusterReconciler) objectWithClusterLabelNameToCluster(obj client.Object) []ctrl.Request {
	labels := obj.GetLabels()
	clusterName, ok := labels[constants.ClusterLabelName]
	if !ok {
		// Object not managed by a eks-a Cluster, don't enqueue
		// We could also use ownership for this
		r.log.Info("Object not managed by an eks-a Cluster, ignoring", "type", fmt.Sprintf("%T", obj), "name", obj.GetName())
		return nil
	}

	return []ctrl.Request{{
		NamespacedName: types.NamespacedName{
			Namespace: "default", // TODO: figure a better way of doing this, eksa objects might not be in default
			Name:      clusterName,
		},
	}}
}

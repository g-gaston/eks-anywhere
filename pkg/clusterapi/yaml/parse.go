package yaml

import (
	"github.com/go-logr/logr"
	etcdv1 "github.com/mrajashree/etcdadm-controller/api/v1beta1"
	"github.com/pkg/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"

	"github.com/aws/eks-anywhere/pkg/clusterapi"
	"github.com/aws/eks-anywhere/pkg/yamlutil"
)

type ControlPlane[C, M clusterapi.Object] interface {
	ClusterControlPlane[C]
	KubeadmControlPlane[M]
	EtcdControlPlane[M]
}

type KubeadmControlPlane[M clusterapi.Object] interface {
	GetCluster() *clusterv1.Cluster
	SetKubeadmControlPlane(*controlplanev1.KubeadmControlPlane)
	SetControlPlaneMachineTemplate(M)
}

type EtcdControlPlane[M clusterapi.Object] interface {
	GetCluster() *clusterv1.Cluster
	SetEtcdAdmCluster(*etcdv1.EtcdadmCluster)
	SetEtcdAdmMachineTemplate(M)
}

type ClusterControlPlane[C clusterapi.Object] interface {
	SetCluster(*clusterv1.Cluster)
	SetProviderCluster(C)
}

// ProcessCluster finds the CAPI cluster in the parsed objects and sets it in ControlPlane
func ProcessCluster[C, M clusterapi.Object](cp ControlPlane[C, M], lookup yamlutil.ObjectLookup) {
	for _, obj := range lookup {
		if obj.GetObjectKind().GroupVersionKind().Kind == "Cluster" {
			cluster := obj.(*clusterv1.Cluster)
			cp.SetCluster(cluster)

			providerCluster := lookup.GetFromRef(*cluster.Spec.InfrastructureRef)
			if providerCluster == nil {
				return
			}

			cp.SetProviderCluster(providerCluster.(C))

			return
		}
	}
}

// ProcessCluster finds the provider cluster and the kubeadm control plane machine template in the parsed objects
// and sets it in ControlPlane
// Both Cluster and KubeadmControlPlane have to be processed before this
// func ProcessProviderCluster[M clusterapi.Object](cp ControlPlane[M], lookup yamlutil.ObjectLookup) {
// 	if cp.Cluster == nil || cp.KubeadmControlPlane == nil {
// 		return
// 	}

// 	providerCluster := lookup.GetFromRef(*cp.Cluster.Spec.InfrastructureRef)
// 	if providerCluster == nil {
// 		return
// 	}

// 	cp.ProviderCluster = providerCluster.(C)

// 	machineTemplate := lookup.GetFromRef(cp.KubeadmControlPlane.Spec.MachineTemplate.InfrastructureRef)
// 	if machineTemplate == nil {
// 		return
// 	}

// 	cp.ControlPlaneMachineTemplate = machineTemplate.(M)
// }

// ProcessKubeadmControlPlane finds the CAPI kubeadm control plane in the parsed objects and sets it in ControlPlane
func ProcessKubeadmControlPlane[C, M clusterapi.Object](cp ControlPlane[C, M], lookup yamlutil.ObjectLookup) {
	cluster := cp.GetCluster()
	if cluster == nil {
		return
	}

	kcpObj := lookup.GetFromRef(*cluster.Spec.ControlPlaneRef)
	if kcpObj == nil {
		return
	}

	kcp := kcpObj.(*controlplanev1.KubeadmControlPlane)

	cp.SetKubeadmControlPlane(kcp)

	machineTemplate := lookup.GetFromRef(kcp.Spec.MachineTemplate.InfrastructureRef)
	if machineTemplate == nil {
		return
	}

	cp.SetControlPlaneMachineTemplate(machineTemplate.(M))
}

// ProcessEtcdCluster finds the CAPI etcdadm cluster (for unstacked clusters) in the parsed objects and sets it in ControlPlane
func ProcessEtcdCluster[C, M clusterapi.Object](cp ControlPlane[C, M], lookup yamlutil.ObjectLookup) {
	cluster := cp.GetCluster()
	if cluster == nil || cluster.Spec.ManagedExternalEtcdRef == nil {
		return
	}

	etcdClusterObj := lookup.GetFromRef(*cluster.Spec.ManagedExternalEtcdRef)
	if etcdClusterObj == nil {
		return
	}

	etcdCluster := etcdClusterObj.(*etcdv1.EtcdadmCluster)

	cp.SetEtcdAdmCluster(etcdCluster)

	etcdMachineTemplate := lookup.GetFromRef(etcdCluster.Spec.InfrastructureTemplate)
	if etcdMachineTemplate == nil {
		return
	}

	cp.SetEtcdAdmMachineTemplate(etcdMachineTemplate.(M))
}

// RegisterControlPlaneMappings records the basic mappings for CAPI cluster, kubeadmcontrolplane
// and etcdadm cluster in a Parser
func RegisterControlPlaneMappings[T any](parser *yamlutil.Parser[T]) error {
	err := parser.RegisterMappings(
		yamlutil.NewMapping(
			"Cluster", func() yamlutil.APIObject {
				return &clusterv1.Cluster{}
			},
		),
		yamlutil.NewMapping(
			"KubeadmControlPlane", func() yamlutil.APIObject {
				return &controlplanev1.KubeadmControlPlane{}
			},
		),
		yamlutil.NewMapping(
			"EtcdadmCluster", func() yamlutil.APIObject {
				return &etcdv1.EtcdadmCluster{}
			},
		),
	)
	if err != nil {
		return errors.Wrap(err, "registering base control plan mappings")
	}

	return nil
}

// NewControlPlaneParser builds a Parser for a particular provider ControlPlane
// It registers the basic shared mappings plus the provider cluster and machine template ones
// Any extra mappings or processors for objects in the ProviderControlPlane (P) will need to be
// registered manually
func NewControlPlaneParser[C, M clusterapi.Object](logger logr.Logger, clusterMapping yamlutil.Mapping[C], machineConfigMapping yamlutil.Mapping[M]) (*yamlutil.Parser[ControlPlane[C, M]], error) {
	parser := yamlutil.NewParser[ControlPlane[C, M]](logger)
	if err := RegisterControlPlaneMappings(parser); err != nil {
		return nil, errors.Wrap(err, "building capi control plane parser")
	}

	err := parser.RegisterMappings(
		clusterMapping.ToGenericMapping(),
		machineConfigMapping.ToGenericMapping(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "registering provider mappings")
	}

	parser.RegisterProcessors(
		// Order is important, register CAPICluster before anything else
		ProcessCluster[C, M],
		ProcessKubeadmControlPlane[C, M],
		ProcessEtcdCluster[C, M],
		// ProcessProviderCluster[C, M],
	)

	return parser, nil
}

// func RegisterControlPlaneParser[C ControlPlane](parser *yamlutil.Parser[C], clusterMapping yamlutil.Mapping[yamlutil.APIObject], machineConfigMapping yamlutil.Mapping[yamlutil.APIObject]) error {
// 	if err := RegisterControlPlaneMappings(parser); err != nil {
// 		return errors.Wrap(err, "building capi control plane parser")
// 	}

// 	err := parser.RegisterMappings(
// 		clusterMapping.ToGenericMapping(),
// 		machineConfigMapping.ToGenericMapping(),
// 	)
// 	if err != nil {
// 		return errors.Wrap(err, "registering provider mappings")
// 	}

// 	parser.RegisterProcessors(
// 		// Order is important, register CAPICluster before anything else
// 		ProcessCluster[C],
// 		ProcessKubeadmControlPlane[C, M],
// 		ProcessEtcdCluster[C, M],
// 		ProcessProviderCluster[C, M],
// 	)

// 	return nil
// }

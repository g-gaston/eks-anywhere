package clusterapi

import (
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ControlPlane groups all the CAPI objects necessary
// to build a functional kubernetes control plane
type ControlPlane struct {
	Cluster                 *clusterv1.Cluster
	ProviderCluster         client.Object
	KubeadmControlPlane     *controlplanev1.KubeadmControlPlane
	ProviderMachineTemplate client.Object
}

func (c *ControlPlane) ClientObjects() []client.Object {
	return []client.Object{c.Cluster, c.ProviderCluster, c.KubeadmControlPlane, c.ProviderMachineTemplate}
}

func (c *ControlPlane) RuntimeObjects() []runtime.Object {
	return []runtime.Object{c.Cluster, c.ProviderCluster, c.KubeadmControlPlane, c.ProviderMachineTemplate}
}

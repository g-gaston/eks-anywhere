package registry

type Builder struct {
	reconciler ClusterReconcilerRegistry
}

func NewBuilder() Builder {
	return Builder{}
}

func (b Builder) Add(kind string, reconciler ProviderClusterReconciler) {
	b.reconciler.add(kind, reconciler)
}

func (b Builder) Build() ClusterReconcilerRegistry {
	return b.reconciler
}

package vsphere

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	vspherev1 "sigs.k8s.io/cluster-api-provider-vsphere/api/v1beta1"

	"github.com/aws/eks-anywhere/pkg/clients/kubernetes"
	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/clusterapi"
	yamlcapi "github.com/aws/eks-anywhere/pkg/clusterapi/yaml"
	"github.com/aws/eks-anywhere/pkg/yamlutil"
)

// BaseControlPlane represents a CAPI docker control plane
type BaseControlPlane = clusterapi.ControlPlane[*vspherev1.VSphereCluster, *vspherev1.VSphereMachineTemplate]

type ControlPlaneBuilder struct {
	baseBuilder *yamlcapi.ControlPlaneBuilder[*vspherev1.VSphereCluster, *vspherev1.VSphereMachineTemplate]
	cp          *ControlPlane
}

func (b *ControlPlaneBuilder) BuildFromParsed(lookup yamlutil.ObjectLookup) error {
	b.baseBuilder.BuildFromParsed(lookup)
	b.cp.BaseControlPlane = *b.baseBuilder.ControlPlane
	processSecret(b.cp, lookup)

	return nil
}

// ControlPlane holds the vsphere specific objects for a CAPI docker control plane
type ControlPlane struct {
	BaseControlPlane
	Secrets []*corev1.Secret
}

func (p ControlPlane) Objects() []kubernetes.Object {
	o := p.BaseControlPlane.Objects()
	for _, s := range p.Secrets {
		o = append(o, s)
	}

	return o
}

// ControlPlaneSpec builds a vsohere ControlPlane definition based on a eks-a cluster spec
func ControlPlaneSpec(ctx context.Context, logger logr.Logger, client kubernetes.Client, spec *cluster.Spec) (*ControlPlane, error) {
	// passing nil just for the example
	templateBuilder := NewVsphereTemplateBuilder(nil, nil, nil, nil, time.Now, false)
	controlPlaneYaml, err := templateBuilder.GenerateCAPISpecControlPlane(
		spec,
		func(values map[string]interface{}) {
			values["controlPlaneTemplateName"] = clusterapi.ControlPlaneMachineTemplateName(spec)
			values["etcdTemplateName"] = clusterapi.EtcdAdmMachineTemplateName(spec.Cluster)
			// add all other necessary values, vsphere requires a bunch of stuff
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "generating vsphere control plane yaml spec")
	}

	parser, builder, err := newControlPlaneParser(logger)
	if err != nil {
		return nil, err
	}

	err = parser.Parse(controlPlaneYaml, builder)
	if err != nil {
		return nil, errors.Wrap(err, "parsing docker control plane yaml")
	}

	cp := builder.cp

	if err = cp.UpdateImmutableObjectNames(ctx, client, getMachineTemplate, machineTemplateEqual); err != nil {
		return nil, errors.Wrap(err, "updating docker immutable object names")
	}

	return cp, nil
}

func newControlPlaneParser(logger logr.Logger) (*yamlutil.Parser, *ControlPlaneBuilder, error) {
	parser, baseBuilder, err := yamlcapi.NewControlPlaneParser(
		logger,
		yamlutil.NewMapping(
			"VSphereCluster",
			func() *vspherev1.VSphereCluster {
				return &vspherev1.VSphereCluster{}
			},
		),
		yamlutil.NewMapping(
			"VSphereMachineTemplate",
			func() *vspherev1.VSphereMachineTemplate {
				return &vspherev1.VSphereMachineTemplate{}
			},
		),
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "building vsphere control plane parser")
	}

	err = parser.RegisterMappings(
		yamlutil.NewMapping("Secret", func() yamlutil.APIObject {
			return &corev1.Secret{}
		}),
	)

	if err != nil {
		return nil, nil, errors.Wrap(err, "registering vsphere control plane mappings in parser")
	}

	builder := &ControlPlaneBuilder{
		baseBuilder: baseBuilder,
		cp:          &ControlPlane{},
	}

	return parser, builder, nil
}

func processSecret(c *ControlPlane, lookup yamlutil.ObjectLookup) {
	for _, obj := range lookup {
		if obj.GetObjectKind().GroupVersionKind().Kind == "Secret" {
			c.Secrets = append(c.Secrets, obj.(*corev1.Secret))
		}
	}
}

func getMachineTemplate(ctx context.Context, client kubernetes.Client, name, namespace string) (*vspherev1.VSphereMachineTemplate, error) {
	m := &vspherev1.VSphereMachineTemplate{}
	if err := client.Get(ctx, name, namespace, m); err != nil {
		return nil, err
	}

	return m, nil
}

func machineTemplateEqual(new, old *vspherev1.VSphereMachineTemplate) bool {
	return equality.Semantic.DeepDerivative(new, old)
}

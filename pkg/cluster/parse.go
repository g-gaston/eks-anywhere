package cluster

import (
	"errors"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
)

type Config struct {
	objects               objects
	cluster               *anywherev1.Cluster
	datacenter            object
	vsphereDatacenter     *anywherev1.VSphereDatacenterConfig
	dockerDatacenter      *anywherev1.DockerDatacenterConfig
	machineConfigs        []runtime.Object
	vsphereMachineConfigs map[string]*anywherev1.VSphereMachineConfig
	oidcConfigs           map[string]*anywherev1.OIDCConfig
	awsIamConfigs         map[string]*anywherev1.AWSIamConfig
	gitOpsConfig          *anywherev1.GitOpsConfig
}

func (c *Config) Cluster() *anywherev1.Cluster {
	return c.cluster
}

func (c *Config) VSphereDatacenter() *anywherev1.VSphereDatacenterConfig {
	return c.vsphereDatacenter
}

func (c *Config) DockerDatacenter() *anywherev1.DockerDatacenterConfig {
	return c.dockerDatacenter
}

func (c *Config) VsphereMachineConfig(name string) *anywherev1.VSphereMachineConfig {
	return c.vsphereMachineConfigs[name]
}

func (c *Config) MachineConfigs() []runtime.Object {
	return c.machineConfigs
}

func (c *Config) OIDCConfig(name string) *anywherev1.OIDCConfig {
	return c.oidcConfigs[name]
}

func (c *Config) AWSIamConfig(name string) *anywherev1.AWSIamConfig {
	return c.awsIamConfigs[name]
}

func (c *Config) GitOpsConfig() *anywherev1.GitOpsConfig {
	return c.gitOpsConfig
}

type object interface {
	runtime.Object
	GetName() string
}

func keyForObject(o object) string {
	return key(o.GetObjectKind().GroupVersionKind().GroupVersion().String(), o.GetObjectKind().GroupVersionKind().Kind, o.GetName())
}

type objects map[string]object

func (o objects) add(obj object) {
	o[keyForObject(obj)] = obj
}

func (o objects) getFromRef(apiVersion string, ref anywherev1.Ref) object {
	return o[keyForRef(apiVersion, ref)]
}

type basicObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

func (k *basicObject) empty() bool {
	return k.APIVersion == "" && k.Kind == ""
}

func key(apiVersion, kind, name string) string {
	// this assumes we don't allow to have objects in multiple namespaces
	return fmt.Sprintf("%s%s%s", apiVersion, kind, name)
}

func keyForRef(apiVersion string, ref anywherev1.Ref) string {
	return key(apiVersion, ref.Kind, ref.Name)
}

func ParseConfig(yamlManifest []byte) (*Config, error) {
	config := &Config{
		objects:               objects{},
		vsphereMachineConfigs: map[string]*anywherev1.VSphereMachineConfig{},
		oidcConfigs:           map[string]*anywherev1.OIDCConfig{},
		awsIamConfigs:         map[string]*anywherev1.AWSIamConfig{},
	}
	yamlObjs := strings.Split(string(yamlManifest), "---")

	for _, yamlObj := range yamlObjs {
		k := &basicObject{}
		err := yaml.Unmarshal([]byte(yamlObj), k)
		if err != nil {
			return nil, err
		}

		// Ignore empty objects.
		// Empty objects are generated if there are weird things in manifest files like e.g. two --- in a row without a yaml doc in the middle
		if k.empty() {
			continue
		}

		var obj object

		switch k.Kind {
		case anywherev1.ClusterKind:
			if config.cluster != nil {
				return nil, errors.New("only one Cluster per yaml manifest is allowed")
			}
			config.cluster = &anywherev1.Cluster{}
			obj = config.cluster
		case anywherev1.VSphereDatacenterKind:
			obj = &anywherev1.VSphereDatacenterConfig{}
		case anywherev1.VSphereMachineConfigKind:
			obj = &anywherev1.VSphereMachineConfig{}
		case anywherev1.DockerDatacenterKind:
			obj = &anywherev1.DockerDatacenterConfig{}
		case anywherev1.AWSIamConfigKind:
			obj = &anywherev1.AWSIamConfig{}
		case anywherev1.OIDCConfigKind:
			obj = &anywherev1.OIDCConfig{}
		case anywherev1.GitOpsConfigKind:
			obj = &anywherev1.GitOpsConfig{}
		default:
			return nil, fmt.Errorf("invalid object with kind %s found on manifest", k.Kind)
		}

		if err := yaml.Unmarshal([]byte(yamlObj), obj); err != nil {
			return nil, err
		}

		config.objects.add(obj)
	}

	if err := processObjects(config); err != nil {
		return nil, err
	}

	return config, nil
}

func processObjects(c *Config) error {
	if c.cluster == nil {
		return errors.New("no Cluster found in manifest")
	}

	// Process datacenter
	c.datacenter = c.objects.getFromRef(c.cluster.APIVersion, c.cluster.Spec.DatacenterRef)
	switch c.cluster.Spec.DatacenterRef.Kind {
	case anywherev1.VSphereDatacenterKind:
		c.vsphereDatacenter = c.datacenter.(*anywherev1.VSphereDatacenterConfig)
	case anywherev1.DockerDatacenterKind:
		c.dockerDatacenter = c.datacenter.(*anywherev1.DockerDatacenterConfig)
	}

	// Process machine configs
	processMachineConfig(c, c.cluster.Spec.ControlPlaneConfiguration.MachineGroupRef)
	if c.cluster.Spec.ExternalEtcdConfiguration != nil {
		processMachineConfig(c, c.cluster.Spec.ExternalEtcdConfiguration.MachineGroupRef)
	}

	for _, w := range c.cluster.Spec.WorkerNodeGroupConfigurations {
		processMachineConfig(c, w.MachineGroupRef)
	}

	// Process IDP
	for _, idr := range c.cluster.Spec.IdentityProviderRefs {
		processIdentityProvider(c, idr)
	}

	// Process GitOps
	processGitOps(c)

	return nil
}

func processMachineConfig(c *Config, machineRef *anywherev1.Ref) {
	if machineRef == nil {
		return
	}

	m := c.objects.getFromRef(c.cluster.APIVersion, *machineRef)
	if m == nil {
		return
	}

	c.machineConfigs = append(c.machineConfigs, m)
	switch machineRef.Kind {
	case anywherev1.VSphereMachineConfigKind:
		c.vsphereMachineConfigs[m.GetName()] = m.(*anywherev1.VSphereMachineConfig)
	}
}

func processIdentityProvider(c *Config, idpRef anywherev1.Ref) {
	idp := c.objects.getFromRef(c.cluster.APIVersion, idpRef)
	if idp == nil {
		return
	}

	switch idpRef.Kind {
	case anywherev1.OIDCConfigKind:
		c.oidcConfigs[idp.GetName()] = idp.(*anywherev1.OIDCConfig)
	case anywherev1.AWSIamConfigKind:
		c.awsIamConfigs[idp.GetName()] = idp.(*anywherev1.AWSIamConfig)
	}
}

func processGitOps(c *Config) {
	if c.cluster.Spec.GitOpsRef == nil {
		return
	}

	gitOps := c.objects.getFromRef(c.cluster.APIVersion, *c.cluster.Spec.GitOpsRef)
	if gitOps == nil {
		return
	}

	switch c.cluster.Spec.GitOpsRef.Kind {
	case anywherev1.GitOpsConfigKind:
		c.gitOpsConfig = gitOps.(*anywherev1.GitOpsConfig)
	}
}

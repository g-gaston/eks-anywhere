package controllers

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/cluster-api/util/yaml"
)

func yamlToUnstructured(yamlObjects []byte) ([]unstructured.Unstructured, error) {
	// Using this capi util for now, not sure if we want to depend on it but it's well written
	return yaml.ToUnstructured(yamlObjects)
}

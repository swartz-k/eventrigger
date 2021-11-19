package k8s

import (
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/gengo/namer"
	"k8s.io/gengo/types"
)

func DecodeAndUnstructure(b []byte) (*unstructured.Unstructured, error) {
	var result map[string]interface{}
	if err := yaml.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: result}, nil
}

func GetGroupVersionResource(obj *unstructured.Unstructured) (gvr schema.GroupVersionResource) {
	gvk := obj.GroupVersionKind()
	resource := namer.NewAllLowercasePluralNamer(nil).Name(&types.Type{
		Name: types.Name{
			Name: gvk.Kind,
		},
	})

	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: resource,
	}
}

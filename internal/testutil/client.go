package testutil

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/client-go/applyconfigurations"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/structured-merge-diff/v6/typed"
)

func NewClientBuilder() *fake.ClientBuilder {
	return fake.NewClientBuilder().WithTypeConverters(
		applyconfigurations.NewTypeConverter(scheme.Scheme),
		unstructuredTypeConverter{TypeConverter: managedfields.NewDeducedTypeConverter()},
	)
}

type unstructuredTypeConverter struct {
	managedfields.TypeConverter
}

func (c unstructuredTypeConverter) ObjectToTyped(obj runtime.Object, opts ...typed.ValidationOptions) (*typed.TypedValue, error) {
	content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	converted := &unstructured.Unstructured{Object: content}
	converted.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	return c.TypeConverter.ObjectToTyped(converted, opts...)
}

/*
Copyright 2023 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package metric

import (
	"fmt"
	"sort"

	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-tools/pkg/crd"
	"sigs.k8s.io/controller-tools/pkg/genall"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/markers"

	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
)

type Generator struct{}

func (Generator) CheckFilter() loader.NodeFilter {
	// Re-use controller-tools filter to filter out unrelated nodes that aren't used
	// in CRD generation, like interfaces and struct fields without JSON tag.
	return crd.Generator{}.CheckFilter()
}

func (g Generator) Generate(ctx *genall.GenerationContext) error {
	// Create the parser which is specific to the metric generator.
	parser := newParser(
		&crd.Parser{
			Collector: ctx.Collector,
			Checker:   ctx.Checker,
		},
	)

	// Loop over all passed packages.
	for _, root := range ctx.Roots {
		// skip packages which don't import metav1 because they can't define a CRD without meta v1.
		metav1 := root.Imports()["k8s.io/apimachinery/pkg/apis/meta/v1"]
		if metav1 == nil {
			continue
		}

		// parse the given package to feed crd.FindKubeKinds to find CRD objects.
		parser.NeedPackage(root)
		kubeKinds := crd.FindKubeKinds(parser.Parser, metav1)
		if len(kubeKinds) == 0 {
			klog.Fatalf("no objects in the roots")
		}

		for _, gv := range kubeKinds {
			// Create customresourcestate.Resource for each CRD which contains all metric
			// definitions for the CRD.
			parser.NeedResourceFor(gv)
		}
	}

	// Build customresourcestate configuration file from generated data.
	metrics := customresourcestate.Metrics{
		Spec: customresourcestate.MetricsSpec{
			Resources: []customresourcestate.Resource{},
		},
	}

	// Sort the resources to get a deterministic output.

	for _, resource := range parser.CustomResourceStates {
		if len(resource.Metrics) > 0 {
			// sort the metrics
			sort.Slice(resource.Metrics, func(i, j int) bool {
				return resource.Metrics[i].Name < resource.Metrics[j].Name
			})

			metrics.Spec.Resources = append(metrics.Spec.Resources, resource)
		}
	}

	sort.Slice(metrics.Spec.Resources, func(i, j int) bool {
		if metrics.Spec.Resources[i].MetricNamePrefix == nil && metrics.Spec.Resources[j].MetricNamePrefix == nil {
			a := metrics.Spec.Resources[i].GroupVersionKind.Group + "/" + metrics.Spec.Resources[i].GroupVersionKind.Version + "/" + metrics.Spec.Resources[i].GroupVersionKind.Kind
			b := metrics.Spec.Resources[j].GroupVersionKind.Group + "/" + metrics.Spec.Resources[j].GroupVersionKind.Version + "/" + metrics.Spec.Resources[j].GroupVersionKind.Kind
			return a < b
		}

		// Either a or b will not be the empty string, so we can compare them.
		var a, b string
		if metrics.Spec.Resources[i].MetricNamePrefix == nil {
			a = *metrics.Spec.Resources[i].MetricNamePrefix
		}
		if metrics.Spec.Resources[j].MetricNamePrefix != nil {
			b = *metrics.Spec.Resources[j].MetricNamePrefix
		}
		return a < b
	})

	// Write the rendered yaml to the context which will result in stdout.
	filePath := "metrics.yaml"
	if err := ctx.WriteYAML(filePath, "", []interface{}{metrics}, genall.WithTransform(addCustomResourceStateKind)); err != nil {
		return fmt.Errorf("WriteYAML to %s: %w", filePath, err)
	}

	return nil
}

// addCustomResourceStateKind adds the correct kind because we don't have a correct
// kubernetes-style object as configuration definition.
func addCustomResourceStateKind(obj map[string]interface{}) error {
	obj["kind"] = "CustomResourceStateMetrics"
	return nil
}

func (g Generator) RegisterMarkers(into *markers.Registry) error {
	for _, m := range markerDefinitions {
		if err := m.Register(into); err != nil {
			return err
		}
	}

	return nil
}

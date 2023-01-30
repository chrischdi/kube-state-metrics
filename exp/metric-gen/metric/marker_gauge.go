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
	"sigs.k8s.io/controller-tools/pkg/markers"

	"k8s.io/klog/v2"
	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
)

const (
	// GaugeMarkerName is a marker for defining metric definitions.
	GaugeMarkerName = "Metrics:gauge"
)

func init() {
	markerDefinitions = append(
		markerDefinitions,
		must(markers.MakeDefinition(GaugeMarkerName, markers.DescribesField, GaugeMarker{})).
			help(GaugeMarker{}.help()),
		must(markers.MakeDefinition(GaugeMarkerName, markers.DescribesType, GaugeMarker{})).
			help(GaugeMarker{}.help()),
	)
}

type GaugeMarker struct {
	Name           string
	Help           string              `marker:"help,optional"`
	JSONPath       JSONPath            `marker:"JSONPath,optional"`
	LabelFromKey   string              `marker:"labelFromKey,optional"`
	LabelsFromPath map[string]JSONPath `marker:"labelsFromPath,optional"`
	NilIsZero      bool                `marker:"nilIsZero,optional"`
	ValueFrom      *JSONPath           `marker:"valueFrom,optional"`
}

func (GaugeMarker) help() *markers.DefinitionHelp {
	return &markers.DefinitionHelp{
		Category: "Metrics",
		DetailedHelp: markers.DetailedHelp{
			Summary: "Defines a Gauge metric and uses the implicit path to the field joined by the provided JSONPath as path for the metric configuration.",
			Details: "",
		},
		FieldHelp: map[string]markers.DetailedHelp{},
	}
}

func (g GaugeMarker) ToGenerator(basePath ...string) *customresourcestate.Generator {
	additionalPath, err := g.JSONPath.Parse()
	if err != nil {
		klog.Fatal(err)
	}
	var valueFrom []string
	if g.ValueFrom != nil {
		valueFrom, err = g.ValueFrom.Parse()
		if err != nil {
			klog.Fatal(err)
		}
	}

	labelsFromPath := map[string][]string{}
	for k, v := range g.LabelsFromPath {
		path := []string{}
		var err error
		if v != "." {
			path, err = v.Parse()
			if err != nil {
				klog.Fatal(err)
			}
		}
		labelsFromPath[k] = path
	}

	path := append(basePath, additionalPath...)

	return &customresourcestate.Generator{
		Name: g.Name,
		Help: g.Help,
		Each: customresourcestate.Metric{
			Type: customresourcestate.MetricTypeGauge,
			Gauge: &customresourcestate.MetricGauge{
				NilIsZero: g.NilIsZero,
				MetricMeta: customresourcestate.MetricMeta{
					Path:           path,
					LabelsFromPath: labelsFromPath,
				},
				LabelFromKey: g.LabelFromKey,
				ValueFrom:    valueFrom,
			},
		},
	}
}

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
	// InfoMarkerName is a marker for defining metric definitions.
	InfoMarkerName = "Metrics:info"
)

func init() {
	markerDefinitions = append(
		markerDefinitions,
		must(markers.MakeDefinition(InfoMarkerName, markers.DescribesField, InfoMarker{})).
			help(InfoMarker{}.help()),
		must(markers.MakeDefinition(InfoMarkerName, markers.DescribesType, InfoMarker{})).
			help(InfoMarker{}.help()),
	)
}

type InfoMarker struct {
	Name           string
	Help           string              `marker:"help,optional"`
	LabelsFromPath map[string]JSONPath `marker:"labelsFromPath,optional"`
	JSONPath       JSONPath            `marker:"JSONPath,optional"`
	LabelFromKey   string              `marker:"labelFromKey,optional"`
}

func (InfoMarker) help() *markers.DefinitionHelp {
	return &markers.DefinitionHelp{
		Category: "Metrics",
		DetailedHelp: markers.DetailedHelp{
			Summary: "Defines a Info metric and uses the implicit path to the field as path for the metric configuration.",
			Details: "",
		},
		FieldHelp: map[string]markers.DetailedHelp{},
	}
}

func (i InfoMarker) ToGenerator(basePath ...string) *customresourcestate.Generator {
	path := basePath
	if i.JSONPath != "" {
		valueFrom, err := i.JSONPath.Parse()
		if err != nil {
			klog.Fatal(err)
		}
		if len(valueFrom) > 0 {
			path = append(path, valueFrom...)
		}
	}

	labelsFromPath := map[string][]string{}
	for k, v := range i.LabelsFromPath {
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

	return &customresourcestate.Generator{
		Name: i.Name,
		Help: i.Help,
		Each: customresourcestate.Metric{
			Type: customresourcestate.MetricTypeInfo,
			Info: &customresourcestate.MetricInfo{
				MetricMeta: customresourcestate.MetricMeta{
					Path:           path,
					LabelsFromPath: labelsFromPath,
				},
				LabelFromKey: i.LabelFromKey,
			},
		},
	}
}

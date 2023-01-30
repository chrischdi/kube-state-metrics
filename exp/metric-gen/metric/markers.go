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

	"k8s.io/client-go/util/jsonpath"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-tools/pkg/markers"

	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
)

const (
	// NameMarkerName is a marker for defining metric definitions.
	NameMarkerName = "Metrics:namePrefix"
	// LabelFromPathMarkerName is a marker for defining labels for paths.
	LabelFromPathMarkerName = "Metrics:labelFromPath"
)

var (
	markerDefinitions = []*definitionWithHelp{
		must(markers.MakeDefinition(NameMarkerName, markers.DescribesType, NamePrefixMarker(""))).
			help(NamePrefixMarker("").Help()),
		must(markers.MakeDefinition(LabelFromPathMarkerName, markers.DescribesType, LabelFromPathMarker{})).
			help(LabelFromPathMarker{}.Help()),
		must(markers.MakeDefinition(LabelFromPathMarkerName, markers.DescribesField, LabelFromPathMarker{})).
			help(LabelFromPathMarker{}.Help()),
		// GroupName is a marker copied from controller-runtime to identify the API Group.
		// It needs to get added as marker so the parser will be able to read the API
		// which is Group set for a package.
		must(markers.MakeDefinition("groupName", markers.DescribesPackage, "")),
	}
)

// +controllertools:marker:generateHelp:category=CRD

// ResourceMarker is a marker that knows how to apply itself to a particular
// version in a CRD Spec.
type ResourceMarker interface {
	// ApplyToCRD applies this marker to the given CRD, in the given version
	// within that CRD.  It's called after everything else in the CRD is populated.
	ApplyToResource(resource *customresourcestate.Resource) error
}

// GeneratorMarker is a marker that knows how to create a generator from itself.
type GeneratorMarker interface {
	// ApplyToCRD applies this marker to the given CRD, in the given version
	// within that CRD.  It's called after everything else in the CRD is populated.
	ToGenerator(basePath ...string) *customresourcestate.Generator
}

type NamePrefixMarker string

func (NamePrefixMarker) Help() *markers.DefinitionHelp {
	return &markers.DefinitionHelp{
		Category: "Metrics",
		DetailedHelp: markers.DetailedHelp{
			Summary: "enables the creation of a customresourcestate Resource for the CRD and uses the given prefix for the metrics.",
			Details: "",
		},
		FieldHelp: map[string]markers.DetailedHelp{},
	}
}

func (n NamePrefixMarker) ApplyToResource(resource *customresourcestate.Resource) error {
	resource.MetricNamePrefix = pointer.String(string(n))
	return nil
}

type LabelFromPathMarker struct {
	// +Metrics:labelFromPath:name=<string>,JSONPath=<string> on API type struct
	Name     string
	JSONPath JSONPath `marker:"JSONPath"`
}

type JSONPath string

func (j JSONPath) Parse() ([]string, error) {
	ret := []string{}

	jp, err := jsonpath.Parse("foo", `{`+string(j)+`}`)
	if err != nil {
		return nil, fmt.Errorf("parse JSONPath: %w", err)
	}

	if len(jp.Root.Nodes) > 1 {
		return nil, fmt.Errorf("expected a single JSONPath, got %d", len(jp.Root.Nodes))
	}

	switch jp.Root.Nodes[0].Type() {
	case jsonpath.NodeList:
		list, ok := jp.Root.Nodes[0].(*jsonpath.ListNode)
		if !ok {
			return nil, fmt.Errorf("unable to typecast to jsonpath.ListNode")
		}
		for _, n := range list.Nodes {
			nf, ok := n.(*jsonpath.FieldNode)
			if !ok {
				return nil, fmt.Errorf("unable to typecast to jsonpath.NodeField")
			}
			ret = append(ret, nf.Value)
		}
	default:
		return nil, fmt.Errorf("unexcepted jsonpath node type: %q", jp.Root.Nodes[0].Type())
	}

	return ret, nil
}

func (LabelFromPathMarker) Help() *markers.DefinitionHelp {
	return &markers.DefinitionHelp{
		Category: "Metrics",
		DetailedHelp: markers.DetailedHelp{
			Summary: "adds an additional label to all metrics of this field or type with a value from the given JSONPath.",
			Details: "",
		},
		FieldHelp: map[string]markers.DetailedHelp{},
	}
}

func (n LabelFromPathMarker) ApplyToResource(resource *customresourcestate.Resource) error {
	if resource.LabelsFromPath == nil {
		resource.LabelsFromPath = map[string][]string{}
	}
	jsonPathElems, err := n.JSONPath.Parse()
	if err != nil {
		return err
	}

	if jsonPath, labelExists := resource.LabelsFromPath[n.Name]; labelExists {
		if len(jsonPathElems) != len(jsonPath) {
			return fmt.Errorf("duplicate definition for label %q", n.Name)
		}
		for i, v := range jsonPath {
			if v != jsonPathElems[i] {
				return fmt.Errorf("duplicate definition for label %q", n.Name)
			}
		}
	}

	resource.LabelsFromPath[n.Name] = jsonPathElems
	return nil
}

type definitionWithHelp struct {
	*markers.Definition
	Help *markers.DefinitionHelp
}

func must(def *markers.Definition, err error) *definitionWithHelp {
	return &definitionWithHelp{
		Definition: markers.Must(def, err),
	}
}

func (d *definitionWithHelp) help(help *markers.DefinitionHelp) *definitionWithHelp {
	d.Help = help
	return d
}

func (d *definitionWithHelp) Register(reg *markers.Registry) error {
	if err := reg.Register(d.Definition); err != nil {
		return err
	}
	if d.Help != nil {
		reg.AddHelp(d.Definition, d.Help)
	}
	return nil
}

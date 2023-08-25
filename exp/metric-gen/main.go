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
package main

import (
	"fmt"
	"os"

	"github.com/prometheus/common/version"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-tools/pkg/genall"
	"sigs.k8s.io/controller-tools/pkg/genall/help"
	prettyhelp "sigs.k8s.io/controller-tools/pkg/genall/help/pretty"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/markers"

	"k8s.io/klog/v2"

	"k8s.io/kube-state-metrics/v2/exp/metric-gen/generator"
)

const (
	generatorName = "metric"
)

var (
	// optionsRegistry contains all the marker definitions used to process command line options
	optionsRegistry = &markers.Registry{}
)

func main() {
	var whichMarkersFlag, versionFlag bool

	pflag.CommandLine.BoolVarP(&whichMarkersFlag, "which-markers", "w", false, "Print out all markers available with the requested generators.")
	pflag.CommandLine.BoolVarP(&versionFlag, "version", "v", false, "Print version information.")

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  metric-gen [flags] /path/to/package [/path/to/package]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		pflag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}

	pflag.Parse()

	if versionFlag {
		fmt.Printf("%s\n", version.Print("metric-gen"))
		klog.FlushAndExit(klog.ExitFlushTimeout, 0)
	}

	// Register the metric generator itself as marker so genall.FromOptions is able to initialize the runtime properly.
	// This also registers the markers inside the optionsRegistry so its available to print the marker docs.
	metricGenerator := generator.CustomResourceConfigGenerator{}
	defn := markers.Must(markers.MakeDefinition(generatorName, markers.DescribesPackage, metricGenerator))
	if err := optionsRegistry.Register(defn); err != nil {
		panic(err)
	}

	if whichMarkersFlag {
		printMarkerDocs()
		return
	}

	// Check if package paths got passed as input parameters.
	if len(os.Args[1:]) == 0 {
		fmt.Fprint(os.Stderr, "error: Please provide package paths as parameters\n\n")
		pflag.Usage()
		os.Exit(1)
	}

	// Load the passed packages as roots.
	roots, err := loader.LoadRoots(os.Args[1:]...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: loading packages %v\n", err)
		os.Exit(1)
	}

	// Set up the generator runtime using controller-tools and passing our optionsRegistry.
	rt, err := genall.FromOptions(optionsRegistry, []string{generatorName})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Setup the generation context with the loaded roots.
	rt.GenerationContext.Roots = roots
	// Setup the runtime to output to stdout.
	rt.OutputRules = genall.OutputRules{Default: genall.OutputToStdout}

	// Run the generator using the runtime.
	if hadErrs := rt.Run(); hadErrs {
		fmt.Fprint(os.Stderr, "generator did not run successfully\n")
		os.Exit(1)
	}
}

// printMarkerDocs prints out marker help for the given generators specified in
// the rawOptions
func printMarkerDocs() error {
	// just grab a registry so we don't lag while trying to load roots
	// (like we'd do if we just constructed the full runtime).
	reg, err := genall.RegistryFromOptions(optionsRegistry, []string{generatorName})
	if err != nil {
		return err
	}

	helpInfo := help.ByCategory(reg, help.SortByCategory)

	for _, cat := range helpInfo {
		if cat.Category == "" {
			continue
		}
		contents := prettyhelp.MarkersDetails(false, cat.Category, cat.Markers)
		if err := contents.WriteTo(os.Stderr); err != nil {
			return err
		}
	}
	return nil
}

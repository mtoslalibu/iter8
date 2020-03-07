package experiment

import (
	"strings"
)

const (
	// ExperimentLabel is used to mark the detected target with experiment name.namespace
	experimentLabel string = "iter8-experiment"
)

// returns the name and namespace of the experiment
func resolveExperiemntLabel(val string) (string, string) {
	out := strings.Split(val, ".")
	if len(out) != 2 {
		return "", ""
	}
	return out[0], out[1]
}

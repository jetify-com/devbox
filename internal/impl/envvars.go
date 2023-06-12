// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"go.jetpack.io/devbox/internal/sheller"
)

const devboxSetPrefix = "__DEVBOX_SET_"

func mapToPairs(m map[string]string) []string {
	pairs := []string{}
	for k, v := range m {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return pairs
}

func pairsToMap(pairs []string) map[string]string {
	vars := map[string]string{}
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			continue
		}
		vars[k] = v
	}
	return vars
}

// exportify takes an array of strings of the form VAR=VAL and returns a bash script
// that exports all the vars after properly escaping them.
func exportify(vars map[string]string) string {
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	strb := strings.Builder{}
	for _, k := range keys {
		strb.WriteString("export ")
		strb.WriteString(k)
		strb.WriteString(`=`)
		strb.WriteString(sheller.QuoteWrap(vars[k]))
		strb.WriteString(";\n")
	}
	return strings.TrimSpace(strb.String())
}

// addEnvIfNotPreviouslySetByDevbox adds the key-value pairs from new to existing,
// but only if the key was not previously set by devbox
// Caveat, this won't mark the values as set by devbox automatically. Instead,
// you need to call markEnvAsSetByDevbox when you are done setting variables.
// This is so you can add variables from multiple sources (e.g. plugin, devbox.json)
// that may build on each other (e.g. PATH=$PATH:...)
func addEnvIfNotPreviouslySetByDevbox(existing, new map[string]string) {
	for k, v := range new {
		if _, alreadySet := existing[devboxSetPrefix+k]; !alreadySet {
			existing[k] = v
		}
	}
}

func markEnvsAsSetByDevbox(envs ...map[string]string) {
	for _, env := range envs {
		for key := range env {
			env[devboxSetPrefix+key] = "1"
		}
	}
}

// IsEnvEnabled checks if the devbox environment is enabled. We use the ogPathKey
// as a proxy for this. This allows us to differentiate between global and
// individual project shells.
func (d *Devbox) IsEnvEnabled() bool {
	return os.Getenv(d.ogPathKey()) != ""
}

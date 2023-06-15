// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"fmt"
	"os"
	"sort"
	"strings"
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

// exportify takes a map of [string]string and returns a single string
// of the form export KEY="VAL"; and escapes all the vals from special characters.
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
		strb.WriteString(`="`)
		for _, r := range vars[k] {
			switch r {
			// Special characters inside double quotes:
			// https://pubs.opengroup.org/onlinepubs/009604499/utilities/xcu_chap02.html#tag_02_02_03
			case '$', '`', '"', '\\', '\n':
				strb.WriteRune('\\')
			}
			strb.WriteRune(r)
		}
		strb.WriteString("\";\n")
	}
	return strings.TrimSpace(strb.String())
}

// exportify takes a map of [string]string and returns an array of string
// of the form KEY="VAL" and escapes all the vals from special characters.
func keyEqualsValue(vars map[string]string) []string {
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	keyValues := make([]string, 0, len(vars))
	sort.Strings(keys)

	for _, k := range keys {
		if isApproved(k) {
			strb := strings.Builder{}
			strb.WriteString(k)
			strb.WriteString(`=`)
			for _, r := range vars[k] {
				switch r {
				// Special characters inside double quotes:
				// https://pubs.opengroup.org/onlinepubs/009604499/utilities/xcu_chap02.html#tag_02_02_03
				case '$', '`', '"', '\\', '\n':
					strb.WriteRune('\\')
				}
				strb.WriteRune(r)
			}
			keyValues = append(keyValues, strb.String())
		}
	}
	return keyValues
}

func isApproved(key string) bool {
	// list to keys
	// should find the corrupt key
	troublingEnvKeys := []string{
		"HOME",
		"NODE_CHANNEL_FD",
	}
	approved := true
	for _, ak := range troublingEnvKeys {
		// DEVBOX_OG_PATH_<hash> being set causes devbox global shellenv or overwrite
		// the PATH after vscode opens and resets it to global shellenv
		// This causes vscode terminal to not be able to find devbox packages
		// after reopen in devbox environment action is called
		if key == ak || strings.HasPrefix(key, "DEVBOX_OG_PATH") {
			approved = false
		}
	}
	return approved
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

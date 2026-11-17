// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"bytes"
	"fmt"
	"net/url"
	"path"
	"strings"
	"text/template"
)

// Vars are the variables available to templates in a package definition.
// See the "Template variables" section of the spec.
type Vars struct {
	Version  string
	System   string
	OS       string
	Arch     string
	URL      string
	Filename string

	InstallDir string
	CacheDir   string

	// Page is the 0-based page number when listing versions from a
	// paginated http_json source.
	Page int
}

var defaultOSMap = map[string]string{
	"darwin": "darwin",
	"linux":  "linux",
}

var defaultArchMap = map[string]string{
	"x86_64":  "amd64",
	"aarch64": "arm64",
}

// templateFuncs follow the sprig argument order (the string comes last) so
// that pipelines read naturally: {{ .Version | trimPrefix "v" }}. The
// text/template builtins (eq, index, urlquery, ...) are available too.
var templateFuncs = template.FuncMap{
	"trimPrefix": func(prefix, s string) string { return strings.TrimPrefix(s, prefix) },
	"trimSuffix": func(suffix, s string) string { return strings.TrimSuffix(s, suffix) },
	"replace":    func(old, replacement, s string) string { return strings.ReplaceAll(s, old, replacement) },
	"lower":      strings.ToLower,
	"upper":      strings.ToUpper,
	"splitList":  func(sep, s string) []string { return strings.Split(s, sep) },
}

// TemplateVars returns the template variables for installing version on
// system. URL, Filename and InstallDir are left empty; callers fill them in
// once they are known.
func (d *Definition) TemplateVars(system, version string) (Vars, error) {
	arch, osName, ok := strings.Cut(system, "-")
	if !ok {
		return Vars{}, fmt.Errorf("invalid system %q", system)
	}
	vars := Vars{
		Version: version,
		System:  system,
		OS:      lookupOr(d.OSMap, osName, defaultOSMap[osName], osName),
		Arch:    lookupOr(d.ArchMap, arch, defaultArchMap[arch], arch),
	}
	return vars, nil
}

func lookupOr(m map[string]string, key string, fallbacks ...string) string {
	if v, ok := m[key]; ok {
		return v
	}
	for _, f := range fallbacks {
		if f != "" {
			return f
		}
	}
	return key
}

// Render executes tmpl with vars. Unknown fields are an error.
func Render(tmpl string, vars Vars) (string, error) {
	if !strings.Contains(tmpl, "{{") {
		return tmpl, nil
	}
	parsed, err := template.New("pkgdef").
		Option("missingkey=error").
		Funcs(templateFuncs).
		Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("template %q: %w", tmpl, err)
	}
	var buf bytes.Buffer
	if err := parsed.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("template %q: %w", tmpl, err)
	}
	return buf.String(), nil
}

// withURL returns a copy of vars with URL and Filename set from rawURL.
func (v Vars) withURL(rawURL string) Vars {
	v.URL = rawURL
	v.Filename = filenameFromURL(rawURL)
	return v
}

func filenameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Path == "" {
		return path.Base(rawURL)
	}
	return path.Base(parsed.Path)
}

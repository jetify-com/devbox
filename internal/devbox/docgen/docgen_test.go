package docgen

import (
	"maps"
	"testing"
)

func TestEnvWithRelativePaths(t *testing.T) {
	projectDir := "/home/user/myproject"

	t.Run("replaces project dir with relative path", func(t *testing.T) {
		env := map[string]string{
			"PGDATA": projectDir + "/.devbox/virtenv/postgresql/data",
			"PGHOST": projectDir + "/.devbox/virtenv/postgresql",
			"PGPORT": "5432",
		}
		got := envWithRelativePaths(env, projectDir)
		want := map[string]string{
			"PGDATA": "./.devbox/virtenv/postgresql/data",
			"PGHOST": "./.devbox/virtenv/postgresql",
			"PGPORT": "5432",
		}
		if !maps.Equal(got, want) {
			t.Errorf("envWithRelativePaths() = %v, want %v", got, want)
		}
	})

	t.Run("does not mutate the input map", func(t *testing.T) {
		original := projectDir + "/.devbox/virtenv/postgresql"
		env := map[string]string{"PGHOST": original}
		envWithRelativePaths(env, projectDir)
		if env["PGHOST"] != original {
			t.Errorf("input map was mutated: PGHOST = %q, want %q", env["PGHOST"], original)
		}
	})

	t.Run("empty project dir returns env unchanged", func(t *testing.T) {
		env := map[string]string{"PGHOST": "/some/abs/path"}
		got := envWithRelativePaths(env, "")
		if !maps.Equal(got, env) {
			t.Errorf("envWithRelativePaths() = %v, want %v", got, env)
		}
	})
}

package templates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTemplatesExist(t *testing.T) {
	curDir := ""
	// Try to find examples dir. After 10 hops, we give up.
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(curDir + "examples"); err == nil {
			break
		}
		curDir += "../"
	}
	for _, path := range templates {
		_, err := os.Stat(filepath.Join(curDir, path))
		if os.IsNotExist(err) {
			t.Errorf("Directory for %s does not exist", path)
		}
	}
}

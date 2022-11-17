package pkgcfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWildcardMatch(t *testing.T) {
	result := wildcardMatch("pkg*.json", "pkg.json")
	assert.True(t, result)

	result = wildcardMatch("pkg*.json", "pkg1.json")
	assert.True(t, result)

	result = wildcardMatch("pkg*.json", "pkg12.json")
	assert.True(t, result)

	result = wildcardMatch("pkg*.json", "pkg.json")
	assert.True(t, result)

	result = wildcardMatch("pkg.json", "pkg.json")
	assert.True(t, result)

	result = wildcardMatch("pkg*.json", "pkg.json1")
	assert.False(t, result)

	result = wildcardMatch("pkg*.json", "1pkg.json1")
	assert.False(t, result)
}

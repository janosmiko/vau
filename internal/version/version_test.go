package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFull(t *testing.T) {
	result := Full()
	assert.Contains(t, result, "vau")
	assert.Contains(t, result, Version)
	assert.Contains(t, result, GitCommit)
	assert.Contains(t, result, BuildDate)
}

func TestShort(t *testing.T) {
	assert.Equal(t, Version, Short())
}

func TestFullWithCustomValues(t *testing.T) {
	oldVersion := Version
	oldCommit := GitCommit
	oldDate := BuildDate
	defer func() {
		Version = oldVersion
		GitCommit = oldCommit
		BuildDate = oldDate
	}()

	Version = "v1.2.3"
	GitCommit = "abc1234"
	BuildDate = "2024-01-01T00:00:00Z"

	result := Full()
	assert.Equal(t, "vau v1.2.3 (commit: abc1234, built: 2024-01-01T00:00:00Z)", result)
	assert.Equal(t, "v1.2.3", Short())
}

package ui

import (
	"testing"

	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
)

var testDockerFields = []model.DockerConfigField{
	{Registry: "ghcr.io", Name: "username", Value: "me"},
	{Registry: "ghcr.io", Name: "password", Value: "topsecret"},
	{Registry: "ghcr.io", Name: "auth", Value: "bWU6dG9wc2VjcmV0"},
}

func TestRenderDockerConfigOverlay_HidesSecretsByDefault(t *testing.T) {
	out := RenderDockerConfigOverlay("app/.dockerconfigjson", testDockerFields, 0, false, 120, 40)
	assert.Contains(t, out, "app/.dockerconfigjson")
	assert.Contains(t, out, "ghcr.io")
	assert.Contains(t, out, "me")
	assert.NotContains(t, out, "topsecret")
	assert.NotContains(t, out, "bWU6dG9wc2VjcmV0")
}

func TestRenderDockerConfigOverlay_StripsControlChars(t *testing.T) {
	fields := []model.DockerConfigField{{Registry: "evil\x1b[2Jreg", Name: "username", Value: "u"}}
	out := RenderDockerConfigOverlay("t\x1b]0;x\x07", fields, 1, false, 120, 40)
	assert.NotContains(t, out, "\x1b[2J")
	assert.NotContains(t, out, "\x1b]0;")
}

func TestRenderDockerConfigOverlay_Revealed(t *testing.T) {
	out := RenderDockerConfigOverlay("x", testDockerFields, 1, true, 120, 40)
	assert.Contains(t, out, "topsecret")
	assert.Contains(t, out, "bWU6dG9wc2VjcmV0")
}

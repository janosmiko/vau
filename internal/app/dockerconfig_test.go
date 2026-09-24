package app

import (
	"encoding/base64"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestParseDockerConfig(t *testing.T) {
	userPass := base64.StdEncoding.EncodeToString([]byte("bot:s3cr:et"))
	rawJSON := `{"auths":{"ghcr.io":{"username":"me","password":"pw","auth":"bWU6cHc="},` +
		`"docker.io":{"auth":"` + userPass + `"}}}`

	want := []model.DockerConfigField{
		{Registry: "docker.io", Name: "username", Value: "bot"},
		{Registry: "docker.io", Name: "password", Value: "s3cr:et"},
		{Registry: "docker.io", Name: "auth", Value: userPass},
		{Registry: "ghcr.io", Name: "username", Value: "me"},
		{Registry: "ghcr.io", Name: "password", Value: "pw"},
		{Registry: "ghcr.io", Name: "auth", Value: "bWU6cHc="},
	}

	tests := []struct {
		name string
		in   string
		want []model.DockerConfigField
		ok   bool
	}{
		{"raw json", rawJSON, want, true},
		{"base64 encoded json", base64.StdEncoding.EncodeToString([]byte(rawJSON)), want, true},
		{"surrounding whitespace", "\n " + rawJSON + " \n", want, true},
		{
			"auth without colon stays masked-only",
			`{"auths":{"r":{"auth":"dG9wc2VjcmV0"}}}`,
			[]model.DockerConfigField{{Registry: "r", Name: "auth", Value: "dG9wc2VjcmV0"}},
			true,
		},
		{"plain string", "hunter2", nil, false},
		{"json without auths", `{"foo":"bar"}`, nil, false},
		{"empty auths", `{"auths":{}}`, nil, false},
		{"registry without credentials", `{"auths":{"r":{}}}`, nil, false},
		{"empty", "", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseDockerConfig(tt.in)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func dockerTestModel() *Model {
	m := newTestModel()
	m.mode = model.ModeSecret
	m.secret = &model.Secret{
		Path: "app/pull",
		Keys: []string{".dockerconfigjson", "plain"},
		Data: map[string]string{
			".dockerconfigjson": `{"auths":{"ghcr.io":{"username":"me","password":"pw"}}}`,
			"plain":             "hunter2",
		},
	}
	return m
}

func TestSecretKey_D_OpensDockerConfig(t *testing.T) {
	m := dockerTestModel()

	m.handleSecretKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	assert.Equal(t, model.ModeDockerConfig, m.mode)
	assert.Len(t, m.dockerFields, 2)
	assert.Equal(t, 0, m.dockerCursor)
	assert.False(t, m.dockerRevealed)
	assert.Equal(t, "app/pull/.dockerconfigjson", m.dockerTitle)
}

func TestSecretKey_D_RejectsNonDockerValue(t *testing.T) {
	m := dockerTestModel()
	m.secretCursor = 1

	m.handleSecretKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	assert.Equal(t, model.ModeSecret, m.mode)
	assert.Equal(t, "Not a dockerconfigjson value", m.errMsg)
}

func TestDockerConfigKey_NavRevealCopyClose(t *testing.T) {
	m := dockerTestModel()
	m.handleSecretKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	m.handleDockerConfigKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	assert.Equal(t, 1, m.dockerCursor)
	m.handleDockerConfigKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	assert.Equal(t, 1, m.dockerCursor, "cursor stops at last field")
	m.handleDockerConfigKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	assert.Equal(t, 0, m.dockerCursor)

	m.handleDockerConfigKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	assert.True(t, m.dockerRevealed)

	_, cmd := m.handleDockerConfigKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	assert.NotNil(t, cmd, "y returns a clipboard command")

	m.handleDockerConfigKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, model.ModeSecret, m.mode)
	assert.Nil(t, m.dockerFields)
	assert.NotNil(t, m.secret, "secret popup stays open")
}

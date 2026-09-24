package app

import (
	"encoding/base64"
	"encoding/json"
	"maps"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
)

func (m *Model) handleDockerConfigKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "h":
		m.mode = model.ModeSecret
		m.dockerFields = nil
	case "j", "down", "ctrl+n":
		if m.dockerCursor < len(m.dockerFields)-1 {
			m.dockerCursor++
		}
	case "k", "up", "ctrl+p":
		if m.dockerCursor > 0 {
			m.dockerCursor--
		}
	case "v", "tab":
		m.dockerRevealed = !m.dockerRevealed
	case "y":
		if m.dockerCursor < len(m.dockerFields) {
			f := m.dockerFields[m.dockerCursor]
			return m, copyToSystemClipboard(f.Value, f.Registry+" "+f.Name)
		}
	}
	return m, nil
}

type dockerAuth struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	Email         string `json:"email"`
	Auth          string `json:"auth"`
	IdentityToken string `json:"identitytoken"`
}

// parseDockerConfig flattens a dockerconfigjson value (raw or base64-encoded
// JSON) into one field per registry credential, sorted by registry.
func parseDockerConfig(val string) ([]model.DockerConfigField, bool) {
	raw := []byte(strings.TrimSpace(val))
	if len(raw) > 0 && raw[0] != '{' {
		decoded, err := base64.StdEncoding.DecodeString(string(raw))
		if err != nil {
			return nil, false
		}
		raw = decoded
	}

	var cfg struct {
		Auths map[string]dockerAuth `json:"auths"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil || len(cfg.Auths) == 0 {
		return nil, false
	}

	var fields []model.DockerConfigField
	for _, registry := range slices.Sorted(maps.Keys(cfg.Auths)) {
		a := cfg.Auths[registry]
		// Kubernetes and docker login often store only "auth" (base64 of user:pass).
		if a.Username == "" && a.Password == "" && a.Auth != "" {
			if dec, err := base64.StdEncoding.DecodeString(a.Auth); err == nil {
				if user, pass, found := strings.Cut(string(dec), ":"); found {
					a.Username, a.Password = user, pass
				}
			}
		}
		for _, f := range [][2]string{
			{"username", a.Username},
			{"password", a.Password},
			{"email", a.Email},
			{"auth", a.Auth},
			{"identitytoken", a.IdentityToken},
		} {
			if f[1] != "" {
				fields = append(fields, model.DockerConfigField{Registry: registry, Name: f[0], Value: f[1]})
			}
		}
	}
	return fields, len(fields) > 0
}

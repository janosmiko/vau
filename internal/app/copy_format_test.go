package app

import (
	"testing"

	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestCopySecretAsYAML(t *testing.T) {
	m := &Model{
		secret: &model.Secret{
			Path: "secret/test",
			Keys: []string{"username", "password", "host"},
			Data: map[string]string{
				"username": "admin",
				"password": "s3cret!",
				"host":     "localhost",
			},
		},
	}

	// Nil secret should return nil cmd.
	m2 := &Model{secret: nil}
	cmd := m2.copySecretAsYAML()
	assert.Nil(t, cmd)

	// Valid secret should return non-nil cmd.
	cmd = m.copySecretAsYAML()
	assert.NotNil(t, cmd)
}

func TestCopySecretAsDotenv(t *testing.T) {
	m := &Model{
		secret: &model.Secret{
			Path: "secret/test",
			Keys: []string{"DB_HOST", "DB_PASS"},
			Data: map[string]string{
				"DB_HOST": "localhost",
				"DB_PASS": "p@ss word",
			},
		},
	}

	// Nil secret should return nil.
	m2 := &Model{secret: nil}
	cmd := m2.copySecretAsDotenv()
	assert.Nil(t, cmd)

	// Valid secret should return non-nil cmd.
	cmd = m.copySecretAsDotenv()
	assert.NotNil(t, cmd)
}

func TestCopySecretAsJSON(t *testing.T) {
	m := &Model{
		secret: &model.Secret{
			Path: "secret/test",
			Keys: []string{"key"},
			Data: map[string]string{"key": "value"},
		},
	}

	cmd := m.copySecretAsJSON()
	assert.NotNil(t, cmd)
}

func TestHandleCopyFormatJSON(t *testing.T) {
	s := &model.Secret{Path: "secret/test", Keys: []string{"k"}, Data: map[string]string{"k": "v"}}
	m := &Model{copySecret: s}
	_, cmd := m.handleCopyFormat("j")
	assert.NotNil(t, cmd)
	assert.Empty(t, m.status)
	assert.Nil(t, m.copySecret)
}

func TestHandleCopyFormatYAML(t *testing.T) {
	s := &model.Secret{Path: "secret/test", Keys: []string{"k"}, Data: map[string]string{"k": "v"}}
	m := &Model{copySecret: s}
	_, cmd := m.handleCopyFormat("y")
	assert.NotNil(t, cmd)
	assert.Empty(t, m.status)
	assert.Nil(t, m.copySecret)
}

func TestHandleCopyFormatDotenv(t *testing.T) {
	s := &model.Secret{Path: "secret/test", Keys: []string{"k"}, Data: map[string]string{"k": "v"}}
	m := &Model{copySecret: s}
	_, cmd := m.handleCopyFormat("d")
	assert.NotNil(t, cmd)
	assert.Empty(t, m.status)
	assert.Nil(t, m.copySecret)
}

func TestHandleCopyFormatEscape(t *testing.T) {
	s := &model.Secret{Path: "secret/test", Keys: []string{"k"}, Data: map[string]string{"k": "v"}}
	m := &Model{copySecret: s, status: "Copy as: ..."}
	_, cmd := m.handleCopyFormat("esc")
	assert.Nil(t, cmd)
	assert.Empty(t, m.status)
	assert.Nil(t, m.copySecret)
}

func TestHandleCopyFormatInvalidKey(t *testing.T) {
	s := &model.Secret{Path: "secret/test", Keys: []string{"k"}, Data: map[string]string{"k": "v"}}
	m := &Model{copySecret: s, status: "Copy as: ..."}
	_, cmd := m.handleCopyFormat("x")
	assert.Nil(t, cmd)
	assert.Empty(t, m.status)
	assert.Nil(t, m.copySecret)
}

func TestHandleCopyFormatNilSecret(t *testing.T) {
	m := &Model{copySecret: nil}
	_, cmd := m.handleCopyFormat("j")
	assert.Nil(t, cmd)
}

func TestHandleCopyFormatFromExplorer(t *testing.T) {
	s := &model.Secret{
		Path: "secret/test",
		Keys: []string{"DB_HOST", "DB_PASS"},
		Data: map[string]string{"DB_HOST": "localhost", "DB_PASS": "s3cret"},
	}
	m := &Model{copySecret: s}

	// JSON format
	_, cmd := m.handleCopyFormat("j")
	assert.NotNil(t, cmd)

	// YAML format
	m.copySecret = s
	_, cmd = m.handleCopyFormat("y")
	assert.NotNil(t, cmd)

	// Dotenv format
	m.copySecret = s
	_, cmd = m.handleCopyFormat("d")
	assert.NotNil(t, cmd)
}

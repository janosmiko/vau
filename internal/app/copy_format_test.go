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
	m := &Model{
		secret: &model.Secret{
			Path: "secret/test",
			Keys: []string{"k"},
			Data: map[string]string{"k": "v"},
		},
	}
	_, cmd := m.handleCopyFormat("j")
	assert.NotNil(t, cmd)
	assert.Empty(t, m.status)
}

func TestHandleCopyFormatYAML(t *testing.T) {
	m := &Model{
		secret: &model.Secret{
			Path: "secret/test",
			Keys: []string{"k"},
			Data: map[string]string{"k": "v"},
		},
	}
	_, cmd := m.handleCopyFormat("y")
	assert.NotNil(t, cmd)
	assert.Empty(t, m.status)
}

func TestHandleCopyFormatDotenv(t *testing.T) {
	m := &Model{
		secret: &model.Secret{
			Path: "secret/test",
			Keys: []string{"k"},
			Data: map[string]string{"k": "v"},
		},
	}
	_, cmd := m.handleCopyFormat("d")
	assert.NotNil(t, cmd)
	assert.Empty(t, m.status)
}

func TestHandleCopyFormatEscape(t *testing.T) {
	m := &Model{
		secret: &model.Secret{
			Path: "secret/test",
			Keys: []string{"k"},
			Data: map[string]string{"k": "v"},
		},
		status: "Copy as: ...",
	}
	_, cmd := m.handleCopyFormat("esc")
	assert.Nil(t, cmd)
	assert.Empty(t, m.status)
}

func TestHandleCopyFormatInvalidKey(t *testing.T) {
	m := &Model{
		secret: &model.Secret{
			Path: "secret/test",
			Keys: []string{"k"},
			Data: map[string]string{"k": "v"},
		},
		status: "Copy as: ...",
	}
	_, cmd := m.handleCopyFormat("x")
	assert.Nil(t, cmd)
	assert.Empty(t, m.status)
}

func TestHandleCopyFormatNilSecret(t *testing.T) {
	m := &Model{secret: nil}
	_, cmd := m.handleCopyFormat("j")
	assert.Nil(t, cmd)
}

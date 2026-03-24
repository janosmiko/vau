package app

import (
	"testing"

	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestFormatSecretAsYAML(t *testing.T) {
	tests := []struct {
		name     string
		secret   *model.Secret
		expected string
	}{
		{
			name: "simple key-values",
			secret: &model.Secret{
				Keys: []string{"host", "port"},
				Data: map[string]string{"host": "localhost", "port": "5432"},
			},
			expected: "host: localhost\nport: 5432\n",
		},
		{
			name: "value with special chars gets quoted",
			secret: &model.Secret{
				Keys: []string{"password"},
				Data: map[string]string{"password": "p@ss:word"},
			},
			expected: "password: \"p@ss:word\"\n",
		},
		{
			name: "empty value gets quoted",
			secret: &model.Secret{
				Keys: []string{"empty"},
				Data: map[string]string{"empty": ""},
			},
			expected: "empty: \"\"\n",
		},
		{
			name: "boolean-like values get quoted",
			secret: &model.Secret{
				Keys: []string{"flag1", "flag2", "nil_val"},
				Data: map[string]string{"flag1": "true", "flag2": "false", "nil_val": "null"},
			},
			expected: "flag1: \"true\"\nflag2: \"false\"\nnil_val: \"null\"\n",
		},
		{
			name: "value with newline gets quoted",
			secret: &model.Secret{
				Keys: []string{"multi"},
				Data: map[string]string{"multi": "line1\nline2"},
			},
			expected: "multi: \"line1\\nline2\"\n",
		},
		{
			name: "value with hash gets quoted",
			secret: &model.Secret{
				Keys: []string{"comment"},
				Data: map[string]string{"comment": "value # not a comment"},
			},
			expected: "comment: \"value # not a comment\"\n",
		},
		{
			name: "no keys",
			secret: &model.Secret{
				Keys: []string{},
				Data: map[string]string{},
			},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatSecretAsYAML(tc.secret)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestFormatSecretAsDotenv(t *testing.T) {
	tests := []struct {
		name     string
		secret   *model.Secret
		expected string
	}{
		{
			name: "simple values unquoted",
			secret: &model.Secret{
				Keys: []string{"HOST", "PORT"},
				Data: map[string]string{"HOST": "localhost", "PORT": "5432"},
			},
			expected: "HOST=localhost\nPORT=5432\n",
		},
		{
			name: "value with spaces gets quoted",
			secret: &model.Secret{
				Keys: []string{"MSG"},
				Data: map[string]string{"MSG": "hello world"},
			},
			expected: "MSG=\"hello world\"\n",
		},
		{
			name: "empty value gets quoted",
			secret: &model.Secret{
				Keys: []string{"EMPTY"},
				Data: map[string]string{"EMPTY": ""},
			},
			expected: "EMPTY=\"\"\n",
		},
		{
			name: "value with double quotes gets escaped",
			secret: &model.Secret{
				Keys: []string{"QUOTED"},
				Data: map[string]string{"QUOTED": "say \"hello\""},
			},
			expected: "QUOTED=\"say \\\"hello\\\"\"\n",
		},
		{
			name: "value with newline gets escaped",
			secret: &model.Secret{
				Keys: []string{"MULTI"},
				Data: map[string]string{"MULTI": "line1\nline2"},
			},
			expected: "MULTI=\"line1\\nline2\"\n",
		},
		{
			name: "value with backslash gets escaped",
			secret: &model.Secret{
				Keys: []string{"PATH"},
				Data: map[string]string{"PATH": "C:\\Users"},
			},
			expected: "PATH=\"C:\\\\Users\"\n",
		},
		{
			name: "value with dollar sign gets quoted",
			secret: &model.Secret{
				Keys: []string{"PRICE"},
				Data: map[string]string{"PRICE": "$100"},
			},
			expected: "PRICE=\"$100\"\n",
		},
		{
			name: "value with hash gets quoted",
			secret: &model.Secret{
				Keys: []string{"COMMENT"},
				Data: map[string]string{"COMMENT": "value#tag"},
			},
			expected: "COMMENT=\"value#tag\"\n",
		},
		{
			name: "no keys",
			secret: &model.Secret{
				Keys: []string{},
				Data: map[string]string{},
			},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatSecretAsDotenv(tc.secret)
			assert.Equal(t, tc.expected, got)
		})
	}
}

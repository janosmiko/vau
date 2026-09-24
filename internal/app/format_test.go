package app

import (
	"testing"

	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Decodes into map[string]any, not map[string]string, so a value resolved
// as bool/int/float/null shows up as a type mismatch instead of comparing
// two equal-looking strings.
func TestFormatSecretAsYAML_RoundTrips(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"plain string", "host", "localhost"},
		{"special chars", "password", "p@ss:word"},
		{"empty value", "empty", ""},
		{"boolean-like true", "flag1", "true"},
		{"boolean-like false", "flag2", "false"},
		{"null-like", "nil_val", "null"},
		{"tilde null-like", "nil_val2", "~"},
		{"yes/no-like", "confirm", "yes"},
		{"octal-looking", "code", "0123"},
		{"scientific-notation-looking", "n", "1e3"},
		{"title-case bool-like", "flag3", "True"},
		{"leading space", "pass", " pass"},
		{"trailing space", "pass2", "pass "},
		{"leading dash", "item", "-value"},
		{"newline", "multi", "line1\nline2"},
		{"hash", "comment", "value # not a comment"},
		{"key with colon-space", "a: b", "v"},
		{"key with hash", "a#b", "v"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			secret := &model.Secret{
				Keys: []string{tc.key},
				Data: map[string]string{tc.key: tc.value},
			}
			got := formatSecretAsYAML(secret)

			var decoded map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(got), &decoded))
			require.Contains(t, decoded, tc.key)
			assert.Equal(t, tc.value, decoded[tc.key])
		})
	}
}

// Ensures the mapping is built in secret.Keys order, not Go's randomized
// map iteration order.
func TestFormatSecretAsYAML_PreservesKeyOrder(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"z", "a", "m"},
		Data: map[string]string{"z": "1", "a": "2", "m": "3"},
	}
	got := formatSecretAsYAML(secret)

	var doc yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(got), &doc))
	require.Len(t, doc.Content, 1)
	mapping := doc.Content[0]
	require.Equal(t, yaml.MappingNode, mapping.Kind)

	var gotKeys []string
	for i := 0; i < len(mapping.Content); i += 2 {
		gotKeys = append(gotKeys, mapping.Content[i].Value)
	}
	assert.Equal(t, secret.Keys, gotKeys)
}

func TestFormatSecretAsYAML_NoKeys(t *testing.T) {
	secret := &model.Secret{Keys: []string{}, Data: map[string]string{}}
	got := formatSecretAsYAML(secret)

	var decoded map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(got), &decoded))
	assert.Empty(t, decoded)
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

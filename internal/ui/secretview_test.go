package ui

import (
	"testing"

	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// tryBase64Decode
// ---------------------------------------------------------------------------

func TestTryBase64Decode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // expected substring in decoded output
	}{
		{
			name:     "valid standard base64",
			input:    "aGVsbG8=",
			contains: "hello",
		},
		{
			name:     "valid base64 with padding",
			input:    "d29ybGQ=",
			contains: "world",
		},
		{
			name:     "empty string returns empty",
			input:    "",
			contains: "",
		},
		{
			name:     "plain text that is not base64 returns original",
			input:    "not!valid!base64$$",
			contains: "not!valid!base64$$",
		},
		{
			name:     "valid base64 with newline in decoded content",
			input:    "bGluZTEKbGluZTI=", // "line1\nline2"
			contains: "line1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tryBase64Decode(tc.input)
			assert.Contains(t, got, tc.contains)
		})
	}
}

func TestTryBase64DecodeNewlineReplacement(t *testing.T) {
	// "line1\nline2" encoded
	got := tryBase64Decode("bGluZTEKbGluZTI=")
	// Newlines should be replaced with the visual marker
	assert.Contains(t, got, string(rune(8629))) // Unicode return symbol
	assert.NotContains(t, got, "\n")
}

func TestTryBase64DecodeURLEncoding(t *testing.T) {
	// URL-safe base64 uses - and _ instead of + and /
	// "hello>world?" in URL-safe base64
	got := tryBase64Decode("aGVsbG8-d29ybGQ_")
	// Should attempt URL decoding -- even if this particular string
	// fails URL decoding, we verify it does not panic.
	assert.NotEmpty(t, got)
}

// ---------------------------------------------------------------------------
// sanitizeForDisplay
// ---------------------------------------------------------------------------

func TestSanitizeForDisplay(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "normal ASCII string unchanged",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "newline replaced with return symbol",
			input: "line1\nline2",
			want:  "line1\u21B5line2",
		},
		{
			name:  "carriage return stripped",
			input: "line1\rline2",
			want:  "line1line2",
		},
		{
			name:  "CRLF becomes return symbol only",
			input: "a\r\nb",
			want:  "a\u21B5b",
		},
		{
			name:  "tab replaced with two spaces",
			input: "col1\tcol2",
			want:  "col1  col2",
		},
		{
			name:  "null byte replaced with dot",
			input: "before\x00after",
			want:  "before\u00B7after",
		},
		{
			name:  "DEL (127) replaced with dot",
			input: "a\x7fb",
			want:  "a\u00B7b",
		},
		{
			name:  "bell char (control) replaced with dot",
			input: "a\x07b",
			want:  "a\u00B7b",
		},
		{
			name:  "mixed control characters",
			input: "a\x01\n\t\x7fb",
			want:  "a\u00B7\u21B5  \u00B7b",
		},
		{
			name:  "unicode preserved",
			input: "cafe\u0301",
			want:  "cafe\u0301",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeForDisplay(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// truncate
// ---------------------------------------------------------------------------

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{
			name:   "string shorter than max",
			s:      "hi",
			maxLen: 10,
			want:   "hi",
		},
		{
			name:   "string equal to max",
			s:      "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "string longer than max with room for ellipsis",
			s:      "hello world",
			maxLen: 8,
			want:   "hello...",
		},
		{
			name:   "string longer than max but max <= 3 no ellipsis",
			s:      "hello",
			maxLen: 3,
			want:   "hel",
		},
		{
			name:   "string longer than max with max = 4",
			s:      "hello world",
			maxLen: 4,
			want:   "h...",
		},
		{
			name:   "empty string",
			s:      "",
			maxLen: 5,
			want:   "",
		},
		{
			name:   "max is zero",
			s:      "hello",
			maxLen: 0,
			want:   "",
		},
		{
			name:   "max is negative",
			s:      "hello",
			maxLen: -1,
			want:   "",
		},
		{
			name:   "max is 1",
			s:      "hello",
			maxLen: 1,
			want:   "h",
		},
		{
			name:   "max is 2",
			s:      "hello",
			maxLen: 2,
			want:   "he",
		},
		{
			name:   "exact boundary at 3",
			s:      "abc",
			maxLen: 3,
			want:   "abc",
		},
		{
			name:   "one over boundary at 3",
			s:      "abcd",
			maxLen: 3,
			want:   "abc",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := truncate(tc.s, tc.maxLen)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// getValueDisplay
// ---------------------------------------------------------------------------

func TestGetValueDisplay(t *testing.T) {
	tests := []struct {
		name     string
		val      string
		revealed bool
		maxW     int
		want     string
	}{
		{
			name:     "revealed returns actual value",
			val:      "my-secret-password",
			revealed: true,
			maxW:     50,
			want:     "my-secret-password",
		},
		{
			name:     "hidden returns masked",
			val:      "my-secret-password",
			revealed: false,
			maxW:     50,
			want:     "********",
		},
		{
			name:     "revealed with truncation",
			val:      "a-very-long-secret-value-that-exceeds-limit",
			revealed: true,
			maxW:     10,
			want:     "a-very-...",
		},
		{
			name:     "hidden ignores max width",
			val:      "short",
			revealed: false,
			maxW:     3,
			want:     "********",
		},
		{
			name:     "revealed empty value",
			val:      "",
			revealed: true,
			maxW:     50,
			want:     "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getValueDisplay(tc.val, tc.revealed, tc.maxW)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// getValueDisplayWithBase64
// ---------------------------------------------------------------------------

func TestGetValueDisplayWithBase64(t *testing.T) {
	tests := []struct {
		name     string
		val      string
		revealed bool
		isBase64 bool
		maxW     int
		contains string
	}{
		{
			name:     "base64 mode decodes and shows prefix",
			val:      "aGVsbG8=",
			revealed: false,
			isBase64: true,
			maxW:     50,
			contains: "[b64] hello",
		},
		{
			name:     "base64 mode off and revealed shows value",
			val:      "aGVsbG8=",
			revealed: true,
			isBase64: false,
			maxW:     50,
			contains: "aGVsbG8=",
		},
		{
			name:     "base64 mode off and hidden shows mask",
			val:      "aGVsbG8=",
			revealed: false,
			isBase64: false,
			maxW:     50,
			contains: "********",
		},
		{
			name:     "base64 mode on overrides revealed=false",
			val:      "d29ybGQ=",
			revealed: false,
			isBase64: true,
			maxW:     50,
			contains: "[b64] world",
		},
		{
			name:     "base64 mode with truncation",
			val:      "aGVsbG8gd29ybGQgZm9vIGJhciBiYXo=", // "hello world foo bar baz"
			revealed: false,
			isBase64: true,
			maxW:     15,
			contains: "[b64]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getValueDisplayWithBase64(tc.val, tc.revealed, tc.isBase64, tc.maxW)
			assert.Contains(t, got, tc.contains)
		})
	}
}

// ---------------------------------------------------------------------------
// RenderSecretOverlay
// ---------------------------------------------------------------------------

func TestRenderSecretOverlayNilSecret(t *testing.T) {
	result := RenderSecretOverlay(
		nil, 0, nil, false, false,
		"", "", -1, nil, 100, 40,
	)
	assert.Contains(t, result, "No secret loaded")
}

func TestRenderSecretOverlaySimpleSecret(t *testing.T) {
	secret := &model.Secret{
		Path: "secret/data/myapp",
		Keys: []string{"username", "password"},
		Data: map[string]string{
			"username": "admin",
			"password": "s3cret",
		},
	}

	result := RenderSecretOverlay(
		secret, 0,
		map[string]bool{"username": true},
		false, false,
		"", "", -1,
		nil,
		100, 40,
	)

	// Should contain the path as title
	assert.Contains(t, result, "secret/data/myapp")
	// Should contain key names
	assert.Contains(t, result, "username")
	assert.Contains(t, result, "password")
	// Revealed key should show value
	assert.Contains(t, result, "admin")
	// Non-revealed key should show mask
	assert.Contains(t, result, "********")
}

func TestRenderSecretOverlayJSONView(t *testing.T) {
	secret := &model.Secret{
		Path: "secret/json/test",
		Keys: []string{"key1"},
		Data: map[string]string{
			"key1": "value1",
		},
	}

	result := RenderSecretOverlay(
		secret, 0, nil, false, true,
		"", "", -1, nil,
		100, 40,
	)

	assert.Contains(t, result, "secret/json/test")
	// JSON view should contain braces
	assert.Contains(t, result, "{")
	assert.Contains(t, result, "}")
}

func TestRenderSecretOverlaySmallScreen(t *testing.T) {
	secret := &model.Secret{
		Path: "secret/small",
		Keys: []string{"k"},
		Data: map[string]string{"k": "v"},
	}

	// Verify it does not panic with a very small screen
	result := RenderSecretOverlay(
		secret, 0, nil, false, false,
		"", "", -1, nil,
		20, 5,
	)
	assert.NotEmpty(t, result)
}

func TestRenderSecretOverlayEmptyKeys(t *testing.T) {
	secret := &model.Secret{
		Path: "secret/empty",
		Keys: []string{},
		Data: map[string]string{},
	}

	result := RenderSecretOverlay(
		secret, 0, nil, false, false,
		"", "", -1, nil,
		100, 40,
	)

	assert.Contains(t, result, "secret/empty")
	assert.Contains(t, result, "empty")
}

func TestRenderSecretOverlayBase64Keys(t *testing.T) {
	secret := &model.Secret{
		Path: "secret/b64",
		Keys: []string{"encoded"},
		Data: map[string]string{
			"encoded": "aGVsbG8=",
		},
	}

	result := RenderSecretOverlay(
		secret, 0,
		nil, false, false,
		"", "", -1,
		map[string]bool{"encoded": true},
		100, 40,
	)

	assert.Contains(t, result, "b64")
}

// ---------------------------------------------------------------------------
// renderSecretTable
// ---------------------------------------------------------------------------

func TestRenderSecretTableSimple(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"host", "port", "db"},
		Data: map[string]string{
			"host": "localhost",
			"port": "5432",
			"db":   "mydb",
		},
	}

	result := renderSecretTable(
		secret, 0,
		map[string]bool{"host": true, "port": true, "db": true},
		nil,
		"", "", -1,
		80, 20,
	)

	assert.Contains(t, result, "Key")
	assert.Contains(t, result, "Value")
	assert.Contains(t, result, "host")
	assert.Contains(t, result, "port")
	assert.Contains(t, result, "db")
	// Revealed values
	assert.Contains(t, result, "localhost")
	assert.Contains(t, result, "5432")
	assert.Contains(t, result, "mydb")
}

func TestRenderSecretTableHiddenValues(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"apikey"},
		Data: map[string]string{
			"apikey": "super-secret-key",
		},
	}

	result := renderSecretTable(
		secret, 0,
		nil, nil,
		"", "", -1,
		80, 20,
	)

	assert.Contains(t, result, "apikey")
	assert.Contains(t, result, "********")
	assert.NotContains(t, result, "super-secret-key")
}

func TestRenderSecretTableSelectedRow(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"alpha", "beta"},
		Data: map[string]string{
			"alpha": "aaa",
			"beta":  "bbb",
		},
	}

	result := renderSecretTable(
		secret, 1,
		map[string]bool{"alpha": true, "beta": true},
		nil,
		"", "", -1,
		80, 20,
	)

	// The selected row indicator should be present (▸ U+25B8)
	assert.Contains(t, result, "▸")
	assert.Contains(t, result, "beta")
}

func TestRenderSecretTableEmptySecret(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{},
		Data: map[string]string{},
	}

	result := renderSecretTable(
		secret, 0, nil, nil,
		"", "", -1, 80, 20,
	)

	assert.Contains(t, result, "empty")
}

func TestRenderSecretTableBase64Display(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"cert"},
		Data: map[string]string{
			"cert": "aGVsbG8=",
		},
	}

	result := renderSecretTable(
		secret, -1, // no selection matches
		nil,
		map[string]bool{"cert": true},
		"", "", -1,
		80, 20,
	)

	assert.Contains(t, result, "[b64]")
	assert.Contains(t, result, "hello")
}

func TestRenderSecretTableEditingKeyColumn(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"mykey"},
		Data: map[string]string{
			"mykey": "myval",
		},
	}

	result := renderSecretTable(
		secret, 0,
		map[string]bool{"mykey": true},
		nil,
		"mykey", "newkey", 0, // editing key column
		80, 20,
	)

	// Should show the editing view text
	assert.Contains(t, result, "newkey")
}

func TestRenderSecretTableEditingValueColumn(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"mykey"},
		Data: map[string]string{
			"mykey": "myval",
		},
	}

	result := renderSecretTable(
		secret, 0,
		nil, nil,
		"mykey", "edited-val", 1, // editing value column
		80, 20,
	)

	assert.Contains(t, result, "edited-val")
}

func TestRenderSecretTableScrolling(t *testing.T) {
	// Create a secret with many keys
	keys := make([]string, 50)
	data := make(map[string]string)
	for i := range 50 {
		k := "key" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		keys[i] = k
		data[k] = "val"
	}
	secret := &model.Secret{Keys: keys, Data: data}

	// Select an item near the bottom with a small height
	result := renderSecretTable(
		secret, 45,
		nil, nil,
		"", "", -1,
		80, 10,
	)

	// Should still render without panic
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Key")
	assert.Contains(t, result, "Value")
}

// ---------------------------------------------------------------------------
// renderSecretPopupJSON
// ---------------------------------------------------------------------------

func TestRenderSecretPopupJSONSimple(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"name", "version"},
		Data: map[string]string{
			"name":    "myapp",
			"version": "1.0",
		},
	}

	result := renderSecretPopupJSON(secret, 20)

	assert.Contains(t, result, "{")
	assert.Contains(t, result, "}")
	assert.Contains(t, result, "name")
	assert.Contains(t, result, "myapp")
	assert.Contains(t, result, "version")
	assert.Contains(t, result, "1.0")
}

func TestRenderSecretPopupJSONCommas(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"a", "b", "c"},
		Data: map[string]string{
			"a": "1",
			"b": "2",
			"c": "3",
		},
	}

	result := renderSecretPopupJSON(secret, 20)

	// Last key should not have a trailing comma; others should
	assert.Contains(t, result, ",")
}

func TestRenderSecretPopupJSONSingleKey(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"only"},
		Data: map[string]string{
			"only": "one",
		},
	}

	result := renderSecretPopupJSON(secret, 20)

	assert.Contains(t, result, "{")
	assert.Contains(t, result, "}")
	assert.Contains(t, result, "only")
	assert.Contains(t, result, "one")
}

func TestRenderSecretPopupJSONHeightLimit(t *testing.T) {
	keys := make([]string, 20)
	data := make(map[string]string)
	for i := range 20 {
		k := string(rune('a' + i))
		keys[i] = k
		data[k] = "val"
	}
	secret := &model.Secret{Keys: keys, Data: data}

	// Height of 5 means limit = 3 entries (height - 2 for braces)
	result := renderSecretPopupJSON(secret, 5)

	assert.Contains(t, result, "{")
	assert.Contains(t, result, "}")
	// Should not contain all 20 keys
	assert.NotContains(t, result, "t") // 20th key would be 't'
}

// ---------------------------------------------------------------------------
// RenderVersionHistoryOverlay
// ---------------------------------------------------------------------------

func TestRenderVersionHistoryOverlayWithVersions(t *testing.T) {
	versions := []model.SecretVersion{
		{
			Version:     "3",
			CreatedTime: "2024-01-15T10:30:00Z",
		},
		{
			Version:      "2",
			CreatedTime:  "2024-01-14T09:00:00Z",
			DeletionTime: "2024-01-15T00:00:00Z",
		},
		{
			Version:     "1",
			CreatedTime: "2024-01-13T08:00:00Z",
			Destroyed:   true,
		},
	}

	result := RenderVersionHistoryOverlay(versions, 0, "secret/data/myapp", 100, 40)

	assert.Contains(t, result, "Version History")
	assert.Contains(t, result, "secret/data/myapp")
	assert.Contains(t, result, "Version")
	assert.Contains(t, result, "Created")
	assert.Contains(t, result, "Status")
	// Version numbers
	assert.Contains(t, result, "v3")
	assert.Contains(t, result, "v2")
	assert.Contains(t, result, "v1")
	// Statuses
	assert.Contains(t, result, "current")
	assert.Contains(t, result, "deleted")
	assert.Contains(t, result, "destroyed")
}

func TestRenderVersionHistoryOverlayEmpty(t *testing.T) {
	result := RenderVersionHistoryOverlay(nil, 0, "secret/empty", 100, 40)

	assert.Contains(t, result, "Version History")
	assert.Contains(t, result, "secret/empty")
	assert.Contains(t, result, "no versions found")
}

func TestRenderVersionHistoryOverlaySelectedRow(t *testing.T) {
	versions := []model.SecretVersion{
		{Version: "2", CreatedTime: "2024-01-15T10:30:00Z"},
		{Version: "1", CreatedTime: "2024-01-14T09:00:00Z"},
	}

	result := RenderVersionHistoryOverlay(versions, 1, "secret/test", 100, 40)

	// The selected row indicator should be present (▸ U+25B8)
	assert.Contains(t, result, "▸")
}

func TestRenderVersionHistoryOverlaySmallScreen(t *testing.T) {
	versions := []model.SecretVersion{
		{Version: "1", CreatedTime: "2024-01-01T00:00:00Z"},
	}

	// Verify it does not panic with minimum dimensions
	result := RenderVersionHistoryOverlay(versions, 0, "p", 20, 5)
	assert.NotEmpty(t, result)
}

func TestRenderVersionHistoryOverlayLongCreatedTime(t *testing.T) {
	versions := []model.SecretVersion{
		{
			Version:     "1",
			CreatedTime: "2024-01-15T10:30:00.123456789Z",
		},
	}

	result := RenderVersionHistoryOverlay(versions, 0, "secret/ts", 100, 40)

	// The long timestamp should be truncated to 19 chars
	assert.Contains(t, result, "2024-01-15T10:30:00")
	assert.NotContains(t, result, ".123456789Z")
}

func TestRenderVersionHistoryOverlayDeletionTimeZero(t *testing.T) {
	versions := []model.SecretVersion{
		{
			Version:      "1",
			CreatedTime:  "2024-01-15T10:30:00Z",
			DeletionTime: "0001-01-01T00:00:00Z",
		},
	}

	result := RenderVersionHistoryOverlay(versions, 0, "secret/zero", 100, 40)

	// Zero deletion time should be treated as "current", not "deleted"
	assert.Contains(t, result, "current")
	assert.NotContains(t, result, "deleted")
}

// ---------------------------------------------------------------------------
// tryBase64Decode (RawStdEncoding fallback)
// ---------------------------------------------------------------------------

func TestTryBase64DecodeRawStdEncoding(t *testing.T) {
	// "hello" in raw standard base64 (no padding)
	got := tryBase64Decode("aGVsbG8")
	assert.Contains(t, got, "hello")
}

// ---------------------------------------------------------------------------
// renderSecretTable (allRevealed=true via RenderSecretOverlay)
// ---------------------------------------------------------------------------

func TestRenderSecretOverlayAllRevealed(t *testing.T) {
	secret := &model.Secret{
		Path: "secret/all-revealed",
		Keys: []string{"key1", "key2"},
		Data: map[string]string{"key1": "val1", "key2": "val2"},
	}
	result := RenderSecretOverlay(
		secret, 0,
		map[string]bool{"key1": true, "key2": true},
		true, false,
		"", "", -1, nil,
		100, 40,
	)
	assert.Contains(t, result, "val1")
	assert.Contains(t, result, "val2")
}

// ---------------------------------------------------------------------------
// RenderVersionHistoryOverlay scrolling
// ---------------------------------------------------------------------------

func TestRenderVersionHistoryOverlayScrolling(t *testing.T) {
	versions := make([]model.SecretVersion, 30)
	for i := range versions {
		versions[i] = model.SecretVersion{
			Version:     string(rune('0' + i%10)),
			CreatedTime: "2024-01-01T00:00:00Z",
		}
	}
	result := RenderVersionHistoryOverlay(versions, 25, "secret/scroll", 100, 20)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Version History")
}

// ---------------------------------------------------------------------------
// RenderSecretOverlay (panel dimension clamps)
// ---------------------------------------------------------------------------

func TestRenderSecretOverlayTinyScreen(t *testing.T) {
	// Extremely small screen triggers panelContentH < 3 and panelContentW < 20 clamps
	secret := &model.Secret{
		Path: "secret/tiny",
		Keys: []string{"k"},
		Data: map[string]string{"k": "v"},
	}
	assert.NotPanics(t, func() {
		RenderSecretOverlay(
			secret, 0, nil, false, false,
			"", "", -1, nil,
			10, 5,
		)
	})
}

// ---------------------------------------------------------------------------
// renderSecretTable (tableHeight < 1 clamp)
// ---------------------------------------------------------------------------

func TestRenderSecretTableTinyHeight(t *testing.T) {
	secret := &model.Secret{
		Keys: []string{"k"},
		Data: map[string]string{"k": "v"},
	}
	result := renderSecretTable(
		secret, 0, nil, nil,
		"", "", -1,
		80, 1, // height=1, tableHeight = 1-2 = -1, clamped to 1
	)
	assert.NotEmpty(t, result)
}

// ---------------------------------------------------------------------------
// RenderVersionHistoryOverlay (tableH < 1 clamp)
// ---------------------------------------------------------------------------

func TestRenderVersionHistoryOverlayTinyScreen(t *testing.T) {
	versions := []model.SecretVersion{
		{Version: "1", CreatedTime: "2024-01-01T00:00:00Z"},
	}
	assert.NotPanics(t, func() {
		RenderVersionHistoryOverlay(versions, 0, "p", 50, 10)
	})
}

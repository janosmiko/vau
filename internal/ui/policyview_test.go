package ui

import (
	"strings"
	"testing"

	"github.com/janosmiko/vau/internal/model"
	"github.com/stretchr/testify/assert"
)

var testParentEntries = []model.Entry{
	{Name: "secret/", IsDir: true},
	{Name: "[Policies]", IsDir: true},
	{Name: "[Auth Methods]", IsDir: true},
}

func TestRenderPolicyListEmpty(t *testing.T) {
	result := RenderPolicyList(testParentEntries, 1, nil, 0, "", nil, 0, "0.1.0", 100, 30)
	assert.Contains(t, result, "Policies/")
	assert.Contains(t, result, "(empty)")
}

func TestRenderPolicyListWithPolicies(t *testing.T) {
	policies := []string{"admin", "default", "readonly"}
	result := RenderPolicyList(testParentEntries, 1, policies, 1, "path \"secret/*\" {}", nil, 0, "0.1.0", 100, 30)
	assert.Contains(t, result, "admin")
	assert.Contains(t, result, "default")
}

func TestRenderPolicyListShowsVersion(t *testing.T) {
	result := RenderPolicyList(testParentEntries, 1, []string{"p1"}, 0, "", nil, 0, "1.2.3", 100, 30)
	assert.Contains(t, result, "vau 1.2.3")
}

func TestRenderPolicyListShowsTabBar(t *testing.T) {
	tabs := []string{"secret/", "kv/"}
	result := RenderPolicyList(testParentEntries, 1, []string{"p1"}, 0, "", tabs, 0, "0.1.0", 100, 30)
	assert.Contains(t, result, "secret/")
	assert.Contains(t, result, "kv/")
}

func TestRenderPolicyListSmallScreen(t *testing.T) {
	result := RenderPolicyList(nil, -1, []string{"p"}, 0, "", nil, 0, "v", 30, 8)
	assert.NotEmpty(t, result)
}

func TestRenderPolicyViewOverlay(t *testing.T) {
	hcl := "path \"secret/*\" {\n  capabilities = [\"read\", \"list\"]\n}"
	result := RenderPolicyViewOverlay("admin", hcl, 0, 100, 40)
	assert.Contains(t, result, "Policy: admin")
	assert.Contains(t, result, "secret/*")
}

func TestRenderPolicyViewOverlayScrolling(t *testing.T) {
	var lines []string
	for i := range 50 {
		lines = append(lines, "line "+strings.Repeat("x", i))
	}
	result := RenderPolicyViewOverlay("test", strings.Join(lines, "\n"), 10, 100, 40)
	assert.Contains(t, result, "11")
}

func TestRenderHCLPreviewEmpty(t *testing.T) {
	result := renderHCLPreview("", 40, 10)
	assert.Contains(t, result, "(no preview)")
}

func TestRenderHCLPreviewLimitsHeight(t *testing.T) {
	var lines []string
	for range 20 {
		lines = append(lines, "line content")
	}
	result := renderHCLPreview(strings.Join(lines, "\n"), 80, 10)
	outputLines := strings.Split(result, "\n")
	// header + separator + content + "..." indicator
	assert.LessOrEqual(t, len(outputLines), 11)
}

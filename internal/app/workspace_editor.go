package app

import (
	"encoding/json"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
	"github.com/janosmiko/vau/internal/vault"
)

func (m *Model) createPolicyWithEditor(name string) tea.Cmd {
	template := "# Policy: " + name + "\n# See: https://developer.hashicorp.com/vault/docs/concepts/policies\n\npath \"secret/*\" {\n  capabilities = [\"read\", \"list\"]\n}\n"
	return m.openEditorWithContent(template, ".hcl", func(content string) tea.Msg {
		err := m.client.PutPolicy(name, content)
		return policyEditorResultMsg{name: name, content: content, isNew: true, err: err}
	})
}

// editPolicyFromList fetches the policy content, then opens editor.
func (m *Model) editPolicyFromList(name string) tea.Cmd {
	return func() tea.Msg {
		content, err := m.client.GetPolicy(name)
		if err != nil {
			return errorMsg(fmt.Sprintf("reading policy %q: %v", name, err))
		}
		// Return a message that triggers the editor open
		return policyContentForEditMsg{name: name, content: content}
	}
}

// editRoleFromList fetches the role data, then opens editor.
func (m *Model) editRoleFromList(authPath, name string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.client.GetRole(authPath, name)
		if err != nil {
			return errorMsg(fmt.Sprintf("reading role %q: %v", name, err))
		}
		return roleDataForEditMsg{authPath: authPath, name: name, data: data}
	}
}

func (m *Model) editPolicyInEditor(name, content string) tea.Cmd {
	return m.openEditorWithContent(content, ".hcl", func(newContent string) tea.Msg {
		if newContent == content {
			return statusMsg("Policy unchanged")
		}
		err := m.client.PutPolicy(name, newContent)
		return policyEditorResultMsg{name: name, content: newContent, isNew: false, err: err}
	})
}

func (m *Model) deletePolicy(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.DeletePolicy(name)
		return policyDeletedMsg{name: name, err: err}
	}
}

// --- Role CRUD ---

// roleTemplate returns a prefilled JSON template based on the auth method type.
func roleTemplate(authPath string) string {
	authType := vault.AuthMethodType(authPath)
	switch authType {
	case "token":
		return `{
  "allowed_policies": ["default"],
  "disallowed_policies": [],
  "orphan": false,
  "renewable": true,
  "token_period": "0",
  "token_explicit_max_ttl": "0"
}
`
	case "approle":
		return `{
  "token_policies": ["default"],
  "token_ttl": "1h",
  "token_max_ttl": "4h",
  "secret_id_ttl": "0",
  "secret_id_num_uses": 0,
  "token_num_uses": 0,
  "bind_secret_id": true
}
`
	case "userpass":
		return `{
  "password": "",
  "token_policies": ["default"],
  "token_ttl": "1h",
  "token_max_ttl": "4h"
}
`
	case "ldap":
		return `{
  "policies": ["default"]
}
`
	case "jwt", "oidc":
		return `{
  "role_type": "` + authType + `",
  "bound_audiences": [],
  "user_claim": "sub",
  "token_policies": ["default"],
  "token_ttl": "1h",
  "token_max_ttl": "4h"
}
`
	case "cert":
		return `{
  "certificate": "",
  "token_policies": ["default"],
  "token_ttl": "1h"
}
`
	case "kubernetes":
		return `{
  "bound_service_account_names": ["*"],
  "bound_service_account_namespaces": ["default"],
  "token_policies": ["default"],
  "token_ttl": "1h",
  "token_max_ttl": "4h"
}
`
	default:
		return "{\n  \n}\n"
	}
}

func (m *Model) createRoleWithEditor(authPath, name string) tea.Cmd {
	template := roleTemplate(authPath)
	return m.openEditorWithContent(template, ".json", func(content string) tea.Msg {
		var data map[string]any
		if err := json.Unmarshal([]byte(content), &data); err != nil {
			return roleEditorResultMsg{authPath: authPath, name: name, err: fmt.Errorf("invalid JSON: %w", err)}
		}
		err := m.client.WriteRole(authPath, name, data)
		return roleEditorResultMsg{authPath: authPath, name: name, data: data, isNew: true, err: err}
	})
}

func (m *Model) editRoleInEditor(authPath, name string, data map[string]any) tea.Cmd {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return func() tea.Msg {
			return roleEditorResultMsg{authPath: authPath, name: name, err: fmt.Errorf("marshal: %w", err)}
		}
	}
	original := string(content)
	return m.openEditorWithContent(original, ".json", func(newContent string) tea.Msg {
		if newContent == original {
			return statusMsg("Role unchanged")
		}
		var newData map[string]any
		if err := json.Unmarshal([]byte(newContent), &newData); err != nil {
			return roleEditorResultMsg{authPath: authPath, name: name, err: fmt.Errorf("invalid JSON: %w", err)}
		}
		err := m.client.WriteRole(authPath, name, newData)
		return roleEditorResultMsg{authPath: authPath, name: name, data: newData, isNew: false, err: err}
	})
}

func (m *Model) deleteRole(authPath, name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.DeleteRole(authPath, name)
		return roleDeletedMsg{authPath: authPath, name: name, err: err}
	}
}

// --- Shared editor helper ---

// openEditorWithContent creates a temp file with the given content and extension,
// opens it in the user's editor, and calls the callback with the edited content.
func (m *Model) openEditorWithContent(content, extension string, callback func(string) tea.Msg) tea.Cmd {
	editor, err := m.editorCommand()
	if err != nil {
		return func() tea.Msg { return errorMsg("editor: " + err.Error()) }
	}

	tmpDir := os.TempDir()
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		tmpDir = xdg
	}

	tmpFile, err := os.CreateTemp(tmpDir, ".vau-edit-*"+extension)
	if err != nil {
		return func() tea.Msg { return errorMsg("temp file: " + err.Error()) }
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return func() tea.Msg { return errorMsg("write temp: " + err.Error()) }
	}
	tmpFile.Close()

	c := editorExecCommand(editor, tmpFile.Name())
	return tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.Remove(tmpFile.Name())
		if err != nil {
			return errorMsg("editor failed: " + err.Error())
		}
		edited, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			return errorMsg("read edited: " + err.Error())
		}
		return callback(string(edited))
	})
}

// --- CRUD message handlers ---

func (m *Model) handlePolicyEditorResult(msg policyEditorResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		m.mode = model.ModePolicyList
		return m, nil
	}
	if msg.isNew {
		m.status = fmt.Sprintf("Created policy: %s", msg.name)
	} else {
		m.status = fmt.Sprintf("Updated policy: %s", msg.name)
		m.policyContent = msg.content
		m.policyScroll = 0
	}
	m.mode = model.ModePolicyList
	return m, m.loadPolicies()
}

func (m *Model) handlePolicyDeletedResult(msg policyDeletedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.status = fmt.Sprintf("Deleted policy: %s", msg.name)
	m.mode = model.ModePolicyList
	return m, m.loadPolicies()
}

func (m *Model) handleRoleEditorResult(msg roleEditorResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		m.mode = model.ModeRoleList
		return m, nil
	}
	if msg.isNew {
		m.status = fmt.Sprintf("Created role: %s", msg.name)
	} else {
		m.status = fmt.Sprintf("Updated role: %s", msg.name)
		m.roleData = msg.data
		m.roleScroll = 0
	}
	m.mode = model.ModeRoleList
	return m, m.loadRoles(msg.authPath)
}

func (m *Model) handleRoleDeletedResult(msg roleDeletedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.status = fmt.Sprintf("Deleted role: %s", msg.name)
	m.mode = model.ModeRoleList
	return m, m.loadRoles(msg.authPath)
}

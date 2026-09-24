package app

import (
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
	"github.com/janosmiko/vau/internal/ui"
)

func (m *Model) loadTokenAccessors() tea.Cmd {
	return func() tea.Msg {
		accessors, err := m.client.ListTokenAccessors()
		return tokenListMsg{accessors: accessors, err: err}
	}
}

func (m *Model) loadTokenData(accessor string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.client.LookupAccessor(accessor)
		return tokenDataMsg{accessor: accessor, data: data, err: err}
	}
}

func (m *Model) loadTokenDataPreview(accessor string) tea.Cmd {
	return func() tea.Msg {
		data, _ := m.client.LookupAccessor(accessor)
		return tokenDataPreviewMsg{accessor: accessor, data: data}
	}
}

func (m *Model) createTokenFromRole(role string) tea.Cmd {
	return func() tea.Msg {
		data, err := m.client.CreateTokenWithRole(role, nil)
		return tokenCreatedMsg{data: data, err: err}
	}
}

func (m *Model) revokeToken(accessor string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.RevokeAccessor(accessor)
		return tokenRevokedMsg{accessor: accessor, err: err}
	}
}

// --- Token message handlers ---

func (m *Model) handleTokenListResult(msg tokenListMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.tokenAccessors = msg.accessors
	m.tokenCursor = 0
	m.tokenDataPreview = nil
	m.errMsg = ""
	if len(m.tokenAccessors) > 0 {
		return m, m.loadTokenDataPreview(m.tokenAccessors[0].Name)
	}
	return m, nil
}

func (m *Model) handleTokenDataResult(msg tokenDataMsg) tea.Model {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m
	}
	m.tokenData = msg.data
	m.tokenScroll = 0
	m.mode = model.ModeTokenView
	m.errMsg = ""
	return m
}

func (m *Model) handleTokenCreatedResult(msg tokenCreatedMsg) tea.Model {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m
	}
	m.tokenCreatedData = msg.data
	m.mode = model.ModeTokenCreated
	m.errMsg = ""
	return m
}

func (m *Model) handleTokenRevokedResult(msg tokenRevokedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.status = fmt.Sprintf("Revoked token: %s", msg.accessor)
	return m, m.loadTokenAccessors()
}

// --- Token key handlers ---

func (m *Model) handleTokenListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if result, handled := m.handleCommonKeys(key); handled {
		return result, nil
	}

	switch {
	case matchKey(key, m.keys.Quit):
		return m, tea.Quit

	case matchKey(key, m.keys.Up):
		if m.tokenCursor < len(m.tokenAccessors)-1 {
			m.tokenCursor++
			m.tokenDataPreview = nil
			return m, m.loadTokenDataPreview(m.tokenAccessors[m.tokenCursor].Name)
		}

	case matchKey(key, m.keys.Down):
		if m.tokenCursor > 0 {
			m.tokenCursor--
			m.tokenDataPreview = nil
			return m, m.loadTokenDataPreview(m.tokenAccessors[m.tokenCursor].Name)
		}

	case matchKey(key, m.keys.HalfDown):
		if n := len(m.tokenAccessors); n > 0 {
			m.tokenCursor = min(m.tokenCursor+10, n-1)
			m.tokenDataPreview = nil
			return m, m.loadTokenDataPreview(m.tokenAccessors[m.tokenCursor].Name)
		}
	case matchKey(key, m.keys.HalfUp):
		if len(m.tokenAccessors) > 0 {
			m.tokenCursor = max(m.tokenCursor-10, 0)
			m.tokenDataPreview = nil
			return m, m.loadTokenDataPreview(m.tokenAccessors[m.tokenCursor].Name)
		}
	case matchKey(key, m.keys.Top):
		if len(m.tokenAccessors) > 0 {
			m.tokenCursor = 0
			m.tokenDataPreview = nil
			return m, m.loadTokenDataPreview(m.tokenAccessors[m.tokenCursor].Name)
		}
	case matchKey(key, m.keys.Bottom):
		if n := len(m.tokenAccessors); n > 0 {
			m.tokenCursor = n - 1
			m.tokenDataPreview = nil
			return m, m.loadTokenDataPreview(m.tokenAccessors[m.tokenCursor].Name)
		}

	case matchKey(key, m.keys.Right) || matchKey(key, m.keys.Open):
		if len(m.tokenAccessors) > 0 && m.tokenCursor < len(m.tokenAccessors) {
			return m, m.loadTokenData(m.tokenAccessors[m.tokenCursor].Name)
		}

	case matchKey(key, m.keys.NewSecret):
		// Create custom token via editor
		template := `{
  "policies": ["default"],
  "ttl": "1h",
  "renewable": true,
  "num_uses": 0
}
`
		return m, m.openEditorWithContent(template, ".json", func(content string) tea.Msg {
			var params map[string]any
			if err := json.Unmarshal([]byte(content), &params); err != nil {
				return errorMsg("invalid JSON: " + err.Error())
			}
			data, err := m.client.CreateToken(params)
			return tokenCreatedMsg{data: data, err: err}
		})

	case matchKey(key, m.keys.Delete):
		if len(m.tokenAccessors) > 0 && m.tokenCursor < len(m.tokenAccessors) {
			accessor := m.tokenAccessors[m.tokenCursor].Name
			m.confirmMsg = fmt.Sprintf("Revoke token %s?", accessor[:min(len(accessor), 12)]+"...")
			m.prevConfirmMode = model.ModeTokenList
			m.confirmAction = func() tea.Cmd {
				return m.revokeToken(accessor)
			}
			m.enterConfirmMode()
			return m, nil
		}

	case matchKey(key, m.keys.Left) || key == "esc":
		m.mode = model.ModeExplorer
		m.atMountLevel = true
		return m, nil
	}
	return m, nil
}

func (m *Model) handleTokenViewKey(msg tea.KeyMsg) tea.Model {
	key := msg.String()

	switch {
	case matchKey(key, m.keys.Up):
		m.tokenScroll++
	case matchKey(key, m.keys.Down):
		m.tokenScroll = max(m.tokenScroll-1, 0)
	case matchKey(key, m.keys.HalfDown):
		m.tokenScroll += 10
	case matchKey(key, m.keys.HalfUp):
		m.tokenScroll = max(m.tokenScroll-10, 0)
	case matchKey(key, m.keys.FullDown):
		m.tokenScroll += 20
	case matchKey(key, m.keys.FullUp):
		m.tokenScroll = max(m.tokenScroll-20, 0)
	case matchKey(key, m.keys.Left) || key == "esc" || matchKey(key, m.keys.Quit):
		m.mode = model.ModeTokenList
		m.tokenData = nil
		m.tokenScroll = 0
	}
	return m
}

// --- Token render helper ---

func (m *Model) renderTokenListView() string {
	explorerHeight := m.height - 2
	parentEntries := m.mountEntries()
	parentIdx := -1
	for i, e := range parentEntries {
		if e.Name == accessTokens {
			parentIdx = i
			break
		}
	}
	tabLabels := m.workspaceTabLabels()
	return ui.RenderTokenList(
		parentEntries, parentIdx,
		m.tokenAccessors, m.tokenCursor, m.tokenDataPreview,
		tabLabels, m.activeTab,
		m.version, m.width, explorerHeight,
	)
}

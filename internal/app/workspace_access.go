package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/vau/internal/model"
	"github.com/janosmiko/vau/internal/ui"
)

// --- Entity loaders ---

func (m *Model) loadEntities() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		entities, err := client.ListEntities()
		return entityListMsg{entities: entities, err: err}
	}
}

func (m *Model) loadEntityData(name string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		data, err := client.GetEntity(name)
		if err == nil && data != nil {
			client.EnrichEntityAliases(data)
		}
		return entityDataMsg{name: name, data: data, err: err}
	}
}

func (m *Model) loadEntityDataPreview(name string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		data, _ := client.GetEntity(name)
		if data != nil {
			client.EnrichEntityAliases(data)
		}
		return entityDataPreviewMsg{name: name, data: data}
	}
}

// --- Group loaders ---

func (m *Model) loadGroups() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		groups, err := client.ListGroups()
		return groupListMsg{groups: groups, err: err}
	}
}

func (m *Model) loadGroupData(name string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		data, err := client.GetGroup(name)
		return groupDataMsg{name: name, data: data, err: err}
	}
}

func (m *Model) loadGroupDataPreview(name string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		data, _ := client.GetGroup(name)
		return groupDataPreviewMsg{name: name, data: data}
	}
}

// --- Entity message handlers ---

func (m *Model) handleEntityListResult(msg entityListMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.entities = msg.entities
	m.entityCursor = 0
	m.entityDataPreview = nil
	m.errMsg = ""
	if len(m.entities) > 0 {
		return m, m.loadEntityDataPreview(m.entities[0].Name)
	}
	return m, nil
}

func (m *Model) handleEntityDataResult(msg entityDataMsg) tea.Model {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m
	}
	m.entityName = msg.name
	m.entityData = msg.data
	m.entityScroll = 0
	m.mode = model.ModeEntityView
	m.errMsg = ""
	return m
}

// --- Group message handlers ---

func (m *Model) handleGroupListResult(msg groupListMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m, nil
	}
	m.groups = msg.groups
	m.groupCursor = 0
	m.groupDataPreview = nil
	m.errMsg = ""
	if len(m.groups) > 0 {
		return m, m.loadGroupDataPreview(m.groups[0].Name)
	}
	return m, nil
}

func (m *Model) handleGroupDataResult(msg groupDataMsg) tea.Model {
	if msg.err != nil {
		m.errMsg = msg.err.Error()
		return m
	}
	m.groupName = msg.name
	m.groupData = msg.data
	m.groupScroll = 0
	m.mode = model.ModeGroupView
	m.errMsg = ""
	return m
}

// --- Entity key handlers ---

func (m *Model) handleEntityListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if result, handled := m.handleCommonKeys(key); handled {
		return result, nil
	}

	switch {
	case matchKey(key, m.keys.Quit):
		return m, tea.Quit
	case matchKey(key, m.keys.Up):
		if m.entityCursor < len(m.entities)-1 {
			m.entityCursor++
			m.entityDataPreview = nil // show loading indicator
			return m, m.loadEntityDataPreview(m.entities[m.entityCursor].Name)
		}
	case matchKey(key, m.keys.Down):
		if m.entityCursor > 0 {
			m.entityCursor--
			m.entityDataPreview = nil
			return m, m.loadEntityDataPreview(m.entities[m.entityCursor].Name)
		}
	case matchKey(key, m.keys.HalfDown):
		if n := len(m.entities); n > 0 {
			m.entityCursor = min(m.entityCursor+10, n-1)
			m.entityDataPreview = nil
			return m, m.loadEntityDataPreview(m.entities[m.entityCursor].Name)
		}
	case matchKey(key, m.keys.HalfUp):
		if len(m.entities) > 0 {
			m.entityCursor = max(m.entityCursor-10, 0)
			m.entityDataPreview = nil
			return m, m.loadEntityDataPreview(m.entities[m.entityCursor].Name)
		}
	case matchKey(key, m.keys.Top):
		if len(m.entities) > 0 {
			m.entityCursor = 0
			m.entityDataPreview = nil
			return m, m.loadEntityDataPreview(m.entities[m.entityCursor].Name)
		}
	case matchKey(key, m.keys.Bottom):
		if n := len(m.entities); n > 0 {
			m.entityCursor = n - 1
			m.entityDataPreview = nil
			return m, m.loadEntityDataPreview(m.entities[m.entityCursor].Name)
		}
	case matchKey(key, m.keys.Right) || matchKey(key, m.keys.Open):
		if len(m.entities) > 0 && m.entityCursor < len(m.entities) {
			return m, m.loadEntityData(m.entities[m.entityCursor].Name)
		}
	case matchKey(key, m.keys.Left) || key == "esc":
		m.mode = model.ModeExplorer
		m.atMountLevel = true
		return m, nil
	}
	return m, nil
}

func (m *Model) handleEntityViewKey(msg tea.KeyMsg) tea.Model {
	key := msg.String()

	switch {
	case matchKey(key, m.keys.Up):
		m.entityScroll++
	case matchKey(key, m.keys.Down):
		m.entityScroll = max(m.entityScroll-1, 0)
	case matchKey(key, m.keys.HalfDown):
		m.entityScroll += 10
	case matchKey(key, m.keys.HalfUp):
		m.entityScroll = max(m.entityScroll-10, 0)
	case matchKey(key, m.keys.FullDown):
		m.entityScroll += 20
	case matchKey(key, m.keys.FullUp):
		m.entityScroll = max(m.entityScroll-20, 0)
	case matchKey(key, m.keys.Left) || key == "esc" || matchKey(key, m.keys.Quit):
		m.mode = model.ModeEntityList
		m.entityData = nil
		m.entityName = ""
		m.entityScroll = 0
	}
	return m
}

// --- Group key handlers ---

func (m *Model) handleGroupListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if result, handled := m.handleCommonKeys(key); handled {
		return result, nil
	}

	switch {
	case matchKey(key, m.keys.Quit):
		return m, tea.Quit
	case matchKey(key, m.keys.Up):
		if m.groupCursor < len(m.groups)-1 {
			m.groupCursor++
			m.groupDataPreview = nil
			return m, m.loadGroupDataPreview(m.groups[m.groupCursor].Name)
		}
	case matchKey(key, m.keys.Down):
		if m.groupCursor > 0 {
			m.groupCursor--
			m.groupDataPreview = nil
			return m, m.loadGroupDataPreview(m.groups[m.groupCursor].Name)
		}
	case matchKey(key, m.keys.HalfDown):
		if n := len(m.groups); n > 0 {
			m.groupCursor = min(m.groupCursor+10, n-1)
			m.groupDataPreview = nil
			return m, m.loadGroupDataPreview(m.groups[m.groupCursor].Name)
		}
	case matchKey(key, m.keys.HalfUp):
		if len(m.groups) > 0 {
			m.groupCursor = max(m.groupCursor-10, 0)
			m.groupDataPreview = nil
			return m, m.loadGroupDataPreview(m.groups[m.groupCursor].Name)
		}
	case matchKey(key, m.keys.Top):
		if len(m.groups) > 0 {
			m.groupCursor = 0
			m.groupDataPreview = nil
			return m, m.loadGroupDataPreview(m.groups[m.groupCursor].Name)
		}
	case matchKey(key, m.keys.Bottom):
		if n := len(m.groups); n > 0 {
			m.groupCursor = n - 1
			m.groupDataPreview = nil
			return m, m.loadGroupDataPreview(m.groups[m.groupCursor].Name)
		}
	case matchKey(key, m.keys.Right) || matchKey(key, m.keys.Open):
		if len(m.groups) > 0 && m.groupCursor < len(m.groups) {
			return m, m.loadGroupData(m.groups[m.groupCursor].Name)
		}
	case matchKey(key, m.keys.Left) || key == "esc":
		m.mode = model.ModeExplorer
		m.atMountLevel = true
		return m, nil
	}
	return m, nil
}

func (m *Model) handleGroupViewKey(msg tea.KeyMsg) tea.Model {
	key := msg.String()

	switch {
	case matchKey(key, m.keys.Up):
		m.groupScroll++
	case matchKey(key, m.keys.Down):
		m.groupScroll = max(m.groupScroll-1, 0)
	case matchKey(key, m.keys.HalfDown):
		m.groupScroll += 10
	case matchKey(key, m.keys.HalfUp):
		m.groupScroll = max(m.groupScroll-10, 0)
	case matchKey(key, m.keys.FullDown):
		m.groupScroll += 20
	case matchKey(key, m.keys.FullUp):
		m.groupScroll = max(m.groupScroll-20, 0)
	case matchKey(key, m.keys.Left) || key == "esc" || matchKey(key, m.keys.Quit):
		m.mode = model.ModeGroupList
		m.groupData = nil
		m.groupName = ""
		m.groupScroll = 0
	}
	return m
}

// --- Entity/Group render helpers ---

func (m *Model) renderEntityListView() string {
	explorerHeight := m.height - 2
	parentEntries := m.mountEntries()
	parentIdx := -1
	for i, e := range parentEntries {
		if e.Name == accessEntities {
			parentIdx = i
			break
		}
	}
	tabLabels := m.workspaceTabLabels()
	return ui.RenderEntityList(
		parentEntries, parentIdx,
		m.entities, m.entityCursor, m.entityDataPreview,
		tabLabels, m.activeTab,
		m.version, m.width, explorerHeight,
	)
}

func (m *Model) renderGroupListView() string {
	explorerHeight := m.height - 2
	parentEntries := m.mountEntries()
	parentIdx := -1
	for i, e := range parentEntries {
		if e.Name == accessGroups {
			parentIdx = i
			break
		}
	}
	tabLabels := m.workspaceTabLabels()
	return ui.RenderGroupList(
		parentEntries, parentIdx,
		m.groups, m.groupCursor, m.groupDataPreview,
		tabLabels, m.activeTab,
		m.version, m.width, explorerHeight,
	)
}

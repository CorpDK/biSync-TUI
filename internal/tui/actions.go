package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CorpDK/bisync-tui/internal/config"
	bisync "github.com/CorpDK/bisync-tui/internal/sync"
	"github.com/CorpDK/bisync-tui/internal/tui/components"
)

func (m AppModel) handleAction(msg components.ActionSelectedMsg) (tea.Model, tea.Cmd) {
	mapping := m.findMapping(msg.MappingName)
	if mapping == nil {
		return m, nil
	}
	switch msg.Action {
	case components.ActionSync:
		return m.submitSync(*mapping, bisync.SyncOptions{})
	case components.ActionDryRun:
		return m.submitSync(*mapping, bisync.SyncOptions{DryRun: true})
	case components.ActionResync:
		return m.showResyncModal(*mapping)
	case components.ActionInit:
		return m.initMapping(*mapping)
	case components.ActionLogs:
		m.detailPanel.SetMode(components.DetailLogs)
		return m, nil
	case components.ActionInfo:
		m.detailPanel.SetMode(components.DetailInfo)
		return m, nil
	case components.ActionRemoteSize:
		return m, m.fetchRemoteSize(*mapping)
	case components.ActionHistory:
		return m, m.loadHistory(mapping.Name)
	case components.ActionAllLogs:
		return m, m.loadAllLogs()
	case components.ActionDiff:
		return m.runDiffPreview()
	case components.ActionEncryption:
		return m.showEncryptionForm(msg.MappingName)
	case components.ActionEditMapping:
		return m.showEditMappingForm(*mapping)
	case components.ActionDeleteMapping:
		return m.showDeleteMappingModal(*mapping)
	}
	return m, nil
}

func (m AppModel) showResyncModal(mapping config.Mapping) (tea.Model, tea.Cmd) {
	modal := components.NewModal(
		"resync-"+mapping.Name,
		"Force Resync",
		"This will re-establish the baseline.\nAny unsynced changes may be overwritten.",
		m.width, m.height,
	)
	m.modal = &modal
	return m, nil
}

func (m AppModel) showDeleteMappingModal(mapping config.Mapping) (tea.Model, tea.Cmd) {
	modal := components.NewModal(
		"delete-mapping-"+mapping.Name,
		"Delete Mapping",
		"Are you sure you want to delete mapping '"+mapping.Name+
			"'?\nThis removes it from your config.",
		m.width, m.height,
	)
	m.modal = &modal
	return m, nil
}

func (m AppModel) handleModalConfirm(msg components.ModalConfirmMsg) (tea.Model, tea.Cmd) {
	switch {
	case strings.HasPrefix(msg.ID, "resync-"):
		name := strings.TrimPrefix(msg.ID, "resync-")
		if mapping := m.findMapping(name); mapping != nil {
			return m.submitSync(*mapping, bisync.SyncOptions{Resync: true})
		}
	case strings.HasPrefix(msg.ID, "delete-remote-"):
		name := strings.TrimPrefix(msg.ID, "delete-remote-")
		m.remoteDetail.SetStatus("◐ Deleting remote '" + name + "'...")
		return m, m.deleteRemote(name)
	case strings.HasPrefix(msg.ID, "delete-mapping-"):
		name := strings.TrimPrefix(msg.ID, "delete-mapping-")
		return m.handleDeleteMapping(name)
	}
	return m, nil
}

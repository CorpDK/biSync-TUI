package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CorpDK/bisync-tui/internal/config"
	"github.com/CorpDK/bisync-tui/internal/tui/components"
	"github.com/CorpDK/bisync-tui/internal/tui/forms"
)

func (m AppModel) showNewMappingForm() (tea.Model, tea.Cmd) {
	form, keys := forms.NewMappingForm()
	overlay := components.NewFormOverlay("create-mapping", form, keys, m.width, m.height)
	m.formOverlay = &overlay
	return m, overlay.Init()
}

func (m AppModel) showEditMappingForm(mapping config.Mapping) (tea.Model, tea.Cmd) {
	form, keys := forms.NewEditMappingForm(forms.MappingValues{
		Name:            mapping.Name,
		Local:           mapping.Local,
		Remote:          mapping.Remote,
		FiltersFile:     mapping.FiltersFile,
		BandwidthLimit:  mapping.BandwidthLimit,
		ConflictResolve: mapping.ConflictResolve,
		BackupEnabled:   mapping.BackupEnabled,
	})
	overlay := components.NewFormOverlay("edit-mapping-"+mapping.Name, form, keys, m.width, m.height)
	m.formOverlay = &overlay
	return m, overlay.Init()
}

func (m AppModel) showEncryptionForm(mappingName string) (tea.Model, tea.Cmd) {
	remotes, err := m.engine.ListRemotes(m.ctx)
	if err != nil {
		m.detailPanel.AppendLog("Error listing remotes: " + err.Error())
		return m, nil
	}
	if len(remotes) == 0 {
		m.detailPanel.AppendLog("No rclone remotes configured. Run 'rclone config' first.")
		return m, nil
	}
	form, keys := forms.NewEncryptionForm(mappingName, remotes)
	overlay := components.NewFormOverlay("setup-encryption-"+mappingName, form, keys, m.width, m.height)
	m.formOverlay = &overlay
	return m, overlay.Init()
}

func (m AppModel) handleFormSubmit(msg components.FormSubmittedMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.ID == "create-mapping":
		return m.handleCreateMapping(msg.Values)
	case msg.ID == "create-profile":
		return m.handleCreateProfile(msg.Values)
	case strings.HasPrefix(msg.ID, "edit-mapping-"):
		name := strings.TrimPrefix(msg.ID, "edit-mapping-")
		return m.handleEditMapping(name, msg.Values)
	case strings.HasPrefix(msg.ID, "setup-encryption-"):
		name := strings.TrimPrefix(msg.ID, "setup-encryption-")
		return m.handleSetupEncryption(name, msg.Values)
	}
	return m, nil
}

func (m AppModel) handleCreateMapping(values map[string]string) (tea.Model, tea.Cmd) {
	mapping := mappingFromValues(values)
	cfgPath := config.ProfilePath("")
	if err := config.AddMapping(cfgPath, mapping); err != nil {
		m.detailPanel.AppendLog("Error adding mapping: " + err.Error())
		return m, nil
	}
	return m.reloadConfig()
}

func (m AppModel) handleEditMapping(originalName string, values map[string]string) (tea.Model, tea.Cmd) {
	mapping := mappingFromValues(values)

	// Preserve existing fields not in the form
	if orig := m.findMapping(originalName); orig != nil {
		mapping.Encryption = orig.Encryption
		if mapping.BackupEnabled && orig.BackupRetention > 0 {
			mapping.BackupRetention = orig.BackupRetention
		}
	}

	cfgPath := config.ProfilePath("")
	if originalName != mapping.Name {
		if err := config.RemoveMapping(cfgPath, originalName); err != nil {
			m.detailPanel.AppendLog("Error removing old mapping: " + err.Error())
			return m, nil
		}
		if err := config.AddMapping(cfgPath, mapping); err != nil {
			m.detailPanel.AppendLog("Error adding renamed mapping: " + err.Error())
			return m, nil
		}
	} else {
		if err := config.UpdateMapping(cfgPath, mapping); err != nil {
			m.detailPanel.AppendLog("Error updating mapping: " + err.Error())
			return m, nil
		}
	}
	return m.reloadConfig()
}

func (m AppModel) handleDeleteMapping(name string) (tea.Model, tea.Cmd) {
	cfgPath := config.ProfilePath("")
	if err := config.RemoveMapping(cfgPath, name); err != nil {
		m.detailPanel.AppendLog("Error deleting mapping: " + err.Error())
		return m, nil
	}
	return m.reloadConfig()
}

func (m AppModel) handleSetupEncryption(mappingName string, values map[string]string) (tea.Model, tea.Cmd) {
	mapping := m.findMapping(mappingName)
	if mapping == nil {
		return m, nil
	}
	mapping.Encryption = config.EncryptionConfig{
		Enabled:     true,
		CryptRemote: values["crypt_remote"],
	}
	cfgPath := config.ProfilePath("")
	if err := config.UpdateMapping(cfgPath, *mapping); err != nil {
		m.detailPanel.AppendLog("Error saving encryption config: " + err.Error())
		return m, nil
	}
	cfg, err := config.LoadProfile("")
	if err != nil {
		m.detailPanel.AppendLog("Error reloading config: " + err.Error())
		return m, nil
	}
	m.config = cfg
	m.detailPanel.AppendLog("Encryption enabled for " + mappingName + " using " + values["crypt_remote"])
	for _, mp := range cfg.Mappings {
		if mp.Name == mappingName {
			ms := m.states[mappingName]
			m.detailPanel.SetMapping(&mp, ms)
			break
		}
	}
	return m, nil
}

func (m AppModel) handleCreateProfile(values map[string]string) (tea.Model, tea.Cmd) {
	name := values["name"]
	path := config.ProfilePath(name)
	if err := config.CreateDefaultConfig(path); err != nil {
		m.detailPanel.AppendLog("Error creating profile: " + err.Error())
		return m, nil
	}
	m.detailPanel.AppendLog("Created profile: " + name)
	return m, nil
}

func (m AppModel) reloadConfig() (tea.Model, tea.Cmd) {
	cfg, err := config.LoadProfile("")
	if err != nil {
		m.detailPanel.AppendLog("Error reloading config: " + err.Error())
		return m, nil
	}
	m.config = cfg
	m.states = m.stateStore.LoadAll(cfg.Mappings)
	m.mappingList = components.NewMappingList(cfg.Mappings, m.states,
		m.mappingList.Width(), m.mappingList.Height())
	m.detailPanel.Reset()
	return m, nil
}

// mappingFromValues creates a Mapping from form values.
func mappingFromValues(values map[string]string) config.Mapping {
	m := config.Mapping{
		Name:            values["name"],
		Local:           values["local"],
		Remote:          values["remote"],
		FiltersFile:     values["filters_file"],
		BandwidthLimit:  values["bandwidth_limit"],
		ConflictResolve: values["conflict_resolve"],
		BackupEnabled:   values["backup_enabled"] == "true",
	}
	if m.BackupEnabled {
		m.BackupRetention = 7
	}
	return m
}


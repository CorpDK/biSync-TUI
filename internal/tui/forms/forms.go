package forms

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

// NewProfileForm builds a form for creating a new config profile.
func NewProfileForm() (*huh.Form, []string) {
	var name string

	keys := []string{"name"}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("name").
				Title("Profile Name").
				Description("A short identifier for this profile (e.g., work, laptop)").
				Value(&name).
				Validate(nonEmpty("profile name")),
		),
	).WithShowHelp(true)

	return form, keys
}

// NewMappingForm builds a form for creating a new sync mapping.
func NewMappingForm() (*huh.Form, []string) {
	var (
		name            string
		local           string
		remote          string
		filtersFile     string
		bandwidthLimit  string
		conflictResolve string
		backupEnabled   string
	)

	keys := []string{"name", "local", "remote", "filters_file", "bandwidth_limit", "conflict_resolve", "backup_enabled"}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("name").
				Title("Mapping Name").
				Description("Unique name for this sync pair").
				Value(&name).
				Validate(nonEmpty("mapping name")),
			huh.NewInput().
				Key("local").
				Title("Local Path").
				Description("Local directory (e.g., ~/Documents)").
				Value(&local).
				Validate(nonEmpty("local path")),
			huh.NewInput().
				Key("remote").
				Title("Remote Path").
				Description("Remote path (e.g., gdrive:MyDrive/Documents)").
				Value(&remote).
				Validate(nonEmpty("remote path")),
		),
		huh.NewGroup(
			huh.NewInput().
				Key("filters_file").
				Title("Filters File (optional)").
				Description("Path to rclone filters file").
				Value(&filtersFile),
			huh.NewInput().
				Key("bandwidth_limit").
				Title("Bandwidth Limit (optional)").
				Description("e.g., 10M").
				Value(&bandwidthLimit),
			huh.NewSelect[string]().
				Key("conflict_resolve").
				Title("Conflict Resolution").
				Options(
					huh.NewOption("Newer file wins", "newer"),
					huh.NewOption("Older file wins", "older"),
					huh.NewOption("Local (Path1) wins", "path1"),
					huh.NewOption("Remote (Path2) wins", "path2"),
				).
				Value(&conflictResolve),
			huh.NewSelect[string]().
				Key("backup_enabled").
				Title("Enable Backups").
				Options(
					huh.NewOption("Yes", "true"),
					huh.NewOption("No", "false"),
				).
				Value(&backupEnabled),
		),
	).WithShowHelp(true)

	return form, keys
}

// MappingValues holds current values for pre-filling the edit mapping form.
type MappingValues struct {
	Name            string
	Local           string
	Remote          string
	FiltersFile     string
	BandwidthLimit  string
	ConflictResolve string
	BackupEnabled   bool
}

// NewEditMappingForm builds a form pre-filled with existing mapping values.
func NewEditMappingForm(v MappingValues) (*huh.Form, []string) {
	name := v.Name
	local := v.Local
	remote := v.Remote
	filtersFile := v.FiltersFile
	bandwidthLimit := v.BandwidthLimit
	conflictResolve := v.ConflictResolve
	if conflictResolve == "" {
		conflictResolve = "newer"
	}
	backupEnabled := "false"
	if v.BackupEnabled {
		backupEnabled = "true"
	}

	keys := []string{"name", "local", "remote", "filters_file", "bandwidth_limit", "conflict_resolve", "backup_enabled"}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("name").
				Title("Mapping Name").
				Description("Unique name for this sync pair").
				Value(&name).
				Validate(nonEmpty("mapping name")),
			huh.NewInput().
				Key("local").
				Title("Local Path").
				Description("Local directory (e.g., ~/Documents)").
				Value(&local).
				Validate(nonEmpty("local path")),
			huh.NewInput().
				Key("remote").
				Title("Remote Path").
				Description("Remote path (e.g., gdrive:MyDrive/Documents)").
				Value(&remote).
				Validate(nonEmpty("remote path")),
		),
		huh.NewGroup(
			huh.NewInput().
				Key("filters_file").
				Title("Filters File (optional)").
				Description("Path to rclone filters file").
				Value(&filtersFile),
			huh.NewInput().
				Key("bandwidth_limit").
				Title("Bandwidth Limit (optional)").
				Description("e.g., 10M").
				Value(&bandwidthLimit),
			huh.NewSelect[string]().
				Key("conflict_resolve").
				Title("Conflict Resolution").
				Options(
					huh.NewOption("Newer file wins", "newer"),
					huh.NewOption("Older file wins", "older"),
					huh.NewOption("Local (Path1) wins", "path1"),
					huh.NewOption("Remote (Path2) wins", "path2"),
				).
				Value(&conflictResolve),
			huh.NewSelect[string]().
				Key("backup_enabled").
				Title("Enable Backups").
				Options(
					huh.NewOption("Yes", "true"),
					huh.NewOption("No", "false"),
				).
				Value(&backupEnabled),
		),
	).WithShowHelp(true)

	return form, keys
}

// NewEncryptionForm builds a form for configuring encryption on a mapping.
// remotes is the list of available rclone remotes to choose from.
func NewEncryptionForm(mappingName string, remotes []string) (*huh.Form, []string) {
	var cryptRemote string

	keys := []string{"crypt_remote"}

	opts := make([]huh.Option[string], 0, len(remotes))
	for _, r := range remotes {
		opts = append(opts, huh.NewOption(r, r))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("crypt_remote").
				Title("Crypt Remote for "+mappingName).
				Description("Select the rclone crypt remote to use").
				Options(opts...).
				Value(&cryptRemote),
		),
	).WithShowHelp(true)

	return form, keys
}

func nonEmpty(field string) func(string) error {
	return func(s string) error {
		if s == "" {
			return fmt.Errorf("%s cannot be empty", field)
		}
		return nil
	}
}

package launcher

import (
	"context"
	"time"
)

// saveSettingsTable persists one settings-owned table and rolls the editor back if core rejects it.
func (a *App) saveSettingsTable(state *formTableEditorState, key, value, previousValue string) {
	coreValue := value
	if key == "IgnoredHotkeyApps" {
		var err error
		coreValue, err = settingsIgnoredHotkeyAppsCoreJSON(value)
		if err != nil {
			_ = a.runOnUI("apply invalid settings table value", func() {
				a.settingSaving = false
				state.target.values[key] = previousValue
				if a.settingsTableEditor == state {
					state.saving = false
					state.status = "Could not save: " + err.Error()
				}
				a.invalidateSettingsWindow()
			})
			return
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err := a.services.UpdateGeneralSetting(ctx, a.sessionID, key, coreValue)
	cancel()

	_ = a.runOnUI("apply settings table save", func() {
		a.settingSaving = false
		if err != nil {
			state.target.values[key] = previousValue
			if a.settingsTableEditor == state {
				if rows, decodeErr := decodeFormTableRows(previousValue); decodeErr == nil {
					state.rows = rows
					state.selected = min(state.selected, len(rows)-1)
				}
				state.saving = false
				state.status = "Could not save: " + err.Error()
			}
		} else {
			if state.target == a.hotkeySettings.Form() || state.target == a.generalQuerySettingsForm() {
				a.applyHotkeySettingsRawLocked(key, coreValue)
			}
			if a.settingsTableEditor == state {
				state.saving = false
				state.status = ""
			}
		}
		a.invalidateSettingsWindow()
	})
}

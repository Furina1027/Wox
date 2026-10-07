package launcher

// selectedPluginID returns the catalog identity behind the current detail pane.
func (a *App) selectedPluginID() string {
	plugins := a.pluginSettings.Plugins()
	selected := a.pluginSettings.Selected()
	if selected < 0 || selected >= len(plugins) {
		return ""
	}
	return plugins[selected].ID
}

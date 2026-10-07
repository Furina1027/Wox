package ui

import (
	"testing"

	"wox/common"
	"wox/setting"
)

func TestValidateAutoThemeDraftRejectsEmptyAndAutoEndpoints(t *testing.T) {
	themes := map[string]common.Theme{
		"light": {ThemeId: "light", ThemeName: "Light"},
		"dark":  {ThemeId: "dark", ThemeName: "Dark"},
		"auto":  {ThemeId: "auto", ThemeName: "Auto", IsAutoAppearance: true},
	}
	if err := validateAutoThemeDraft("", "light", "dark", themes); err == nil {
		t.Fatal("empty name should fail")
	}
	if err := validateAutoThemeDraft("Mine", "missing", "dark", themes); err == nil {
		t.Fatal("missing endpoint should fail")
	}
	if err := validateAutoThemeDraft("Mine", "auto", "dark", themes); err == nil {
		t.Fatal("auto endpoint should fail")
	}
	if err := validateAutoThemeDraft("Mine", "light", "dark", themes); err != nil {
		t.Fatalf("valid draft failed: %v", err)
	}
}

func TestComposeUserAutoThemeKeepsAppearanceFields(t *testing.T) {
	theme := composeUserAutoTheme("user-auto", "Evening", common.Theme{
		SchemaVersion: 2, ThemeId: setting.DefaultAutoThemeId, ThemeName: "Wox Auto",
		IsAutoAppearance: true, LightThemeId: "old-light", DarkThemeId: "old-dark",
		AppBackgroundColor: "#111111FF", IsSystem: true,
	}, "light", "dark")
	if theme.ThemeId != "user-auto" || theme.ThemeName != "Evening" || theme.IsSystem || !theme.IsAutoAppearance {
		t.Fatalf("identity = %#v", theme)
	}
	if theme.LightThemeId != "light" || theme.DarkThemeId != "dark" || theme.AppBackgroundColor != "#111111FF" {
		t.Fatalf("pair or fallback colors = %#v", theme)
	}
}

func TestReplaceRemovedAutoEndpointsUsesSystemDefaults(t *testing.T) {
	light, dark, changed := replaceRemovedAutoEndpoints("custom", "custom", "dark")
	if !changed || light != setting.DefaultLightThemeId || dark != "dark" {
		t.Fatalf("light replacement = %s %s %v", light, dark, changed)
	}
	light, dark, changed = replaceRemovedAutoEndpoints("custom", "light", "custom")
	if !changed || light != "light" || dark != setting.DefaultDarkThemeId {
		t.Fatalf("dark replacement = %s %s %v", light, dark, changed)
	}
	if _, _, changed = replaceRemovedAutoEndpoints("custom", "light", "dark"); changed {
		t.Fatal("unrelated uninstall should not change auto endpoints")
	}
}

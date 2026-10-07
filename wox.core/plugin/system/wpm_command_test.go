package system

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"wox/common"
	"wox/plugin"

	"github.com/stretchr/testify/require"
)

type wpmCommandTestAPI struct {
	plugin.API
	query common.PlainQuery
}

// TestPluginTemplates verifies both runtimes and legacy/current manifest placeholders.
func TestPluginTemplates(t *testing.T) {
	for _, runtime := range []plugin.Runtime{plugin.PLUGIN_RUNTIME_PYTHON, plugin.PLUGIN_RUNTIME_NODEJS} {
		t.Run(string(runtime), func(t *testing.T) {
			found := false
			for _, template := range pluginTemplates {
				if template.Runtime == runtime {
					found = true
					require.Equal(t, "https://codeload.github.com/Wox-launcher/"+template.Name+"/zip/refs/heads/main", template.Url)
				}
			}
			require.True(t, found)
			for _, manifest := range []string{
				`{"Id":"{{.Id}}","Name":"{{.Name}}","Runtime":"` + strings.ToLower(string(runtime)) + `","TriggerKeywords":["{{.TriggerKeyword}}"],"Description":"{{.Description}}","Author":"{{.Author}}","Website":"{{.Website}}"}`,
				`{"Id":"[Id]","Name":"[Name]","Runtime":"[Runtime]","TriggerKeywords":["[Trigger Keyword]"],"Description":"[Description]","Author":"[Author]","Website":"[Website]"}`,
			} {
				rendered := renderPluginTemplateManifest(manifest, `Test "Plugin"`, runtime)
				var metadata plugin.Metadata
				require.NoError(t, json.Unmarshal([]byte(rendered), &metadata))
				require.Equal(t, `Test "Plugin"`, string(metadata.Name))
				require.Equal(t, strings.ToLower(string(runtime)), string(metadata.Runtime))
				require.Equal(t, []string{"np"}, metadata.TriggerKeywords)
				require.Equal(t, `Test "Plugin"`, string(metadata.Description))
				require.Equal(t, "Wox User", metadata.Author)
				require.Empty(t, metadata.Website)
				require.NotContains(t, rendered, "{{.")
				require.NotEmpty(t, metadata.Id)
			}
		})
	}
}
func (a *wpmCommandTestAPI) ChangeQuery(ctx context.Context, query common.PlainQuery) {
	a.query = query
}

// TestWPMCommandDiscovery covers filtering and entering commands through each alias.
func TestWPMCommandDiscovery(t *testing.T) {
	ctx := context.Background()
	api := &wpmCommandTestAPI{}
	w := &WPMPlugin{api: api}
	for _, keyword := range []string{"store", "wpm", "pm"} {
		hint, _ := plugin.MatchQueryHint(keyword+" ", []*plugin.Instance{{Metadata: w.GetMetadata()}})
		require.NotNil(t, hint)
		require.Equal(t, []string{"create", "install", "uninstall"}, hint.Elements[1].Suggestions)
		require.True(t, hint.CommandSuggestions, "command completion must retain its trailing space")
		for _, tc := range []struct {
			search string
			count  int
		}{
			{"", 3},
			{"inst", 2},
			{" INST ", 2},
			{" DEV. ", 0},
			{"dev.add", 0},
			{"create", 1},
			{"missing-command", 0},
		} {
			t.Run(keyword+"/"+tc.search, func(t *testing.T) {
				response := w.Query(ctx, plugin.Query{Type: plugin.QueryTypeInput, TriggerKeyword: keyword, Search: tc.search})
				require.Len(t, response.Results, tc.count)
				for _, result := range response.Results {
					require.NotEmpty(t, result.SubTitle)
					require.Len(t, result.Actions, 1)
					require.True(t, result.Actions[0].PreventHideAfterAction)
					result.Actions[0].Action(ctx, plugin.ActionContext{})
					require.Equal(t, common.PlainQuery{QueryType: plugin.QueryTypeInput, QueryText: keyword + " " + result.Title + " "}, api.query)
				}
			})
		}
	}
	response := w.Query(ctx, plugin.Query{Type: plugin.QueryTypeInput, TriggerKeyword: "store", Command: "create"})
	require.Len(t, response.Results, 1)
	require.Equal(t, "i18n:plugin_wpm_enter_plugin_name", response.Results[0].Title)

	createResults := w.createCommand(ctx, plugin.Query{TriggerKeyword: "wpm", Search: "demo"})
	require.Len(t, createResults, 4)
	for _, result := range createResults {
		require.NotContains(t, result.Title, "script_template")
		require.NotContains(t, result.Group, "group_script_plugins")
	}
}

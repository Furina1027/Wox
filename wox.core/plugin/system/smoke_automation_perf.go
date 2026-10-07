//go:build wox_automation

package system

import (
	"fmt"

	"wox/common"
	"wox/common/icons"
	"wox/plugin"
)

const (
	smokeAutomationListCommand      = "list-500"
	smokeAutomationGridCommand      = "grid-500"
	smokeAutomationWarmCacheCommand = "warm-cache"
	smokeAutomationListCount        = 500
	smokeAutomationGridCount        = 500
	smokeAutomationWarmCacheCount   = 8
)

func queryListFixture() plugin.QueryResponse {
	results := make([]plugin.QueryResult, 0, smokeAutomationListCount)
	for index := range smokeAutomationListCount {
		results = append(results, plugin.QueryResult{
			Id:       fmt.Sprintf("perf-list-%04d", index),
			Title:    fmt.Sprintf("Perf list result %04d", index),
			SubTitle: "Deterministic list fixture",
			Icon:     icons.Get(icons.PluginApp),
		})
	}
	return plugin.NewQueryResponse(results)
}

func queryGridFixture() plugin.QueryResponse {
	results := make([]plugin.QueryResult, 0, smokeAutomationGridCount)
	for index := range smokeAutomationGridCount {
		group := fmt.Sprintf("Group %02d", index/50)
		results = append(results, plugin.QueryResult{
			Id:         fmt.Sprintf("perf-grid-%04d", index),
			Title:      fmt.Sprintf("Grid %04d", index),
			SubTitle:   group,
			Icon:       icons.Get(icons.PluginApp),
			Group:      group,
			GroupScore: int64(1000 - index/50),
		})
	}
	return plugin.QueryResponse{
		Results: results,
		Layout: plugin.QueryLayout{
			GridLayout: &plugin.MetadataFeatureParamsGridLayout{
				Columns: 6, ShowTitle: true, ItemMargin: 6, AspectRatio: 1,
			},
		},
	}
}

// queryChatFixture publishes an observable streaming state and completes it after the last update.
func queryWarmCacheFixture() plugin.QueryResponse {
	icons := []common.WoxImage{icons.Get(icons.PluginApp), icons.Get(icons.PluginCalculator)}
	titles := []string{"Warm cache alpha", "Warm cache beta"}
	results := make([]plugin.QueryResult, 0, smokeAutomationWarmCacheCount)
	for index := range smokeAutomationWarmCacheCount {
		results = append(results, plugin.QueryResult{
			Id:       fmt.Sprintf("perf-warm-%d", index),
			Title:    titles[index%len(titles)],
			SubTitle: "Repeated text and image fixture",
			Icon:     icons[index%len(icons)],
		})
	}
	return plugin.NewQueryResponse(results)
}

// chatFixturePreview exercises distinct historical Markdown and a growing final answer.

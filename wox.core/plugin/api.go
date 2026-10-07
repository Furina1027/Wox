package plugin

import (
	"context"
	"fmt"
	"wox/common"
	"wox/common/icons"
	"wox/setting/definition"
	"wox/util"
	"wox/util/clipboard"
)

type LogLevel = string

type RegisterTriggerKeywordOption struct {
	Keyword   string
	QueryHint *common.QueryHint
	// QueryVariables is core-only context requested by built-in searches before launcher activation.
	QueryVariables []QueryVariable `json:"-"`
}

type RegisterTriggerKeywordResult struct {
	Success bool
}

type UnregisterTriggerKeywordOption struct {
	Keyword string
}

type UnregisterTriggerKeywordResult struct {
	Success bool
}

type DragOutStatus string

const (
	DragOutStatusSuccess        DragOutStatus = "success"
	DragOutStatusCancel         DragOutStatus = "cancel"
	DragOutStatusCancelInSource DragOutStatus = "cancel_in_source"
)

// DragOutEvent is a post-drag notification. It does not enable, disable, or cancel the drag.
type DragOutEvent struct {
	ResultId string        `json:"ResultId"`
	Files    []string      `json:"Files"`
	Status   DragOutStatus `json:"Status"`
}

type DragOutListenOption struct {
	Callback func(ctx context.Context, event DragOutEvent)
}

type DragOutListenResult struct {
	Success bool
}

const (
	LogLevelInfo    LogLevel = "Info"
	LogLevelError   LogLevel = "Error"
	LogLevelDebug   LogLevel = "Debug"
	LogLevelWarning LogLevel = "Warning"
)

type CopyType string

const (
	CopyTypePlainText CopyType = "text"
	CopyTypeImage     CopyType = "image"
)

// API exposes the runtime services that plugins can call back into.
type API interface {
	ChangeQuery(ctx context.Context, query common.PlainQuery)
	HideApp(ctx context.Context)
	ShowApp(ctx context.Context)
	Notify(ctx context.Context, description string)
	PushAttention(ctx context.Context, request PushAttentionRequest)
	Log(ctx context.Context, level LogLevel, msg string)
	GetTranslation(ctx context.Context, key string) string
	GetSetting(ctx context.Context, key string) string
	// SaveSetting is kept for compatibility with plugins targeting Wox before v2.4.0.
	//
	// Deprecated: use SetSetting.
	SaveSetting(ctx context.Context, key string, value string, isPlatformSpecific bool)
	// SetSetting saves a plugin setting with explicit platform and device-local behavior.
	// Available since Wox v2.4.0.
	SetSetting(ctx context.Context, option SetSettingOption) SetSettingResult
	OnSettingChanged(ctx context.Context, callback func(ctx context.Context, key string, value string))
	OnGetDynamicSetting(ctx context.Context, callback func(ctx context.Context, key string) definition.PluginSettingDefinitionItem)
	OnDeepLink(ctx context.Context, callback func(ctx context.Context, arguments map[string]string))
	OnUnload(ctx context.Context, callback func(ctx context.Context))
	OnMRURestore(ctx context.Context, callback func(ctx context.Context, mruData MRUData) (*QueryResult, error))

	// RegisterPluginTool publishes a callable tool owned by the current plugin.
	// Requires Wox >= 2.4.5.
	RegisterPluginTool(ctx context.Context, option RegisterPluginToolOption) RegisterPluginToolResult
	// UnregisterPluginTool removes a tool previously registered by this plugin.
	// Repeating the operation succeeds. Requires Wox >= 2.4.5.
	UnregisterPluginTool(ctx context.Context, option UnregisterPluginToolOption) UnregisterPluginToolResult
	// ListPluginTools returns currently callable tools from initialized, enabled plugins.
	// Requires Wox >= 2.4.5.
	ListPluginTools(ctx context.Context, option ListPluginToolsOption) ListPluginToolsResult
	// InvokePluginTool executes another plugin's registered tool after schema validation.
	// Requires Wox >= 2.4.5.
	InvokePluginTool(ctx context.Context, option InvokePluginToolOption) InvokePluginToolResult

	// ShowToolbarMsg creates or updates the toolbar msg for the current plugin query context.
	// It is only accepted while the caller is the active plugin in the current session.
	// Leaving that plugin query context clears the toolbar msg automatically.
	ShowToolbarMsg(ctx context.Context, msg ToolbarMsg)

	// ClearToolbarMsg removes a toolbar msg previously shown by this plugin by its id.
	ClearToolbarMsg(ctx context.Context, toolbarMsgId string)

	// OnEnterPluginQuery registers a callback that fires once when the session enters
	// this plugin's query context.
	OnEnterPluginQuery(ctx context.Context, callback func(ctx context.Context))

	// OnLeavePluginQuery registers a callback that fires once when the session leaves
	// this plugin's query context.
	OnLeavePluginQuery(ctx context.Context, callback func(ctx context.Context))
	// OnDragOut registers a notification after a result file drag started from
	// this plugin reaches a terminal OS status. QueryResultDragData is what
	// makes a result draggable; this callback cannot block or cancel that drag.
	// Native drags stay copy-only, so a plugin that wants take-out to empty its
	// own source should remove those files when Status is success.
	OnDragOut(ctx context.Context, option DragOutListenOption) DragOutListenResult
	RegisterQueryCommands(ctx context.Context, commands []MetadataCommand)
	// RegisterTriggerKeyword returns Success=false for invalid keywords or keywords owned by another enabled plugin.
	// Re-registering this plugin's own keyword succeeds without adding a duplicate.
	RegisterTriggerKeyword(ctx context.Context, option RegisterTriggerKeywordOption) RegisterTriggerKeywordResult
	// UnregisterTriggerKeyword releases only this plugin's runtime registration.
	// An already absent registration is also considered successful.
	UnregisterTriggerKeyword(ctx context.Context, option UnregisterTriggerKeywordOption) UnregisterTriggerKeywordResult

	// GetUpdatableResult retrieves the current state of a result from the result cache.
	// Returns nil if the result is not found (no longer visible in UI).
	// Returns a pointer to UpdatableResult containing the current state if found.
	//
	// The returned UpdatableResult can be modified and passed to UpdateResult() to update the UI.
	//
	// Example - Toggle favorite state:
	//   Action: func(ctx context.Context, actionContext ActionContext) {
	//       // Get current result state
	//       updatableResult := api.GetUpdatableResult(ctx, actionContext.ResultId)
	//       if updatableResult == nil {
	//           return // Result no longer visible
	//       }
	//
	//       // Toggle favorite
	//       if isFavorite {
	//           removeFavorite()
	//           // Update action name and icon
	//           (*updatableResult.Actions)[actionIndex].Name = "Add to favorite"
	//           (*updatableResult.Actions)[actionIndex].Icon = AddToFavIcon
	//           // Remove favorite tail
	//           *updatableResult.Tails = removeMatchingTail(*updatableResult.Tails, favoriteTail)
	//       } else {
	//           addFavorite()
	//           // Update action name and icon
	//           (*updatableResult.Actions)[actionIndex].Name = "Remove from favorite"
	//           (*updatableResult.Actions)[actionIndex].Icon = RemoveFromFavIcon
	//           // Add favorite tail
	//           *updatableResult.Tails = append(*updatableResult.Tails, favoriteTail)
	//       }
	//
	//       // Update the result
	//       api.UpdateResult(ctx, *updatableResult)
	//   }
	GetUpdatableResult(ctx context.Context, resultId string) *UpdatableResult

	// UpdateResult updates a query result that is currently displayed in the UI.
	//
	// This method is designed for showing real-time progress updates during long-running operations,
	// such as file downloads, plugin installations, or API calls. It directly pushes updates to the UI
	// without polling, making it ideal for one-time or event-driven updates.
	//
	// Returns:
	//   - true: The result was successfully updated (still visible in the UI)
	//   - false: The result is no longer visible in the UI (caller should stop updating)
	//
	// When to use UpdateResult:
	//   - Progress updates during Action execution (e.g., "Downloading... 50%")
	//   - One-time status updates (e.g., "Installation complete")
	//   - Event-driven updates with clear start/end (e.g., file change notifications)
	//   - Periodic updates (e.g., CPU/memory monitoring) - start a timer in Init() and track result IDs
	//
	// Best practices:
	//   - Set PreventHideAfterAction: true in your action to keep the result visible
	//   - Only call this within Action handlers or background goroutines spawned by actions
	//   - Check the return value - if false, stop updating to avoid resource leaks
	//   - Only update fields that have changed (use nil for fields you don't want to update)
	//
	// Example:
	//   Action: func(ctx context.Context, actionContext ActionContext) {
	//       title := "Installing..."
	//       api.UpdateResult(ctx, UpdatableResult{Id: actionContext.ResultId, Title: &title})
	//
	//       go func() {
	//           title := "Downloading..."
	//           if !api.UpdateResult(ctx, UpdatableResult{Id: actionContext.ResultId, Title: &title}) {
	//               return // Result no longer visible, stop updating
	//           }
	//           // ... perform download ...
	//           title = "Installation complete"
	//           api.UpdateResult(ctx, UpdatableResult{Id: actionContext.ResultId, Title: &title})
	//       }()
	//   }
	UpdateResult(ctx context.Context, result UpdatableResult) bool

	// PushResults pushes additional query results to UI for the given query.
	// Returns true if results were accepted by UI (query still active), false otherwise.
	PushResults(ctx context.Context, query Query, results []QueryResult) bool

	// IsVisible returns whether the primary launcher is currently visible.
	// Intended for periodic result refresh (CPU/memory, timer, media) and
	// in-app notification routing — not for suppressing secondary flows.
	// Without a session id this is primary-only; secondary panels do not count.
	//
	// Example:
	//   func (p *Plugin) refreshData(ctx context.Context) {
	//       if !p.api.IsVisible(ctx) {
	//           return // Primary launcher is hidden, skip update
	//       }
	//       // ... update data ...
	//   }
	IsVisible(ctx context.Context) bool

	// RefreshQuery re-executes the current query with the existing query text.
	// This is useful when plugin data changes and you want to update the displayed results.
	//
	// Parameters:
	//   - ctx: Context
	//   - param: RefreshQueryParam to control refresh behavior
	//
	// Example - Refresh after marking item as favorite:
	//   Action: func(ctx context.Context, actionContext ActionContext) {
	//       markAsFavorite(item)
	//       // Refresh query and preserve user's current selection
	//       api.RefreshQuery(ctx, RefreshQueryParam{PreserveSelectedIndex: true})
	//   }
	//
	// Example - Refresh after deleting item:
	//   Action: func(ctx context.Context, actionContext ActionContext) {
	//       deleteItem(item)
	//       // Refresh query and reset to first item
	//       api.RefreshQuery(ctx, RefreshQueryParam{PreserveSelectedIndex: false})
	//   }
	//
	// Example - Refresh after moving the selected item:
	//   api.RefreshQuery(ctx, RefreshQueryParam{SelectedResultId: item.Id})
	// The rebuilt result must use the same QueryResult.Id. The highlight follows
	// that result when its index changes.
	RefreshQuery(ctx context.Context, param RefreshQueryParam)

	// RefreshGlance asks Wox UI to pull the latest Global Glance data for this plugin.
	// It deliberately does not push UI content so user slot settings remain authoritative.
	RefreshGlance(ctx context.Context, ids []string)

	// Copy copies the given content to the system clipboard.
	// Supports text, image, or both simultaneously.
	Copy(ctx context.Context, params CopyParams)

	// Screenshot captures a user-selected screen area and returns the saved image path.
	Screenshot(ctx context.Context, option ScreenshotOption) ScreenshotResult

	// GetCacheFolder returns this plugin's dedicated cache directory under
	// ~/.wox/cache/plugins/<plugin-id>/. The folder is created if it does not exist.
	// Use it for machine-local files such as downloads and search caches. Do not store
	// user settings here; those belong in GetSetting/SetSetting so they can sync.
	// Wox deletes the folder when the plugin is uninstalled.
	GetCacheFolder(ctx context.Context) string

	// GetThemeColors returns the current launcher palette as opaque #RRGGBB colors.
	// Use it for HTML/webview previews so plugin surfaces follow light and dark themes.
	// Requires Wox >= 2.4.5.
	GetThemeColors(ctx context.Context, option GetThemeColorsOption) GetThemeColorsResult
}

type CopyParams struct {
	Type     CopyType
	Text     string
	WoxImage *common.WoxImage
}

// ScreenshotOption controls optional screenshot behavior.
type ScreenshotOption struct {
	// HideAnnotationToolbar keeps plugin capture flows focused on raw image selection when callers,
	// such as OCR plugins, do not need Wox's markup tools. Cancel and confirm remain visible so the
	// user still has an explicit escape/finish path when AutoConfirm is not requested.
	HideAnnotationToolbar bool `json:"hideAnnotationToolbar"`
	// AutoConfirm completes the screenshot as soon as the user finishes drawing the selection.
	// The previous API always required a manual confirm click, which is unnecessary for callers that
	// only need the selected image path and do their own processing after capture.
	AutoConfirm bool `json:"autoConfirm"`
}

// ScreenshotResult reports the screenshot capture outcome and saved image path.
type ScreenshotResult struct {
	Success        bool
	ScreenshotPath string
	ErrMsg         string
}

// GetThemeColorsOption is reserved so later filters can be added without a new API.
// Requires Wox >= 2.4.5.
type GetThemeColorsOption struct{}

// GetThemeColorsResult is the opaque launcher palette for plugin-authored HTML.
// Requires Wox >= 2.4.5.
type GetThemeColorsResult struct {
	Background    string `json:"Background"`
	Text          string `json:"Text"`
	SecondaryText string `json:"SecondaryText"`
	Border        string `json:"Border"`
	Accent        string `json:"Accent"`
	AccentText    string `json:"AccentText"`
	Selection     string `json:"Selection"`
	Dark          bool   `json:"Dark"`
}

// SetSettingOption controls how a plugin setting is persisted.
// Available since Wox v2.4.0.
type SetSettingOption struct {
	Key              string `json:"key"`
	Value            string `json:"value"`
	PlatformSpecific bool   `json:"platformSpecific,omitempty"`
	IsLocal          bool   `json:"isLocal,omitempty"`
}

// SetSettingResult reports whether a plugin setting was persisted.
// Available since Wox v2.4.0.
type SetSettingResult struct {
	Success bool
	ErrMsg  string
}

// APIImpl is the concrete API implementation bound to one plugin instance.
type APIImpl struct {
	pluginInstance       *Instance
	toolCallStartTimeMap *util.HashMap[string, int64] // store the start time of tool calls
}

func (a *APIImpl) ChangeQuery(ctx context.Context, query common.PlainQuery) {
	if query.QueryType != QueryTypeInput && query.QueryType != QueryTypeSelection {
		util.GetLogger().Error(ctx, "ChangeQuery requires QueryType input or selection")
		return
	}

	query.QueryHint = query.QueryHint.NormalizeForQuery(query.QueryType, query.QueryText)
	if query.QueryHint != nil {
		query.QueryHint = query.QueryHint.Clone()
		for i := range query.QueryHint.Elements {
			query.QueryHint.Elements[i].Placeholder = common.I18nString(a.pluginInstance.TranslateMetadataText(ctx, query.QueryHint.Elements[i].Placeholder))
		}
	}
	GetPluginManager().GetUI().ChangeQuery(ctx, query)
}

func (a *APIImpl) HideApp(ctx context.Context) {
	GetPluginManager().GetUI().HideApp(ctx)
}

func (a *APIImpl) ShowApp(ctx context.Context) {
	GetPluginManager().GetUI().ShowApp(ctx, common.ShowContext{
		SelectAll: true,
	})
}

func (a *APIImpl) Notify(ctx context.Context, message string) {
	icon := a.pluginInstance.Metadata.Icon
	if parsedIcon, err := common.ParseWoxImage(icon); err == nil {
		convertedIcon := a.pluginInstance.ConvertIcon(ctx, parsedIcon)
		icon = convertedIcon.String()
	}

	GetPluginManager().GetUI().Notify(ctx, common.NotifyMsg{
		PluginId:       a.pluginInstance.Metadata.Id,
		Text:           a.GetTranslation(ctx, message),
		Icon:           icon,
		DisplaySeconds: 5,
	})
}

// PushAttention persists a plugin-owned item and refreshes the launcher unread badge.
func (a *APIImpl) PushAttention(ctx context.Context, request PushAttentionRequest) {
	if a.pluginInstance == nil {
		return
	}

	request.Title = a.GetTranslation(ctx, request.Title)
	if request.Description != "" {
		request.Description = a.GetTranslation(ctx, request.Description)
	}

	defaultIcon := a.pluginInstance.Metadata.GetIconOrDefault(a.pluginInstance.PluginDirectory, icons.Get(icons.BrandWox))
	_, err := GetAttentionManager().Push(ctx, AttentionPluginSource{
		PluginID:        a.pluginInstance.Metadata.Id,
		PluginDirectory: a.pluginInstance.PluginDirectory,
		DefaultIcon:     defaultIcon,
	}, request)
	if err != nil {
		a.Log(ctx, LogLevelWarning, fmt.Sprintf("failed to push attention item: %v", err))
		return
	}

	PublishAttentionUnreadCount(ctx)
}

// PublishAttentionUnreadCount pushes the current unread attention count to the UI.
func PublishAttentionUnreadCount(ctx context.Context) {
	count := 0
	if !IsAttentionPluginDisabled() {
		unread, err := GetAttentionManager().UnreadCount(ctx)
		if err != nil {
			util.GetLogger().Warn(ctx, fmt.Sprintf("failed to count unread attention items: %v", err))
			return
		}
		count = int(unread)
	}

	ui := GetPluginManager().GetUI()
	if ui == nil {
		return
	}
	ui.UpdateAttentionUnreadCount(ctx, count)
}

// Log writes plugin output into the shared Wox log tagged with the plugin name.
// Plugins used to get a private logger and log file each; that duplicated every
// line already written here and cost a goroutine, a pipe, and an open file per plugin.
func (a *APIImpl) Log(ctx context.Context, level LogLevel, msg string) {
	logCtx := util.WithComponentContext(ctx, a.pluginInstance.GetName(ctx))
	switch level {
	case LogLevelError:
		logger.Error(logCtx, msg)
	case LogLevelInfo:
		logger.Info(logCtx, msg)
	case LogLevelDebug:
		logger.Debug(logCtx, msg)
	case LogLevelWarning:
		logger.Warn(logCtx, msg)
	}
}

func (a *APIImpl) GetTranslation(ctx context.Context, key string) string {
	return a.pluginInstance.Metadata.translate(ctx, common.I18nString(key))
}

func (a *APIImpl) GetSetting(ctx context.Context, key string) string {
	// try to get platform specific setting first
	platformSpecificKey := key + "@" + util.GetCurrentPlatform()
	v, exist := a.pluginInstance.Setting.Get(platformSpecificKey)
	if exist {
		return v
	}

	v, exist = a.pluginInstance.Setting.Get(key)
	if exist {
		return v
	}
	return ""
}

// SaveSetting persists a synchronized plugin setting for compatibility with older plugins.
//
// Deprecated: use SetSetting.
func (a *APIImpl) SaveSetting(ctx context.Context, key string, value string, isPlatformSpecific bool) {
	result := a.SetSetting(ctx, SetSettingOption{
		Key:              key,
		Value:            value,
		PlatformSpecific: isPlatformSpecific,
	})
	if !result.Success {
		a.Log(ctx, LogLevelError, fmt.Sprintf("failed to save setting %q: %s", key, result.ErrMsg))
	}
}

// SetSetting persists a plugin setting according to the request's platform and local-only options.
func (a *APIImpl) SetSetting(ctx context.Context, option SetSettingOption) SetSettingResult {
	if option.Key == "" {
		return SetSettingResult{ErrMsg: "setting key cannot be empty"}
	}

	finalKey := option.Key
	if option.PlatformSpecific {
		finalKey = option.Key + "@" + util.GetCurrentPlatform()
	} else {
		// if not platform specific, remove platform specific setting, otherwise it will be loaded first
		platformSpecificKey := option.Key + "@" + util.GetCurrentPlatform()
		var deleteErr error
		if option.IsLocal {
			deleteErr = a.pluginInstance.Setting.DeleteLocal(platformSpecificKey)
		} else {
			deleteErr = a.pluginInstance.Setting.Delete(platformSpecificKey)
		}
		if deleteErr != nil {
			return SetSettingResult{ErrMsg: fmt.Sprintf("failed to remove platform-specific setting: %s", deleteErr)}
		}
	}

	existValue, exist := a.pluginInstance.Setting.Get(finalKey)
	var saveErr error
	if option.IsLocal {
		saveErr = a.pluginInstance.Setting.SetLocal(finalKey, option.Value)
	} else {
		saveErr = a.pluginInstance.Setting.Set(finalKey, option.Value)
	}
	if saveErr != nil {
		return SetSettingResult{ErrMsg: fmt.Sprintf("failed to persist setting: %s", saveErr)}
	}

	if !exist || existValue != option.Value {
		for _, callback := range a.pluginInstance.SettingChangeCallbacks {
			util.Go(ctx, "plugin setting change callback", func() {
				callback(ctx, option.Key, option.Value)
			})
		}
	}
	return SetSettingResult{Success: true}
}

func (a *APIImpl) OnSettingChanged(ctx context.Context, callback func(ctx context.Context, key string, value string)) {
	a.pluginInstance.SettingChangeCallbacks = append(a.pluginInstance.SettingChangeCallbacks, callback)
}

func (a *APIImpl) OnGetDynamicSetting(
	ctx context.Context,
	callback func(ctx context.Context, key string) definition.PluginSettingDefinitionItem,
) {
	a.pluginInstance.DynamicSettingCallbacks = append(a.pluginInstance.DynamicSettingCallbacks, callback)
}

func (a *APIImpl) OnDeepLink(ctx context.Context, callback func(ctx context.Context, arguments map[string]string)) {
	if !a.pluginInstance.Metadata.IsSupportFeature(MetadataFeatureDeepLink) {
		a.Log(ctx, LogLevelError, "plugin has no access to deep link feature")
		return
	}

	a.pluginInstance.DeepLinkCallbacks = append(a.pluginInstance.DeepLinkCallbacks, callback)
}

func (a *APIImpl) OnUnload(ctx context.Context, callback func(ctx context.Context)) {
	a.pluginInstance.UnloadCallbacks = append(a.pluginInstance.UnloadCallbacks, callback)
}

func (a *APIImpl) ShowToolbarMsg(ctx context.Context, msg ToolbarMsg) {
	GetPluginManager().ShowToolbarMsg(ctx, a.pluginInstance, msg)
}

func (a *APIImpl) ClearToolbarMsg(ctx context.Context, toolbarMsgId string) {
	GetPluginManager().ClearToolbarMsg(ctx, a.pluginInstance, toolbarMsgId)
}

func (a *APIImpl) OnEnterPluginQuery(ctx context.Context, callback func(ctx context.Context)) {
	a.pluginInstance.EnterPluginQueryCallbacks = append(a.pluginInstance.EnterPluginQueryCallbacks, callback)
}

func (a *APIImpl) OnLeavePluginQuery(ctx context.Context, callback func(ctx context.Context)) {
	a.pluginInstance.LeavePluginQueryCallbacks = append(a.pluginInstance.LeavePluginQueryCallbacks, callback)
}

func (a *APIImpl) OnDragOut(ctx context.Context, option DragOutListenOption) DragOutListenResult {
	if option.Callback == nil {
		return DragOutListenResult{Success: false}
	}
	a.pluginInstance.DragOutCallbacks = append(a.pluginInstance.DragOutCallbacks, option.Callback)
	return DragOutListenResult{Success: true}
}

func (a *APIImpl) RegisterQueryCommands(ctx context.Context, commands []MetadataCommand) {
	a.pluginInstance.RuntimeQueryCommands = append([]MetadataCommand(nil), commands...)
}

func (a *APIImpl) RegisterTriggerKeyword(ctx context.Context, option RegisterTriggerKeywordOption) RegisterTriggerKeywordResult {
	return RegisterTriggerKeywordResult{Success: GetPluginManager().registerTriggerKeyword(a.pluginInstance, option)}
}

func (a *APIImpl) UnregisterTriggerKeyword(ctx context.Context, option UnregisterTriggerKeywordOption) UnregisterTriggerKeywordResult {
	a.pluginInstance.unregisterTriggerKeyword(option.Keyword)
	return UnregisterTriggerKeywordResult{Success: true}
}

func (a *APIImpl) OnMRURestore(ctx context.Context, callback func(ctx context.Context, mruData MRUData) (*QueryResult, error)) {
	if !a.pluginInstance.Metadata.IsSupportFeature(MetadataFeatureMRU) {
		a.Log(ctx, LogLevelError, "plugin has no access to MRU feature")
		return
	}

	a.pluginInstance.MRURestoreCallbacks = append(a.pluginInstance.MRURestoreCallbacks, callback)
}

func (a *APIImpl) UpdateResult(ctx context.Context, result UpdatableResult) bool {
	// Preserve an explicit query scope: stable result ids can occur in several
	// cached queries, and id-only lookup can select an obsolete action snapshot.
	if util.GetContextSessionId(ctx) == "" || util.GetContextQueryId(ctx) == "" {
		if sessionId, queryId := GetPluginManager().GetQueryInfoByResultId(result.Id); sessionId != "" {
			ctx = util.WithQueryIdContext(util.WithSessionContext(ctx, sessionId), queryId)
		}
	}
	if util.GetContextSessionId(ctx) != "" && util.GetContextQueryId(ctx) != "" {
		if _, found := GetPluginManager().findResultCacheByIdWithContext(ctx, result.Id); !found {
			return false
		}
	}
	polishedResult := GetPluginManager().PolishUpdatableResult(ctx, a.pluginInstance, result)
	success := GetPluginManager().GetUI().UpdateResult(ctx, polishedResult)
	return success
}

func (a *APIImpl) PushResults(ctx context.Context, query Query, results []QueryResult) bool {
	if query.Id == "" {
		a.Log(ctx, LogLevelWarning, "PushResults ignored: query id is empty")
		return false
	}
	if query.SessionId == "" {
		a.Log(ctx, LogLevelWarning, "PushResults ignored: session id is empty")
		return false
	}
	if util.GetContextSessionId(ctx) == "" {
		ctx = util.WithQueryIdContext(util.WithSessionContext(ctx, query.SessionId), query.Id)
	}

	// Bug fix: core no longer owns "current query" state because backend query
	// pipelines are concurrent. Push by query id and let UI accept or reject
	// the payload against the visible query, matching normal Query responses.
	layout := GetPluginManager().getCachedLayoutForPluginQuery(ctx, a.pluginInstance, query)
	for i := range results {
		results[i] = GetPluginManager().PolishResult(ctx, a.pluginInstance, query, layout, results[i])
	}

	polishedResults := GetPluginManager().BuildQueryResultsSnapshot(query.SessionId, query.Id)
	payload := PushResultsPayload{
		QueryId: query.Id,
		Results: polishedResults,
	}
	return GetPluginManager().GetUI().PushResults(ctx, payload)
}

func (a *APIImpl) GetUpdatableResult(ctx context.Context, resultId string) *UpdatableResult {
	return GetPluginManager().GetUpdatableResult(ctx, resultId)
}

func (a *APIImpl) IsVisible(ctx context.Context) bool {
	return GetPluginManager().GetUI().IsVisible(ctx)
}

func (a *APIImpl) RefreshQuery(ctx context.Context, param RefreshQueryParam) {
	GetPluginManager().GetUI().RefreshQuery(ctx, common.RefreshQueryOptions{
		PreserveSelectedIndex: param.PreserveSelectedIndex,
		SelectedResultId:      param.SelectedResultId,
	})
}

func (a *APIImpl) RefreshGlance(ctx context.Context, ids []string) {
	if a.pluginInstance == nil {
		return
	}
	GetPluginManager().GetUI().RefreshGlance(ctx, a.pluginInstance.Metadata.Id, ids)
}

func (a *APIImpl) Copy(ctx context.Context, params CopyParams) {
	if params.Type == CopyTypePlainText {
		err := clipboard.WriteText(params.Text)
		if err != nil {
			a.Log(ctx, LogLevelError, fmt.Sprintf("failed to copy text to clipboard: %v", err))
		}
		return
	}

	if params.Type == CopyTypeImage {
		if params.WoxImage.IsAnimatedGif() {
			path, cleanup, err := params.WoxImage.ResolveAnimatedGIFPath(ctx)
			if err != nil {
				a.Log(ctx, LogLevelError, fmt.Sprintf("failed to resolve animated gif for clipboard: %v", err))
				return
			}
			defer cleanup()
			if err := clipboard.WriteAnimatedGIF(path); err != nil {
				a.Log(ctx, LogLevelError, fmt.Sprintf("failed to copy animated gif to clipboard: %v", err))
			}
			return
		}

		img, err := params.WoxImage.ToImage()
		if err != nil {
			a.Log(ctx, LogLevelError, fmt.Sprintf("failed to convert woximage to image: %v", err))
			return
		}
		err = clipboard.Write(&clipboard.ImageData{
			Image: img,
		})
		if err != nil {
			a.Log(ctx, LogLevelError, fmt.Sprintf("failed to copy image to clipboard: %v", err))
		}
		return
	}
}

func (a *APIImpl) Screenshot(ctx context.Context, option ScreenshotOption) ScreenshotResult {
	request := common.DefaultCaptureScreenshotRequest()
	// Plugin screenshots return a saved file path and leave clipboard handling to the caller.
	request.Output = "file"
	// Screenshot API options are translated in core where the plugin caller is known. Keeping UI
	// on a request-only contract avoids making the UI infer SDK defaults from plugin runtime details.
	request.HideAnnotationToolbar = option.HideAnnotationToolbar
	request.AutoConfirm = option.AutoConfirm
	if !a.pluginInstance.IsSystemPlugin {
		// Third-party screenshot callers need a visible identity marker in the floating toolbox.
		// The UI cannot reliably infer the plugin from the generic CaptureScreenshot method,
		// so core resolves the metadata icon here and sends only the render-ready WoxImage.
		callerIcon := a.pluginInstance.Metadata.GetIconOrDefault(a.pluginInstance.PluginDirectory, icons.Get(icons.BrandWox))
		request.CallerIcon = &callerIcon
	}

	result, err := GetPluginManager().GetUI().CaptureScreenshot(ctx, request)
	if err != nil {
		return ScreenshotResult{
			Success: false,
			ErrMsg:  err.Error(),
		}
	}

	switch result.Status {
	case common.CaptureScreenshotStatusCompleted:
		if result.ScreenshotPath == "" {
			return ScreenshotResult{
				Success: false,
				ErrMsg:  "screenshot completed without an export path",
			}
		}

		return ScreenshotResult{
			Success:        true,
			ScreenshotPath: result.ScreenshotPath,
			ErrMsg:         result.ClipboardWarningMessage,
		}
	case common.CaptureScreenshotStatusCancelled:
		return ScreenshotResult{
			Success: false,
			ErrMsg:  "cancelled",
		}
	case common.CaptureScreenshotStatusFailed:
		errMsg := result.ErrorMessage
		if errMsg == "" {
			errMsg = "screenshot session failed"
		}
		return ScreenshotResult{
			Success: false,
			ErrMsg:  errMsg,
		}
	default:
		return ScreenshotResult{
			Success: false,
			ErrMsg:  fmt.Sprintf("unexpected screenshot status: %s", result.Status),
		}
	}
}

// GetThemeColors maps the active Wox theme to opaque HTML-safe colors.
func (a *APIImpl) GetThemeColors(ctx context.Context, _ GetThemeColorsOption) GetThemeColorsResult {
	ui := GetPluginManager().GetUI()
	if ui == nil {
		return themeColorsResult(common.Theme{}.PluginColors())
	}
	theme := ui.GetCurrentTheme(ctx)
	result := themeColorsResult(theme.PluginColors())
	a.Log(ctx, LogLevelDebug, fmt.Sprintf("theme colors themeId=%s dark=%t background=%s text=%s", theme.ThemeId, result.Dark, result.Background, result.Text))
	return result
}

func themeColorsResult(colors common.ThemePluginColors) GetThemeColorsResult {
	return GetThemeColorsResult{
		Background:    colors.Background,
		Text:          colors.Text,
		SecondaryText: colors.SecondaryText,
		Border:        colors.Border,
		Accent:        colors.Accent,
		AccentText:    colors.AccentText,
		Selection:     colors.Selection,
		Dark:          colors.Dark,
	}
}

// GetCacheFolder returns this plugin's cache directory, creating it if needed.
func (a *APIImpl) GetCacheFolder(ctx context.Context) string {
	if a.pluginInstance == nil {
		return ""
	}

	folder, err := util.GetLocation().EnsurePluginCacheDirectory(a.pluginInstance.Metadata.Id)
	if err != nil {
		a.Log(ctx, LogLevelError, fmt.Sprintf("failed to create plugin cache folder: %v", err))
		return ""
	}
	return folder
}

func NewAPI(instance *Instance) API {
	return &APIImpl{
		pluginInstance:       instance,
		toolCallStartTimeMap: util.NewHashMap[string, int64](),
	}
}

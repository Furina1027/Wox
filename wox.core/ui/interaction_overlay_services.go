package ui

import (
	"context"
	"errors"
	"fmt"
	"image"
	"net/url"
	"strings"

	"wox/common"
	"wox/plugin"
	"wox/ui/contract"
	"wox/util"
	"wox/util/emojiimage"
	"wox/util/overlay"
	"wox/util/overlay/imageoverlay"
	"wox/util/tooltip"
	"wox/util/websiteicon"

	"github.com/disintegration/imaging"
)

// ShowTooltip presents one native tooltip anchored in screen coordinates.
func (s *CoreServices) ShowTooltip(ctx context.Context, sessionID string, options contract.TooltipOptions) error {
	options.Name = strings.TrimSpace(options.Name)
	options.Text = strings.TrimSpace(options.Text)
	if options.Name == "" || options.Text == "" {
		return errors.New("tooltip name and text are required")
	}
	tooltip.Show(uiServiceContext(ctx, sessionID), tooltip.Options{
		Name: options.Name, Text: options.Text, Side: options.Side, HotkeyLabels: options.HotkeyLabels,
		AnchorX: options.AnchorX, AnchorY: options.AnchorY, AnchorWidth: options.AnchorWidth, AnchorHeight: options.AnchorHeight,
		OwnerX: options.OwnerX, OwnerY: options.OwnerY, OwnerWidth: options.OwnerWidth, OwnerHeight: options.OwnerHeight,
		IgnoreOwnerLeave: options.IgnoreOwnerLeave,
	})
	return nil
}

// HideTooltip closes one native tooltip by name.
func (s *CoreServices) HideTooltip(_ context.Context, _ string, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("tooltip name is required")
	}
	tooltip.Close(name)
	return nil
}

// GlanceItems refreshes selected plugin glance items.
func (s *CoreServices) GlanceItems(ctx context.Context, sessionID string, keys []plugin.GlanceKey, reason plugin.GlanceRefreshReason) ([]plugin.GlanceItemUI, error) {
	return plugin.GetPluginManager().GetGlanceItems(uiServiceContext(ctx, sessionID), keys, reason), nil
}

// ExecuteGlanceAction executes one selected plugin glance action.
func (s *CoreServices) ExecuteGlanceAction(ctx context.Context, sessionID string, pluginID string, glanceID string, actionID string) error {
	if pluginID == "" || glanceID == "" || actionID == "" {
		return errors.New("pluginId, glanceId and actionId are required")
	}
	return plugin.GetPluginManager().ExecuteGlanceAction(uiServiceContext(ctx, sessionID), pluginID, glanceID, actionID)
}

// LoadLazyResultImage resolves one manager-issued lazy result image token.
func (s *CoreServices) LoadLazyResultImage(ctx context.Context, sessionID string, token string) (common.WoxImage, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return common.WoxImage{}, errors.New("token is empty")
	}
	return plugin.GetPluginManager().LoadLazyResultIcon(uiServiceContext(ctx, sessionID), token)
}

// ResolveImage converts core-owned URL, emoji, and file-icon sources into raster payloads.
func (s *CoreServices) ResolveImage(ctx context.Context, sessionID string, source common.WoxImage, size int) (common.WoxImage, error) {
	ctx = uiServiceContext(ctx, sessionID)
	if source.IsEmpty() {
		return common.WoxImage{}, errors.New("image is empty")
	}
	if size <= 0 {
		size = 128
	}
	size = min(max(size, 16), 2048)
	if source.ImageType == common.WoxImageTypeFileIcon {
		resolved := common.ConvertFileIconToAbsolutePathWithSize(ctx, source, size)
		if resolved.ImageType == common.WoxImageTypeFileIcon || resolved.IsEmpty() {
			return common.WoxImage{}, errors.New("failed to resolve file icon")
		}
		return resolved, nil
	}
	if source.ImageType != common.WoxImageTypeUrl && source.ImageType != common.WoxImageTypeEmoji {
		return common.WoxImage{}, fmt.Errorf("image type %s does not require core resolution", source.ImageType)
	}
	if source.ImageType == common.WoxImageTypeUrl {
		// Keep remote GIFs as cached files so UI can play every frame. Rasterizing
		// here would collapse them to a PNG of the first frame.
		converted := common.ConvertIconWithSize(ctx, source, "", size)
		if converted.IsAnimatedGif() && converted.ImageType == common.WoxImageTypeAbsolutePath && !converted.IsEmpty() {
			return converted, nil
		}
	}
	var decoded image.Image
	var err error
	if source.ImageType == common.WoxImageTypeEmoji {
		// Flutter renders emoji with the platform color font. Reuse that font here so glyph coverage and visual metrics match without a network fetch.
		decoded, err = emojiimage.Render(source.ImageData, size)
	}
	if decoded == nil {
		decoded, err = source.ToImageWithContext(ctx)
	}
	if err != nil {
		return common.WoxImage{}, err
	}
	if decoded.Bounds().Dx() > size || decoded.Bounds().Dy() > size {
		decoded = imaging.Fit(decoded, size, size, imaging.Lanczos)
	}
	return common.NewWoxImage(decoded)
}

// ResultPreview resolves one deferred plugin preview.
func (s *CoreServices) ResultPreview(ctx context.Context, sessionID string, querySessionID string, queryID string, resultID string) (plugin.WoxPreview, error) {
	if querySessionID == "" || queryID == "" || resultID == "" {
		return plugin.WoxPreview{}, errors.New("sessionId, queryId and id are required")
	}
	return plugin.GetPluginManager().GetResultPreview(uiServiceContext(ctx, sessionID), querySessionID, queryID, resultID)
}

// ShowPreviewImage opens one full-size image in the native overlay.
func (s *CoreServices) ShowPreviewImage(ctx context.Context, sessionID string, image common.WoxImage) error {
	if image.IsEmpty() {
		return errors.New("preview image is empty")
	}
	return imageoverlay.Show(uiServiceContext(ctx, sessionID), imageoverlay.Options{
		Image: image, FitToScreen: true, Topmost: true, Movable: true, CloseOnEscape: true, Anchor: overlay.AnchorCenter,
	})
}

// FetchWebsiteIcon embeds a direct image URL or website favicon so saved settings survive cache cleanup and sync.
func (s *CoreServices) FetchWebsiteIcon(ctx context.Context, sessionID string, websiteURL string) (common.WoxImage, error) {
	parsed, err := url.Parse(websiteURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil {
		return common.WoxImage{}, errors.New("invalid URL")
	}
	ctx = uiServiceContext(ctx, sessionID)
	icon, err := websiteicon.FetchDirectImage(ctx, websiteURL)
	if err == nil {
		return icon, nil
	}
	util.GetLogger().Debug(ctx, "fetch settings icon: direct image failed: "+err.Error())
	if !errors.Is(err, websiteicon.ErrNotAnImage) && websiteicon.IsDirectImageURL(websiteURL) {
		return common.WoxImage{}, err
	}
	icon, err = websiteicon.Fetch(ctx, websiteURL)
	if err != nil {
		return common.WoxImage{}, err
	}
	decoded, err := icon.ToImageWithContext(ctx)
	if err != nil {
		return common.WoxImage{}, err
	}
	return common.NewWoxImage(decoded)
}

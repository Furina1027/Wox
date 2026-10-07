package preview

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"testing"

	woxcomponent "wox/ui/launcher/component"
	woxui "wox/ui/runtime"
	woxwidget "wox/ui/widget"
)

func TestPreviewLoadingCentersSharedIndicator(t *testing.T) {
	color := woxui.Color{R: 12, G: 18, B: 24, A: 255}
	loading := PreviewLoading(320, 180, color).(woxwidget.Align)
	if loading.Width != 320 || loading.Height != 180 || loading.Horizontal != 0.5 || loading.Vertical != 0.5 {
		t.Fatalf("preview loading align = %#v, want a centered placeholder", loading)
	}
	if indicator := loading.Child.(woxwidget.LoopAnimation); indicator.Key != "wox-loading-indicator" {
		t.Fatalf("preview loading child = %#v, want WoxLoadingIndicator", indicator)
	}
}

func TestPreviewImageOmitsOverlayGestureWithoutOnTap(t *testing.T) {
	view := builtPreviewImage(PreviewImageProps{Width: 200, Height: 100, Image: &woxui.Image{Width: 10, Height: 20}})
	if view.OnTap != nil {
		t.Fatal("preview image should not open an overlay when OnTap is unset")
	}
	if view.OnPointer == nil {
		t.Fatal("preview image should consume wheel zoom")
	}
}

func TestPreviewImageWrapsOverlayGestureWithOnTap(t *testing.T) {
	tapped := false
	view := builtPreviewImage(PreviewImageProps{
		Width: 200, Height: 100, Image: &woxui.Image{Width: 10, Height: 10}, OnTap: func() { tapped = true },
	})
	if view.ID != "preview-image-overlay" || view.OnTap == nil {
		t.Fatalf("preview image gesture = %+v, want overlay tap", view)
	}
	view.OnTap()
	if !tapped {
		t.Fatal("preview image tap did not fire")
	}
}

func TestPreviewImageWrapsAnimatedGIF(t *testing.T) {
	animated := decodePreviewTestGIF(t)
	view := PreviewImage(PreviewImageProps{Width: 200, Height: 100, Image: animated}).(woxwidget.Stateful)
	child := view.CreateState().Build(woxwidget.StateContext{}, PreviewImageProps{Width: 200, Height: 100, Image: animated})
	if _, ok := child.(woxwidget.FrameAnimation); !ok {
		t.Fatalf("preview image child = %T, want FrameAnimation", child)
	}
}

func decodePreviewTestGIF(t *testing.T) *woxui.Image {
	t.Helper()
	red := image.NewPaletted(image.Rect(0, 0, 8, 8), color.Palette{color.RGBA{}, color.RGBA{R: 255, A: 255}})
	blue := image.NewPaletted(image.Rect(0, 0, 8, 8), color.Palette{color.RGBA{}, color.RGBA{B: 255, A: 255}})
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			red.Set(x, y, color.RGBA{R: 255, A: 255})
			blue.Set(x, y, color.RGBA{B: 255, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := gif.EncodeAll(&encoded, &gif.GIF{
		Image:     []*image.Paletted{red, blue},
		Delay:     []int{10, 10},
		LoopCount: 0,
		Config:    image.Config{ColorModel: red.Palette, Width: 8, Height: 8},
	}); err != nil {
		t.Fatalf("encode gif: %v", err)
	}
	decoded, err := woxui.DecodeImage(bytes.NewReader(encoded.Bytes()))
	if err != nil || !decoded.IsAnimated() {
		t.Fatalf("decode gif: animated=%t err=%v", decoded != nil && decoded.IsAnimated(), err)
	}
	return decoded
}

func builtPreviewImage(props PreviewImageProps) woxwidget.Gesture {
	view := PreviewImage(props).(woxwidget.Stateful)
	return view.CreateState().Build(woxwidget.StateContext{}, props).(woxwidget.Gesture)
}

func TestPreviewSurfaceUsesFlutterTranslucentFill(t *testing.T) {
	theme := woxcomponent.Theme{
		Background:   woxui.Color{R: 12, G: 18, B: 24, A: 180},
		PreviewText:  woxui.Color{R: 220, G: 230, B: 240, A: 64},
		PreviewSplit: woxui.Color{R: 100, G: 110, B: 120, A: 32},
	}
	surface := previewSurface(woxwidget.Container{}, theme, 320, 180).(woxwidget.Container)

	if surface.Radius != previewSurfaceRadius || surface.BorderWidth != previewSurfaceBorderWidth || surface.Padding != woxwidget.UniformInsets(previewSurfaceBorderWidth) {
		t.Fatalf("preview shell = radius %v border %v padding %#v, want concentric %v/%v inset", surface.Radius, surface.BorderWidth, surface.Padding, previewSurfaceRadius, previewSurfaceBorderWidth)
	}
	if surface.BorderColor.A != 115 || surface.BorderWidth != 1 {
		t.Fatalf("preview border = %#v at %v, want Flutter 0.45 alpha 1px stroke", surface.BorderColor, surface.BorderWidth)
	}
	if surface.Color != (woxui.Color{R: 220, G: 230, B: 240, A: 9}) {
		t.Fatalf("preview fill = %#v, want preview text color at Flutter 0.035 alpha", surface.Color)
	}
	if _, nestedFill := surface.Child.(woxwidget.Container); nestedFill {
		t.Fatal("preview uses a nested fill to simulate its border")
	}
}

func TestPreviewTagsScrollHorizontallyWhenOverflowing(t *testing.T) {
	tags := make([]PreviewTag, 8)
	for index := range tags {
		tags[index] = PreviewTag{Label: "tag"}
	}
	view := PreviewTags(tags, woxcomponent.Theme{}, &woxui.Window{}, 120, nil).(woxwidget.ScrollView)
	if view.Key != "preview-tags" || !view.Horizontal || !view.MapVerticalWheel {
		t.Fatalf("preview tags = key %q horizontal %v map-wheel %v, want a retained horizontal strip that accepts a vertical wheel", view.Key, view.Horizontal, view.MapVerticalWheel)
	}
	if view.Width != 120 || view.ContentWidth <= view.Width {
		t.Fatalf("preview tags geometry = viewport %.0f content %.0f, want overflowing content", view.Width, view.ContentWidth)
	}
}

func TestPreviewTagHoverUsesTooltip(t *testing.T) {
	var hovered bool
	var tooltip string
	var anchor woxui.Rect
	view := PreviewTags([]PreviewTag{{Label: "51 chars", Tooltip: "Character count"}}, woxcomponent.Theme{}, &woxui.Window{}, 300, func(inside bool, text string, bounds woxui.Rect) {
		hovered, tooltip, anchor = inside, text, bounds
	}).(woxwidget.ScrollView)
	row := view.Child.(woxwidget.Flex)
	wrapper := row.Children[0].(woxwidget.Container)
	semantics := wrapper.Child.(woxwidget.Semantics)
	if semantics.AutomationID != "preview-tag-0" || semantics.Role != woxui.AccessibilityRoleText || semantics.Label != "51 chars" || semantics.Description != "Character count" {
		t.Fatalf("preview tag semantics = %+v, want a stable hoverable tooltip target", semantics)
	}
	gesture := semantics.Child.(woxwidget.Gesture)
	wantAnchor := woxui.Rect{X: 2, Y: 3, Width: 40, Height: 26}
	gesture.OnHoverAt(true, wantAnchor)

	if !hovered || tooltip != "Character count" || anchor != wantAnchor {
		t.Fatalf("hover = %v, %q, %#v; want tooltip and anchor", hovered, tooltip, anchor)
	}
}

func TestTerminalPreviewUsesFramelessFlutterSurface(t *testing.T) {
	theme := woxcomponent.Theme{PreviewText: woxui.Color{R: 220, G: 230, B: 240, A: 255}, PreviewSplit: woxui.Color{R: 100, G: 110, B: 120, A: 255}}
	tooltip := ""
	view := TerminalPreviewView(TerminalPreviewProps{
		Width: 500, Height: 300, SessionID: "test", Command: "ping example.com", SearchOpen: true, Theme: theme,
		SearchHotkey: "Cmd+Shift+F", FullscreenHotkey: "Cmd+B", OnTagHover: func(_ bool, text string, _ woxui.Rect) { tooltip = text },
	}).(woxwidget.Container)
	if view.BorderWidth != 0 || view.Color.A != 0 || view.Padding.Left != 10 || view.Padding.Top != 10 || view.Padding.Right != 12 {
		t.Fatalf("terminal outer surface = border %.0f fill %#v padding %#v; want Flutter frameless padding", view.BorderWidth, view.Color, view.Padding)
	}
	stack := view.Child.(woxwidget.Stack)
	if stack.Width != 478 {
		t.Fatalf("terminal content width = %.0f, want 478 after outer padding", stack.Width)
	}
	content := stack.Children[0].Child.(woxwidget.Flex)
	header := content.Children[0].(woxwidget.Container).Child.(woxwidget.Container)
	if header.BorderWidth != 1 || header.Color.A == 0 {
		t.Fatalf("terminal header = border %.0f fill %#v; want framed status bar", header.BorderWidth, header.Color)
	}
	headerStack := header.Child.(woxwidget.Stack)
	command := headerStack.Children[1]
	if command.Left != 17 || command.Right != 79 || !command.StretchWidth {
		t.Fatalf("terminal command layout = left/right %.0f/%.0f stretch %v, want 17/79/true", command.Left, command.Right, command.StretchWidth)
	}
	commandAlign := command.Child.(woxwidget.Align)
	if command.Top != 0 || commandAlign.Height != 34 || commandAlign.Vertical != 0.5 {
		t.Fatalf("terminal command alignment = top %.0f child %#v, want full-height vertical center", command.Top, command.Child)
	}
	findAlign := headerStack.Children[2].Child.(woxwidget.Align)
	find := findAlign.Child.(woxwidget.Stateful).Widget.(woxcomponent.IconButtonProps)
	if !headerStack.Children[2].AnchorRight || headerStack.Children[2].Right != 34 || !headerStack.Children[3].AnchorRight {
		t.Fatalf("terminal action anchors = find %v/%.0f fullscreen %v, want true/34/true", headerStack.Children[2].AnchorRight, headerStack.Children[2].Right, headerStack.Children[3].AnchorRight)
	}
	if findAlign.Height != 34 || findAlign.Vertical != 0.5 {
		t.Fatalf("terminal find alignment = %#v, want full-height vertical center", findAlign)
	}
	if find.Label != "Find" || find.Icon == nil || find.OnHoverAt == nil {
		t.Fatalf("terminal find action = %+v; want shared icon button", find)
	}
	find.OnHoverAt(true, woxui.Rect{})
	if tooltip != "Cmd+Shift+F" {
		t.Fatalf("terminal find tooltip = %q, want Cmd+Shift+F", tooltip)
	}
	fullscreen := headerStack.Children[3].Child.(woxwidget.Align).Child.(woxwidget.Stateful).Widget.(woxcomponent.IconButtonProps)
	if fullscreen.Label != "Toggle fullscreen" || fullscreen.Icon == nil || fullscreen.OnHoverAt == nil {
		t.Fatalf("terminal fullscreen action = %+v; want shared icon button", fullscreen)
	}
	fullscreen.OnHoverAt(true, woxui.Rect{})
	if tooltip != "Cmd+B" {
		t.Fatalf("terminal fullscreen tooltip = %q, want Cmd+B", tooltip)
	}
	search := content.Children[1].(woxwidget.Container).Child.(woxwidget.Container).Child.(woxwidget.Flex)
	if _, ok := search.Children[0].(woxwidget.Stateful).Widget.(woxcomponent.TextFieldProps); !ok {
		t.Fatal("terminal find input does not reuse WoxTextField")
	}
	count := search.Children[1].(woxwidget.Semantics)
	if count.AutomationID != "launcher.preview.terminal.search.match-count" || count.Value != "0/0" {
		t.Fatalf("terminal match count semantics = %+v, want stable 0/0 state", count)
	}
	body := content.Children[2].(woxwidget.Container)
	if body.BorderWidth != 0 || body.Color.A != 0 {
		t.Fatalf("terminal output surface = border %.0f fill %#v; want transparent output", body.BorderWidth, body.Color)
	}
}

func TestTerminalHighlightSegmentsFollowWrappedLines(t *testing.T) {
	segments := terminalHighlightSegments("first ms second ms", []string{"first ms", "second ms"}, []TerminalMatch{{Start: 6, End: 8}, {Start: 16, End: 18}})
	if len(segments) != 2 || segments[0] != (terminalHighlightSegment{line: 0, start: 6, end: 8, matchIndex: 0}) || segments[1] != (terminalHighlightSegment{line: 1, start: 7, end: 9, matchIndex: 1}) {
		t.Fatalf("terminal highlight segments = %+v, want both wrapped matches", segments)
	}
}

// TestPreviewTagsFitScaledFooter checks that rounded density sizes share the same reserved space.
func TestPreviewTagsFitScaledFooter(t *testing.T) {
	for _, scale := range []float32{1, 1.1, 1.25, 1.5, 2} {
		theme := woxcomponent.Theme{Controls: woxcomponent.ControlTheme{DensityScale: scale}}
		panel := PreviewView(PreviewProps{Width: 320, Height: 240, Theme: theme, Tags: []PreviewTag{{Label: "OCR"}}}).(woxwidget.Container)
		stack := panel.Child.(woxwidget.Stack)
		tags := stack.Children[1]
		strip := tags.Child.(woxwidget.ScrollView)
		if tags.Top+strip.Height != stack.Height {
			t.Fatalf("density %v: footer bottom %v, available %v", scale, tags.Top+strip.Height, stack.Height)
		}
	}
}

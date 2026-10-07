package launcher

import (
	"fmt"
	"testing"

	woxui "wox/ui/runtime"
)

// TestLauncherBoundsKeepsNewerPreview reproduces a result arriving before the hotkey caller resumes.
// TestGridBoundsIgnoreHiddenPreview checks the same resize path used after query results arrive.
func TestGridBoundsIgnoreHiddenPreview(t *testing.T) {
	err := woxui.Run(func() error {
		window, err := woxui.Open(woxui.WindowOptions{Title: "Wox grid bounds test", Size: woxui.Size{Width: 800, Height: 599}})
		if err != nil {
			return err
		}
		defer window.Close()
		app := &App{
			window: window, editor: woxui.NewTextEditor("drop "), palette: defaultPalette(),
			layout:  queryLayout{GridLayout: &gridLayout{Columns: 5, ShowTitle: true}},
			results: []queryResult{{ID: "file"}},
		}
		if err := app.applyWindowBounds(); err != nil {
			return err
		}
		withoutPreview, err := window.Bounds()
		if err != nil {
			return err
		}
		app.results[0].Preview = queryPreview{PreviewType: "file", PreviewData: "file.exe"}
		if err := app.applyWindowBounds(); err != nil {
			return err
		}
		withHiddenPreview, err := window.Bounds()
		if err != nil {
			return err
		}
		if !launcherBoundsEffectivelyEqual(withoutPreview, withHiddenPreview) {
			return fmt.Errorf("hidden grid preview expanded window: without=%+v with=%+v", withoutPreview, withHiddenPreview)
		}
		app.layout = queryLayout{}
		if err := app.applyWindowBounds(); err != nil {
			return err
		}
		withVisiblePreview, err := window.Bounds()
		if err != nil {
			return err
		}
		if withVisiblePreview.Height <= withHiddenPreview.Height {
			return fmt.Errorf("visible list preview lost its height: grid=%+v list=%+v", withHiddenPreview, withVisiblePreview)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

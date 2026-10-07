package launcher

import "testing"

func TestMeasureFormLabelWidthUsesSurfaceMinimumWithoutWindow(t *testing.T) {
	app := &App{}
	if got := app.measureFormLabelWidth(nil, nil, 60, 0); got != 60 {
		t.Fatalf("form label width = %.0f, want action form minimum 60", got)
	}
}

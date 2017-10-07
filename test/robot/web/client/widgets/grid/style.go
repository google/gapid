// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package grid

import "github.com/google/gapid/test/robot/web/client/dom"

// Icons holds the characters to use to draw the icons using the icons font.
type Icons struct {
	Succeeded rune
	Failed    rune
	Unknown   rune
}

// Style holds parameters to style the grid.
type Style struct {
	GridPadding                     float64   // Padding in pixels from the top-left of the canvas.
	CellSize                        float64   // Width and height in pixels of each cell.
	CellShadowColor                 dom.Color // The color of the shadow of a raised cell / header.
	HeaderFont                      dom.Font  // The header font.
	HeaderFontColor                 dom.Color // The header font color.
	GridLineColor                   dom.Color // The line color for the grid.
	GridLineWidth                   float64   // The line width for the grid.
	BackgroundColor                 dom.Color // The regular background color of cells and headers.
	CurrentSucceededBackgroundColor dom.Color // The background color used for tasks that have succeeded and are current.
	CurrentSucceededForegroundColor dom.Color // The foreground color used for tasks that have succeeded and are current.
	StaleSucceededBackgroundColor   dom.Color // The background color used for tasks that have succeeded and are stale.
	StaleSucceededForegroundColor   dom.Color // The foreground color used for tasks that have succeeded and are stale.
	CurrentFailedBackgroundColor    dom.Color // The background color used for tasks that have failed and are current.
	CurrentFailedForegroundColor    dom.Color // The foreground color used for tasks that have failed and are current.
	StaleFailedBackgroundColor      dom.Color // The background color used for tasks that have failed and are stale.
	StaleFailedForegroundColor      dom.Color // The foreground color used for tasks that have failed and are stale.
	InProgressForegroundColor       dom.Color // The foreground color used for tasks that last failed and are currently in progress.
	RegressedForegroundColor        dom.Color // The foreground color used for tasks that last succeeded and now are failing.
	FixedForegroundColor            dom.Color // The foreground color used for tasks that last failed and now are succeeding.
	UnknownBackgroundColor          dom.Color // The background color used for tasks that are in an unknown state.
	UnknownForegroundColor          dom.Color // The foreground color used for tasks that are in an unknown state.
	StaleUnknownForegroundColor     dom.Color // The foreground color used for tasks that are in an unknown state and are stale.
	SelectedBackgroundColor         dom.Color // The background color used for cells and headers when selected.
	IconsFont                       dom.Font  // The font to use for icon drawing.
	Icons                           Icons     // The character table used for icon drawing.
}

func (s *Style) statsStyle(stats taskStats) (icon rune, backgroundColor, foregroundColor dom.Color) {
	switch {
	case stats.numFailedWasSucceeded > 0:
		return s.Icons.Failed, s.CurrentFailedBackgroundColor, s.RegressedForegroundColor
	case stats.numCurrentFailed > 0:
		return s.Icons.Failed, s.CurrentFailedBackgroundColor, s.CurrentFailedForegroundColor
	case stats.numSucceededWasFailed > 0:
		return s.Icons.Succeeded, s.CurrentSucceededBackgroundColor, s.FixedForegroundColor
	case stats.numInProgressWasFailed+stats.numStaleFailed > 0:
		return s.Icons.Failed, s.StaleFailedBackgroundColor, s.StaleFailedForegroundColor
	case stats.numInProgressWasSucceeded+stats.numStaleSucceeded > 0:
		return s.Icons.Succeeded, s.StaleSucceededBackgroundColor, s.StaleSucceededForegroundColor
	case stats.numCurrentSucceeded > 0:
		return s.Icons.Succeeded, s.CurrentSucceededBackgroundColor, s.CurrentSucceededForegroundColor
	case stats.numInProgressWasUnknown+stats.numStaleUnknown > 0:
		return s.Icons.Unknown, s.UnknownBackgroundColor, s.UnknownForegroundColor
	case stats.numStaleUnknown > 0:
		return s.Icons.Unknown, s.UnknownBackgroundColor, s.StaleUnknownForegroundColor
	default:
		return s.Icons.Unknown, s.BackgroundColor, s.UnknownForegroundColor
	}
}

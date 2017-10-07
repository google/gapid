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

import (
	"time"

	"github.com/google/gapid/test/robot/web/client/dom"
)

// Grid is a two-dimensional grid that uses an HTML canvas for display.
// Call New() to create a default initialized Grid.
type Grid struct {
	canvas         *dom.Canvas
	datasets       []*dataset  // The grid dataset(s).
	animating      bool        // If true then the grid will repeatedly redraw
	time           float64     // time in seconds since the grid was created
	startTime      time.Time   // time when the grid was created
	tickPending    bool        // True if there's already a queued tick call
	drawEverything bool        // True if the next call to draw() should include non-visible regions
	transition     *transition // The current animation transition

	// Style holds display style parameters.
	Style Style

	// OnCellClicked is called when a cell is clicked.
	OnCellClicked func(CellIndex, *CellData)

	// OnColumnClicked is called when a column header is clicked.
	OnColumnClicked func(Key, *HeaderData)

	// OnRowClicked is called when a row header is clicked.
	OnRowClicked func(Key, *HeaderData)
}

// New returns a new Grid widget.
func New() *Grid {
	style := Style{
		GridPadding:                     4,
		CellSize:                        48,
		CellShadowColor:                 dom.RGBA(0, 0, 0, 0.3),
		HeaderFont:                      dom.NewFont(16, "Verdana"),
		HeaderFontColor:                 dom.Black,
		GridLineColor:                   dom.RGB(0.5, 0.5, 0.5),
		GridLineWidth:                   0.4,
		BackgroundColor:                 dom.White,
		CurrentSucceededBackgroundColor: dom.RGBA(0.91, 0.96, 0.91, 1.0),
		CurrentSucceededForegroundColor: dom.RGBA(0.30, 0.69, 0.31, 0.9),
		StaleSucceededBackgroundColor:   dom.RGBA(0.91, 0.96, 0.91, 0.3),
		StaleSucceededForegroundColor:   dom.RGBA(0.30, 0.69, 0.31, 0.3),
		CurrentFailedBackgroundColor:    dom.RGBA(1.00, 0.80, 0.82, 1.0),
		CurrentFailedForegroundColor:    dom.RGBA(0.95, 0.26, 0.21, 0.9),
		StaleFailedBackgroundColor:      dom.RGBA(1.00, 0.80, 0.82, 0.3),
		StaleFailedForegroundColor:      dom.RGBA(0.95, 0.26, 0.21, 0.3),
		InProgressForegroundColor:       dom.RGBA(0.00, 0.50, 1.00, 0.9),
		RegressedForegroundColor:        dom.RGBA(1.00, 0.40, 0.41, 0.9),
		FixedForegroundColor:            dom.RGBA(0.18, 0.85, 0.20, 0.9),
		UnknownBackgroundColor:          dom.RGBA(1.00, 1.00, 1.00, 1.0),
		UnknownForegroundColor:          dom.RGBA(0.60, 0.60, 0.60, 0.9),
		StaleUnknownForegroundColor:     dom.RGBA(0.60, 0.60, 0.60, 0.3),
		SelectedBackgroundColor:         dom.RGB(0.89, 0.95, 0.97),
		IconsFont:                       dom.NewFont(25, "Material Icons"),
		Icons: Icons{
			Succeeded: '\uE876',
			Failed:    '\uE5CD',
			Unknown:   '\uE8FD',
		},
	}
	grid := &Grid{
		canvas:    dom.NewCanvas(4, 4),
		startTime: time.Now(),
		Style:     style,
	}
	grid.canvas.OnMouseDown(grid.onMouseDown)
	grid.canvas.OnMouseUp(grid.onMouseUp)
	grid.canvas.OnClick(grid.onClick)
	grid.canvas.OnMouseMove(grid.onMouseMove)
	dom.Win.OnScroll(grid.tick)
	return grid
}

// Element returns the DOM element that holds the grid.
func (g *Grid) Element() *dom.Element {
	return g.canvas.Element
}

func (g *Grid) topDataset() *dataset {
	if c := len(g.datasets); c > 0 {
		return g.datasets[c-1]
	}
	return nil
}

func (g *Grid) resize(w, h int) {
	g.canvas.Resize(w, h)
	g.draw(0, true)
}

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
	"fmt"

	"github.com/google/gapid/test/robot/web/client/dom"
)

func (g *Grid) mousePosition(ev dom.MouseEvent) *dom.Point {
	return dom.NewPoint(float64(ev.PageX-g.canvas.OffsetLeft), float64(ev.PageY-g.canvas.OffsetTop))
}

func (g *Grid) onMouseDown(ev dom.MouseEvent) {
	p := g.mousePosition(ev)
	switch ev.Button {
	case dom.LeftMouseButton:
		d := g.topDataset()
		if row := d.rowAt(p); row != nil {
			row.clickRipples.add(p)
		}
		if column := d.columnAt(p); column != nil {
			column.clickRipples.add(p)
		}
		if cell := d.cellAt(p); cell != nil {
			cell.clickRipples.add(p)
		}
		g.tick()
	case dom.RightMouseButton:
		d := g.topDataset()
		if h := d.columnAt(p); h != nil {
			println(fmt.Sprintf("%+v", h.key))
		}
		if h := d.rowAt(p); h != nil {
			println(fmt.Sprintf("%+v", h.key))
		}
		if cell := d.cellAt(p); cell != nil {
			println(fmt.Sprintf("%+v", cell.data.Key))
		}
		//for i, t := range cell.data.Tasks {
		//	println(fmt.Sprintf("%d: %+v", i, t.Data))
		//}
	}
}

func (g *Grid) onMouseUp(ev dom.MouseEvent) {}

func (g *Grid) onMouseMove(ev dom.MouseEvent) {
	p := g.mousePosition(ev)
	d := g.topDataset()
	d.highlightedRow = d.rowAt(p)
	d.highlightedColumn = d.columnAt(p)
	d.highlightedCell = d.cellAt(p)
	g.tick()
}

func (g *Grid) onClick(ev dom.MouseEvent) {
	p := g.mousePosition(ev)
	if ev.Button == dom.LeftMouseButton {
		d := g.topDataset()
		if row := d.rowAt(p); row != nil {
			if f := g.OnRowClicked; f != nil {
				f(row.key, row.data)
			}
		}
		if column := d.columnAt(p); column != nil {
			if f := g.OnColumnClicked; f != nil {
				f(column.key, column.data)
			}
		}
		if cell := d.cellAt(p); cell != nil {
			if f := g.OnCellClicked; f != nil {
				f(cell.index, cell.data)
			}
		}
	}
}

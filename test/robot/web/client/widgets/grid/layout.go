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
	"math"

	"github.com/google/gapid/test/robot/web/client/dom"
)

func (d *dataset) layout(ctx *dom.Context2D, style *Style) (width, height int) {
	rowTextWidths, maxRowTextWidth := measureHeaders(ctx, style, d.rows)
	colTextHeights, maxColTextHeight := measureHeaders(ctx, style, d.columns)
	maxRowTextWidth += 20
	maxColTextHeight += 20

	cellSize, padding := style.CellSize, style.GridPadding
	rowsWidth := maxRowTextWidth + cellSize
	columnsHeight := maxColTextHeight + cellSize

	x, y := padding, padding+columnsHeight
	for i, h := range d.rows {
		h.rect = dom.NewRect(x, y, x+rowsWidth, y+cellSize)
		h.textOffset = dom.NewPoint((maxRowTextWidth-rowTextWidths[i])/2, cellSize/2)
		h.clusterRect = dom.NewRectWH(x+rowsWidth-cellSize-5, y, cellSize, cellSize)
		y += cellSize
	}

	x, y = padding+rowsWidth, padding
	for i, h := range d.columns {
		h.rect = dom.NewRect(x, y, x+cellSize, y+columnsHeight)
		h.textOffset = dom.NewPoint(cellSize/2, (maxColTextHeight-colTextHeights[i])/2)
		h.clusterRect = dom.NewRectWH(x, y+columnsHeight-cellSize-5, cellSize, cellSize)
		x += cellSize
	}

	x, y = padding+rowsWidth, padding+columnsHeight
	for i, c := range d.cells {
		col, row := d.cellColumnAndRow(i)
		x, y := x+cellSize*float64(col), y+cellSize*float64(row)
		c.rect = dom.NewRect(x, y, x+cellSize, y+cellSize)
	}

	return int(padding*2 + rowsWidth + cellSize*float64(len(d.columns))),
		int(padding*2 + columnsHeight + cellSize*float64(len(d.rows)))
}

func measureHeaders(ctx *dom.Context2D, s *Style, headers []*header) (lengths []float64, max float64) {
	ctx.Save()
	ctx.Font = s.HeaderFont
	lengths = make([]float64, len(headers))
	for i, h := range headers {
		length := math.Ceil(ctx.MeasureText(h.data.Name))
		lengths[i] = length
		if length > max {
			max = length
		}
	}
	ctx.Restore()
	return lengths, max
}

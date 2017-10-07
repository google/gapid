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
	"time"

	"github.com/google/gapid/test/robot/web/client/dom"
)

type cluster struct {
	stats taskStats
	rect  *dom.Rect
}

func (g *Grid) queueTick() {
	if g.tickPending {
		return
	}
	dom.Win.RequestAnimationFrame(func() {
		g.tickPending = false
		g.tick()
	})
	g.tickPending = true
}

func (g *Grid) tick() {
	if g.tickPending {
		return
	}

	time := float64(time.Since(g.startTime)) / float64(time.Second)
	dt := time - g.time
	g.time = time

	// Update transitions
	if g.transition != nil {
		if g.transition.update(dt) {
			g.transition = g.transition.next
		}
	}

	// Set to true when drawing animations.
	g.animating = g.transition != nil

	g.draw(dt, false)

	if g.animating {
		g.queueTick()
	}
}

func (g *Grid) draw(dt float64, everything bool) {
	if g.tickPending {
		g.drawEverything = everything || g.drawEverything
		return
	}
	everything = everything || g.drawEverything
	g.drawEverything = false

	ctx := g.canvas.Context2D()
	ctx.ClearRect(g.canvas.VisibleRect())

	for _, d := range g.datasets {
		g.drawDataset(ctx, d, dt, false)
	}
}

func (g *Grid) drawDataset(ctx *dom.Context2D, d *dataset, dt float64, everything bool) {
	ctx.Save()
	ctx.GlobalAlpha = d.alpha

	visibleRect := g.canvas.VisibleRect()

	// Draw all the non-highlighted row headers
	for _, h := range d.rows {
		if everything || visibleRect.Overlaps(h.rect) {
			if d.highlightedRow != h {
				g.drawRowHeader(ctx, dt, h, false)
			}
		}
	}

	// Draw all the non-highlighted column headers
	for _, h := range d.columns {
		if everything || visibleRect.Overlaps(h.rect) {
			if d.highlightedColumn != h {
				g.drawColHeader(ctx, dt, h, false)
			}
		}
	}

	// Draw all the non-highlighted cells
	for _, c := range d.cells {
		if everything || visibleRect.Overlaps(c.rect) {
			if d.highlightedCell != c {
				g.drawCell(ctx, dt, c, false)
			}
		}
	}

	// Draw the highlighted row header
	if d.highlightedRow != nil {
		g.drawRowHeader(ctx, dt, d.highlightedRow, true)
	}
	// Draw the highlighted column header
	if d.highlightedColumn != nil {
		g.drawColHeader(ctx, dt, d.highlightedColumn, true)
	}
	// Draw the highlighted cell header
	if d.highlightedCell != nil {
		g.drawCell(ctx, dt, d.highlightedCell, true)
	}

	ctx.Restore()
}

func (g *Grid) drawShadow(ctx *dom.Context2D, r *dom.Rect) {
	ctx.ShadowBlur = 30
	ctx.ShadowOffsetX = 3
	ctx.ShadowOffsetY = 3
	ctx.ShadowColor = g.Style.CellShadowColor
	ctx.FillStyle = dom.White
	ctx.FillRect(r)
	ctx.ShadowBlur = 0
	ctx.ShadowOffsetX = 0
	ctx.ShadowOffsetY = 0
}

func (g *Grid) drawRowHeader(ctx *dom.Context2D, dt float64, h *header, highlight bool) {
	alpha := ctx.GlobalAlpha * h.alpha
	if alpha == 0 {
		return
	}

	ctx.Save()
	ctx.GlobalAlpha = alpha

	if highlight {
		g.drawShadow(ctx, h.rect)
	}

	// draw background
	ctx.GlobalAlpha = alpha * h.backgroundAlpha
	_, backgroundColor, _ := g.Style.statsStyle(h.cluster.stats)
	ctx.FillStyle = backgroundColor
	ctx.FillRect(h.rect)

	// draw ripples
	g.drawClickRipples(ctx, dt, &h.clickRipples, func() { ctx.Rect(h.rect) })

	// draw grid
	ctx.GlobalAlpha = alpha
	ctx.StrokeStyle = g.Style.GridLineColor
	ctx.LineWidth = g.Style.GridLineWidth
	ctx.StrokeRect(h.rect)

	// draw cluster
	ctx.GlobalAlpha = alpha * h.clusterAlpha
	g.drawCluster(ctx, h.cluster, h.clusterRect)

	// draw text
	ctx.GlobalAlpha = alpha * h.textAlpha
	ctx.Translate(h.rect.TL)
	ctx.Font = g.Style.HeaderFont
	ctx.FillStyle = g.Style.HeaderFontColor
	ctx.TextBaseline = dom.TextBaselineMiddle
	ctx.FillText(h.data.Name, h.textOffset)
	ctx.Restore()
}

func (g *Grid) drawColHeader(ctx *dom.Context2D, dt float64, h *header, highlight bool) {
	alpha := ctx.GlobalAlpha * h.alpha
	if alpha == 0 {
		return
	}

	ctx.Save()
	ctx.GlobalAlpha = alpha

	if highlight {
		g.drawShadow(ctx, h.rect)
	}

	// draw background
	ctx.GlobalAlpha = alpha * h.backgroundAlpha
	_, backgroundColor, _ := g.Style.statsStyle(h.cluster.stats)
	ctx.FillStyle = backgroundColor
	ctx.FillRect(h.rect)

	// draw ripples
	g.drawClickRipples(ctx, dt, &h.clickRipples, func() { ctx.Rect(h.rect) })

	// draw grid
	ctx.GlobalAlpha = alpha
	ctx.StrokeStyle = g.Style.GridLineColor
	ctx.LineWidth = g.Style.GridLineWidth
	ctx.StrokeRect(h.rect)

	// draw cluster
	ctx.GlobalAlpha = alpha * h.clusterAlpha
	g.drawCluster(ctx, h.cluster, h.clusterRect)

	// draw text
	ctx.GlobalAlpha = alpha * h.textAlpha
	ctx.Translate(h.rect.TL)
	ctx.Font = g.Style.HeaderFont
	ctx.FillStyle = g.Style.HeaderFontColor
	ctx.TextBaseline = dom.TextBaselineMiddle
	ctx.Translate(h.textOffset)
	ctx.Rotate(90)
	ctx.FillText(h.data.Name, &dom.Point{})
	ctx.Restore()
}

func (g *Grid) drawCell(ctx *dom.Context2D, dt float64, c *cell, highlight bool) {
	alpha := ctx.GlobalAlpha * c.alpha
	if alpha == 0 {
		return
	}

	ctx.Save()
	ctx.GlobalAlpha = alpha * c.nonClusterAlpha

	if highlight {
		g.drawShadow(ctx, c.rect)
	}

	// draw background
	_, backgroundColor, _ := g.Style.statsStyle(c.cluster.stats)
	ctx.FillStyle = backgroundColor
	ctx.FillRect(c.rect)

	// draw ripples
	g.drawClickRipples(ctx, dt, &c.clickRipples, func() { ctx.Rect(c.rect) })

	// draw grid
	ctx.StrokeStyle = g.Style.GridLineColor
	ctx.LineWidth = g.Style.GridLineWidth
	ctx.StrokeRect(c.rect)

	// draw cluster
	ctx.GlobalAlpha = alpha
	g.drawCluster(ctx, c.cluster, c.rect)

	ctx.Restore()
}

func (g *Grid) drawCluster(ctx *dom.Context2D, c *cluster, r *dom.Rect) {
	ctx.Save()

	halfWidth := r.W() / 2
	centre := r.Center()

	icon, _, foregroundColor := g.Style.statsStyle(c.stats)

	ctx.LineWidth = 5
	radius := halfWidth * 0.8
	dashLen := (2.0 * math.Pi * radius) / 10
	angle, countToAngle := -90.0, 360.0/float64(c.stats.numTasks)
	drawSegment := func(color dom.Color, dash bool, count int) {
		if count == 0 {
			return
		}
		ctx.Save()
		step := countToAngle * float64(count)
		angleStart, angleEnd := angle, angle-step
		ctx.BeginPath()
		ctx.Arc(centre, radius, angleStart, angleEnd, true)
		ctx.StrokeStyle = color
		ctx.Stroke()

		if dash {
			ctx.LineDashOffset = g.time * 5
			ctx.LineWidth = 3
			ctx.StrokeStyle = g.Style.InProgressForegroundColor
			ctx.SetLineDash([]float64{dashLen * .4, dashLen * .6})
			ctx.BeginPath()
			ctx.Arc(centre, radius, angleStart, angleEnd, true)
			ctx.Stroke()
			g.animating = true
		}

		angle = angleEnd
		ctx.Restore()
	}

	drawSegment(g.Style.StaleUnknownForegroundColor, false, c.stats.numStaleUnknown)
	drawSegment(g.Style.CurrentSucceededForegroundColor, false, c.stats.numCurrentSucceeded)
	drawSegment(g.Style.StaleSucceededForegroundColor, true, c.stats.numInProgressWasSucceeded+c.stats.numInProgressWasUnknown)
	drawSegment(g.Style.StaleSucceededForegroundColor, false, c.stats.numStaleSucceeded)
	drawSegment(g.Style.StaleFailedForegroundColor, false, c.stats.numStaleFailed)
	drawSegment(g.Style.StaleFailedForegroundColor, true, c.stats.numInProgressWasFailed)
	drawSegment(g.Style.CurrentFailedForegroundColor, false, c.stats.numCurrentFailed)
	drawSegment(g.Style.FixedForegroundColor, false, c.stats.numSucceededWasFailed)
	drawSegment(g.Style.RegressedForegroundColor, false, c.stats.numFailedWasSucceeded)

	if icon != 0 {
		ctx.Translate(centre)
		g.drawIcon(ctx, icon, foregroundColor)
	}

	ctx.Restore()
}

func (g *Grid) drawIcon(ctx *dom.Context2D, icon rune, color dom.Color) {
	ctx.Save()
	ctx.Font = g.Style.IconsFont
	ctx.FillStyle = color
	ctx.TextAlign = dom.TextAlignCenter
	ctx.TextBaseline = dom.TextBaselineMiddle
	ctx.FillText(string(icon), &dom.Point{})
	ctx.Restore()
}

func (g *Grid) drawClickRipples(ctx *dom.Context2D, dt float64, l *clickRipples, clip func()) {
	l.update(dt)
	if len(*l) == 0 {
		return
	}
	g.animating = true
	ctx.Save()
	if clip != nil {
		ctx.BeginPath()
		clip()
		ctx.Clip()
	}
	for _, r := range *l {
		g.drawClickRipple(ctx, r)
	}
	ctx.Restore()
}

func (g *Grid) drawClickRipple(ctx *dom.Context2D, r *clickRipple) {
	ctx.Save()
	ctx.GlobalAlpha = r.alpha()
	ctx.BeginPath()
	ctx.Arc(r.center, r.radius(), 0, 360, false)
	ctx.FillStyle = g.Style.SelectedBackgroundColor
	ctx.Fill()
	ctx.Restore()
}

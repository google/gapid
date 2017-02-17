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

	"github.com/google/gapid/core/math/sint"
	"github.com/google/gapid/test/robot/web/client/dom"
)

type transitionUpdate func(deltatime float64) (done bool)

type transition struct {
	update     transitionUpdate
	next, last *transition
}

func newTransition(f transitionUpdate) *transition {
	t := &transition{update: f}
	t.last = t
	return t
}

func (t *transition) then(next *transition) *transition {
	t.last.next = next
	t.last = t.last.next
	return t
}

func newTimedTransition(time float64, f func(normalizedProgress float64)) *transition {
	t := 0.0
	return newTransition(func(deltatime float64) (done bool) {
		t += deltatime
		if t >= time {
			f(1.0)
			return true
		}
		f(t / time)
		return false
	})
}

type rectAnim struct {
	src *dom.Rect
	dst *dom.Rect
	val *dom.Rect
}

func (g *Grid) setTransition(old, new *dataset) {
	w, h := new.layout(g.canvas.Context2D(), &g.Style)

	if old == nil { // Nothing to transition from.
		g.datasets = []*dataset{new}
		g.resize(w, h)
		return
	}

	minColCount := sint.Min(len(old.columns), len(new.columns))
	minRowCount := sint.Min(len(old.rows), len(new.rows))

	oldToNewHeader := map[*header]*header{}
	for r := 0; r < minRowCount; r++ {
		oldToNewHeader[old.rows[r]] = new.rows[r]
	}
	for r := 0; r < minColCount; r++ {
		oldToNewHeader[old.columns[r]] = new.columns[r]
	}

	targetRects := map[Key]*dom.Rect{}
	for _, n := range new.rows {
		if n.key != nil {
			targetRects[n.key] = n.clusterRect
		}
	}
	for _, n := range new.columns {
		if n.key != nil {
			targetRects[n.key] = n.clusterRect
		}
	}
	for _, n := range new.cells {
		if n.data.Key != nil {
			targetRects[n.data.Key] = n.rect
		}
	}

	rectAnims := []rectAnim{}
	for _, o := range old.cells {
		if dst, ok := targetRects[o.data.Key]; ok {
			rectAnims = append(rectAnims, rectAnim{
				src: o.rect.Clone(),
				dst: dst,
				val: o.rect,
			})
		}
	}
	for o, n := range oldToNewHeader {
		rectAnims = append(rectAnims, rectAnim{
			src: o.rect.Clone(),
			dst: n.rect,
			val: o.rect,
		}, rectAnim{
			src: o.clusterRect.Clone(),
			dst: n.clusterRect,
			val: o.clusterRect,
		})
	}

	g.transition = newTransition(func(dt float64) bool {
		// Start by resizing the canvas to hold the old and new datasets.
		g.resize(sint.Max(w, g.canvas.Width), sint.Max(h, g.canvas.Height))
		return true
	}).then(newTimedTransition(0.5, func(t float64) {
		alpha := 1 - t
		// Fade out headers that are changing name, fade out all header clusters.
		for o, n := range oldToNewHeader {
			if o.data.Name != n.data.Name {
				o.textAlpha = alpha
			}
			o.backgroundAlpha = alpha
			o.clusterAlpha = alpha
		}
		// Fade out all headers that don't appear in the new dataset.
		for r, rc := minRowCount, len(old.rows); r < rc; r++ {
			old.rows[r].alpha = alpha
		}
		for c, cc := minColCount, len(old.columns); c < cc; c++ {
			old.columns[c].alpha = alpha
		}
		// Fade out all cells that don't appear in the new dataset.
		for _, o := range old.cells {
			if _, ok := targetRects[o.data.Key]; !ok {
				o.alpha = alpha
			}
		}
	})).then(newTransition(func(dt float64) bool {
		// Switch all the header names
		for o, n := range oldToNewHeader {
			o.data.Name = n.data.Name
			o.textOffset = n.textOffset
		}
		return true
	})).then(newTimedTransition(0.5, func(t float64) {
		tweenWeight := smoothstep(t)
		// Fade in text if it changed.
		for o := range oldToNewHeader {
			o.textAlpha = math.Max(o.textAlpha, t)
		}
		// Fade out all cell content except for the cluster.
		for _, o := range old.cells {
			o.nonClusterAlpha = 1 - t
		}
		// Animate rects.
		for _, r := range rectAnims {
			tweenRect(r.val, r.src, r.dst, tweenWeight)
		}
	})).then(newTransition(func(dt float64) bool {
		// Add the new dataset to the grid, initially fully transparent.
		old.alpha = 1
		new.alpha = 0
		g.datasets = []*dataset{old, new}
		return true
	})).then(newTimedTransition(0.5, func(t float64) {
		// Fade in the new dataset and fade out the old dataset.
		old.alpha = 1 - t
		new.alpha = t
	})).then(newTransition(func(dt float64) bool {
		// Switch completely to the new dataset.
		g.datasets = []*dataset{new}
		g.resize(w, h)
		return true
	}))
}

func smoothstep(x float64) float64 {
	return x * x * x * (x*(x*6-15) + 10)
}

func tween(a, b float64, weight float64) float64 {
	return a + (b-a)*weight
}

func tweenPoint(out, src, dst *dom.Point, weight float64) {
	out.X = tween(src.X, dst.X, weight)
	out.Y = tween(src.Y, dst.Y, weight)
}

func tweenRect(out, src, dst *dom.Rect, weight float64) {
	tweenPoint(out.TL, src.TL, dst.TL, weight)
	tweenPoint(out.BR, src.BR, dst.BR, weight)
}

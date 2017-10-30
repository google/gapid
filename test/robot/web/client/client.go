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

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/gapid/test/robot/web/client/dom"
	"github.com/google/gapid/test/robot/web/client/widgets/grid"
	"github.com/google/gapid/test/robot/web/client/widgets/objview"
)

type traceInfo struct {
	target  Item
	subject Item
}

type task struct {
	trace traceInfo

	kind   Item
	host   Item
	pkg    Item
	result grid.Result
	status grid.Status

	parent *task

	underlying map[string]interface{}
}

func (t *task) Representation() interface{} {
	tr := map[string]interface{}{}
	tr["trace target"] = t.trace.target.Underlying()
	tr["trace subject"] = t.trace.subject.Underlying()
	tr["host"] = t.host.Underlying()
	tr["package"] = t.pkg.Underlying()
	return []interface{}{tr, t.underlying}
}

type enum []Item

func (e enum) indexOf(s Item) int {
	for i, v := range e {
		if v.Id() == s.Id() {
			return i
		}
	}
	return -1
}

type dimension struct {
	name       string
	enumData   enum
	valueOf    func(*task) Item
	itemMap    map[string]Item
	enumSrc    func() enum
	enumSort   func(a, b string) bool
	selectAuto func(c *constraints, d *dimension)
}

func (d *dimension) getEnum() enum {
	if d.enumData == nil {
		d.enumData = d.enumSrc()
	}
	return d.enumData
}

type Item interface {
	Id() string
	Display() string
	Underlying() interface{}
}

type item struct {
	id         string
	display    string
	underlying interface{}
}

func (i item) Underlying() interface{} {
	if i.underlying == nil {
		return i.Id()
	}
	return i.underlying
}

func (i item) Id() string {
	return i.id
}

func (i item) Display() string {
	if i.display == "" {
		return i.id
	}
	return i.display
}

var nilItem = item{id: "<nil>", underlying: map[string]interface{}{}}

func (d *dimension) GetItem(val interface{}) Item {
	if val == nil {
		return nilItem
	}
	id := val.(string)
	if d.itemMap == nil {
		d.itemMap = make(map[string]Item)
		for _, it := range d.getEnum() {
			d.itemMap[it.Id()] = it
		}
	}
	return d.itemMap[id]
}

type constraints map[*dimension]Item

func (s constraints) nextUnconstrained(exclude ...*dimension) *dimension {
nextDimension:
	for _, d := range dimensions {
		for _, e := range exclude {
			if d == e {
				continue nextDimension
			}
		}
		if _, ok := s[d]; !ok {
			return d
		}
	}
	return nil
}

func (s constraints) constrained(d *dimension) bool {
	_, found := s[d]
	return found
}

func (s constraints) match(t *task) bool {
	for d, v := range s {
		if v.Id() != d.valueOf(t).Id() {
			return false
		}
	}
	return true
}

func (s constraints) clone() constraints {
	out := constraints{}
	for d, v := range s {
		out[d] = v
	}
	return out
}

func (s constraints) add(d *dimension, v Item) constraints {
	s[d] = v
	return s
}

func (s constraints) key() grid.Key {
	parts := make([]string, 0, len(s))
	for _, d := range dimensions {
		if v, ok := s[d]; ok {
			parts = append(parts, fmt.Sprintf("%s=%v", d.name, v))
		}
	}
	return strings.Join(parts, ", ")
}

func combineContraints(a, b constraints) constraints {
	out := constraints{}
	for d, v := range a {
		out[d] = v
	}
	for d, v := range b {
		out[d] = v
	}
	return out
}

type page struct {
	grid              *grid.Grid
	tasks             []*task
	columnDimension   *dimension
	rowDimension      *dimension
	columnConstraints map[grid.Key]constraints
	rowConstraints    map[grid.Key]constraints
	constraints       constraints
	onViewChanged     func(c constraints, column, row *dimension)
}

func (p *page) view() (c constraints, column, row *dimension) {
	return p.constraints.clone(), p.columnDimension, p.rowDimension
}

func (p *page) setView(c constraints, column, row *dimension) {
	if p.constraints != nil && p.constraints.key() == c.key() && p.columnDimension == column && p.rowDimension == row {
		return // no change
	}

	// Unconstrain new dimensions
	delete(c, column)
	delete(c, row)

	if len(dimensions) == len(c) {
		delete(c, dimensions[0]) // We need at least one free dimension.
	}

	if column == nil {
		column = p.constraints.nextUnconstrained(row)
	}
	if row == nil {
		row = p.constraints.nextUnconstrained(column)
	}

	if p.constraints.key() != c.key() || p.columnDimension != column || p.rowDimension != row {
		p.constraints, p.columnDimension, p.rowDimension = c, column, row
		p.refresh()
	}

	// Broadcast change even if the restricted view hasn't changed so that the
	// UI reflects the actual view.
	p.onViewChanged(c, column, row)
}

func (p *page) refresh() {
	data := grid.Data{
		Columns: map[grid.Key]*grid.HeaderData{},
		Rows:    map[grid.Key]*grid.HeaderData{},
	}

	p.columnConstraints = map[grid.Key]constraints{}

	switch {
	case p.columnDimension == nil:
	case p.constraints.constrained(p.columnDimension):
		k := p.constraints.key()
		p.columnConstraints[k] = p.constraints
		data.Columns[k] = &grid.HeaderData{Name: string(p.constraints[p.columnDimension].Display())}
	default:
		for _, v := range p.columnDimension.getEnum() {
			c := p.constraints.clone().add(p.columnDimension, v)
			k := c.key()
			p.columnConstraints[k] = c
			data.Columns[k] = &grid.HeaderData{Name: v.Display()}
		}
	}

	p.rowConstraints = map[grid.Key]constraints{}

	switch {
	case p.rowDimension == nil:
	case p.constraints.constrained(p.rowDimension):
		k := p.constraints.key()
		p.rowConstraints[k] = p.constraints
		data.Rows[k] = &grid.HeaderData{Name: p.constraints[p.rowDimension].Display()}
	default:
		for _, v := range p.rowDimension.getEnum() {
			c := p.constraints.clone().add(p.rowDimension, v)
			k := c.key()
			p.rowConstraints[k] = c
			data.Rows[k] = &grid.HeaderData{Name: v.Display()}
		}
	}

	data.Cells = make(map[grid.CellIndex]*grid.CellData)
	for _, task := range p.tasks {
		t := grid.Task{Result: task.result, Status: task.status, Data: task}
		for rowKey, rowConstraint := range p.rowConstraints {
			if rowConstraint.match(task) {
				row := data.Rows[rowKey]
				row.Tasks = append(row.Tasks, t)
			}
		}
		for colKey, colConstraint := range p.columnConstraints {
			if colConstraint.match(task) {
				col := data.Columns[colKey]
				col.Tasks = append(col.Tasks, t)
			}
		}
		for rowKey, rowConstraint := range p.rowConstraints {
			if rowConstraint.match(task) {
				for colKey, colConstraint := range p.columnConstraints {
					if colConstraint.match(task) {
						idx := grid.CellIndex{Column: colKey, Row: rowKey}
						cell, ok := data.Cells[idx]
						if !ok {
							cell = &grid.CellData{Key: combineContraints(colConstraint, rowConstraint).key()}
							data.Cells[idx] = cell
						}
						cell.Tasks = append(cell.Tasks, t)
					}
				}
			}
		}
	}

	p.grid.SetData(data, p.rowDimension.enumSort, p.columnDimension.enumSort)
}

func robotEntityLink(path string, s interface{}) interface{} {
	id := s.(string)
	a := dom.NewA()
	a.Set("href", fmt.Sprintf("/entities/%s", id))
	a.Append(id)
	return a
}

func robotTextPreview(path string, s interface{}) interface{} {
	id := s.(string)

	div := dom.NewDiv()
	div.Element.Style.MaxWidth = 600
	div.Element.Style.MaxHeight = 420
	div.Element.Style.Overflow = "auto"
	div.Element.Style.WhiteSpace = "pre"
	go func() {
		full_text, err := queryRestEndpoint(fmt.Sprintf("/entities/%s", id))
		if err != nil {
			panic(err)
		}

		div.Append(string(full_text))
	}()
	return div
}

func robotVideoView(path string, s interface{}) interface{} {
	id := s.(string)
	v := dom.NewVideo(600, 420, fmt.Sprintf("/entities/%s", id), "mp4")
	v.Append("Your browser does not support embedded video tags")
	return v
}

func setupGrid(tasks []*task) *page {
	const (
		optAuto  = "!!auto"
		optAll   = "!!all"
		optXAxis = "!!x-axis"
		optYAxis = "!!y-axis"
	)

	g := grid.New()
	p := &page{grid: g, tasks: tasks}

	div := dom.NewDiv()

	// The object viewer Representation() for a task looks like:
	// [ {"trace host": ..., "trace target": ...,}, {"id": ..., "input": ..., "host": ..., "target": ..., } ]
	// We customize the display of some of these. For example '/1/target' means the second element of the top-level
	// array, and then the 'target' field in that struct. We format that one as a link to push the target device in
	// the viewer, and then collapse the MemoryLayout parts (see devFmts).
	objView := objview.NewView()
	devFmts := objview.NewFormatters().Add("/information/Configuration/ABIs/\\d+/MemoryLayout", objView.Expandable)
	taskFmts := objview.NewFormatters().Add(
		"^/1/target$",
		func(path string, a interface{}) interface{} {
			id := a.(string)
			return objView.NewPusher(id, "target", targetDimension.GetItem(id).Underlying, devFmts)
		},
	).Add(
		"^/1/host$",
		func(path string, a interface{}) interface{} {
			id := a.(string)
			return objView.NewPusher(id, "host", hostDimension.GetItem(id).Underlying, devFmts)
		},
	).Add("/1/input/((gapi[irst])|gapid_apk|trace|subject|interceptor|vulkanLayer)", robotEntityLink).
		Add("/1/input/layout", objView.Expandable).
		Add("/1/output/(log|report)", robotTextPreview).
		Add("/1/output/video", robotVideoView).
		Add("/0/", objView.Expandable)

	filters := map[*dimension]*dom.Select{}
	for _, d := range dimensions {
		d := d
		row := dom.NewDiv()
		label := dom.NewSpan()
		label.Text().Set(d.name + ": ")
		selecter := dom.NewSelect()
		selecter.Append(dom.NewOption("<auto>", optAuto))
		selecter.Append(dom.NewOption("<all>", optAll))
		selecter.Append(dom.NewOption("<x-axis>", optXAxis))
		selecter.Append(dom.NewOption("<y-axis>", optYAxis))
		for _, e := range d.getEnum() {
			selecter.Append(dom.NewOption(e.Display(), e.Id()))
		}
		filters[d] = selecter
		selecter.OnChange(func() {
			c, column, row := p.view()
			// Clear old value
			delete(c, d)
			if column == d {
				column = nil
			}
			if row == d {
				row = nil
			}
			// Set new value
			switch selecter.Value {
			case optXAxis:
				column = d
			case optYAxis:
				row = d
			case optAuto:
				if d.selectAuto != nil {
					d.selectAuto(&c, d)
					break
				}
				fallthrough
			case optAll:
				delete(c, d)
			default:
				c[d] = d.GetItem(selecter.Value)
			}
			p.setView(c, column, row)
		})
		row.Append(label)
		row.Append(selecter)
		div.Append(row)
	}
	updateFilters := func(c constraints, column, row *dimension) {
		for _, d := range dimensions {
			constraint, constrained := c[d]
			switch {
			case constrained:
				filters[d].Value = constraint.Id()
			case d == column:
				filters[d].Value = optXAxis
			case d == row:
				filters[d].Value = optYAxis
			default:
				filters[d].Value = optAll
			}
		}
	}
	div.Append(g)

	body := dom.Doc().Body()
	body.Style.BackgroundColor = dom.RGB(0.98, 0.98, 0.98)
	objView.Div.Element.Style.Position = "sticky"
	objView.Div.Element.Style.Top = "0"
	objView.Div.Element.Style.Float = "right"
	body.Append(objView)
	body.Append(div)

	dom.Win.Location.OnHashChange(func(dom.HashChangeEvent) {
		p.setView(decodeState(strings.TrimLeft(dom.Win.Location.Hash, "#")))
	})
	p.onViewChanged = func(c constraints, column, row *dimension) {
		dom.Win.Location.Hash = encodeState(c, column, row)
		updateFilters(p.constraints, column, row)
	}
	g.OnColumnClicked = func(k grid.Key, _ *grid.HeaderData) {
		c := p.columnConstraints[k]
		_, _, row := p.view()
		if u := c.nextUnconstrained(row); u != nil {
			p.setView(c, u, row)
		}
	}
	g.OnRowClicked = func(k grid.Key, _ *grid.HeaderData) {
		c := p.rowConstraints[k]
		_, column, _ := p.view()
		if u := c.nextUnconstrained(column); u != nil {
			p.setView(c, column, u)
		}
	}
	g.OnCellClicked = func(i grid.CellIndex, d *grid.CellData) {
		if len(d.Tasks) == 1 {
			t := d.Tasks[0].Data.(*task)
			objView.Set(t.kind.Display(), t, taskFmts)
			return
		}

		c := combineContraints(p.columnConstraints[i.Column], p.rowConstraints[i.Row])
		_, column, row := p.view()
		if u := c.nextUnconstrained(); u != nil {
			column = u
		}
		if u := c.nextUnconstrained(column); u != nil {
			row = u
		}
		p.setView(c, column, row)
	}

	p.setView(decodeState(strings.TrimLeft(dom.Win.Location.Hash, "#")))
	return p
}

func main() {
	dom.Win.OnLoad(func() {
		go func() {
			var seenSeq int64 = -1
			var p *page
			for ticker := time.Tick(time.Second * 10); true; <-ticker {
				// TODO: error handling so we can come back on server restart
				seenData := queryObject(fmt.Sprintf("/status/?seen=%d", seenSeq))
				receivedSeq := (int64)(seenData["seq"].(float64))
				if seenSeq != receivedSeq {
					seenSeq = receivedSeq

					clearDimensionData()
					tasks := getRobotTasks()

					if p == nil {
						p = setupGrid(tasks)
					} else {
						p.tasks = tasks
						p.refresh()
					}
				}

			}
		}()
	})
}

func encodeState(c constraints, columns, rows *dimension) string {
	parts := []string{}
	if columns != nil {
		parts = append(parts, "columns="+columns.name)
	}
	if rows != nil {
		parts = append(parts, "rows="+rows.name)
	}
	for _, d := range dimensions {
		if v, ok := c[d]; ok {
			parts = append(parts, fmt.Sprintf("%s=%v", d.name, v.Id()))
		}
	}
	return strings.Join(parts, "&")
}

func decodeState(s string) (c constraints, columns, rows *dimension) {
	vals := map[string]string{}
	for _, s := range strings.Split(s, "&") {
		pair := strings.Split(s, "=")
		if len(pair) != 2 {
			continue
		}
		vals[pair[0]] = string(pair[1])
	}

	c = constraints{}
	for _, d := range dimensions {
		if name, ok := vals["columns"]; ok && d.name == string(name) {
			columns = d
		} else if name, ok := vals["rows"]; ok && d.name == string(name) {
			rows = d
		} else if value, ok := vals[d.name]; ok {
			c[d] = d.GetItem(value)
		}
	}

	return c, columns, rows
}

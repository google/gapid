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
	name     string
	enumData enum
	valueOf  func(*task) Item
	itemMap  map[string]Item
	enumSrc  func() enum
	enumSort func(a, b string) bool
	defVal   func() string
}

func (d *dimension) getEnum() enum {
	if d.enumData == nil {
		d.enumData = d.enumSrc()
	}
	return d.enumData
}

func (d *dimension) defaultId() string {
	if d.defVal != nil {
		return d.defVal()
	}
	return ""
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
	a := robotEntityLink(path, s)
	div.Append(a)
	textDiv := dom.NewDiv()
	textDiv.Element.Style.MaxWidth = 600
	textDiv.Element.Style.MaxHeight = 420
	textDiv.Element.Style.Overflow = "auto"
	textDiv.Element.Style.WhiteSpace = "pre"
	go func() {
		// if the text is larger than a megabyte, cut off early.
		fullText, err := queryRestEndpointWithCutoff(fmt.Sprintf("/entities/%s", id), 1024*1024)
		if err != nil {
			textDiv.Append("Error previewing text:" + err.Error())
			return
		}

		textDiv.Append(string(fullText))
	}()
	div.Append(textDiv)
	return div
}

func robotVideoView(path string, s interface{}) interface{} {
	id := s.(string)
	v := dom.NewVideo(600, 420, fmt.Sprintf("/entities/%s", id), "mp4")
	v.Append("Your browser does not support embedded video tags")
	return v
}

type filter struct {
	selecter *dom.Select
	options  map[string]string
	dim      *dimension
	sort     func(a, b string) bool
}

type view struct {
	div        *dom.Div
	objView    *objview.View
	formatters *objview.Formatters
	filters    []*filter
	column     *filter
	row        *filter
}

func newView() *view {
	v := &view{div: dom.NewDiv(), objView: objview.NewView()}

	// The object viewer Representation() for a task looks like:
	// [ {"trace host": ..., "trace target": ...,}, {"id": ..., "input": ..., "host": ..., "target": ..., } ]
	// We customize the display of some of these. For example '/1/target' means the second element of the top-level
	// array, and then the 'target' field in that struct. We format that one as a link to push the target device in
	// the viewer, and then collapse the MemoryLayout parts (see devFmts).
	devFmts := objview.NewFormatters().Add("/information/Configuration/ABIs/\\d+/MemoryLayout", v.objView.Expandable)
	v.formatters = objview.NewFormatters().Add(
		"/1/trace_target",
		func(path string, a interface{}) interface{} {
			id := a.(string)
			return v.objView.NewPusher(id, "trace_target", targetDimension.GetItem(id).Underlying, devFmts)
		},
	).Add(
		"/1/host",
		func(path string, a interface{}) interface{} {
			id := a.(string)
			return v.objView.NewPusher(id, "host", hostDimension.GetItem(id).Underlying, devFmts)
		},
	).Add("/1/input/((gapi[irst])|gapid_apk|trace|subject|interceptor|vulkanLayer)", robotEntityLink).
		Add("/1/input/layout", v.objView.Expandable).
		Add("/1/output/(log|report)", robotTextPreview).
		Add("/1/output/err", func(path string, a interface{}) interface{} {
			err := a.(string)
			div := dom.NewDiv()
			div.Element.Style.MaxWidth = 600
			div.Element.Style.MaxHeight = 420
			div.Element.Style.Overflow = "auto"
			div.Element.Style.WhiteSpace = "pre"
			div.Append(err)
			return div
		}).
		Add("/1/output/video", robotVideoView).
		Add("/0/", v.objView.Expandable)

	body := dom.Doc().Body()
	body.Style.BackgroundColor = dom.RGB(0.98, 0.98, 0.98)
	v.objView.Div.Element.Style.Position = "sticky"
	v.objView.Div.Element.Style.Top = "0"
	v.objView.Div.Element.Style.Float = "right"
	body.Append(v.objView)
	body.Append(v.div)

	return v
}

func (v *view) optAllValue() string {
	return "!!all"
}

func (v *view) optXAxisValue() string {
	return "!!x-axis"
}

func (v *view) optYAxisValue() string {
	return "!!y-axis"
}

func (v *view) addFilter(d *dimension) *filter {
	row := dom.NewDiv()
	label := dom.NewSpan()
	label.Text().Set(d.name + ": ")
	selecter := dom.NewSelect()
	selecter.Append(dom.NewOption("<all>", v.optAllValue()))
	selecter.Append(dom.NewOption("<x-axis>", v.optXAxisValue()))
	selecter.Append(dom.NewOption("<y-axis>", v.optYAxisValue()))

	f := &filter{
		selecter: selecter,
		options:  make(map[string]string),
		dim:      d,
		sort:     d.enumSort,
	}

	for _, e := range d.getEnum() {
		selecter.Append(dom.NewOption(e.Display(), e.Id()))
		f.options[e.Display()] = e.Id()
	}

	if id := d.defaultId(); id != "" {
		f.selecter.Value = id
	} else if v.column == nil {
		f.selecter.Value = v.optXAxisValue()
		v.column = f
	} else if v.row == nil {
		f.selecter.Value = v.optYAxisValue()
		v.row = f
	} else {
		f.selecter.Value = v.optAllValue()
	}

	v.filters = append(v.filters, f)

	row.Append(label)
	row.Append(selecter)
	v.div.Append(row)

	return f
}

func (v *view) setAxis(f *filter) *filter {
	if f.selecter.Value == v.optXAxisValue() {
		if v.column == f {
			// no change
			return nil
		} else if v.row == f {
			// perform a swap
			return nil
		} else {
			oldAxis := v.column
			v.column = f
			return oldAxis
		}
	} else if f.selecter.Value == v.optYAxisValue() {
		if v.row == f {
			// no change
			return nil
		} else if v.column == f {
			// perform a swap
			return nil
		} else {
			oldAxis := v.row
			v.row = f
			return oldAxis
		}
	} else {
		return nil
	}
}

func (v *view) refreshGrid(tasks []*task, g *grid.Grid) {
	data := grid.Data{
		Columns: map[grid.Key]*grid.HeaderData{},
		Rows:    map[grid.Key]*grid.HeaderData{},
		Cells:   map[grid.CellIndex]*grid.CellData{},
	}

	for d, v := range v.column.options {
		data.Columns[v] = &grid.HeaderData{Name: d}
	}

	for d, v := range v.row.options {
		data.Rows[v] = &grid.HeaderData{Name: d}
	}

nextTask:
	for _, task := range tasks {
		t := grid.Task{Result: task.result, Status: task.status, Data: task}
		colKey, rowKey := "", ""
		for _, f := range v.filters {
			if f.selecter.Value == v.optAllValue() {
				continue
			}
			dimValue := f.dim.valueOf(task).Id()
			switch f.selecter.Value {
			case v.optXAxisValue():
				colKey = dimValue
			case v.optYAxisValue():
				rowKey = dimValue
			default:
				if f.selecter.Value != dimValue {
					// doesn't match the filter, don't add this task
					continue nextTask
				}
			}
		}
		col := data.Columns[colKey]
		col.Tasks = append(col.Tasks, t)
		row := data.Rows[rowKey]
		row.Tasks = append(row.Tasks, t)
		idx := grid.CellIndex{Column: colKey, Row: rowKey}
		if cell, ok := data.Cells[idx]; !ok {
			cell = &grid.CellData{Key: idx, Tasks: grid.TaskList{t}}
			data.Cells[idx] = cell
		} else {
			cell.Tasks = append(cell.Tasks, t)
		}
	}

	g.SetData(data, v.column.sort, v.row.sort)
}

type controller struct {
	tasks []*task
	free  map[*filter]*filter
	v     *view
	g     *grid.Grid
}

func newController(tasks []*task) *controller {
	return &controller{
		tasks: tasks,
		free:  map[*filter]*filter{},
		v:     newView(),
		g:     nil,
	}
}

func (c *controller) nextFree() *filter {
	for _, k := range c.free {
		return k
	}
	return nil
}

func (c *controller) resolveFilter(changed *filter) {
	switch changed.selecter.Value {
	case c.v.optAllValue():
		c.free[changed] = changed
	case c.v.optXAxisValue(), c.v.optYAxisValue():
		if axis := c.v.setAxis(changed); axis != nil {
			c.free[axis] = axis
			delete(c.free, changed)
		}
	default:
		delete(c.free, changed)
	}
}

func (c *controller) resolveAxis(oldAxis *filter, newValue string) *filter {
	if f := c.nextFree(); f != nil {
		switch oldAxis.selecter.Value {
		case c.v.optXAxisValue():
			f.selecter.Value = c.v.optXAxisValue()
		case c.v.optYAxisValue():
			f.selecter.Value = c.v.optYAxisValue()
		default:
			return nil
		}
		c.resolveFilter(f)
		c.setFilterValue(oldAxis, newValue)
		return f
	}
	return nil
}

func (c *controller) setFilterValue(f *filter, newValue string) {
	if c.v.column == f || c.v.row == f {
		c.resolveAxis(f, newValue)
	} else {
		f.selecter.Value = newValue
		c.resolveFilter(f)
	}
}

func (c *controller) addDimension(d *dimension) {
	filter := c.v.addFilter(d)

	c.resolveFilter(filter)

	filter.selecter.OnChange(func() {
		c.resolveFilter(filter)
		c.commitViewState()
	})
}

func (c *controller) addGrid() {
	c.g = grid.New()

	c.g.OnColumnClicked = func(k grid.Key, _ *grid.HeaderData) {
		c.resolveAxis(c.v.column, k.(string))
		c.commitViewState()
	}
	c.g.OnRowClicked = func(k grid.Key, _ *grid.HeaderData) {
		c.resolveAxis(c.v.row, k.(string))
		c.commitViewState()
	}
	c.g.OnCellClicked = func(i grid.CellIndex, d *grid.CellData) {
		if len(d.Tasks) == 1 {
			t := d.Tasks[0].Data.(*task)
			c.v.objView.Set(t.kind.Display(), t, c.v.formatters)
			return
		}
		c.resolveAxis(c.v.column, i.Column.(string))
		c.resolveAxis(c.v.row, i.Row.(string))
		c.commitViewState()
	}

	c.v.div.Append(c.g)
}

func (c *controller) commitViewState() {
	c.v.refreshGrid(c.tasks, c.g)

	if c.v.column == nil || c.v.row == nil {
		panic("column or row not set! This should not happen.")
	}

	parts := []string{}
	parts = append(parts, "columns="+c.v.column.dim.name)
	parts = append(parts, "rows="+c.v.row.dim.name)
	for _, f := range c.v.filters {
		if _, ok := c.free[f]; !ok {
			parts = append(parts, f.dim.name+"="+f.selecter.Value)
		}
	}
	dom.Win.Location.Hash = strings.Join(parts, "&")
}

func (c *controller) decodeViewState() {
	s := strings.TrimLeft(dom.Win.Location.Hash, "#")
	vals := map[string]string{}
	for _, s := range strings.Split(s, "&") {
		pair := strings.Split(s, "=")
		if len(pair) != 2 {
			continue
		}
		vals[pair[0]] = string(pair[1])
	}

	for _, f := range c.v.filters {
		if name, ok := vals["columns"]; ok && f.dim.name == string(name) {
			c.setFilterValue(f, c.v.optXAxisValue())
		} else if name, ok := vals["rows"]; ok && f.dim.name == string(name) {
			c.setFilterValue(f, c.v.optYAxisValue())
		} else if value, ok := vals[f.dim.name]; ok {
			c.setFilterValue(f, value)
		} else {
			c.setFilterValue(f, c.v.optAllValue())
		}
	}

	c.commitViewState()
	dom.Win.Location.OnHashChange(func(dom.HashChangeEvent) { c.decodeViewState() })
}

func setupController(tasks []*task) *controller {
	c := newController(tasks)
	for _, d := range dimensions {
		c.addDimension(d)
	}
	c.addGrid()
	c.commitViewState()
	c.decodeViewState()
	return c
}

func main() {
	dom.Win.OnLoad(func() {
		go func() {
			var seenSeq int64 = -1
			var c *controller
			for ticker := time.Tick(time.Second * 10); true; <-ticker {
				// TODO: error handling so we can come back on server restart
				seenData := queryObject(fmt.Sprintf("/status/?seen=%d", seenSeq))
				receivedSeq := (int64)(seenData["seq"].(float64))
				if seenSeq != receivedSeq {
					seenSeq = receivedSeq

					clearDimensionData()
					tasks := getRobotTasks()

					if c == nil {
						c = setupController(tasks)
					} else {
						c.tasks = tasks
						c.v.refreshGrid(c.tasks, c.g)
					}
				}
			}
		}()
	})
}

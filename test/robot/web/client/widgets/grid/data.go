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
	"sort"

	"github.com/google/gapid/test/robot/web/client/dom"
)

// Data holds all the presentable data for the grid.
type Data struct {
	Columns map[Key]*HeaderData
	Rows    map[Key]*HeaderData
	Cells   map[CellIndex]*CellData
}

// HeaderData holds information about a single row or column header.
type HeaderData struct {
	// Label used for displaying the header.
	Name string

	// The list of tasks that belong to this cell.
	Tasks TaskList
}

// Key is a unique identifier for headers and cells.
type Key interface{}

// CellIndex locates a single cell in the grid.
type CellIndex struct {
	Column Key
	Row    Key
}

// CellData holds the list of tasks to display for that cell.
type CellData struct {
	// The list of tasks that belong to this cell.
	Tasks TaskList

	// Optional Key used for transition animations when changing data.
	Key Key
}

// Task holds information about a single task in a cell.
type Task struct {
	// Last completed task result.
	Result Result

	// The current task status.
	Status Status

	// User data.
	Data interface{}
}

// Result is an enumerator of task results.
type Result int

const (
	// Unknown represents the unknown result of a task.
	Unknown = Result(iota)

	// Succeeded represents a task that has succeeded.
	Succeeded

	// Failed represents a task that has failed.
	Failed
)

// Status is an enumerator of task statuses.
type Status int

const (
	// Current represents a task that has a result for the latest data.
	Current = Status(iota)

	// Stale represents a task that has a result for data that is not current.
	Stale

	// InProgress represents a task that is currently being run.
	// The task's result will be for data that is not current.
	InProgress

	// Changed represents a task with a result for the latest data that is different from its stale data
	Changed
)

// TaskList is a list of tasks.
type TaskList []Task

// Count returns the number of tasks that pass the predicate.
func (l *TaskList) Count(pred func(Task) bool) int {
	i := 0
	for _, j := range *l {
		if pred(j) {
			i++
		}
	}
	return i
}

func (l TaskList) stats() taskStats {
	return taskStats{
		numCurrentSucceeded:       l.Count(taskCurrentSucceeded),
		numStaleSucceeded:         l.Count(taskStaleSucceeded),
		numInProgressWasSucceeded: l.Count(taskInProgressWasSucceeded),
		numSucceededWasFailed:     l.Count(taskSucceededWasFailed),
		numInProgressWasUnknown:   l.Count(taskInProgressWasUnknown),
		numInProgressWasFailed:    l.Count(taskInProgressWasFailed),
		numStaleFailed:            l.Count(taskStaleFailed),
		numCurrentFailed:          l.Count(taskCurrentFailed),
		numFailedWasSucceeded:     l.Count(taskFailedWasSucceeded),
		numStaleUnknown:           l.Count(taskStaleUnknown),
		numTasks:                  len(l),
	}
}

func taskCurrentSucceeded(t Task) bool       { return t.Result == Succeeded && t.Status == Current }
func taskStaleSucceeded(t Task) bool         { return t.Result == Succeeded && t.Status == Stale }
func taskInProgressWasSucceeded(t Task) bool { return t.Result == Succeeded && t.Status == InProgress }
func taskSucceededWasFailed(t Task) bool     { return t.Result == Succeeded && t.Status == Changed }
func taskCurrentFailed(t Task) bool          { return t.Result == Failed && t.Status == Current }
func taskStaleFailed(t Task) bool            { return t.Result == Failed && t.Status == Stale }
func taskInProgressWasFailed(t Task) bool    { return t.Result == Failed && t.Status == InProgress }
func taskFailedWasSucceeded(t Task) bool     { return t.Result == Failed && t.Status == Changed }
func taskInProgressWasUnknown(t Task) bool   { return t.Result == Unknown && t.Status == InProgress }
func taskStaleUnknown(t Task) bool           { return t.Result == Unknown && t.Status == Stale }

type cell struct {
	index           CellIndex
	data            *CellData
	clickRipples    clickRipples
	alpha           float64
	nonClusterAlpha float64
	cluster         *cluster
	rect            *dom.Rect
}

func newCell(i CellIndex, d *CellData) *cell {
	return &cell{
		index:           i,
		data:            d,
		cluster:         &cluster{stats: d.Tasks.stats()},
		alpha:           1,
		nonClusterAlpha: 1,
	}
}

type taskStats struct {
	numCurrentSucceeded       int
	numStaleSucceeded         int
	numInProgressWasSucceeded int
	numSucceededWasFailed     int
	numInProgressWasUnknown   int
	numInProgressWasFailed    int
	numStaleFailed            int
	numCurrentFailed          int
	numFailedWasSucceeded     int
	numStaleUnknown           int
	numTasks                  int
}

type header struct {
	key             Key
	data            *HeaderData
	index           int
	clickRipples    clickRipples
	alpha           float64
	textAlpha       float64
	backgroundAlpha float64
	clusterAlpha    float64
	tasks           TaskList
	cluster         *cluster
	rect            *dom.Rect
	textOffset      *dom.Point
	clusterRect     *dom.Rect
}

func newHeader(k Key, d *HeaderData) *header {
	return &header{
		key:             k,
		data:            d,
		cluster:         &cluster{},
		alpha:           1,
		textAlpha:       1,
		backgroundAlpha: 1,
		clusterAlpha:    1,
	}
}

type dataset struct {
	columns           []*header
	rows              []*header
	cells             []*cell // columns * (rows * (cell))
	alpha             float64
	highlightedCell   *cell   // The highlighted cell, or nil
	highlightedRow    *header // The highlighted row, or nil
	highlightedColumn *header // The highlighted column, or nil
}

func (d *dataset) cellIndex(col, row int) (idx int) {
	return col*len(d.rows) + row
}

func (d *dataset) cellColumnAndRow(idx int) (col, row int) {
	rows := len(d.rows)
	return idx / rows, idx % rows
}

func (d *dataset) rowAt(p *dom.Point) *header {
	for _, h := range d.rows {
		if h.rect.Contains(p) {
			return h
		}
	}
	return nil
}

func (d *dataset) columnAt(p *dom.Point) *header {
	for _, h := range d.columns {
		if h.rect.Contains(p) {
			return h
		}
	}
	return nil
}

func (d *dataset) cellAt(p *dom.Point) *cell {
	for _, c := range d.cells {
		if c.rect.Contains(p) {
			return c
		}
	}
	return nil
}

func buildData(in Data, rowSort, columnSort headerLess) *dataset {
	out := &dataset{alpha: 1}
	// Build all the columns.
	keyToCol := map[Key]*header{}
	out.columns = make([]*header, 0, len(in.Columns))
	for k, h := range in.Columns {
		col := newHeader(k, h)
		col.tasks = h.Tasks
		out.columns = append(out.columns, col)
		keyToCol[k] = col
	}
	// Build all the rows.
	keyToRow := map[Key]*header{}
	out.rows = make([]*header, 0, len(in.Rows))
	for k, h := range in.Rows {
		row := newHeader(k, h)
		row.tasks = h.Tasks
		out.rows = append(out.rows, row)
		keyToRow[k] = row
	}
	// Sort all the columns and rows.
	sort.Sort(&headerSorter{out.columns, columnSort})
	sort.Sort(&headerSorter{out.rows, rowSort})
	for i, c := range out.columns {
		c.index = i
	}
	for i, r := range out.rows {
		r.index = i
	}
	// Sort all the cells.
	out.cells = make([]*cell, len(out.rows)*len(out.columns))
	for i, c := range in.Cells {
		col, ok := keyToCol[i.Column]
		if !ok {
			continue
		}
		row, ok := keyToRow[i.Row]
		if !ok {
			continue
		}
		cellIdx := out.cellIndex(col.index, row.index)
		out.cells[cellIdx] = newCell(i, c)
	}
	// Cache stats for all tasks in all columns and rows
	for _, h := range out.columns {
		h.cluster.stats = h.tasks.stats()
	}
	for _, h := range out.rows {
		h.cluster.stats = h.tasks.stats()
	}
	// Create an empty cell for any missing cells.
	for i, c := range out.cells {
		if c == nil {
			out.cells[i] = newCell(CellIndex{}, &CellData{})
		}
	}
	return out
}

// SetData assigns the data to the grid.
func (g *Grid) SetData(data Data, rowSort, columnSort func(a, b string) bool) {
	if rowSort == nil {
		rowSort = sortAlphabetic
	}
	if columnSort == nil {
		columnSort = sortAlphabetic
	}
	new, old := buildData(data, makeHeaderLess(rowSort), makeHeaderLess(columnSort)), g.topDataset()
	g.setTransition(old, new)
	g.tick()
}

type headerLess func(a, b *header) bool

func sortAlphabetic(a, b string) bool { return a < b }

func makeHeaderLess(sort func(a, b string) bool) headerLess {
	return func(a, b *header) bool {
		return sort(a.data.Name, b.data.Name)
	}
}

type headerSorter struct {
	list []*header
	less headerLess
}

func (s *headerSorter) Len() int           { return len(s.list) }
func (s *headerSorter) Less(i, j int) bool { return s.less(s.list[i], s.list[j]) }
func (s *headerSorter) Swap(i, j int)      { s.list[i], s.list[j] = s.list[j], s.list[i] }

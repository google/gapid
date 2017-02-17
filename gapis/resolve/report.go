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

package resolve

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/gapid/core/fault/severity"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/stringtable"
)

// Report resolves the report for the given capture and optional device.
func Report(ctx log.Context, c *path.Capture, d *path.Device) (*service.Report, error) {
	obj, err := database.Build(ctx, &ReportResolvable{c, d})
	if err != nil {
		return nil, err
	}
	return obj.(*service.Report), nil
}

// Resolve implements the database.Resolver interface.
func (r *ReportResolvable) Resolve(ctx log.Context) (interface{}, error) {
	ctx = capture.Put(ctx, r.Capture)

	c, err := capture.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	list, err := c.Atoms(ctx)
	if err != nil {
		return nil, err
	}

	atoms := list.Atoms

	builder := service.NewReportBuilder()

	var lastError interface{}
	var currentAtom uint64
	items := []*service.ReportItemRaw{}
	state := c.NewState()
	state.OnError = func(err interface{}) {
		lastError = err
	}
	state.NewMessage = func(s severity.Level, m *stringtable.Msg) uint32 {
		items = append(items, service.WrapReportItem(
			&service.ReportItem{
				Severity: service.Severity(s),
				Command:  currentAtom,
			}, m))
		return uint32(len(items) - 1)
	}
	state.AddTag = func(i uint32, t *stringtable.Msg) {
		items[i].Tags = append(items[i].Tags, t)
	}

	mutate := func(i int, a atom.Atom) {
		defer func() {
			if err := recover(); err != nil {
				items = append(items, service.WrapReportItem(
					&service.ReportItem{
						Severity: service.Severity_CriticalLevel,
						Command:  uint64(i),
					}, messages.ErrCritical(fmt.Sprintf("%s", err))))
			}
		}()
		if as := a.Extras().Aborted(); as != nil && as.IsAssert {
			items = append(items, service.WrapReportItem(
				&service.ReportItem{
					Severity: service.Severity_CriticalLevel,
					Command:  uint64(i),
				}, messages.ErrTraceAssert(as.Reason)))
		}
		err := a.Mutate(ctx, state, nil /* no builder, just mutate */)
		if len(items) == 0 {
			var m *stringtable.Msg
			if err != nil && !atom.IsAbortedError(err) {
				m = messages.ErrMessage(err)
			} else if lastError != nil {
				m = messages.ErrMessage(fmt.Sprintf("%v", lastError))
			}
			if m != nil {
				items = append(items, service.WrapReportItem(
					&service.ReportItem{
						Severity: service.Severity_ErrorLevel,
						Command:  uint64(i),
					}, m))
			}
		}
	}
	// Gather report items from the state mutator, and collect together all the
	// APIs in use.
	apis := map[gfxapi.API]struct{}{}
	for i, a := range atoms {
		if api := a.API(); api != nil {
			apis[api] = struct{}{}
		}
		currentAtom = uint64(i)
		mutate(i, a)
		for _, item := range items {
			item.Tags = append(item.Tags, getAtomNameTag(a))
			builder.Add(ctx, item)
		}
		items, lastError = items[:0], nil
	}

	if r.Device != nil {
		// Request is for a replay report too.
		intent := replay.Intent{
			Capture: r.Capture,
			Device:  r.Device,
		}

		mgr := replay.GetManager(ctx)

		// Capture can use multiple APIs. Iterate the APIs in use looking for
		// those that support the QueryIssues interface. Call QueryIssues for each
		// of these APIs, and use reflect.Select to gather all the issues.
		issues := []reflect.SelectCase{}
		pending := make([]gfxapi.API, 0, len(apis))
		for api := range apis {
			if qi, ok := api.(replay.QueryIssues); ok {
				c := make(chan replay.Issue, 256)
				go qi.QueryIssues(ctx, intent, mgr, c)
				issues = append(issues, reflect.SelectCase{
					Dir:  reflect.SelectRecv,
					Chan: reflect.ValueOf(c),
				})
				pending = append(pending, api)
			}
		}

		for len(pending) > 0 {
			i, v, ok := reflect.Select(issues)
			if !ok {
				pending[i] = pending[len(pending)-1]
				pending = pending[0 : len(pending)-1]
				issues[i] = issues[len(issues)-1]
				issues = issues[0 : len(issues)-1]
				continue
			}
			issue := v.Interface().(replay.Issue)
			item := service.WrapReportItem(
				&service.ReportItem{
					Severity: issue.Severity,
					Command:  uint64(issue.Atom),
				}, messages.ErrReplayDriver(issue.Error.Error()))
			if int(issue.Atom) < len(atoms) {
				item.Tags = append(item.Tags, getAtomNameTag(atoms[issue.Atom]))
			}
			builder.Add(ctx, item)
		}

		// Items are now all out of order. Sort them.
		builder.SortReport()
	}

	return builder.Build(), nil
}

func getAtomNameTag(a atom.Atom) *stringtable.Msg {
	atomName := a.Class().Schema().Name()
	return messages.TagAtomName(strings.ToLower(atomName[:1]) + atomName[1:])
}

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
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"time"

	"os"

	"github.com/google/gapid/test/robot/web/client/widgets/grid"
)

var (
	traceKind  = item{id: "trace"}
	reportKind = item{id: "report"}
	replayKind = item{id: "replay"}

	subjectDimension = &dimension{
		name: "subject",
		valueOf: func(t *task) Item {
			return t.trace.subject
		},
		enumSrc: func() enum {
			return itemGetter("{{.id}}", "{{.Information.APK.package}}")(queryArray("/subjects/"))
		},
	}
	targetDimension = &dimension{
		name: "traceTarget",
		valueOf: func(t *task) Item {
			return t.trace.target
		},
		enumSrc: func() enum {
			return itemGetter("{{.id}}", "{{.information.Configuration.Hardware.Name}}")(queryArray("/devices/"))
		},
	}
	hostDimension = &dimension{
		name: "traceHost",
		valueOf: func(t *task) Item {
			return t.trace.host
		},
		enumSrc: func() enum {
			return itemGetter("{{.id}}", "{{.information.Configuration.Hardware.Name}}")(queryArray("/devices/"))
		},
	}
	kindDimension = &dimension{
		name:     "kind",
		enumData: enum{traceKind, replayKind, reportKind},
		valueOf: func(t *task) Item {
			return t.kind
		},
	}
	gapidApkDimension = &dimension{
		name: "gapid_apk",
		valueOf: func(t *task) Item {
			return t.trace.gapidApk
		},
		enumSrc: func() enum {
			return toolFieldMerger("gapid_apk", queryArray("/packages/"))
		},
	}
	gapitDimension = &dimension{
		name: "gapit",
		valueOf: func(t *task) Item {
			return t.trace.gapit
		},
		enumSrc: func() enum {
			return toolFieldMerger("gapit", queryArray("/packages/"))
		},
	}

	dimensions = []*dimension{subjectDimension, targetDimension, hostDimension, kindDimension, gapidApkDimension, gapitDimension}
)

func itemGetter(idPattern string, displayPattern string) func([]interface{}) enum {
	mustTemplate := func(s string) *template.Template {
		return template.Must(template.New(fmt.Sprintf("t%d", time.Now().Unix())).Parse(s))
	}
	exec := func(t *template.Template, item interface{}) string {
		var b bytes.Buffer
		t.Execute(&b, item)
		return b.String()
	}
	idt := mustTemplate(idPattern)
	dispt := mustTemplate(displayPattern)
	return func(entries []interface{}) enum {
		result := enum{}
		for _, it := range entries {
			result = append(result, item{id: exec(idt, it), display: exec(dispt, it), underlying: it})
		}
		return result
	}
}

func toolFieldMerger(field string, entries []interface{}) enum {
	// TODO(valbulescu): maybe parse into the actual protos instead of using JSON
	result := enum{}
	for _, it := range entries {
		tool := (it.(map[string]interface{}))["tool"].([]interface{})
		for _, t := range tool {
			fieldVal, ok := (t.(map[string]interface{}))[field]
			if ok {
				result = append(result, item{id: fieldVal.(string)})
			}
		}
	}
	return result
}

func queryRestEndpoint(path string) ([]byte, error) {
	resp, err := http.Get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func queryArray(path string) []interface{} {
	// TODO: Cache this, as we're using the same path for multiple dimensions.
	body, err := queryRestEndpoint(path)
	if err != nil {
		panic(err)
	}

	arr := []interface{}{}
	if err := json.Unmarshal(body, &arr); err != nil {
		panic(err)
	}
	return arr
}

func queryObject(path string) map[string]interface{} {
	resp, err := http.Get(path)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	arr := map[string]interface{}{}
	if err := json.Unmarshal(body, &arr); err != nil {
		panic(err)
	}
	return arr
}

func clearDimensionData() {
	for _, d := range dimensions {
		if d.enumSrc != nil {
			d.enumData = nil
		}
		d.itemMap = nil
	}
}

func newTask(entry map[string]interface{}, kind Item) *task {
	t := &task{
		underlying: entry,
		kind:       kind,
		trace:      traceInfo{host: nilItem, subject: nilItem, target: nilItem, gapidApk: nilItem, gapit: nilItem},
	}

	if st, ok := entry["status"].(float64); ok {
		switch int(st) {
		case 1:
			t.status = grid.InProgress
			t.result = grid.Unknown
		case 2:
			t.status = grid.Current
			t.result = grid.Succeeded
		case 3:
			t.status = grid.Current
			t.result = grid.Failed
		}
	} else {
		t.status = grid.Stale
		t.result = grid.Failed
	}
	return t
}

func robotTasksPerKind(kind Item, path string, fun func(map[string]interface{}, *task)) []*task {
	tasks := []*task{}
	for _, e := range queryArray(path) {
		e := e.(map[string]interface{})
		t := newTask(e, kind)
		fun(e, t)
		tasks = append(tasks, t)
	}
	return tasks
}

func getRobotTasks() []*task {
	traceMap := map[string]*task{}
	tasks := []*task{}

	traceProc := func(e map[string]interface{}, t *task) {
		ei := e["input"].(map[string]interface{})
		t.trace = traceInfo{
			host:     hostDimension.GetItem(e["host"]),
			subject:  subjectDimension.GetItem(ei["subject"]),
			target:   targetDimension.GetItem(e["target"]),
			gapidApk: gapidApkDimension.GetItem(ei["gapid_apk"]),
			gapit:    gapitDimension.GetItem(ei["gapit"]),
		}
		if eo := e["output"]; eo != nil {
			if traceOutput := eo.(map[string]interface{})["trace"]; traceOutput != nil {
				traceMap[traceOutput.(string)] = t
			}
		}
	}
	tasks = append(tasks, robotTasksPerKind(traceKind, "/traces/", traceProc)...)

	subTaskProc := func(e map[string]interface{}, t *task) {
		ei := (e["input"].(map[string]interface{}))
		id := ei["trace"].(string)
		if traceTask, found := traceMap[id]; found {
			t.trace = traceTask.trace
		} else {
			fmt.Fprintf(os.Stderr, "Trace %s not found when processing action\n", id)
		}
	}
	tasks = append(tasks, robotTasksPerKind(replayKind, "/replays/", subTaskProc)...)
	tasks = append(tasks, robotTasksPerKind(reportKind, "/reports/", subTaskProc)...)

	return tasks
}

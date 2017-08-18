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
	"reflect"
	"time"

	"os"

	"github.com/google/gapid/test/robot/web/client/widgets/grid"
)

var (
	traceKind  = item{id: "trace"}
	reportKind = item{id: "report"}
	replayKind = item{id: "replay"}

	packageDisplayTemplate = "{{if .information.tag}}{{.information.tag}}" +
		"{{else if and (isUserType .information.type) (.information.cl)}}{{.information.cl}}" +
		"{{else if  .information.uploader}}{{.information.uploader}} - {{.id}}" +
		"{{else}}unknown - {{.id}}" +
		"{{end}}"

	machineDisplayTemplate = "{{if .information.Name}}{{.information.Name}}" +
		"{{else if .information.Configuration.Hardware.Name}}{{.information.Configuration.Hardware.Name}}" +
		"{{else}}{{.information.Configuration.OS}} - {{.information.id.data}}" +
		"{{end}}"

	kindDimension = &dimension{
		name:     "kind",
		enumData: enum{traceKind, replayKind, reportKind},
		valueOf: func(t *task) Item {
			return t.kind
		},
	}
	subjectDimension = &dimension{
		name: "subject",
		valueOf: func(t *task) Item {
			return t.trace.subject
		},
		enumSrc: func() enum {
			return itemGetter("{{.id}}", "{{.Information.APK.package}}", template.FuncMap{})(queryArray("/subjects/"))
		},
	}
	targetDimension = &dimension{
		name: "traceTarget",
		valueOf: func(t *task) Item {
			return t.trace.target
		},
		enumSrc: func() enum {
			return itemGetter("{{.id}}", machineDisplayTemplate, template.FuncMap{})(queryArray("/devices/"))
		},
	}
	hostDimension = &dimension{
		name: "host",
		valueOf: func(t *task) Item {
			return t.host
		},
		enumSrc: func() enum {
			return itemGetter("{{.id}}", machineDisplayTemplate, template.FuncMap{})(queryArray("/devices/"))
		},
	}
	packageDimension = &dimension{
		name: "package",
		valueOf: func(t *task) Item {
			return t.pkg
		},
		enumSrc: func() enum {
			return itemGetter("{{.id}}", packageDisplayTemplate, template.FuncMap{"isUserType": isUserType})(queryArray("/packages/"))
		},
	}

	dimensions = []*dimension{kindDimension, subjectDimension, targetDimension, hostDimension, packageDimension}
)

func isUserType(t reflect.Value) bool {
	// cannot currently use build.Type_UserType to check the type need to fix that.
	return t.Kind() == reflect.Float64 && t.Float() == float64(2)
}

func itemGetter(idPattern string, displayPattern string, functions template.FuncMap) func([]interface{}) enum {
	mustTemplate := func(s string) *template.Template {
		return template.Must(template.New(fmt.Sprintf("t%d", time.Now().Unix())).Funcs(functions).Parse(s))
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
		trace:      traceInfo{target: nilItem, pkg: nilItem, subject: nilItem},
		kind:       kind,
		host:       nilItem,
		pkg:        nilItem,
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
			target:  targetDimension.GetItem(e["target"]),
			subject: subjectDimension.GetItem(ei["subject"]),
		}
		t.host = hostDimension.GetItem(e["host"])
		t.pkg = packageDimension.GetItem(ei["package"])
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
		t.host = hostDimension.GetItem(e["host"])
		t.pkg = packageDimension.GetItem(ei["package"])
	}
	tasks = append(tasks, robotTasksPerKind(replayKind, "/replays/", subTaskProc)...)
	tasks = append(tasks, robotTasksPerKind(reportKind, "/reports/", subTaskProc)...)

	return tasks
}

// Copyright (C) 2018 Google Inc.
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

package status

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

const (
	eventDurtationStart = "B"
	eventDurtationEnd   = "E"
	eventInstance       = "i"
	eventMemoryDump     = "v"
)

type traceEvent struct {
	Name      string                 `json:"name,omitempty"`
	ProcessID uint64                 `json:"pid"`
	TaskID    uint64                 `json:"tid"`
	EventType string                 `json:"ph"`
	Timestamp int64                  `json:"ts"`
	Scope     string                 `json:"s,omitempty"`
	Args      map[string]interface{} `json:"args,omitempty"`
	ID        string                 `json:"id,omitempty"`
}

// RegisterTracer registers a status listener that writes all status updates in
// the Chrome Trace Event Format to the writer w.
// See https://docs.google.com/document/d/1CvAClvFfyA5R-PhYUmn5OOQtYMH4h6I0nSsKchNAySU
// for documentation on the trace event format.
func RegisterTracer(w io.Writer) Unregister {
	l := &statusTracer{
		writer:    w,
		processID: os.Getpid(),
		start:     time.Now(),
		taskIDs:   map[*Task]uint64{},
	}
	w.Write([]byte("["))

	app.Traverse(l.begin)

	return RegisterListener(l)
}

type statusTracer struct {
	writer          io.Writer
	start           time.Time
	processID       int
	taskIDs         map[*Task]uint64
	freeTaskIDs     []uint64
	nextAllocTaskID uint64
	nextMemoryID    uint64
	mutex           sync.Mutex
}

func (s *statusTracer) allocTaskID() uint64 {
	if len(s.freeTaskIDs) > 0 {
		tid := s.freeTaskIDs[len(s.freeTaskIDs)-1]
		s.freeTaskIDs = s.freeTaskIDs[:len(s.freeTaskIDs)-1]
		return tid
	}

	tid := s.nextAllocTaskID
	s.nextAllocTaskID++
	return tid
}

func (s *statusTracer) freeTaskID(tid uint64) {
	s.freeTaskIDs = append(s.freeTaskIDs, tid)
}

func (s *statusTracer) OnTaskStart(ctx context.Context, t *Task) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.begin(t)
}

func (s *statusTracer) OnTaskProgress(ctx context.Context, t *Task) {}
func (s *statusTracer) OnTaskBlock(ctx context.Context, t *Task)    {}
func (s *statusTracer) OnTaskUnblock(ctx context.Context, t *Task)  {}
func (s *statusTracer) OnReplayStatusUpdate(ctx context.Context, r *Replay, label uint64, totalInstrs, finishedInstrs uint32) {
}

func (s *statusTracer) OnTaskFinish(ctx context.Context, t *Task) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.end(t)
}

func (s *statusTracer) OnEvent(ctx context.Context, t *Task, n string, scope EventScope) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.event(ctx, t, n, scope)
}

func (s *statusTracer) OnMemorySnapshot(ctx context.Context, stats runtime.MemStats) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.memorySnapshot(ctx, stats)
}

func (s *statusTracer) begin(t *Task) {
	e := traceEvent{
		Name:      t.name,
		EventType: eventDurtationStart,
		Timestamp: time.Since(s.start).Nanoseconds() / 1000,
		ProcessID: uint64(s.processID),
	}

	if t.parent == &app {
		e.TaskID = s.allocTaskID()
	} else {
		e.TaskID = s.taskIDs[t.parent]
	}
	s.taskIDs[t] = e.TaskID

	b, _ := json.Marshal(e)
	s.writer.Write([]byte("\n"))
	s.writer.Write(b)
	s.writer.Write([]byte(","))
}

func (s *statusTracer) end(t *Task) {
	e := traceEvent{
		Name:      "",
		TaskID:    s.taskIDs[t],
		EventType: eventDurtationEnd,
		Timestamp: time.Since(s.start).Nanoseconds() / 1000,
		ProcessID: uint64(s.processID),
	}
	delete(s.taskIDs, t)

	if t.parent == &app {
		s.freeTaskID(e.TaskID)
	}

	b, _ := json.Marshal(e)
	s.writer.Write([]byte("\n"))
	s.writer.Write(b)
	s.writer.Write([]byte(","))
}

func (s *statusTracer) event(ctx context.Context, t *Task, n string, scope EventScope) {
	tid := uint64(0)
	sc := "g"
	switch scope {
	case TaskScope:
		tid, sc = t.id, "t"
	case GlobalScope:
		sc = "g"
	case ProcessScope:
		sc = "p"
	}
	e := traceEvent{
		Name:      n,
		TaskID:    tid,
		EventType: eventInstance,
		Timestamp: time.Since(s.start).Nanoseconds() / 1000,
		ProcessID: uint64(s.processID),
		Scope:     sc,
	}
	b, _ := json.Marshal(e)
	s.writer.Write([]byte("\n"))
	s.writer.Write(b)
	s.writer.Write([]byte(","))
}

func (s *statusTracer) memorySnapshot(ctx context.Context, stats runtime.MemStats) {
	id := s.nextMemoryID
	s.nextMemoryID++

	processTotals := make(map[string]string)
	processTotals["heap_in_use"] = fmt.Sprintf("%x", stats.Alloc)
	processTotals["heap_virtual_memory"] = fmt.Sprintf("%x", stats.HeapSys)
	processTotals["heap_releasable_memory"] = fmt.Sprintf("%x", stats.HeapIdle-stats.HeapReleased)
	processTotals["gc_memory"] = fmt.Sprintf("%x", stats.GCSys)

	dumps := make(map[string]interface{})
	dumps["process_totals"] = processTotals
	e := traceEvent{
		Name:      "periodic_interval",
		EventType: eventMemoryDump,
		Timestamp: time.Since(s.start).Nanoseconds() / 1000,
		ProcessID: uint64(s.processID),
		ID:        fmt.Sprintf("mem%+v", id),
		Args:      map[string]interface{}{"dumps": dumps},
	}
	b, _ := json.Marshal(e)
	s.writer.Write([]byte("\n"))
	s.writer.Write(b)
	s.writer.Write([]byte(","))
}

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
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/service"
)

type statusVerb struct {
	StatusFlags
}

func init() {
	verb := &statusVerb{}
	verb.StatusUpdateInterval = 1000

	app.AddVerb(&app.Verb{
		Name:      "status",
		ShortHelp: "Attaches to an existing gapis, and provides status",
		Action:    verb,
	})
}

func clear() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func readableBytes(nBytes uint64) string {
	suffixes := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	i := 0
	nBytesRemainder := uint64(0)
	for nBytes > 1024 {
		nBytesRemainder = nBytes & 0x3FF
		nBytes >>= 10
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%v%v", nBytes, suffixes[i])
	} else {
		return fmt.Sprintf("%.3f%v", float32(nBytes)+(float32(nBytesRemainder)/1024.0), suffixes[i])
	}
}

func newTask(id, parent uint64, name string, background bool) *tsk {
	return &tsk{
		id:         id,
		parentID:   parent,
		name:       name,
		background: background,
		children:   map[uint64]*tsk{},
	}
}

type tsk struct {
	id         uint64
	parentID   uint64
	name       string
	background bool
	progress   int32
	blocked    bool
	children   map[uint64]*tsk
}

type u64List []uint64

// Len is the number of elements in the collection.
func (s u64List) Len() int { return len(s) }

// Less reports whether the element with
// index i should sort before the element with index j.
func (s u64List) Less(i, j int) bool { return s[i] < s[j] }

// Swap swaps the elements with indexes i and j.
func (s u64List) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (verb *statusVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	client, err := getGapis(ctx, verb.Gapis, GapirFlags{})
	if err != nil {
		return log.Err(ctx, err, "Failed to connect to the GAPIS server")
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.E(ctx, "Error closing client: %v", err)
		}
	}()

	statusMutex := sync.Mutex{}

	ancestors := make(map[uint64][]uint64)
	activeTasks := make(map[uint64]*tsk)
	totalBlocked := 0
	currentMemoryUsage := uint64(0)
	maxMemoryUsage := uint64(0)
	replayTotalInstrs := uint32(0)
	replayFinishedInstrs := uint32(0)

	var findTask func(map[uint64]*tsk, []uint64) *tsk

	findTask = func(base map[uint64]*tsk, indices []uint64) *tsk {
		if next, ok := base[indices[0]]; ok {
			if len(indices) == 1 {
				return next
			}
			return findTask(next.children, indices[1:])
		}
		return nil
	}

	var forLineage func(map[uint64]*tsk, []uint64, func(*tsk))
	forLineage = func(base map[uint64]*tsk, indices []uint64, f func(*tsk)) {
		if nextTsk, ok := base[indices[0]]; ok {
			f(nextTsk)
			if len(indices) == 1 {
				return
			}
			forLineage(nextTsk.children, indices[1:], f)
		}
	}

	var print func(map[uint64]*tsk, int, bool)
	print = func(m map[uint64]*tsk, d int, background bool) {
		keys := u64List{}
		for k := range m {
			keys = append(keys, k)
		}
		sort.Sort(keys)
		for _, k := range keys {
			v := m[k]
			if v.blocked {
				continue
			}
			if v.background != background {
				continue
			}
			tabs := ""
			for i := 0; i < d; i++ {
				tabs = "	" + tabs
			}
			perc := ""
			if v.progress != 0 {
				perc = fmt.Sprintf("  <%d%%>", v.progress)
			}
			fmt.Printf("%s %v%s\n", tabs, v.name, perc)
			print(v.children, d+1, background)
		}
	}

	stopPolling := task.Async(ctx, func(ctx context.Context) error {
		return task.Poll(ctx, time.Duration(verb.StatusUpdateInterval)*time.Millisecond, func(context.Context) error {
			statusMutex.Lock()
			defer statusMutex.Unlock()
			clear()
			fmt.Printf("Memory Usage: %v  Max: %v\n", readableBytes(currentMemoryUsage), readableBytes(maxMemoryUsage))
			fmt.Printf("Active Tasks: \n")
			print(activeTasks, 1, false)
			fmt.Printf("Background Tasks: \n")
			print(activeTasks, 1, true)
			fmt.Printf("Blocked Task Count: %d\n", totalBlocked)
			return nil
		})
	})
	defer stopPolling()

	ec := make(chan error)
	ctx, cancel := context.WithCancel(ctx)
	crash.Go(func() {
		err := client.Status(ctx,
			time.Duration(verb.MemoryUpdateInterval/2)*time.Millisecond,
			time.Duration(verb.StatusUpdateInterval/2)*time.Millisecond,
			func(tu *service.TaskUpdate) {
				statusMutex.Lock()
				defer statusMutex.Unlock()

				if tu.Status == service.TaskStatus_STARTING {
					// If this is a top-level task, add it to our list of top-level tasks.
					if tu.Parent == 0 {
						activeTasks[tu.Id] = newTask(tu.Id, 0, tu.Name, tu.Background)
					} else {
						if p, ok := ancestors[tu.Parent]; ok {
							// If we can find this tasks parent, then add it in the tree
							if a := findTask(activeTasks, append(p, tu.Parent)); a != nil {
								a.children[tu.Id] = newTask(tu.Id, tu.Parent, tu.Name, tu.Background || a.background)
								ans := append([]uint64{}, ancestors[tu.Parent]...)
								ancestors[tu.Id] = append(ans, tu.Parent)
							} else {
								// If we don't have the parent for whatever reason,
								//   treat this as a top-level task.
								activeTasks[tu.Id] = newTask(
									tu.Id,
									0,
									tu.Name,
									tu.Background)
							}
						} else if a, ok := activeTasks[tu.Parent]; ok {
							// If the parent of this task is a top level task,
							//   then add it there.
							a.children[tu.Id] = newTask(
								tu.Id,
								tu.Parent,
								tu.Name,
								tu.Background || a.background)
							ans := append([]uint64{}, ancestors[tu.Parent]...)
							ancestors[tu.Id] = append(ans, tu.Parent)
						} else {
							// Fallback to adding this as its own top-level task.
							activeTasks[tu.Id] = newTask(
								tu.Id,
								0,
								tu.Name,
								tu.Background)
						}
					}
				} else if tu.Status == service.TaskStatus_FINISHED {
					// Remove this from all parents.
					//   Make sure to fix up our "totalBlocked" if our
					//    blocked task finished.
					loc := []uint64{}
					if a, ok := ancestors[tu.Id]; ok {
						loc = a
					}
					loc = append(loc, tu.Id)
					forLineage(activeTasks, loc, func(t *tsk) {
						if t.blocked {
							if totalBlocked > 0 {
								totalBlocked--
							}
						}
					})
					if len(loc) > 1 {
						// Find the parent, and remove us
						if t := findTask(activeTasks, loc[:len(loc)-1]); t != nil {
							delete(t.children, tu.Id)
						}
					} else {
						delete(activeTasks, tu.Id)
					}
				} else if tu.Status == service.TaskStatus_PROGRESS {
					// Simply update the progress for our task
					loc := []uint64{}
					if a, ok := ancestors[tu.Id]; ok {
						loc = a
					}
					loc = append(loc, tu.Id)
					if a := findTask(activeTasks, loc); a != nil {
						a.progress = tu.CompletePercent
					}
				} else if tu.Status == service.TaskStatus_BLOCKED {
					// If a task becomes blocked, then we should block
					//  it and all of its ancestors.
					loc := []uint64{}
					if a, ok := ancestors[tu.Id]; ok {
						loc = a
					}
					loc = append(loc, tu.Id)
					forLineage(activeTasks, loc, func(t *tsk) {
						totalBlocked++
						t.blocked = true
					})
				} else if tu.Status == service.TaskStatus_UNBLOCKED {
					// If a task becomes unblocked, then we should unblock
					//  it and all of its ancestors.
					loc := []uint64{}
					if a, ok := ancestors[tu.Id]; ok {
						loc = a
					}
					loc = append(loc, tu.Id)
					forLineage(activeTasks, loc, func(t *tsk) {
						if totalBlocked > 0 {
							totalBlocked--
						}
						t.blocked = false
					})
				} else if tu.Status == service.TaskStatus_EVENT {
					fmt.Printf("EVENT--> %+v\n", tu.Event)
				}
			}, func(tu *service.MemoryStatus) {
				if tu.TotalHeap > maxMemoryUsage {
					maxMemoryUsage = tu.TotalHeap
				}
				currentMemoryUsage = tu.TotalHeap
			}, func(tu *service.ReplayUpdate) {
				replayTotalInstrs = tu.TotalInstrs
				replayFinishedInstrs = tu.FinishedInstrs
			})
		ec <- err
	})

	var sigChan chan os.Signal
	sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	select {
	case <-sigChan:
		cancel()
	case err := <-ec:
		if err != nil {
			return log.Err(ctx, err, "Failed to connect to the GAPIS status stream")
		}
	}
	return nil
}

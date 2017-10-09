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

package master

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event"
	"github.com/google/gapid/test/robot/search"
	"github.com/google/gapid/test/robot/search/eval"
)

const masterName = "Master"
const keepaliveFrequency = time.Second * 10

var satelliteClass = reflect.TypeOf(&Satellite{})

type local struct {
	satelliteLock sync.Mutex
	satellites    []*satellite
	nextID        int32
	keepalive     *time.Ticker
	onChange      event.Broadcast
}

// NewLocal creates a new local Master that manages it's own satellites.
func NewLocal(ctx context.Context) Master {
	l := &local{
		keepalive: time.NewTicker(keepaliveFrequency),
	}
	crash.Go(func() { l.run(ctx) })
	return l
}

func (m *local) Close(ctx context.Context) {
	m.keepalive.Stop()
}

// Search implements Master.Search
// It searches the set of active satellites, and supports monitoring of satellites as they start orbiting.
func (m *local) Search(ctx context.Context, query *search.Query, handler SatelliteHandler) error {
	filter := eval.Filter(ctx, query, satelliteClass, event.AsHandler(ctx, handler))
	initial := m.producer(ctx)
	if query.Monitor {
		return event.Monitor(ctx, &m.satelliteLock, m.onChange.Listen, initial, filter)
	}
	return event.Feed(ctx, filter, initial)
}

// Orbit implements Master.Orbit
// It will start orbiting the master, and will not return until it leaves orbit.
func (m *local) Orbit(ctx context.Context, services ServiceList, commands CommandHandler) error {
	sat := m.addSatellite(ctx, services)
	defer m.removeSatellite(ctx, sat)
	crash.Go(func() {
		sat.sendCommand(ctx, &Command{Do: &Command_Identify{Identify: &Identify{Name: sat.info.Name}}})
	})
	sat.processCommands(ctx, commands)
	return nil
}

// Shutdown implements Master.Shutdown
// It broadcasts the shutdown message to all satellites currently orbiting the master.
func (m *local) Shutdown(ctx context.Context, request *ShutdownRequest) (*ShutdownResponse, error) {
	command := &Command{Do: &Command_Shutdown{Shutdown: request.Shutdown}}
	response := &ShutdownResponse{}
	return response, m.broadcast(ctx, command, request.To)
}

func (m *local) getSatellites() []*satellite {
	m.satelliteLock.Lock()
	defer m.satelliteLock.Unlock()
	return append([]*satellite(nil), m.satellites...)
}

func (m *local) producer(ctx context.Context) event.Producer {
	i := 0
	return func(ctx context.Context) interface{} {
		if i >= len(m.satellites) {
			return nil
		}
		res := m.satellites[i]
		i++
		return res
	}
}

func (m *local) addSatellite(ctx context.Context, services ServiceList) *satellite {
	m.satelliteLock.Lock()
	defer m.satelliteLock.Unlock()
	// generate the name and modify the list under the lock
	name := ""
	switch {
	case services.Master:
		name = fmt.Sprintf("Master_%d", m.nextID)
	case services.Worker:
		name = fmt.Sprintf("Worker_%d", m.nextID)
	case services.Web:
		name = fmt.Sprintf("Web_%d", m.nextID)
	}
	m.nextID++
	sat := newSatellite(ctx, name, services)
	m.satellites = append(m.satellites, sat)
	return sat
}

func (m *local) removeSatellite(ctx context.Context, sat *satellite) error {
	m.satelliteLock.Lock()
	defer m.satelliteLock.Unlock()
	// find and remove the satellite from the list
	for i, e := range m.satellites {
		if e == sat {
			m.satellites = append(m.satellites[:i], m.satellites[i+1:]...)
			break
		}
	}
	return nil
}

func serverInList(name string, servers []string) bool {
	if len(servers) == 0 {
		return true
	}
	for _, s := range servers {
		if s == name {
			return true
		}
	}
	return false
}

func (m *local) broadcast(ctx context.Context, command *Command, to []string) error {
	// First send the command to all the registered services that match the to list
	for _, sat := range m.getSatellites() {
		if !serverInList(sat.info.Name, to) {
			continue
		}
		sat.sendCommand(ctx, command)
	}
	return nil
}

func (m *local) run(ctx context.Context) {
	ping := &Command{Do: &Command_Ping{Ping: &Ping{}}}
	// each time a tick happens
	for _ = range m.keepalive.C {
		// ping all the satellites to see if they are still up
		for _, sat := range m.getSatellites() {
			sat.sendCommand(ctx, ping)
		}
	}
}

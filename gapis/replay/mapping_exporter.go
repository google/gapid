// Copyright (C) 2019 Google Inc.
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

package replay

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"
	"sort"
	"sync"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapir"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/service"
)

type MappingExporter struct {
	mappings *map[uint64][]service.VulkanHandleMappingItem
	thread   uint64
	path     string
	results  map[uint64]service.VulkanHandleMappingItem
	mutex    sync.RWMutex
}

func NewMappingExporter(ctx context.Context, mappings *map[uint64][]service.VulkanHandleMappingItem) *MappingExporter {
	return &MappingExporter{
		mappings: mappings,
		thread:   0,
		path:     "",
		results:  make(map[uint64]service.VulkanHandleMappingItem),
		mutex:    sync.RWMutex{},
	}
}

func NewMappingExporterWithPrint(ctx context.Context, path string) *MappingExporter {
	mapping := make(map[uint64][]service.VulkanHandleMappingItem)
	return &MappingExporter{
		mappings: &mapping,
		thread:   0,
		path:     path,
		results:  make(map[uint64]service.VulkanHandleMappingItem),
		mutex:    sync.RWMutex{},
	}
}

func (m *MappingExporter) processNotification(ctx context.Context, s *api.GlobalState, n gapir.Notification) {
	notificationData := n.GetData()
	notificationID := n.GetId()
	mappingData := notificationData.GetData()

	m.mutex.Lock()

	result, ok := m.results[notificationID]
	if !ok {
		log.I(ctx, "Invalid notificationID %d", notificationID)
		return
	}

	byteOrder := s.MemoryLayout.GetEndian()
	r := endian.Reader(bytes.NewReader(mappingData), byteOrder)
	replayValue := r.Uint64()
	result.ReplayValue = replayValue

	if _, ok := (*m.mappings)[replayValue]; !ok {
		(*m.mappings)[replayValue] = make([]service.VulkanHandleMappingItem, 0, 0)
	}

	(*m.mappings)[replayValue] = append((*m.mappings)[replayValue], result)

	delete(m.results, notificationID)

	m.mutex.Unlock()

	if len(m.results) == 0 {
		if len(m.path) > 0 {
			printToFile(ctx, m.path, m.mappings)
		}
	}
}

func (m *MappingExporter) ExtractRemappings(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
	for k, v := range b.Remappings {
		typ := reflect.TypeOf(k)
		var size uint64
		if t, ok := k.(memory.SizedTy); ok {
			size = t.TypeSize(s.MemoryLayout)
		} else {
			size = uint64(typ.Size())
		}
		if size != 1 && size != 2 && size != 4 && size != 8 {
			// Ignore objects that are not handles
			continue
		}

		notificationID := b.GetNotificationID()
		traceValue := reflect.ValueOf(k).Uint()
		m.results[notificationID] = service.VulkanHandleMappingItem{HandleType: typ.Name(), TraceValue: traceValue, ReplayValue: 0}
		b.Notification(notificationID, v, size)
		err := b.RegisterNotificationReader(notificationID, func(n gapir.Notification) {
			m.processNotification(ctx, s, n)
		})

		if err != nil {
			log.W(ctx, "Vulkan Mapping Notification could not registered: ", err)
			return err
		}
	}

	return nil
}

func (m *MappingExporter) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	m.thread = cmd.Thread()
	out.MutateAndWrite(ctx, id, cmd)
}

func printToFile(ctx context.Context, path string, mappings *map[uint64][]service.VulkanHandleMappingItem) {
	f, err := os.Create(path)
	if err != nil {
		log.E(ctx, "Failed to create mapping file %v: %v", path, err)
		return
	}

	mappingsToFile := make([]string, 0, 0)

	for _, v := range *mappings {
		for i := range v {
			m := v[i]
			mappingsToFile = append(mappingsToFile, fmt.Sprintf("%v(%v): %v\n", m.HandleType, m.TraceValue, m.ReplayValue))
		}
	}

	sort.Strings(mappingsToFile)
	for _, l := range mappingsToFile {
		fmt.Fprint(f, l)
	}
	f.Close()
}

func (m *MappingExporter) Flush(ctx context.Context, out transform.Writer) {
	out.MutateAndWrite(ctx, api.CmdNoID, Custom{m.thread,
		func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
			return m.ExtractRemappings(ctx, s, b)
		},
	})
}

func (m *MappingExporter) PreLoop(ctx context.Context, out transform.Writer) {}
func (m *MappingExporter) PostLoop(ctx context.Context, out transform.Writer) {
	out.MutateAndWrite(ctx, api.CmdNoID, Custom{m.thread,
		func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
			return m.ExtractRemappings(ctx, s, b)
		},
	})
}

func (t *MappingExporter) BuffersCommands() bool {
	return false
}

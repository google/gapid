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

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapir"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
	"github.com/google/gapid/gapis/service"
)

type mappingHandle struct {
	traceValue    uint64
	replayAddress value.Pointer
	size          uint64
	name          string
}

type MappingExporter struct {
	mappings       *map[uint64][]service.VulkanHandleMappingItem
	thread         uint64
	path           string
	traceValues    []mappingHandle
	notificationID uint64
}

func NewMappingExporter(ctx context.Context, mappings *map[uint64][]service.VulkanHandleMappingItem) *MappingExporter {
	return &MappingExporter{
		mappings:       mappings,
		thread:         0,
		path:           "",
		traceValues:    make([]mappingHandle, 0, 0),
		notificationID: 0,
	}
}

func NewMappingExporterWithPrint(ctx context.Context, path string) *MappingExporter {
	mapping := make(map[uint64][]service.VulkanHandleMappingItem)
	return &MappingExporter{
		mappings:       &mapping,
		thread:         0,
		path:           path,
		traceValues:    make([]mappingHandle, 0, 0),
		notificationID: 0,
	}
}

func (m *MappingExporter) processNotification(ctx context.Context, s *api.GlobalState, n gapir.Notification) {
	if m.notificationID != n.GetId() {
		log.I(ctx, "Invalid notificationID %d", m.notificationID)
		return
	}

	notificationData := n.GetData()
	mappingData := notificationData.GetData()

	byteOrder := s.MemoryLayout.GetEndian()
	r := endian.Reader(bytes.NewReader(mappingData), byteOrder)

	for _, handle := range m.traceValues {
		var replayValue uint64
		switch handle.size {
		case 1:
			replayValue = uint64(r.Uint8())
		case 2:
			replayValue = uint64(r.Uint16())
		case 4:
			replayValue = uint64(r.Uint32())
		case 8:
			replayValue = r.Uint64()
		default:
			log.F(ctx, true, "Invalid Handle size %s: %d", handle.name, handle.size)
		}

		if _, ok := (*m.mappings)[replayValue]; !ok {
			(*m.mappings)[replayValue] = make([]service.VulkanHandleMappingItem, 0, 0)
		}

		(*m.mappings)[replayValue] = append(
			(*m.mappings)[replayValue],
			service.VulkanHandleMappingItem{HandleType: handle.name, TraceValue: handle.traceValue, ReplayValue: replayValue},
		)
	}

	m.notificationID = 0

	if len(m.path) > 0 {
		printToFile(ctx, m.path, m.mappings)
	}
}

func (m *MappingExporter) ExtractRemappings(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
	bufferSize := uint64(0)

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

		traceValue := reflect.ValueOf(k).Uint()
		m.traceValues = append(m.traceValues, mappingHandle{traceValue: traceValue, replayAddress: v, size: size, name: typ.Name()})
		bufferSize += size
	}

	handleBuffer := b.AllocateMemory(bufferSize)
	target := handleBuffer

	for _, handle := range m.traceValues {
		b.Memcpy(target, handle.replayAddress, handle.size)
		target = target.Offset(handle.size)
	}

	m.notificationID = b.GetNotificationID()
	b.Notification(m.notificationID, handleBuffer, bufferSize)
	err := b.RegisterNotificationReader(m.notificationID, func(n gapir.Notification) {
		m.processNotification(ctx, s, n)
	})

	if err != nil {
		log.W(ctx, "Vulkan Mapping Notification could not be registered: ", err)
		return err
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

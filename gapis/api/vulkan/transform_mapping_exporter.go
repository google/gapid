// Copyright (C) 2020 Google Inc.
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

package vulkan

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

var _ transform.Transform = &mappingExporter{}

type mappingHandle struct {
	traceValue    uint64
	replayAddress value.Pointer
	size          uint64
	name          string
}

type mappingExporter struct {
	mappings       *map[uint64][]service.VulkanHandleMappingItem
	thread         uint64
	path           string
	traceValues    []mappingHandle
	notificationID uint64
}

func newMappingExporter(ctx context.Context, mappings *map[uint64][]service.VulkanHandleMappingItem) *mappingExporter {
	return &mappingExporter{
		mappings:       mappings,
		thread:         0,
		path:           "",
		traceValues:    make([]mappingHandle, 0, 0),
		notificationID: 0,
	}
}

func newMappingExporterWithPrint(ctx context.Context, path string) *mappingExporter {
	mapping := make(map[uint64][]service.VulkanHandleMappingItem)
	return &mappingExporter{
		mappings:       &mapping,
		thread:         0,
		path:           path,
		traceValues:    make([]mappingHandle, 0, 0),
		notificationID: 0,
	}
}

func (mappingTransform *mappingExporter) RequiresAccurateState() bool {
	return false
}

func (mappingTransform *mappingExporter) RequiresInnerStateMutation() bool {
	return false
}

func (mappingTransform *mappingExporter) SetInnerStateMutationFunction(mutator transform.StateMutator) {
	// This transform do not require inner state mutation
}

func (mappingTransform *mappingExporter) BeginTransform(ctx context.Context, inputState *api.GlobalState) error {
	return nil
}

func (mappingTransform *mappingExporter) EndTransform(ctx context.Context, inputState *api.GlobalState) ([]api.Cmd, error) {
	cb := CommandBuilder{Thread: mappingTransform.thread, Arena: inputState.Arena}
	newCmd := cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
		return mappingTransform.extractRemappings(ctx, s, b)
	})

	return []api.Cmd{newCmd}, nil
}

func (mappingTransform *mappingExporter) ClearTransformResources(ctx context.Context) {
	// No resource allocated
}

func (mappingTransform *mappingExporter) TransformCommand(ctx context.Context, id transform.CommandID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	if mappingTransform.thread == 0 && len(inputCommands) > 0 {
		mappingTransform.thread = inputCommands[0].Thread()
	}

	return inputCommands, nil
}

func (mappingTransform *mappingExporter) extractRemappings(ctx context.Context, inputState *api.GlobalState, b *builder.Builder) error {
	bufferSize := uint64(0)

	for k, v := range b.Remappings {
		typ := reflect.TypeOf(k)
		size := memory.SizeOf(typ, inputState.MemoryLayout)

		if size != 1 && size != 2 && size != 4 && size != 8 {
			// Ignore objects that are not handles
			continue
		}

		traceValue := reflect.ValueOf(k).Uint()
		mappingTransform.traceValues = append(mappingTransform.traceValues, mappingHandle{traceValue: traceValue, replayAddress: v, size: size, name: typ.Name()})
		bufferSize += size
	}

	handleBuffer := b.AllocateMemory(bufferSize)
	target := handleBuffer

	for _, handle := range mappingTransform.traceValues {
		b.Memcpy(target, handle.replayAddress, handle.size)
		target = target.Offset(handle.size)
	}

	mappingTransform.notificationID = b.GetNotificationID()
	b.Notification(mappingTransform.notificationID, handleBuffer, bufferSize)
	err := b.RegisterNotificationReader(mappingTransform.notificationID, func(n gapir.Notification) {
		mappingTransform.processNotification(ctx, inputState, n)
	})

	if err != nil {
		log.W(ctx, "Vulkan Mapping Notification could not be registered: ", err)
		return err
	}

	return nil
}

func (mappingTransform *mappingExporter) processNotification(ctx context.Context, inputState *api.GlobalState, notification gapir.Notification) {
	if mappingTransform.notificationID != notification.GetId() {
		log.I(ctx, "Invalid notificationID %d", mappingTransform.notificationID)
		return
	}

	notificationData := notification.GetData()
	mappingData := notificationData.GetData()

	byteOrder := inputState.MemoryLayout.GetEndian()
	reader := endian.Reader(bytes.NewReader(mappingData), byteOrder)

	for _, handle := range mappingTransform.traceValues {
		var replayValue uint64
		switch handle.size {
		case 1:
			replayValue = uint64(reader.Uint8())
		case 2:
			replayValue = uint64(reader.Uint16())
		case 4:
			replayValue = uint64(reader.Uint32())
		case 8:
			replayValue = reader.Uint64()
		default:
			log.F(ctx, true, "Invalid Handle size %s: %d", handle.name, handle.size)
		}

		if _, ok := (*mappingTransform.mappings)[replayValue]; !ok {
			(*mappingTransform.mappings)[replayValue] = make([]service.VulkanHandleMappingItem, 0, 0)
		}

		(*mappingTransform.mappings)[replayValue] = append(
			(*mappingTransform.mappings)[replayValue],
			service.VulkanHandleMappingItem{HandleType: handle.name, TraceValue: handle.traceValue, ReplayValue: replayValue},
		)
	}

	mappingTransform.notificationID = 0

	if len(mappingTransform.path) > 0 {
		printToFile2(ctx, mappingTransform.path, mappingTransform.mappings)
	}
}

func printToFile2(ctx context.Context, path string, mappings *map[uint64][]service.VulkanHandleMappingItem) {
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

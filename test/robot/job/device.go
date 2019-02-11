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

package job

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/google/gapid/core/event"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/test/robot/record"
	"github.com/google/gapid/test/robot/search"
	"github.com/google/gapid/test/robot/search/eval"
)

type devices struct {
	mu       sync.Mutex
	ledger   record.Ledger
	entries  []*Device
	byID     map[string]*Device
	onChange event.Broadcast
}

func (l *devices) init(ctx context.Context, library record.Library) error {
	ledger, err := library.Open(ctx, "devices", &Device{})
	if err != nil {
		return err
	}
	l.ledger = ledger
	l.byID = map[string]*Device{}
	apply := event.AsHandler(ctx, l.apply)
	if err := ledger.Read(ctx, apply); err != nil {
		return err
	}
	ledger.Watch(ctx, apply)
	return nil
}

func (l *devices) apply(ctx context.Context, entry *Device) error {
	l.entries = append(l.entries, entry)
	l.byID[entry.Id] = entry
	l.onChange.Send(ctx, entry)
	return nil
}

func (l *devices) search(ctx context.Context, query *search.Query, handler DeviceHandler) error {
	filter := eval.Filter(ctx, query, reflect.TypeOf(&Device{}), event.AsHandler(ctx, handler))
	initial := event.AsProducer(ctx, l.entries)
	if query.Monitor {
		return event.Monitor(ctx, &l.mu, l.onChange.Listen, initial, filter)
	}
	return event.Feed(ctx, filter, initial)
}

func (l *devices) uniqueName(ctx context.Context, name string) string {
	index := 0
	id := name
	for {
		if _, found := l.byID[id]; !found {
			return id
		}
		index++
		id = fmt.Sprintf("%s-%d", name, index)
	}
}

func sameDevice(match *Device, info *device.Instance) bool {
	if match == nil {
		return info == nil
	}
	if info == nil {
		return false
	}
	return match.Information.SameAs(info)
}

func (l *devices) find(ctx context.Context, info *device.Instance) *Device {
	for _, entry := range l.entries {
		if sameDevice(entry, info) {
			return entry
		}
	}
	return nil
}

func (l *devices) get(ctx context.Context, info *device.Instance) (*Device, error) {
	if info == nil {
		return nil, nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := l.find(ctx, info)
	if entry == nil {
		entry = &Device{
			Id:          l.uniqueName(ctx, info.ID.ID().String()),
			Information: info,
		}
		if err := l.ledger.Add(ctx, entry); err != nil {
			return nil, err
		}
	} else {
		entry.Information = info
	}
	return entry, nil
}

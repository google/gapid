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

package service

import (
	"bytes"
	"context"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/stringtable"
)

// TODO: Move to the builder package.

// Msg constructs stringtable.Msg instance from a MsgRef.
func (r *Report) Msg(ref *MsgRef) *stringtable.Msg {
	m := &stringtable.Msg{
		Identifier: r.Strings[ref.Identifier],
		Arguments:  map[string]*stringtable.Value{},
	}
	for _, arg := range ref.Arguments {
		m.Arguments[r.Strings[arg.Key]] = r.Values[arg.Value]
	}
	return m
}

// key returns a string that uniquely identifies the message content.
func (r *MsgRef) key() string {
	b := bytes.Buffer{}
	b.WriteRune(rune(r.Identifier))
	b.WriteRune(':')
	for _, arg := range r.Arguments {
		b.WriteRune(rune(arg.Key))
		b.WriteRune(':')
		b.WriteRune(rune(arg.Value))
	}
	return b.String()
}

// ReportBuilder helps construct reports.
type ReportBuilder struct {
	report  *Report                // Report to build.
	strings map[string]uint32      // Map of string to index in Report.strings.
	values  map[interface{}]uint32 // Map of value hash to index in Report.values.
}

// ReportItemRaw represents ReportItem, raw message and array of raw tags.
type ReportItemRaw struct {
	Item    *ReportItem
	Message *stringtable.Msg
	Tags    []*stringtable.Msg
}

// NewReportBuilder creates and initializes new report builder.
func NewReportBuilder() *ReportBuilder {
	builder := &ReportBuilder{}
	builder.report = &Report{}
	builder.strings = map[string]uint32{}
	builder.values = map[interface{}]uint32{}
	return builder
}

// Add processes tags, adds references to item and adds item to report.
func (b *ReportBuilder) Add(ctx context.Context, element *ReportItemRaw) {
	if err := b.processMessages(element.Item, element.Message, element.Tags); err == nil {
		b.report.Items = append(b.report.Items, element.Item)
	} else {
		log.E(ctx, "Error %v during adding an item to a report", err)
	}
}

// Build performs final processing and returns report.
func (b *ReportBuilder) Build() *Report {
	if err := b.processGroups(); err != nil {
		return nil
	}
	return b.report
}

// WrapReportItem wraps ReportItem into raw representation of ReportItemTagged
// which contains raw messages instead of references.
func WrapReportItem(item *ReportItem, m *stringtable.Msg) *ReportItemRaw {
	return &ReportItemRaw{
		Item:    item,
		Message: m,
	}
}

// processMessages checks if message's and tags' identifier and arguments
// are unique and treat them appropriately. It also adds message references
// to report item.
func (b *ReportBuilder) processMessages(item *ReportItem, message *stringtable.Msg, tags []*stringtable.Msg) error {
	ref, err := b.processMessage(message)
	if err != nil {
		return err
	}
	item.Message = ref
	for _, m := range tags {
		ref, err = b.processMessage(m)
		if err != nil {
			return err
		}
		item.Tags = append(item.Tags, ref)
	}
	return nil
}

func (b *ReportBuilder) getOrAddString(s string) uint32 {
	i, ok := b.strings[s]
	if !ok {
		i = uint32(len(b.report.Strings))
		b.strings[s] = i
		b.report.Strings = append(b.report.Strings, s)
	}
	return i
}

func (b *ReportBuilder) getOrAddValue(v *stringtable.Value) uint32 {
	key := v.Unpack()
	i, ok := b.values[key]
	if !ok {
		i = uint32(len(b.report.Values))
		b.values[key] = i
		b.report.Values = append(b.report.Values, v)
	}
	return i
}

func (b *ReportBuilder) processMessage(msg *stringtable.Msg) (*MsgRef, error) {
	ref := &MsgRef{
		Identifier: b.getOrAddString(msg.Identifier),
		Arguments:  make([]*MsgRefArgument, 0, len(msg.Arguments)),
	}
	for k, v := range msg.Arguments {
		ref.Arguments = append(ref.Arguments, &MsgRefArgument{
			Key:   b.getOrAddString(k),
			Value: b.getOrAddValue(v),
		})
	}
	return ref, nil
}

// processGroups forms group iterating over ReportItem and comparing their messages.
func (b *ReportBuilder) processGroups() error {
	groupItems := map[string][]uint32{}
	groups := map[string]*MsgRef{}
	emitted := map[string]bool{}

	for i, item := range b.report.Items {
		ref := item.Message
		key := ref.key()
		if _, ok := groupItems[key]; ok {
			groupItems[key] = append(groupItems[key], uint32(i))
		} else {
			groupItems[key] = []uint32{uint32(i)}
			groups[key] = ref
		}
	}

	for _, item := range b.report.Items {
		key := item.Message.key()
		if _, ok := emitted[key]; !ok {
			emitted[key] = true
			b.report.Groups = append(b.report.Groups, &ReportGroup{
				Name:  groups[key],
				Items: groupItems[key],
			})
		}
	}

	return nil
}

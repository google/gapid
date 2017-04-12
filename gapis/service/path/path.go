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

package path

import (
	"fmt"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/protoutil"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapis/service/pod"
)

// Node is the interface for types that represent a reference to a capture,
// atom list, single atom, memory, state or sub-object. A path can be
// passed between client and server using RPCs in order to describe some data
// in a capture.
type Node interface {
	// Text returns the string representation of the path.
	// The returned string must be consistent for equal paths.
	Text() string

	// Parent returns the path that this path derives from.
	// If this path is a root, then Base returns nil.
	Parent() Node

	// Path returns this path node as a path.
	Path() *Any

	// Clone returns a deep-copy of the path.
	// Clone() Any

	// Validate checks the path for correctness, returning an error if any
	// issues are found.
	// Validate() error
}

func (n *ArrayIndex) Path() *Any   { return &Any{&Any_ArrayIndex{n}} }
func (n *As) Path() *Any           { return &Any{&Any_As{n}} }
func (n *Blob) Path() *Any         { return &Any{&Any_Blob{n}} }
func (n *Capture) Path() *Any      { return &Any{&Any_Capture{n}} }
func (n *Command) Path() *Any      { return &Any{&Any_Command{n}} }
func (n *Commands) Path() *Any     { return &Any{&Any_Commands{n}} }
func (n *Context) Path() *Any      { return &Any{&Any_Context{n}} }
func (n *Contexts) Path() *Any     { return &Any{&Any_Contexts{n}} }
func (n *Device) Path() *Any       { return &Any{&Any_Device{n}} }
func (n *Field) Path() *Any        { return &Any{&Any_Field{n}} }
func (n *Hierarchies) Path() *Any  { return &Any{&Any_Hierarchies{n}} }
func (n *Hierarchy) Path() *Any    { return &Any{&Any_Hierarchy{n}} }
func (n *ImageInfo) Path() *Any    { return &Any{&Any_ImageInfo{n}} }
func (n *MapIndex) Path() *Any     { return &Any{&Any_MapIndex{n}} }
func (n *Memory) Path() *Any       { return &Any{&Any_Memory{n}} }
func (n *Mesh) Path() *Any         { return &Any{&Any_Mesh{n}} }
func (n *Parameter) Path() *Any    { return &Any{&Any_Parameter{n}} }
func (n *Report) Path() *Any       { return &Any{&Any_Report{n}} }
func (n *ResourceData) Path() *Any { return &Any{&Any_ResourceData{n}} }
func (n *Resources) Path() *Any    { return &Any{&Any_Resources{n}} }
func (n *Slice) Path() *Any        { return &Any{&Any_Slice{n}} }
func (n *State) Path() *Any        { return &Any{&Any_State{n}} }
func (n *Thumbnail) Path() *Any    { return &Any{&Any_Thumbnail{n}} }

func (n ArrayIndex) Parent() Node   { return oneOfNode(n.Array) }
func (n As) Parent() Node           { return oneOfNode(n.From) }
func (n Blob) Parent() Node         { return nil }
func (n Capture) Parent() Node      { return nil }
func (n Command) Parent() Node      { return n.Commands }
func (n Commands) Parent() Node     { return n.Capture }
func (n Context) Parent() Node      { return n.Contexts }
func (n Contexts) Parent() Node     { return n.Capture }
func (n Device) Parent() Node       { return nil }
func (n Field) Parent() Node        { return oneOfNode(n.Struct) }
func (n Hierarchies) Parent() Node  { return n.Capture }
func (n Hierarchy) Parent() Node    { return n.Hierarchies }
func (n ImageInfo) Parent() Node    { return nil }
func (n MapIndex) Parent() Node     { return oneOfNode(n.Map) }
func (n Memory) Parent() Node       { return n.After }
func (n Mesh) Parent() Node         { return oneOfNode(n.Object) }
func (n Parameter) Parent() Node    { return n.Command }
func (n Report) Parent() Node       { return n.Capture }
func (n ResourceData) Parent() Node { return n.After }
func (n Resources) Parent() Node    { return n.Capture }
func (n Slice) Parent() Node        { return oneOfNode(n.Array) }
func (n State) Parent() Node        { return n.After }
func (n Thumbnail) Parent() Node    { return oneOfNode(n.Object) }

func (n ArrayIndex) Text() string  { return fmt.Sprintf("%v[%v]", n.Parent().Text(), n.Index) }
func (n As) Text() string          { return fmt.Sprintf("%v.as<%v>", n.Parent().Text(), protoutil.OneOf(n.To)) }
func (n Blob) Text() string        { return fmt.Sprintf("blob<%x>", n.Id.Data) }
func (n Capture) Text() string     { return fmt.Sprintf("capture<%x>", n.Id.Data) }
func (n Command) Text() string     { return fmt.Sprintf("%v[%v]", n.Parent().Text(), n.Index) }
func (n Commands) Text() string    { return fmt.Sprintf("%v.commands", n.Parent().Text()) }
func (n Context) Text() string     { return fmt.Sprintf("%v[%x]", n.Parent().Text(), n.Id.Data) }
func (n Contexts) Text() string    { return fmt.Sprintf("%v.contexts", n.Parent().Text()) }
func (n Device) Text() string      { return fmt.Sprintf("device<%x>", n.Id.Data) }
func (n Field) Text() string       { return fmt.Sprintf("%v.%v", n.Parent().Text(), n.Name) }
func (n Hierarchies) Text() string { return fmt.Sprintf("%v.hierarchies", n.Parent().Text()) }
func (n Hierarchy) Text() string   { return fmt.Sprintf("%v[%x]", n.Parent().Text(), n.Id.Data) }
func (n ImageInfo) Text() string   { return fmt.Sprintf("image-info<%x>", n.Id.Data) }
func (n MapIndex) Text() string    { return fmt.Sprintf("%v[%x]", n.Parent().Text(), n.Key) }
func (n Memory) Text() string      { return fmt.Sprintf("%v.memory-after", n.Parent().Text()) }
func (n Mesh) Text() string        { return fmt.Sprintf("%v.mesh", n.Parent().Text()) }
func (n Parameter) Text() string   { return fmt.Sprintf("%v.%v", n.Parent().Text(), n.Name) }
func (n Report) Text() string      { return fmt.Sprintf("%v.report", n.Parent().Text()) }
func (n ResourceData) Text() string {
	return fmt.Sprintf("%v.resource-data<%x>", n.Parent().Text(), n.Id.Data)
}
func (n Resources) Text() string { return fmt.Sprintf("%v.resources", n.Parent().Text()) }
func (n Slice) Text() string     { return fmt.Sprintf("%v[%v:%v]", n.Parent().Text(), n.Start, n.End) }
func (n State) Text() string     { return fmt.Sprintf("%v.state-after", n.Parent().Text()) }
func (n Thumbnail) Text() string { return fmt.Sprintf("%v.thumbnail", n.Parent().Text()) }

func (n *ArrayIndex) SetParent(p Node) {
	switch p := p.(type) {
	case *Field:
		n.Array = &ArrayIndex_Field{p}
	case *Slice:
		n.Array = &ArrayIndex_Slice{p}
	case *ArrayIndex:
		n.Array = &ArrayIndex_ArrayIndex{p}
	case *MapIndex:
		n.Array = &ArrayIndex_MapIndex{p}
	case *Parameter:
		n.Array = &ArrayIndex_Parameter{p}
	case *Report:
		n.Array = &ArrayIndex_Report{p}
	default:
		panic(fmt.Errorf("Cannot set ArrayIndex.Array to %T", p))
	}
}

func (n *Field) SetParent(p Node) {
	switch p := p.(type) {
	case *Field:
		n.Struct = &Field_Field{p}
	case *Slice:
		n.Struct = &Field_Slice{p}
	case *ArrayIndex:
		n.Struct = &Field_ArrayIndex{p}
	case *MapIndex:
		n.Struct = &Field_MapIndex{p}
	case *Parameter:
		n.Struct = &Field_Parameter{p}
	case *State:
		n.Struct = &Field_State{p}
	default:
		panic(fmt.Errorf("Cannot set Field.Struct to %T", p))
	}
}

func (n *MapIndex) SetParent(p Node) {
	switch p := p.(type) {
	case *Field:
		n.Map = &MapIndex_Field{p}
	case *Slice:
		n.Map = &MapIndex_Slice{p}
	case *ArrayIndex:
		n.Map = &MapIndex_ArrayIndex{p}
	case *MapIndex:
		n.Map = &MapIndex_MapIndex{p}
	case *Parameter:
		n.Map = &MapIndex_Parameter{p}
	case *State:
		n.Map = &MapIndex_State{p}
	default:
		panic(fmt.Errorf("Cannot set MapIndex.Map to %T", p))
	}
}

func oneOfNode(v interface{}) Node {
	return protoutil.OneOf(v).(Node)
}

// Node returns the path node for p.
func (p *Any) Node() Node { return oneOfNode(p.Path) }

// Text returns the textual representation of the path.
func (p *Any) Text() string { return p.Node().Text() }

// FindCommand traverses the path nodes looking for a Command path node.
// If no Command path node was found then nil is returned.
func FindCommand(n Node) *Command {
	for n != nil {
		if c, ok := n.(*Command); ok {
			return c
		}
		n = n.Parent()
	}
	return nil
}

// FindCapture traverses the path nodes looking for a Capture path node.
// If no Capture path node was found then nil is returned.
func FindCapture(n Node) *Capture {
	for n != nil {
		if c, ok := n.(*Capture); ok {
			return c
		}
		n = n.Parent()
	}
	return nil
}

// NewCapture returns a new Capture path node with the given ID.
func NewCapture(id id.ID) *Capture {
	return &Capture{Id: NewID(id)}
}

// NewDevice returns a new Device path node with the given ID.
func NewDevice(id id.ID) *Device {
	return &Device{Id: NewID(id)}
}

// NewBlob returns a new Blob path node with the given ID.
func NewBlob(id id.ID) *Blob {
	return &Blob{Id: NewID(id)}
}

// NewImageInfo returns a new ImageInfo path node with the given ID.
func NewImageInfo(id id.ID) *ImageInfo {
	return &ImageInfo{Id: image.NewID(id)}
}

// Resources returns the path node to the capture's resources.
func (n *Capture) Resources() *Resources {
	return &Resources{Capture: n}
}

// Report returns the path node to the capture's report.
func (n *Capture) Report(d *Device) *Report {
	return &Report{Capture: n, Device: d}
}

// Contexts returns the path node to the capture's contexts.
func (n *Capture) Contexts() *Contexts {
	return &Contexts{Capture: n}
}

// Hierarchies returns the path node to the capture's hierarchies.
func (n *Capture) Hierarchies() *Hierarchies {
	return &Hierarchies{Capture: n}
}

// Commands returns the path node to the capture's commands.
func (n *Capture) Commands() *Commands {
	return &Commands{Capture: n}
}

// Index returns the path node to a single command in the a list of commands.
func (n *Commands) Index(i uint64) *Command {
	return &Command{Commands: n, Index: i}
}

// MemoryAfter returns the path node to the memory after this command.
func (n *Command) MemoryAfter(pool uint32, addr, size uint64) *Memory {
	return &Memory{
		Address: addr,
		Pool:    pool,
		Size:    size,
		After:   n,
	}
}

func (n *Command) ResourceAfter(id *ID) *ResourceData {
	return &ResourceData{
		Id:    id,
		After: n,
	}
}

// Mesh returns the path node to the mesh of this command.
func (n *Command) Mesh(faceted bool) *Mesh {
	return &Mesh{
		Options: &MeshOptions{faceted},
		Object:  &Mesh_Command{n},
	}
}

// StateAfter returns the path node to the state after this command.
func (n *Command) StateAfter() *State {
	return &State{After: n}
}

func (n *Command) Next() *Command {
	return &Command{Commands: n.Commands, Index: n.Index + 1}
}

// Parameter returns the path node to the parameter with the given name.
func (n *Command) Parameter(name string) *Parameter {
	return &Parameter{Name: name, Command: n}
}

func (n *Parameter) ArrayIndex(i uint64) *ArrayIndex {
	return &ArrayIndex{Index: i, Array: &ArrayIndex_Parameter{n}}
}

func (n *Parameter) Field(name string) *Field {
	return &Field{Name: name, Struct: &Field_Parameter{n}}
}

func (n *Parameter) MapIndex(k interface{}) *MapIndex {
	out := &MapIndex{Map: &MapIndex_Parameter{n}}
	if v := pod.NewValue(k); v != nil {
		out.Key = &MapIndex_Pod{v}
		return out
	}

	panic(fmt.Errorf("Cannot set MapIndex.Key to %T", k))
}

func (n *Parameter) Slice(s, e uint64) *Slice {
	return &Slice{Start: s, End: e, Array: &Slice_Parameter{n}}
}

func (n *Field) ArrayIndex(i uint64) *ArrayIndex {
	return &ArrayIndex{Index: i, Array: &ArrayIndex_Field{n}}
}

func (n *Field) Field(name string) *Field {
	return &Field{Name: name, Struct: &Field_Field{n}}
}

func (n *Field) MapIndex(k interface{}) *MapIndex {
	out := &MapIndex{Map: &MapIndex_Field{n}}
	if v := pod.NewValue(k); v != nil {
		out.Key = &MapIndex_Pod{v}
		return out
	}

	panic(fmt.Errorf("Cannot set MapIndex.Key to %T", k))
}

func (n *Field) Slice(s, e uint64) *Slice {
	return &Slice{Start: s, End: e, Array: &Slice_Field{n}}
}

func (n *State) Field(name string) *Field {
	return &Field{Name: name, Struct: &Field_State{n}}
}

func (n *MapIndex) Field(name string) *Field {
	return &Field{Name: name, Struct: &Field_MapIndex{n}}
}

func (n *MapIndex) KeyValue() interface{} {
	switch k := protoutil.OneOf(n.Key).(type) {
	case nil:
		return nil
	case *pod.Value:
		return k.Get()
	default:
		panic(fmt.Errorf("Unsupport MapIndex key type %T", k))
	}
}

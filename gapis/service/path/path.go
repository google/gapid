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
	"math"
	"strings"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/protoutil"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapis/service/box"
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

	// Validate checks the path for correctness, returning an error if any
	// issues are found.
	Validate() error
}

func printIndices(index []uint64) string {
	parts := make([]string, len(index))
	for i, v := range index {
		parts[i] = fmt.Sprint(v)
	}
	return strings.Join(parts, ".")
}

func (n *API) Path() *Any                       { return &Any{&Any_Api{n}} }
func (n *ArrayIndex) Path() *Any                { return &Any{&Any_ArrayIndex{n}} }
func (n *As) Path() *Any                        { return &Any{&Any_As{n}} }
func (n *Blob) Path() *Any                      { return &Any{&Any_Blob{n}} }
func (n *Capture) Path() *Any                   { return &Any{&Any_Capture{n}} }
func (n *ConstantSet) Path() *Any               { return &Any{&Any_ConstantSet{n}} }
func (n *Command) Path() *Any                   { return &Any{&Any_Command{n}} }
func (n *Commands) Path() *Any                  { return &Any{&Any_Commands{n}} }
func (n *CommandTree) Path() *Any               { return &Any{&Any_CommandTree{n}} }
func (n *CommandTreeNode) Path() *Any           { return &Any{&Any_CommandTreeNode{n}} }
func (n *CommandTreeNodeForCommand) Path() *Any { return &Any{&Any_CommandTreeNodeForCommand{n}} }
func (n *Context) Path() *Any                   { return &Any{&Any_Context{n}} }
func (n *Contexts) Path() *Any                  { return &Any{&Any_Contexts{n}} }
func (n *Device) Path() *Any                    { return &Any{&Any_Device{n}} }
func (n *Events) Path() *Any                    { return &Any{&Any_Events{n}} }
func (n *Field) Path() *Any                     { return &Any{&Any_Field{n}} }
func (n *ImageInfo) Path() *Any                 { return &Any{&Any_ImageInfo{n}} }
func (n *MapIndex) Path() *Any                  { return &Any{&Any_MapIndex{n}} }
func (n *Memory) Path() *Any                    { return &Any{&Any_Memory{n}} }
func (n *Mesh) Path() *Any                      { return &Any{&Any_Mesh{n}} }
func (n *Parameter) Path() *Any                 { return &Any{&Any_Parameter{n}} }
func (n *Report) Path() *Any                    { return &Any{&Any_Report{n}} }
func (n *ResourceData) Path() *Any              { return &Any{&Any_ResourceData{n}} }
func (n *Resources) Path() *Any                 { return &Any{&Any_Resources{n}} }
func (n *Slice) Path() *Any                     { return &Any{&Any_Slice{n}} }
func (n *State) Path() *Any                     { return &Any{&Any_State{n}} }
func (n *StateTree) Path() *Any                 { return &Any{&Any_StateTree{n}} }
func (n *StateTreeNode) Path() *Any             { return &Any{&Any_StateTreeNode{n}} }
func (n *Thumbnail) Path() *Any                 { return &Any{&Any_Thumbnail{n}} }

func (n API) Parent() Node                       { return nil }
func (n ArrayIndex) Parent() Node                { return oneOfNode(n.Array) }
func (n As) Parent() Node                        { return oneOfNode(n.From) }
func (n Blob) Parent() Node                      { return nil }
func (n Capture) Parent() Node                   { return nil }
func (n ConstantSet) Parent() Node               { return n.Api }
func (n Command) Parent() Node                   { return n.Capture }
func (n Commands) Parent() Node                  { return n.Capture }
func (n CommandTree) Parent() Node               { return n.Capture }
func (n CommandTreeNode) Parent() Node           { return nil }
func (n CommandTreeNodeForCommand) Parent() Node { return n.Command }
func (n Context) Parent() Node                   { return n.Capture }
func (n Contexts) Parent() Node                  { return n.Capture }
func (n Device) Parent() Node                    { return nil }
func (n Events) Parent() Node                    { return n.Commands }
func (n Field) Parent() Node                     { return oneOfNode(n.Struct) }
func (n ImageInfo) Parent() Node                 { return nil }
func (n MapIndex) Parent() Node                  { return oneOfNode(n.Map) }
func (n Memory) Parent() Node                    { return n.After }
func (n Mesh) Parent() Node                      { return oneOfNode(n.Object) }
func (n Parameter) Parent() Node                 { return n.Command }
func (n Report) Parent() Node                    { return n.Capture }
func (n ResourceData) Parent() Node              { return n.After }
func (n Resources) Parent() Node                 { return n.Capture }
func (n Slice) Parent() Node                     { return oneOfNode(n.Array) }
func (n State) Parent() Node                     { return n.After }
func (n StateTree) Parent() Node                 { return n.After }
func (n StateTreeNode) Parent() Node             { return nil }
func (n Thumbnail) Parent() Node                 { return oneOfNode(n.Object) }

func (n ArrayIndex) Text() string { return fmt.Sprintf("%v[%v]", n.Parent().Text(), n.Index) }
func (n API) Text() string        { return fmt.Sprintf("api<%v>", n.Id) }
func (n As) Text() string         { return fmt.Sprintf("%v.as<%v>", n.Parent().Text(), protoutil.OneOf(n.To)) }
func (n Blob) Text() string       { return fmt.Sprintf("blob<%x>", n.Id) }
func (n Capture) Text() string    { return fmt.Sprintf("capture<%x>", n.Id) }
func (n ConstantSet) Text() string {
	return fmt.Sprintf("%v.constant-set<%v>", n.Parent().Text(), n.Index)
}
func (n Command) Text() string {
	return fmt.Sprintf("%v.commands[%v]", n.Parent().Text(), printIndices(n.Index))
}
func (n Commands) Text() string {
	return fmt.Sprintf("%v.commands[%v-%v]", n.Parent().Text(), printIndices(n.From), printIndices(n.To))
}
func (n CommandTree) Text() string { return fmt.Sprintf("%v.command-tree") }
func (n CommandTreeNode) Text() string {
	return fmt.Sprintf("command-tree<%v>[%v]", n.Tree, printIndices(n.Index))
}
func (n CommandTreeNodeForCommand) Text() string {
	return fmt.Sprintf("%v.command-tree-node<%v>", n.Command.Text(), n.Tree)
}
func (n Context) Text() string   { return fmt.Sprintf("%v.[%x]", n.Parent().Text(), n.Id) }
func (n Contexts) Text() string  { return fmt.Sprintf("%v.contexts", n.Parent().Text()) }
func (n Device) Text() string    { return fmt.Sprintf("device<%x>", n.Id) }
func (n Events) Text() string    { return fmt.Sprintf(".events", n.Parent().Text()) }
func (n Field) Text() string     { return fmt.Sprintf("%v.%v", n.Parent().Text(), n.Name) }
func (n ImageInfo) Text() string { return fmt.Sprintf("image-info<%x>", n.Id) }
func (n MapIndex) Text() string  { return fmt.Sprintf("%v[%x]", n.Parent().Text(), n.Key) }
func (n Memory) Text() string    { return fmt.Sprintf("%v.memory-after", n.Parent().Text()) }
func (n Mesh) Text() string      { return fmt.Sprintf("%v.mesh", n.Parent().Text()) }
func (n Parameter) Text() string { return fmt.Sprintf("%v.%v", n.Parent().Text(), n.Name) }
func (n Report) Text() string    { return fmt.Sprintf("%v.report", n.Parent().Text()) }
func (n ResourceData) Text() string {
	return fmt.Sprintf("%v.resource-data<%x>", n.Parent().Text(), n.Id)
}
func (n Resources) Text() string { return fmt.Sprintf("%v.resources", n.Parent().Text()) }
func (n Slice) Text() string     { return fmt.Sprintf("%v[%v:%v]", n.Parent().Text(), n.Start, n.End) }
func (n State) Text() string     { return fmt.Sprintf("%v.state-after", n.Parent().Text()) }
func (n StateTree) Text() string { return fmt.Sprintf("%v.state-tree") }
func (n StateTreeNode) Text() string {
	return fmt.Sprintf("state-tree<%v>[%v]", n.Tree, printIndices(n.Index))
}
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

func (n *Slice) SetParent(p Node) {
	switch p := p.(type) {
	case *Field:
		n.Array = &Slice_Field{p}
	case *Slice:
		n.Array = &Slice_Slice{p}
	case *ArrayIndex:
		n.Array = &Slice_ArrayIndex{p}
	case *MapIndex:
		n.Array = &Slice_MapIndex{p}
	case *Parameter:
		n.Array = &Slice_Parameter{p}
	default:
		panic(fmt.Errorf("Cannot set Slice.Array to %T", p))
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

// NewField returns a new Field path.
func NewField(name string, s Node) *Field {
	out := &Field{Name: name}
	out.SetParent(s)
	return out
}

// NewArrayIndex returns a new ArrayIndex path.
func NewArrayIndex(idx uint64, a Node) *ArrayIndex {
	out := &ArrayIndex{Index: idx}
	out.SetParent(a)
	return out
}

// NewMapIndex returns a new MapIndex path.
func NewMapIndex(k interface{}, m Node) *MapIndex {
	if v := box.NewValue(k); v != nil {
		out := &MapIndex{Key: &MapIndex_Box{v}}
		out.SetParent(m)
		return out
	}
	panic(fmt.Errorf("Cannot set MapIndex.Key to %T", k))
}

// NewSlice returns a new Slice path.
func NewSlice(s, e uint64, n Node) *Slice {
	out := &Slice{Start: s, End: e}
	out.SetParent(n)
	return out
}

// ConstantSet returns a path to the API's i'th ConstantSet.
func (n *API) ConstantSet(i int) *ConstantSet {
	return &ConstantSet{
		Api:   n,
		Index: uint32(i),
	}
}

// Resources returns the path node to the capture's resources.
func (n *Capture) Resources() *Resources {
	return &Resources{Capture: n}
}

// Report returns the path node to the capture's report.
func (n *Capture) Report(d *Device, f *CommandFilter) *Report {
	return &Report{Capture: n, Device: d, Filter: f}
}

// Contexts returns the path node to the capture's contexts.
func (n *Capture) Contexts() *Contexts {
	return &Contexts{Capture: n}
}

// Commands returns the path node to the capture's commands.
func (n *Capture) Commands() *Commands {
	return &Commands{
		Capture: n,
		From:    []uint64{0},
		To:      []uint64{math.MaxUint64},
	}
}

// CommandRange returns the path node to a range of the capture's commands.
func (n *Capture) CommandRange(from, to uint64) *Commands {
	return &Commands{
		Capture: n,
		From:    []uint64{from},
		To:      []uint64{to},
	}
}

// CommandTree returns the path to the root node of a capture's command tree
// optionally filtered by f.
func (n *Capture) CommandTree(f *CommandFilter) *CommandTree {
	return &CommandTree{Capture: n, Filter: f}
}

// Child returns the path to the i'th child of the CommandTreeNode.
func (n *CommandTreeNode) Child(i uint64) *CommandTreeNode {
	newIndex := make([]uint64, len(n.Index)+1)
	copy(newIndex, n.Index)
	newIndex[len(n.Index)] = i
	return &CommandTreeNode{Tree: n.Tree, Index: newIndex}
}

// Command returns the path node to a single command in the capture.
func (n *Capture) Command(i uint64, subidx ...uint64) *Command {
	index := append([]uint64{i}, subidx...)
	return &Command{Capture: n, Index: index}
}

// Context returns the path node to the a context with the given ID.
func (n *Capture) Context(id *ID) *Context {
	return &Context{Capture: n, Id: id}
}

// Resource returns the path node to the resource with the given ID.
func (n *Capture) Resource(id *ID) *Resource {
	return &Resource{Capture: n, Id: id}
}

// MemoryAfter returns the path node to the memory after this command.
func (n *Command) MemoryAfter(pool uint32, addr, size uint64) *Memory {
	return &Memory{addr, size, pool, n, false}
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

// StateTreeAfter returns the path node to the state tree after this command.
func (n *Command) StateTreeAfter() *StateTree {
	return &StateTree{After: n}
}

// First returns the path to the first command.
func (n *Commands) First() *Command {
	return &Command{Capture: n.Capture, Index: n.From}
}

// Last returns the path to the last command.
func (n *Commands) Last() *Command {
	return &Command{Capture: n.Capture, Index: n.To}
}

// Child returns the path to the i'th child of the StateTreeNode.
func (n *StateTreeNode) Child(i uint64) *StateTreeNode {
	newIndex := make([]uint64, len(n.Index)+1)
	copy(newIndex, n.Index)
	newIndex[len(n.Index)] = i
	return &StateTreeNode{Tree: n.Tree, Index: newIndex}
}

// Parameter returns the path node to the parameter with the given name.
func (n *Command) Parameter(name string) *Parameter {
	return &Parameter{Name: name, Command: n}
}

func (n *State) Field(name string) *Field             { return NewField(name, n) }
func (n *Parameter) ArrayIndex(i uint64) *ArrayIndex  { return NewArrayIndex(i, n) }
func (n *Parameter) Field(name string) *Field         { return NewField(name, n) }
func (n *Parameter) MapIndex(k interface{}) *MapIndex { return NewMapIndex(k, n) }
func (n *Parameter) Slice(s, e uint64) *Slice         { return NewSlice(s, e, n) }
func (n *Field) ArrayIndex(i uint64) *ArrayIndex      { return NewArrayIndex(i, n) }
func (n *Field) Field(name string) *Field             { return NewField(name, n) }
func (n *Field) MapIndex(k interface{}) *MapIndex     { return NewMapIndex(k, n) }
func (n *Field) Slice(s, e uint64) *Slice             { return NewSlice(s, e, n) }
func (n *MapIndex) ArrayIndex(i uint64) *ArrayIndex   { return NewArrayIndex(i, n) }
func (n *MapIndex) Field(name string) *Field          { return NewField(name, n) }
func (n *MapIndex) MapIndex(k interface{}) *MapIndex  { return NewMapIndex(k, n) }
func (n *MapIndex) Slice(s, e uint64) *Slice          { return NewSlice(s, e, n) }

func (n *MapIndex) KeyValue() interface{} {
	switch k := protoutil.OneOf(n.Key).(type) {
	case nil:
		return nil
	case *box.Value:
		return k.Get()
	default:
		panic(fmt.Errorf("Unsupport MapIndex key type %T", k))
	}
}

// As requests the ImageInfo converted to the specified format.
func (n *ImageInfo) As(f *image.Format) *As {
	return &As{
		To:   &As_ImageFormat{f},
		From: &As_ImageInfo{n},
	}
}

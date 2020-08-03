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
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/protoutil"
	"github.com/google/gapid/core/data/slice"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapis/service/box"
	"github.com/google/gapid/gapis/vertex"
)

// Node is the interface for types that represent a reference to a capture,
// command list, single command, memory, state or sub-object. A path can be
// passed between client and server using RPCs in order to describe some data
// in a capture.
type Node interface {
	// Parent returns the path that this path derives from.
	// If this path is a root, then Base returns nil.
	Parent() Node

	// SetParent sets the path that this derives from.
	SetParent(Node)

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

func (n *API) Path() *Any                       { return &Any{Path: &Any_API{n}} }
func (n *ArrayIndex) Path() *Any                { return &Any{Path: &Any_ArrayIndex{n}} }
func (n *As) Path() *Any                        { return &Any{Path: &Any_As{n}} }
func (n *Blob) Path() *Any                      { return &Any{Path: &Any_Blob{n}} }
func (n *Capture) Path() *Any                   { return &Any{Path: &Any_Capture{n}} }
func (n *ConstantSet) Path() *Any               { return &Any{Path: &Any_ConstantSet{n}} }
func (n *Command) Path() *Any                   { return &Any{Path: &Any_Command{n}} }
func (n *Commands) Path() *Any                  { return &Any{Path: &Any_Commands{n}} }
func (n *CommandTree) Path() *Any               { return &Any{Path: &Any_CommandTree{n}} }
func (n *CommandTreeNode) Path() *Any           { return &Any{Path: &Any_CommandTreeNode{n}} }
func (n *CommandTreeNodeForCommand) Path() *Any { return &Any{Path: &Any_CommandTreeNodeForCommand{n}} }
func (n *Device) Path() *Any                    { return &Any{Path: &Any_Device{n}} }
func (n *DeviceTraceConfiguration) Path() *Any  { return &Any{Path: &Any_TraceConfig{n}} }
func (n *Events) Path() *Any                    { return &Any{Path: &Any_Events{n}} }
func (n *FramebufferObservation) Path() *Any    { return &Any{Path: &Any_FBO{n}} }
func (n *FramebufferAttachments) Path() *Any    { return &Any{Path: &Any_FramebufferAttachments{n}} }
func (n *FramebufferAttachment) Path() *Any     { return &Any{Path: &Any_FramebufferAttachment{n}} }
func (n *Field) Path() *Any                     { return &Any{Path: &Any_Field{n}} }
func (n *GlobalState) Path() *Any               { return &Any{Path: &Any_GlobalState{n}} }
func (n *ImageInfo) Path() *Any                 { return &Any{Path: &Any_ImageInfo{n}} }
func (n *MapIndex) Path() *Any                  { return &Any{Path: &Any_MapIndex{n}} }
func (n *Memory) Path() *Any                    { return &Any{Path: &Any_Memory{n}} }
func (n *MemoryAsType) Path() *Any              { return &Any{Path: &Any_MemoryAsType{n}} }
func (n *Mesh) Path() *Any                      { return &Any{Path: &Any_Mesh{n}} }
func (n *Metrics) Path() *Any                   { return &Any{Path: &Any_Metrics{n}} }
func (n *Parameter) Path() *Any                 { return &Any{Path: &Any_Parameter{n}} }
func (n *Pipelines) Path() *Any                 { return &Any{Path: &Any_Pipelines{n}} }
func (n *Report) Path() *Any                    { return &Any{Path: &Any_Report{n}} }
func (n *ResourceData) Path() *Any              { return &Any{Path: &Any_ResourceData{n}} }
func (n *Messages) Path() *Any                  { return &Any{Path: &Any_Messages{n}} }
func (n *MultiResourceData) Path() *Any         { return &Any{Path: &Any_MultiResourceData{n}} }
func (n *Resources) Path() *Any                 { return &Any{Path: &Any_Resources{n}} }
func (n *Result) Path() *Any                    { return &Any{Path: &Any_Result{n}} }
func (n *Slice) Path() *Any                     { return &Any{Path: &Any_Slice{n}} }
func (n *State) Path() *Any                     { return &Any{Path: &Any_State{n}} }
func (n *StateTree) Path() *Any                 { return &Any{Path: &Any_StateTree{n}} }
func (n *StateTreeNode) Path() *Any             { return &Any{Path: &Any_StateTreeNode{n}} }
func (n *StateTreeNodeForPath) Path() *Any      { return &Any{Path: &Any_StateTreeNodeForPath{n}} }
func (n *Stats) Path() *Any                     { return &Any{Path: &Any_Stats{n}} }
func (n *Thumbnail) Path() *Any                 { return &Any{Path: &Any_Thumbnail{n}} }
func (n *Type) Path() *Any                      { return &Any{Path: &Any_Type{n}} }

func (n API) Parent() Node                       { return nil }
func (n ArrayIndex) Parent() Node                { return oneOfNode(n.Array) }
func (n As) Parent() Node                        { return oneOfNode(n.From) }
func (n Blob) Parent() Node                      { return nil }
func (n Capture) Parent() Node                   { return nil }
func (n ConstantSet) Parent() Node               { return n.API }
func (n Command) Parent() Node                   { return n.Capture }
func (n Commands) Parent() Node                  { return n.Capture }
func (n CommandTree) Parent() Node               { return n.Capture }
func (n CommandTreeNode) Parent() Node           { return nil }
func (n CommandTreeNodeForCommand) Parent() Node { return n.Command }
func (n Device) Parent() Node                    { return nil }
func (n DeviceTraceConfiguration) Parent() Node  { return n.Device }
func (n Events) Parent() Node                    { return n.Capture }
func (n FramebufferObservation) Parent() Node    { return n.Command }
func (n FramebufferAttachments) Parent() Node    { return n.After }
func (n FramebufferAttachment) Parent() Node     { return n.After }
func (n Field) Parent() Node                     { return oneOfNode(n.Struct) }
func (n GlobalState) Parent() Node               { return n.After }
func (n ImageInfo) Parent() Node                 { return nil }
func (n MapIndex) Parent() Node                  { return oneOfNode(n.Map) }
func (n Memory) Parent() Node                    { return n.After }
func (n MemoryAsType) Parent() Node              { return n.After }
func (n Mesh) Parent() Node                      { return oneOfNode(n.Object) }
func (n Metrics) Parent() Node                   { return n.Command }
func (n Messages) Parent() Node                  { return n.Capture }
func (n Parameter) Parent() Node                 { return n.Command }
func (n Pipelines) Parent() Node                 { return n.After }
func (n Report) Parent() Node                    { return n.Capture }
func (n ResourceData) Parent() Node              { return n.After }
func (n MultiResourceData) Parent() Node         { return n.After }
func (n Resources) Parent() Node                 { return n.Capture }
func (n Result) Parent() Node                    { return n.Command }
func (n Slice) Parent() Node                     { return oneOfNode(n.Array) }
func (n State) Parent() Node                     { return n.After }
func (n StateTree) Parent() Node                 { return n.State }
func (n StateTreeNode) Parent() Node             { return nil }
func (n StateTreeNodeForPath) Parent() Node      { return nil }
func (n Stats) Parent() Node                     { return n.Capture }
func (n Thumbnail) Parent() Node                 { return oneOfNode(n.Object) }
func (n Type) Parent() Node                      { return nil }

func (n *API) SetParent(p Node)                       {}
func (n *Blob) SetParent(p Node)                      {}
func (n *Capture) SetParent(p Node)                   {}
func (n *ConstantSet) SetParent(p Node)               { n.API, _ = p.(*API) }
func (n *Command) SetParent(p Node)                   { n.Capture, _ = p.(*Capture) }
func (n *Commands) SetParent(p Node)                  { n.Capture, _ = p.(*Capture) }
func (n *CommandTree) SetParent(p Node)               { n.Capture, _ = p.(*Capture) }
func (n *CommandTreeNode) SetParent(p Node)           {}
func (n *CommandTreeNodeForCommand) SetParent(p Node) { n.Command, _ = p.(*Command) }
func (n *Device) SetParent(p Node)                    {}
func (n *DeviceTraceConfiguration) SetParent(p Node)  { n.Device, _ = p.(*Device) }
func (n *Events) SetParent(p Node)                    { n.Capture, _ = p.(*Capture) }
func (n *FramebufferObservation) SetParent(p Node)    { n.Command, _ = p.(*Command) }
func (n *FramebufferAttachments) SetParent(p Node)    { n.After, _ = p.(*Command) }
func (n *FramebufferAttachment) SetParent(p Node)     { n.After, _ = p.(*Command) }
func (n *GlobalState) SetParent(p Node)               { n.After, _ = p.(*Command) }
func (n *ImageInfo) SetParent(p Node)                 {}
func (n *Memory) SetParent(p Node)                    { n.After, _ = p.(*Command) }
func (n *MemoryAsType) SetParent(p Node)              { n.After, _ = p.(*Command) }
func (n *Metrics) SetParent(p Node)                   { n.Command, _ = p.(*Command) }
func (n *Messages) SetParent(p Node)                  { n.Capture, _ = p.(*Capture) }
func (n *Pipelines) SetParent(p Node)                 { n.After, _ = p.(*Command) }
func (n *Parameter) SetParent(p Node)                 { n.Command, _ = p.(*Command) }
func (n *Report) SetParent(p Node)                    { n.Capture, _ = p.(*Capture) }
func (n *ResourceData) SetParent(p Node)              { n.After, _ = p.(*Command) }
func (n *MultiResourceData) SetParent(p Node)         { n.After, _ = p.(*Command) }
func (n *Resources) SetParent(p Node)                 { n.Capture, _ = p.(*Capture) }
func (n *Result) SetParent(p Node)                    { n.Command, _ = p.(*Command) }
func (n *State) SetParent(p Node)                     { n.After, _ = p.(*Command) }
func (n *StateTree) SetParent(p Node)                 { n.State, _ = p.(*State) }
func (n *StateTreeNode) SetParent(p Node)             {}
func (n *StateTreeNodeForPath) SetParent(p Node)      {}
func (n *Stats) SetParent(p Node)                     { n.Capture, _ = p.(*Capture) }
func (n *Type) SetParent(p Node)                      {}

// Format implements fmt.Formatter to print the path.
func (n ArrayIndex) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v[%v]", n.Parent(), n.Index)
}

// Format implements fmt.Formatter to print the path.
func (n API) Format(f fmt.State, c rune) { fmt.Fprintf(f, "api<%v>", n.ID) }

// Format implements fmt.Formatter to print the path.
func (n As) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v.as<%v>", n.Parent(), protoutil.OneOf(n.To))
}

// Format implements fmt.Formatter to print the path.
func (n Blob) Format(f fmt.State, c rune) { fmt.Fprintf(f, "blob<%x>", n.ID) }

// Format implements fmt.Formatter to print the path.
func (n Capture) Format(f fmt.State, c rune) { fmt.Fprintf(f, "capture<%x>", n.ID) }

// Format implements fmt.Formatter to print the path.
func (n ConstantSet) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v.constant-set<%v>", n.Parent(), n.Index)
}

// Format implements fmt.Formatter to print the path.
func (n Command) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v.commands[%v]", n.Parent(), printIndices(n.Indices))
}

// Format implements fmt.Formatter to print the path.
func (n Commands) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v.commands[%v-%v]", n.Parent(), printIndices(n.From), printIndices(n.To))
}

// Format implements fmt.Formatter to print the path.
func (n CommandTree) Format(f fmt.State, c rune) { fmt.Fprintf(f, "%v.command-tree", n.Capture) }

// Format implements fmt.Formatter to print the path.
func (n CommandTreeNode) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "command-tree<%v>[%v]", n.Tree, printIndices(n.Indices))
}

// Format implements fmt.Formatter to print the path.
func (n CommandTreeNodeForCommand) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v.command-tree-node<%v>", n.Command, n.Tree)
}

// Format implements fmt.Formatter to print the path.
func (n Device) Format(f fmt.State, c rune) { fmt.Fprintf(f, "device<%x>", n.ID) }

// Format implements fmt.Formatter to print the path.
func (n Events) Format(f fmt.State, c rune) { fmt.Fprintf(f, "%v.events", n.Parent()) }

// Format implements fmt.Formatter to print the path.
func (n Field) Format(f fmt.State, c rune) { fmt.Fprintf(f, "%v.%v", n.Parent(), n.Name) }

// Format implements fmt.Formatter to print the path.
func (n FramebufferAttachments) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v.framebuffer-attachments", n.Parent())
}

func (n FramebufferAttachment) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v.framevuffer-attachment<%x>", n.Parent(), n.Index)
}

// Format implements fmt.Formatter to print the path.
func (n GlobalState) Format(f fmt.State, c rune) { fmt.Fprintf(f, "%v.global-state", n.Parent()) }

// Format implements fmt.Formatter to print the path.
func (n ImageInfo) Format(f fmt.State, c rune) { fmt.Fprintf(f, "image-info<%x>", n.ID) }

// Format implements fmt.Formatter to print the path.
func (n MapIndex) Format(f fmt.State, c rune) { fmt.Fprintf(f, "%v[%x]", n.Parent(), n.Key) }

// Format implements fmt.Formatter to print the path.
func (n Memory) Format(f fmt.State, c rune) { fmt.Fprintf(f, "%v.memory-after", n.Parent()) }

// Format implements fmt.Formatter to print the path
func (n MemoryAsType) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v.memory-as-type-after", n.Parent())
}

// Format implements fmt.Formatter to print the message path.
func (n Messages) Format(f fmt.State, c rune) { fmt.Fprintf(f, "%v.messages", n.Parent()) }

// Format implements fmt.Formatter to print the path.
func (n Mesh) Format(f fmt.State, c rune) { fmt.Fprintf(f, "%v.mesh", n.Parent()) }

// Format implements fmt.Formatter to print the path.
func (n Parameter) Format(f fmt.State, c rune) { fmt.Fprintf(f, "%v.%v", n.Parent(), n.Name) }

// Format implements fmt.Formatter to print the path.
func (n Report) Format(f fmt.State, c rune) { fmt.Fprintf(f, "%v.report", n.Parent()) }

// Format implements fmt.Formatter to print the path.
func (n ResourceData) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v.resource-data<%x>", n.Parent(), n.ID)
}

// Format implements fmt.Formatter to print the path.
func (n MultiResourceData) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v.resource-data<%x>", n.Parent(), n.IDs)
}

// Format implements fmt.Formatter to print the path.
func (n Pipelines) Format(f fmt.State, c rune) { fmt.Fprintf(f, "%v.pipelines", n.Parent()) }

// Format implements fmt.Formatter to print the path.
func (n Resources) Format(f fmt.State, c rune) { fmt.Fprintf(f, "%v.resources", n.Parent()) }

// Format implements fmt.Formatter to print the path.
func (n Result) Format(f fmt.State, c rune) { fmt.Fprintf(f, "%v.result", n.Parent()) }

// Format implements fmt.Formatter to print the path.
func (n Slice) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v[%v:%v]", n.Parent(), n.Start, n.End)
}

// Format implements fmt.Formatter to print the path.
func (n State) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "%v.state", n.Parent())
}

// Format implements fmt.Formatter to print the path.
func (n StateTree) Format(f fmt.State, c rune) { fmt.Fprintf(f, "%v.tree", n.State) }

// Format implements fmt.Formatter to print the path.
func (n StateTreeNode) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "state-tree<%v>[%v]", n.Tree, printIndices(n.Indices))
}

// Format implements fmt.Formatter to print the path.
func (n StateTreeNodeForPath) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "state-tree-for<%v, %v>", n.Tree, n.Member)
}

// Format implements fmt.Formatter to print the path.
func (n Stats) Format(f fmt.State, c rune) { fmt.Fprintf(f, "%v.stats", n.Parent()) }

// Format implements fmt.Formatter to print the path.
func (n Thumbnail) Format(f fmt.State, c rune) { fmt.Fprintf(f, "%v.thumbnail", n.Parent()) }

func (n Type) Format(f fmt.State, c rune) { fmt.Fprintf(f, "%v.type", n.TypeIndex) }

func (n *As) SetParent(p Node) {
	switch p := p.(type) {
	case nil:
		n.From = nil
	case *Field:
		n.From = &As_Field{p}
	case *Slice:
		n.From = &As_Slice{p}
	case *ArrayIndex:
		n.From = &As_ArrayIndex{p}
	case *MapIndex:
		n.From = &As_MapIndex{p}
	case *ImageInfo:
		n.From = &As_ImageInfo{p}
	case *ResourceData:
		n.From = &As_ResourceData{p}
	case *Mesh:
		n.From = &As_Mesh{p}
	default:
		panic(fmt.Errorf("Cannot set As.From to %T", p))
	}
}

func (n *ArrayIndex) SetParent(p Node) {
	switch p := p.(type) {
	case nil:
		n.Array = nil
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
	case nil:
		n.Struct = nil
	case *Field:
		n.Struct = &Field_Field{p}
	case *GlobalState:
		n.Struct = &Field_GlobalState{p}
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
	case nil:
		n.Map = nil
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

func (n *Mesh) SetParent(p Node) {
	switch p := p.(type) {
	case nil:
		n.Object = nil
	case *Command:
		n.Object = &Mesh_Command{p}
	case *CommandTreeNode:
		n.Object = &Mesh_CommandTreeNode{p}
	default:
		panic(fmt.Errorf("Cannot set Mesh.Object to %T", p))
	}
}

func (n *Slice) SetParent(p Node) {
	switch p := p.(type) {
	case nil:
		n.Array = nil
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

func (n *Thumbnail) SetParent(p Node) {
	switch p := p.(type) {
	case nil:
		n.Object = nil
	case *ResourceData:
		n.Object = &Thumbnail_Resource{p}
	case *Command:
		n.Object = &Thumbnail_Command{p}
	case *CommandTreeNode:
		n.Object = &Thumbnail_CommandTreeNode{p}
	default:
		panic(fmt.Errorf("Cannot set Thumbnail.Object to %T", p))
	}
}

func oneOfNode(v interface{}) Node {
	return protoutil.OneOf(v).(Node)
}

// Node returns the path node for p.
func (p *Any) Node() Node { return oneOfNode(p.Path) }

// Format implements fmt.Formatter to print the path.
func (p *Any) Format(f fmt.State, c rune) {
	fmt.Fprint(f, p.Node())
}

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

// NewAPI returns a new API path node with the given ID.
func NewAPI(id id.ID) *API {
	return &API{ID: NewID(id)}
}

// NewCapture returns a new Capture path node with the given ID.
func NewCapture(id id.ID) *Capture {
	return &Capture{ID: NewID(id)}
}

// NewDevice returns a new Device path node with the given ID.
func NewDevice(id id.ID) *Device {
	return &Device{ID: NewID(id)}
}

// NewBlob returns a new Blob path node with the given ID.
func NewBlob(id id.ID) *Blob {
	return &Blob{ID: NewID(id)}
}

// NewImageInfo returns a new ImageInfo path node with the given ID.
func NewImageInfo(id id.ID) *ImageInfo {
	return &ImageInfo{ID: image.NewID(id)}
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
		API:   n,
		Index: int32(i),
	}
}

// Resources returns the path node to the capture's resources.
func (n *Capture) Resources() *Resources {
	return &Resources{Capture: n}
}

// Report returns the path node to the capture's report.
func (n *Capture) Report(d *Device, display bool) *Report {
	return &Report{Capture: n, Device: d, DisplayToSurface: display}
}

// Messages returns the path node to the capture's messages.
func (n *Capture) Messages() *Messages {
	return &Messages{Capture: n}
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

// SubCommandRange returns the path node to a range of the capture's subcommands
func (n *Capture) SubCommandRange(from, to []uint64) *Commands {
	return &Commands{
		Capture: n,
		From:    append([]uint64{}, from...),
		To:      append([]uint64{}, to...),
	}
}

// CommandTree returns the path to the root node of a capture's command tree
// optionally filtered by f.
func (n *Capture) CommandTree(f *CommandFilter) *CommandTree {
	return &CommandTree{Capture: n, Filter: f}
}

// Child returns the path to the i'th child of the CommandTreeNode.
func (n *CommandTreeNode) Child(i uint64) *CommandTreeNode {
	newIndices := make([]uint64, len(n.Indices)+1)
	copy(newIndices, n.Indices)
	newIndices[len(n.Indices)] = i
	return &CommandTreeNode{Tree: n.Tree, Indices: newIndices}
}

// Command returns the path node to a single command in the capture.
func (n *Capture) Command(i uint64, subidx ...uint64) *Command {
	indices := append([]uint64{i}, subidx...)
	return &Command{Capture: n, Indices: indices}
}

// Thread returns the path node to the thread with the given ID.
func (n *Capture) Thread(id uint64) *Thread {
	return &Thread{Capture: n, ID: id}
}

// MemoryAfter returns the path node to the memory after this command.
func (n *Command) MemoryAfter(pool uint32, addr, size uint64) *Memory {
	return &Memory{Address: addr, Size: size, Pool: pool, After: n}
}

// ResourceAfter returns the path node to the resource with the given identifier
// after this command.
func (n *Command) ResourceAfter(id *ID) *ResourceData {
	return &ResourceData{
		ID:    id,
		After: n,
	}
}

// ResourcesAfter returns the path node to the resources with the given
// identifiers after this command.
func (n *Command) ResourcesAfter(ids []*ID) *MultiResourceData {
	return &MultiResourceData{
		IDs:   ids,
		After: n,
	}
}

// FramebufferAttachmentsAfter returns the path node to the framebuffer attachments
// list afthis this command
func (n *Command) FramebufferAttachmentsAfter() *FramebufferAttachments {
	return &FramebufferAttachments{
		After: n,
	}
}

// FramebufferAttachmentAfter returns the path to the framebuffer attachment
// with the given index after this command
func (n *Command) FramebufferAttachmentAfter(index uint32) *FramebufferAttachment {
	return &FramebufferAttachment{
		After: n,
		Index: index,
	}
}

// FramebufferObservation returns the path node to framebuffer observation
// after this command.
func (n *Command) FramebufferObservation() *FramebufferObservation {
	return &FramebufferObservation{Command: n}
}

// Mesh returns the path node to the mesh of this command.
func (n *Command) Mesh(options *MeshOptions) *Mesh {
	return &Mesh{
		Options: options,
		Object:  &Mesh_Command{n},
	}
}

// NewMeshOptions returns a new MeshOptions object.
func NewMeshOptions(faceted bool) *MeshOptions {
	return &MeshOptions{
		Faceted: faceted,
	}
}

// Hints returns the vertex semantics hints from the mesh options.
func (o *MeshOptions) Hints() map[string]vertex.Semantic_Type {
	m := map[string]vertex.Semantic_Type{}
	if o == nil {
		return m
	}

	for _, hint := range o.VertexSemantics {
		m[hint.Name] = hint.Type
	}
	return m
}

// GlobalStateAfter returns the path node to the state after this command.
func (n *Command) GlobalStateAfter() *GlobalState {
	return &GlobalState{After: n}
}

// StateAfter returns the path node to the state after this command.
func (n *Command) StateAfter() *State {
	return &State{After: n}
}

// First returns the path to the first command.
func (n *Commands) First() *Command {
	return &Command{Capture: n.Capture, Indices: n.From}
}

// Last returns the path to the last command.
func (n *Commands) Last() *Command {
	return &Command{Capture: n.Capture, Indices: n.To}
}

// Index returns the path to the i'th child of the StateTreeNode.
func (n *StateTreeNode) Index(i ...uint64) *StateTreeNode {
	newIndices := make([]uint64, len(n.Indices)+len(i))
	copy(newIndices, n.Indices)
	copy(newIndices[len(n.Indices):], i)
	return &StateTreeNode{Tree: n.Tree, Indices: newIndices}
}

// Parameter returns the path node to the parameter with the given name.
func (n *Command) Parameter(name string) *Parameter {
	return &Parameter{Name: name, Command: n}
}

// Result returns the path node to the command's result.
func (n *Command) Result() *Result {
	return &Result{Command: n}
}

// Tree returns the path node to the state tree for this state.
func (n *State) Tree() *StateTree {
	return &StateTree{State: n}
}

func (n *GlobalState) Field(name string) *Field       { return NewField(name, n) }
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
func (n *Slice) ArrayIndex(i uint64) *ArrayIndex      { return NewArrayIndex(i, n) }
func (n *Slice) Slice(s, e uint64) *Slice             { return NewSlice(s, e, n) }

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

// ToList unchains the parents of each node, returning them as a list, starting
// with the root node.
func ToList(n Node) []Node {
	out := []Node{}
	for ; n != nil; n = n.Parent() {
		out = append(out, n)
	}
	slice.Reverse(out)
	return out
}

// HasRoot returns true iff p starts with root, using equal as the node
// comparision function.
func HasRoot(p, root Node) (res bool) {
	a, b := ToList(p), ToList(root)
	if len(b) > len(a) {
		return false
	}
	for i, n := range b {
		if !ShallowEqual(n, a[i]) {
			return false
		}
	}
	return true
}

// ShallowEqual returns true if paths a and b are equal (ignoring parents).
func ShallowEqual(a, b Node) bool {
	a, b = proto.Clone(a.(proto.Message)).(Node), proto.Clone(b.(proto.Message)).(Node)
	a.SetParent(nil)
	b.SetParent(nil)
	return reflect.DeepEqual(a, b)
}

func (i *ID) SameAs(o *ID) bool {
	if i == nil || o == nil {
		return false
	}
	if len(i.Data) != len(o.Data) {
		return false
	}
	for ii := range i.Data {
		if i.Data[ii] != o.Data[ii] {
			return false
		}
	}
	return true
}

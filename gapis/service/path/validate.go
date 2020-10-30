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
	"reflect"

	"github.com/google/gapid/core/data/protoutil"
)

func isNil(f interface{}) bool {
	if f == nil {
		return true
	}
	v := reflect.ValueOf(f)
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return true
		}
		v = v.Elem()
	}
	return false
}

func checkNotNilAndValidate(n Node, f interface{}, name string) error {
	if isNil(f) {
		return fmt.Errorf("Invalid path '%v': %v must not be nil", n, name)
	}
	if fn, ok := f.(Node); ok {
		return fn.Validate()
	}
	return nil
}

type isValider interface {
	IsValid() bool
}

func checkIsValid(n Node, v isValider, name string) error {
	if v == nil || !v.IsValid() {
		return fmt.Errorf("Invalid path '%v': ID '%v' is invalid", n, name)
	}
	return nil
}

func checkNotEmptyString(n Node, s string, name string) error {
	if len(s) == 0 {
		return fmt.Errorf("Invalid path '%v': String '%v' must be non-empty", n, name)
	}
	return nil
}

func checkGreaterThan(n Node, a, b int, name string) error {
	if a <= b {
		return fmt.Errorf("Invalid path '%v': %v must be greater than %v", n, name, b)
	}
	return nil
}

func anyErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

// Validate checks the path is valid.
func (p *Any) Validate() error {
	if n, ok := protoutil.OneOf(p.Path).(Node); ok {
		return n.Validate()
	}
	return fmt.Errorf("Any holds no path node")
}

// Validate checks the path is valid.
func (n *ArrayIndex) Validate() error {
	return checkNotNilAndValidate(n, protoutil.OneOf(n.Array), "array")
}

// Validate checks the path is valid.
func (n *API) Validate() error {
	return anyErr(
		checkIsValid(n, n.ID, "api"),
	)
}

// Validate checks the path is valid.
func (n *As) Validate() error {
	return anyErr(
		checkNotNilAndValidate(n, protoutil.OneOf(n.From), "from"),
		checkNotNilAndValidate(n, protoutil.OneOf(n.To), "to"),
	)
}

// Validate checks the path is valid.
func (n *Blob) Validate() error {
	return checkIsValid(n, n.ID, "id")
}

// Validate checks the path is valid.
func (n *Capture) Validate() error {
	return checkIsValid(n, n.ID, "id")
}

// Validate checks the path is valid.
func (n *Command) Validate() error {
	return anyErr(
		checkNotNilAndValidate(n, n.Capture, "capture"),
		checkGreaterThan(n, len(n.Indices), 0, "length(index)"),
	)
}

// Validate checks the path is valid.
func (n *Commands) Validate() error {
	return anyErr(
		checkNotNilAndValidate(n, n.Capture, "capture"),
		checkGreaterThan(n, len(n.From), 0, "length(from)"),
		checkGreaterThan(n, len(n.To), 0, "length(to)"),
	)
}

// Validate checks the path is valid.
func (n *CommandTree) Validate() error {
	return checkNotNilAndValidate(n, n.Capture, "capture")
}

// Validate checks the path is valid.
func (n *CommandTreeNode) Validate() error {
	return checkIsValid(n, n.Tree, "tree")
}

// Validate checks the path is valid.
func (n *CommandTreeNodeForCommand) Validate() error {
	return anyErr(
		checkNotNilAndValidate(n, n.Command, "command"),
		checkIsValid(n, n.Tree, "tree"),
	)
}

// Validate checks the path is valid.
func (n *ConstantSet) Validate() error {
	return anyErr(
		checkNotNilAndValidate(n, n.API, "api"),
	)
}

// Validate checks the path is valid.
func (n *Device) Validate() error {
	return checkIsValid(n, n.ID, "id")
}

// Validate checks the path is valid.
func (n *DeviceTraceConfiguration) Validate() error {
	return checkNotNilAndValidate(n, n.Device, "device")
}

// Validate checks the path is valid.
func (n *Events) Validate() error {
	return checkNotNilAndValidate(n, protoutil.OneOf(n.Capture), "capture")
}

// Validate checks the path is valid.
func (n *FramebufferObservation) Validate() error {
	return checkNotNilAndValidate(n, protoutil.OneOf(n.Command), "command")
}

// Validate checks the path is valid.
func (n *FramebufferAttachments) Validate() error {
	return checkNotNilAndValidate(n, protoutil.OneOf(n.After), "command")
}

// Validate checks the path is valid.
func (n *FramebufferAttachment) Validate() error {
	return checkNotNilAndValidate(n, protoutil.OneOf(n.After), "command")
}

// Validate checks the path is valid.
func (n *Field) Validate() error {
	return anyErr(
		checkNotNilAndValidate(n, protoutil.OneOf(n.Struct), "struct"),
		checkNotEmptyString(n, n.Name, "name"),
	)
}

// Validate checks the path is valid.
func (n *Framegraph) Validate() error {
	return checkNotNilAndValidate(n, n.Capture, "capture")
}

// Validate checks the path is valid.
func (n *GlobalState) Validate() error {
	return checkNotNilAndValidate(n, n.After, "after")
}

// Validate checks the path is valid.
func (n *ImageInfo) Validate() error {
	return checkNotNilAndValidate(n, n.ID, "id")
}

// Validate checks the path is valid.
func (n *MapIndex) Validate() error {
	return anyErr(
		checkNotNilAndValidate(n, protoutil.OneOf(n.Map), "map"),
		checkNotNilAndValidate(n, protoutil.OneOf(n.Key), "key"),
	)
}

// Validate checks the path is valid.
func (n *Memory) Validate() error {
	return checkNotNilAndValidate(n, n.After, "after")
}

// Validate checks the path is valid.
func (n *MemoryAsType) Validate() error {
	return checkNotNilAndValidate(n, n.After, "after")
}

// Validate checks the path is valid.
func (n *Mesh) Validate() error {
	return checkNotNilAndValidate(n, protoutil.OneOf(n.Object), "object")
}

// Validate checks the path is valid.
func (n *Messages) Validate() error {
	return checkNotNilAndValidate(n, n.Capture, "capture")
}

// Validate checks the path is valid.
func (n *Metrics) Validate() error {
	return checkNotNilAndValidate(n, n.Command, "command")
}

// Validate checks the path is valid.
func (n *Parameter) Validate() error {
	return anyErr(
		checkNotNilAndValidate(n, protoutil.OneOf(n.Command), "command"),
		checkNotEmptyString(n, n.Name, "name"),
	)
}

// Validate checks the path is valid.
func (n *Pipelines) Validate() error {
	return checkNotNilAndValidate(n, n.After, "after")
}

// Validate checks the path is valid.
func (n *Report) Validate() error {
	return checkNotNilAndValidate(n, n.Capture, "capture")
}

// Validate checks the path is valid.
func (n *ResourceData) Validate() error {
	return anyErr(
		checkNotNilAndValidate(n, n.After, "after"),
		checkIsValid(n, n.ID, "id"),
	)
}

// Validate checks the path is valid.
func (n *MultiResourceData) Validate() error {
	if err := checkNotNilAndValidate(n, n.After, "after"); err != nil {
		return err
	}
	for _, id := range n.IDs {
		if err := checkIsValid(n, id, "id"); err != nil {
			return err
		}
	}
	return nil
}

// Validate checks the path is valid.
func (n *Resources) Validate() error {
	return checkNotNilAndValidate(n, n.Capture, "capture")
}

// Validate checks the path is valid.
func (n *Result) Validate() error {
	return checkNotNilAndValidate(n, n.Command, "command")
}

// Validate checks the path is valid.
func (n *Slice) Validate() error {
	return checkNotNilAndValidate(n, protoutil.OneOf(n.Array), "array")
}

// Validate checks the path is valid.
func (n *State) Validate() error {
	return checkNotNilAndValidate(n, n.After, "after")
}

// Validate checks the path is valid.
func (n *StateTree) Validate() error {
	return checkNotNilAndValidate(n, n.State, "state")
}

// Validate checks the path is valid.
func (n *StateTreeNode) Validate() error {
	return checkIsValid(n, n.Tree, "tree")
}

// Validate checks the path is valid.
func (n *StateTreeNodeForPath) Validate() error {
	return anyErr(
		checkIsValid(n, n.Tree, "tree"),
		checkNotNilAndValidate(n, n.Member.Node(), "path"),
	)
}

// Validate checks the path is valid.
func (n *Stats) Validate() error {
	return checkNotNilAndValidate(n, n.Capture, "capture")
}

// Validate checks the path is valid.
func (n *Thumbnail) Validate() error {
	return checkNotNilAndValidate(n, protoutil.OneOf(n.Object), "object")
}

// Validate checks the path is valid.
func (n *Type) Validate() error {
	if n != nil {
		return nil
	}
	return fmt.Errorf("Invalid path '%v': type must not be nil", n)
}

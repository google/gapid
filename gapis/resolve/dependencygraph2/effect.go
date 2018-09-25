// Copyright (C) 2018 Google Inc.
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

package dependencygraph2

import (
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/memory"
)

// Effect is a record of a read or write of a piece of state.
type Effect interface {
	// GetNodeID returns the dependency graph node associated with this effect
	GetNodeID() NodeID
	effect()
}

// ReadEffect is a record of a read of a piece of state.
type ReadEffect interface {
	Effect
	readEffect()
}

// WriteEffect is a record of a write of a piece of state.
type WriteEffect interface {
	Effect
	writeEffect()
}

// ReverseEffect is a record of an effect whose dependency goes in both directions (read <-> write)
type ReverseEffect interface {
	Effect
	reverseEffect()
}

type FragmentEffect interface {
	Effect
	// GetFragment returns the Fragment which is read or written by this effect
	GetFragment() api.Fragment
}

// ReadFragmentEffect is a record of a read of a piece of state in the state graph (as opposed to a memory range)
type ReadFragmentEffect struct {
	NodeID   NodeID
	Fragment api.Fragment
}

var _ = FragmentEffect(ReadFragmentEffect{})
var _ = ReadEffect(ReadFragmentEffect{})

// GetNodeID returns the dependency graph node associated with this effect
func (e ReadFragmentEffect) GetNodeID() NodeID {
	return e.NodeID
}

// GetFragment returns the Fragment which is read or written by this effect
func (e ReadFragmentEffect) GetFragment() api.Fragment {
	return e.Fragment
}

func (ReadFragmentEffect) effect()     {}
func (ReadFragmentEffect) readEffect() {}

// WriteFragmentEffect is a record of a write to a piece of state in the state graph (as opposed to a memory range)
type WriteFragmentEffect struct {
	NodeID   NodeID
	Fragment api.Fragment
}

var _ = FragmentEffect(WriteFragmentEffect{})
var _ = WriteEffect(WriteFragmentEffect{})

// GetNodeID returns the dependency graph node associated with this effect
func (e WriteFragmentEffect) GetNodeID() NodeID {
	return e.NodeID
}

// GetFragment returns the Fragment which is read or written by this effect
func (e WriteFragmentEffect) GetFragment() api.Fragment {
	return e.Fragment
}

func (WriteFragmentEffect) effect()      {}
func (WriteFragmentEffect) writeEffect() {}

// ReadMemEffect is a record of a read of a memory range (either application or device memory)
type ReadMemEffect struct {
	NodeID NodeID
	Slice  memory.Slice
}

var _ = ReadEffect(ReadMemEffect{})

// GetNodeID returns the dependency graph node associated with this effect
func (e ReadMemEffect) GetNodeID() NodeID {
	return e.NodeID
}
func (ReadMemEffect) effect()     {}
func (ReadMemEffect) readEffect() {}

// WriteMemEffect is a record of a write to a memory range (either application or device memory)
type WriteMemEffect struct {
	NodeID NodeID
	Slice  memory.Slice
}

var _ = WriteEffect(WriteMemEffect{})

// GetNodeID returns the dependency graph node associated with this effect
func (e WriteMemEffect) GetNodeID() NodeID {
	return e.NodeID
}
func (WriteMemEffect) effect()      {}
func (WriteMemEffect) writeEffect() {}

// OpenForwardDependencyEffect is a record of the *opening* of a forward dependency.
// The opening effect must occur before the corresponding *closing* effect
// (represented by CloseForwardDependencyEffect); the opening node associated
// with the opening of the foward dependency will depend on the node associated
// with the closing.
// E.g. vkAcquireNextImageKHR opens a foward dependency, which is closed by the
// corresponding vkQueuePresentKHR call.
type OpenForwardDependencyEffect struct {
	NodeID       NodeID
	DependencyID interface{}
}

var _ = WriteEffect(OpenForwardDependencyEffect{})
var _ = ReverseEffect(OpenForwardDependencyEffect{})

// GetNodeID returns the dependency graph node associated with this effect
func (e OpenForwardDependencyEffect) GetNodeID() NodeID {
	return e.NodeID
}
func (OpenForwardDependencyEffect) effect()        {}
func (OpenForwardDependencyEffect) writeEffect()   {}
func (OpenForwardDependencyEffect) reverseEffect() {}

// CloseForwardDependencyEffect is a record of the *closing* of a forward dependency.
// See OpenForwardDependencyEffect above.
type CloseForwardDependencyEffect struct {
	NodeID       NodeID
	DependencyID interface{}
}

var _ = ReadEffect(CloseForwardDependencyEffect{})
var _ = ReverseEffect(CloseForwardDependencyEffect{})

// GetNodeID returns the dependency graph node associated with this effect
func (e CloseForwardDependencyEffect) GetNodeID() NodeID {
	return e.NodeID
}
func (CloseForwardDependencyEffect) effect()        {}
func (CloseForwardDependencyEffect) readEffect()    {}
func (CloseForwardDependencyEffect) reverseEffect() {}

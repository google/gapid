////////////////////////////////////////////////////////////////////////////////
// Automatically generated file. Do not modify!
////////////////////////////////////////////////////////////////////////////////

package gles

import (
	"context"
	"unsafe"

	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/gapis/api/gles/gles_pb"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/memory/memory_pb"
)

// Just in case it is not used
var _ memory.PoolID
var _ memory_pb.Pointer
var _ unsafe.Pointer

func init() {
	protoconv.Register(
	func(ctx context.Context, ref_cb func(interface{}) uint64, in *State) (*gles_pb.InitialState, error) { return in.ToProto(ref_cb), nil },
	func(ctx context.Context, ref_cb func(uint64, interface{}), in *gles_pb.InitialState) (*State, error) { v := StateFrom(in, ref_cb); return &v, nil },
)
}

// ToProto returns the storage form of the State.
func (ϟa *State) ToProto(ref_cb func(interface{}) uint64) *gles_pb.InitialState {
to := &gles_pb.InitialState{}
return to
}

// StateFrom builds a State from the storage form.
func StateFrom(from *gles_pb.InitialState,ref_cb func(uint64, interface{})) State {
ϟa := State{}
return ϟa
}

////////////////////////////////////////////////////////////////////////////////
// Automatically generated file. Do not modify!
////////////////////////////////////////////////////////////////////////////////

package gvr

import (
	"context"
	"unsafe"

	"github.com/google/gapid/core/data/protoconv"
	"github.com/google/gapid/gapis/api/gvr/gvr_pb"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/memory/memory_pb"
)

// Just in case it is not used
var _ memory.PoolID
var _ memory_pb.Pointer
var _ unsafe.Pointer

func init() {
	protoconv.Register(
	func(ctx context.Context, ref_cb func(interface{}) uint64, in *State) (*gvr_pb.InitialState, error) { return in.ToProto(ref_cb), nil },
	func(ctx context.Context, ref_cb func(uint64, interface{}), in *gvr_pb.InitialState) (*State, error) { v := StateFrom(in, ref_cb); return &v, nil },
)
}

// ToProto returns the storage form of the State.
func (ϟa *State) ToProto(ref_cb func(interface{}) uint64) *gvr_pb.InitialState {
to := &gvr_pb.InitialState{}
return to
}

// StateFrom builds a State from the storage form.
func StateFrom(from *gvr_pb.InitialState,ref_cb func(uint64, interface{})) State {
ϟa := State{}
return ϟa
}

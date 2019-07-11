// Copyright (C) 2019 Google Inc.
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
	"context"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/protocol"
	"github.com/google/gapid/gapis/replay/value"
)

type stateWatcher struct {
	memoryWrites map[memory.PoolID]*interval.U64SpanList
}

func (b *stateWatcher) OnBeginCmd(ctx context.Context, cmdID api.CmdID, cmd api.Cmd) {
}

func (b *stateWatcher) OnEndCmd(ctx context.Context, cmdID api.CmdID, cmd api.Cmd) {
}
func (b *stateWatcher) OnBeginSubCmd(ctx context.Context, subIdx api.SubCmdIdx, recordIdx api.RecordIdx) {
}
func (b *stateWatcher) OnRecordSubCmd(ctx context.Context, recordIdx api.RecordIdx) {
}
func (b *stateWatcher) OnEndSubCmd(ctx context.Context) {
}
func (b *stateWatcher) OnReadFrag(ctx context.Context, owner api.RefObject, frag api.Fragment, valueRef api.RefObject, track bool) {
}

func (b *stateWatcher) OnWriteFrag(ctx context.Context, owner api.RefObject, frag api.Fragment, oldValueRef api.RefObject, newValueRef api.RefObject, track bool) {

}

func (b *stateWatcher) OnWriteSlice(ctx context.Context, slice memory.Slice) {
	span := interval.U64Span{
		Start: slice.Base(),
		End:   slice.Base() + slice.Size(),
	}
	poolID := slice.Pool()
	if _, ok := b.memoryWrites[poolID]; !ok {
		b.memoryWrites[poolID] = &interval.U64SpanList{}
	}
	interval.Merge(b.memoryWrites[poolID], span, true)
}

func (b *stateWatcher) OnReadSlice(ctx context.Context, slice memory.Slice) {
}

func (b *stateWatcher) OnWriteObs(ctx context.Context, obs []api.CmdObservation) {
}

func (b *stateWatcher) OnReadObs(ctx context.Context, obs []api.CmdObservation) {
}

func (b *stateWatcher) OpenForwardDependency(ctx context.Context, dependencyID interface{}) {
}

func (b *stateWatcher) CloseForwardDependency(ctx context.Context, dependencyID interface{}) {
}

func (b *stateWatcher) DropForwardDependency(ctx context.Context, dependencyID interface{}) {
}

// Transfrom
type frameLoop struct {
	capture        *capture.GraphicsCapture
	cmds           []api.Cmd
	numInitialCmds int
	loopCount      int32
	loopStartCmd   api.Cmd
	loopEndCmd     api.Cmd
	backupState    *api.GlobalState
	watcher        *stateWatcher

	loopCountPtr value.Pointer

	frameNum uint32
}

func newFrameLoop(ctx context.Context, c *capture.GraphicsCapture, numInitialCmds int, Cmds []api.Cmd, loopCount int32) *frameLoop {
	f := &frameLoop{
		capture:        c,
		cmds:           Cmds,
		numInitialCmds: numInitialCmds,
		loopCount:      loopCount,
		watcher: &stateWatcher{
			memoryWrites: make(map[memory.PoolID]*interval.U64SpanList),
		},
	}

	f.loopStartCmd, f.loopEndCmd = f.getLoopStartAndEndCmd(ctx, Cmds)

	return f
}

func (f *frameLoop) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	ctx = log.Enter(ctx, "frameLoop Transform")

	if cmd == f.loopStartCmd {
		log.I(ctx, "Loop: start loop starts at frame %v, id %v and cmd %v", f.frameNum, id, cmd)
		f.backupState = f.capture.NewUninitializedState(ctx)
		f.backupState.Memory = out.State().Memory.Clone()
		for k, v := range out.State().APIs {
			s := v.Clone(f.backupState.Arena)
			s.SetupInitialState(ctx)
			f.backupState.APIs[k] = s
		}
		st := GetState(f.backupState)
		sb := st.newStateBuilder(ctx, newTransformerOutput(out))
		defer sb.ta.Dispose()
		// TODO: Detect changed resources and backup them.

		// Add jump label
		sb.write(sb.cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
			f.loopCountPtr = b.AllocateMemory(4)
			b.Push(value.S32(f.loopCount))
			b.Store(f.loopCountPtr)
			b.JumpLabel(uint32(0x1))
			return nil
		}))
		out.NotifyPreLoop(ctx)
		out.MutateAndWrite(ctx, id, cmd)
		return

	}
	if cmd == f.loopEndCmd {
		log.I(ctx, "Loop: last frame is %v, id %v and cmd is %v", f.frameNum, id, cmd)
		st := GetState(f.backupState)
		sb := st.newStateBuilder(ctx, newTransformerOutput(out))
		defer sb.ta.Dispose()
		out.MutateAndWrite(ctx, id, cmd)
		// Notify this is the end part of the loop to next transformer
		out.NotifyPostLoop(ctx)

		// TODO: reset changed resources

		// Add jump instruction
		sb.write(sb.cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
			b.Load(protocol.Type_Int32, f.loopCountPtr)
			b.Sub(1)
			b.Clone(0)
			b.Store(f.loopCountPtr)
			b.JumpNZ(uint32(0x1))
			return nil
		}))
		return
	}

	if _, ok := cmd.(*VkQueuePresentKHR); ok {
		f.frameNum++
	}

	out.MutateAndWrite(ctx, id, cmd)

}

func (f *frameLoop) Flush(ctx context.Context, out transform.Writer)    {}
func (f *frameLoop) PreLoop(ctx context.Context, out transform.Writer)  {}
func (f *frameLoop) PostLoop(ctx context.Context, out transform.Writer) {}

// TODO: Get the start and end command of the loop, returns the first and the last commands for now.
func (f *frameLoop) getLoopStartAndEndCmd(ctx context.Context, Cmds []api.Cmd) (startCmd, endCmd api.Cmd) {
	startCmd = Cmds[f.numInitialCmds]
	endCmd = Cmds[len(Cmds)-1]

	return startCmd, endCmd
}

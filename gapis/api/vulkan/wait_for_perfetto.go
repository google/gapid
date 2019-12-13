package vulkan

import (
	"bytes"
	"context"

	"github.com/google/gapid/gapir"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/trace"
)

type WaitForPerfetto struct {
	wff   replay.WaitForFence
	cmdId api.CmdID
}

func addVkDeviceWaitIdle(ctx context.Context, out transform.Writer) {
	s := out.State()
	so := getStateObject(s)
	id := api.CmdNoID
	cb := CommandBuilder{Thread: 0, Arena: s.Arena}

	// Wait for all queues in all devices to finish their jobs first.
	for handle := range so.Devices().All() {
		out.MutateAndWrite(ctx, id, cb.VkDeviceWaitIdle(handle, VkResult_VK_SUCCESS))
	}
}

func (t *WaitForPerfetto) waitTest(ctx context.Context, id api.CmdID, cmd api.Cmd) bool {
	if id == t.cmdId {
		return true
	}
	return false
}

func (t *WaitForPerfetto) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	if t.waitTest(ctx, id, cmd) {
		addVkDeviceWaitIdle(ctx, out)
	}
	t.wff.Transform(ctx, id, cmd, out)
}

func (t *WaitForPerfetto) Flush(ctx context.Context, out transform.Writer) {
	addVkDeviceWaitIdle(ctx, out)
	t.wff.Flush(ctx, out)
}

func (t *WaitForPerfetto) PreLoop(ctx context.Context, out transform.Writer)  {}
func (t *WaitForPerfetto) PostLoop(ctx context.Context, out transform.Writer) {}

func NewWaitForPerfetto(traceOptions *service.TraceOptions, h *replay.SignalHandler, buffer *bytes.Buffer, cmdId api.CmdID) *WaitForPerfetto {
	tcb := func(ctx context.Context, p *gapir.FenceReadyRequest) {
		go func() {
			trace.TraceBuffered(ctx, traceOptions.Device, h.StartSignal, h.StopSignal, h.ReadyFunc, traceOptions, buffer)
			if !h.DoneSignal.Fired() {
				h.DoneFunc(ctx)
			}
		}()
		h.ReadySignal.Wait(ctx)
	}

	fcb := func(ctx context.Context, p *gapir.FenceReadyRequest) {
		if !h.StopSignal.Fired() {
			h.StopFunc(ctx)
		}
	}
	wfp := WaitForPerfetto{cmdId: cmdId}
	wfp.wff = replay.WaitForFence{tcb, fcb, wfp.waitTest}

	return &wfp
}

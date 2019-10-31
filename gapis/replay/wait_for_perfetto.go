package replay

import (
	"context"

	"github.com/google/gapid/gapir"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/trace"
)

func WaitForPerfetto(traceOptions *service.TraceOptions, h *SignalHandler) *WaitForFence {
	tcb := func(ctx context.Context, p *gapir.FenceReadyRequest) {
		go func() {
			trace.Trace(ctx, traceOptions.Device, h.StartSignal, h.StopSignal, h.ReadyFunc, traceOptions, &h.Written)
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

	waitTest := func(ctx context.Context, id api.CmdID, cmd api.Cmd) bool {
		if id == 0 {
			return true
		}
		return false
	}

	return &WaitForFence{tcb, fcb, waitTest}
}

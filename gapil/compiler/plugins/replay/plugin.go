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

package replay

import (
	"github.com/google/gapid/core/codegen"
	"github.com/google/gapid/gapil/compiler"
	"github.com/google/gapid/gapil/semantic"
)

// opcodes is the name of the context field that holds the generated opcodes
// buffer.
const (
	// GetReplayOpcodes is the name of the function that retrieves the replay
	// opcodes from the context. It has the signature:
	//
	// buffer_t* get_replay_opcodes(context_t*)
	GetReplayOpcodes = "get_replay_opcodes"

	initialOpcodesCap = 64 << 10

	data = "replay_data" // Additional context field.

	// Fields of context.replay_data:
	opcodes = "opcodes" // buffer of opcodes currently being built
	call    = "call"    // void (*call)(context*)
)

// Plugin is the replay plugin for the gapil compiler.
func Plugin() compiler.Plugin {
	return &replayer{}
}

var (
	_ compiler.ContextDataPlugin      = (*replayer)(nil)
	_ compiler.FunctionExposerPlugin  = (*replayer)(nil)
	_ compiler.OnBeginCommandListener = (*replayer)(nil)
	_ compiler.OnFenceListener        = (*replayer)(nil)
	_ compiler.OnEndCommandListener   = (*replayer)(nil)
)

func (r *replayer) OnPreBuildContext(c *compiler.C) {}

func (r *replayer) ContextData(c *compiler.C) []compiler.ContextField {
	r.callFPtrTy = c.T.Pointer(c.T.Function(nil, c.T.CtxPtr))
	return []compiler.ContextField{
		{
			Name: data,
			Type: c.T.Struct("replay_data",
				codegen.Field{Name: opcodes, Type: c.T.Buf},
				codegen.Field{Name: call, Type: r.callFPtrTy},
			),
			Init: func(s *compiler.S, dataPtr *codegen.Value) {
				c.InitBuffer(s, dataPtr.Index(0, opcodes), s.Scalar(uint32(initialOpcodesCap)))
			},
			Term: func(s *compiler.S, dataPtr *codegen.Value) {
				c.TermBuffer(s, dataPtr.Index(0, opcodes))
			},
		},
	}
}

func (r *replayer) Functions() map[string]*codegen.Function {
	return map[string]*codegen.Function{
		GetReplayOpcodes: r.getOpcodes,
	}
}

func (r *replayer) OnBeginCommand(s *compiler.S, cmd *semantic.Function) {
	callFunc := r.buildCall(cmd)
	s.Ctx.Index(0, data, call).Store(s.FuncAddr(callFunc))
	r.emitLabel(s)
}

func (r *replayer) OnFence(s *compiler.S) {
	r.emitCall(s)
}

func (r *replayer) OnEndCommand(s *compiler.S, cmd *semantic.Function) {

}

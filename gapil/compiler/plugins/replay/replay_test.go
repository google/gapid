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

package replay_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapil"
	"github.com/google/gapid/gapil/compiler"
	"github.com/google/gapid/gapil/compiler/plugins/replay"
	"github.com/google/gapid/gapil/compiler/testutils"
	"github.com/google/gapid/gapil/executor"
	"github.com/google/gapid/gapil/semantic"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/replay/opcode"
	"github.com/google/gapid/gapis/replay/protocol"
)

const baseCmdID = 0x1000

var (
	D = testutils.Encode
	P = testutils.Pad
	R = testutils.R
	W = testutils.W
)

func TestReplay(t *testing.T) {
	ctx := log.Testing(t)

	for _, test := range []test{
		{
			name: "void C()",
			src:  "cmd void C() {}",
			expected: []opcode.Opcode{
				opcode.Label{Value: baseCmdID},
				opcode.Call{},
			},
		}, {
			name: "Check Call FunctionID",
			src:  "cmd void A() {}\ncmd void B() {}\ncmd void C() {}",
			expected: []opcode.Opcode{
				opcode.Label{Value: baseCmdID + 0},
				opcode.Call{FunctionID: 0},
				opcode.Label{Value: baseCmdID + 1},
				opcode.Call{FunctionID: 1},
				opcode.Label{Value: baseCmdID + 2},
				opcode.Call{FunctionID: 2},
			},
		}, {
			name: "Check Call APIIndex",
			src:  "api_index 3\ncmd void A() {}",
			expected: []opcode.Opcode{
				opcode.Label{Value: baseCmdID},
				opcode.Call{ApiIndex: 3},
			},
		}, {
			name: "Push Unsigned No Expand",
			src:  "cmd void A(u32 a, u32 b) {}",
			data: D(
				uint32(0xaaaaa), // Repeating pattern of 1010
				uint32(0x55555), // Repeating pattern of 0101
			),
			expected: []opcode.Opcode{
				opcode.Label{Value: baseCmdID},
				opcode.PushI{DataType: protocol.Type_Uint32, Value: 0xaaaaa},
				opcode.PushI{DataType: protocol.Type_Uint32, Value: 0x55555},
				opcode.Call{},
			},
		}, {
			name: "Push Unsigned One Expand",
			src:  "cmd void A(u32 a, u32 b, u32 c, u32 d) {}",
			data: D(
				uint32(0x100000),   // One bit beyond what can fit in a PushI
				uint32(0x4000000),  // One bit beyond what can fit in a Extend payload
				uint32(0xaaaaaaaa), // 1010101010...
				uint32(0x55555555), // 0101010101...
			),
			expected: []opcode.Opcode{
				opcode.Label{Value: baseCmdID},
				opcode.PushI{DataType: protocol.Type_Uint32, Value: 0},
				opcode.Extend{Value: 0x100000},

				opcode.PushI{DataType: protocol.Type_Uint32, Value: 1},
				opcode.Extend{Value: 0},

				opcode.PushI{DataType: protocol.Type_Uint32, Value: 0x2a},
				opcode.Extend{Value: 0x2aaaaaa},

				opcode.PushI{DataType: protocol.Type_Uint32, Value: 0x15},
				opcode.Extend{Value: 0x1555555},
				opcode.Call{},
			},
		}, {
			name: "Push Signed Positive No Expand",
			src:  "cmd void A(s32 a, s32 b) {}",
			data: D(
				int32(0x2aaaa), // 0010101010...
				int32(0x55555), // 1010101010...
			),
			expected: []opcode.Opcode{
				opcode.Label{Value: baseCmdID},
				opcode.PushI{DataType: protocol.Type_Int32, Value: 0x2aaaa},
				opcode.PushI{DataType: protocol.Type_Int32, Value: 0x55555},
				opcode.Call{},
			},
		}, {
			name: "Push Signed Positive One Expand",
			src:  "cmd void A(s32 a, s32 b, s32 c, s32 d) {}",
			data: D(
				int32(0x80000),    // One bit beyond what can fit in a PushI
				int32(0x4000000),  // One bit beyond what can fit in a Extend payload
				int32(0x2aaaaaaa), // 0010101010...
				int32(0x55555555), // 0101010101...
			),
			expected: []opcode.Opcode{
				opcode.Label{Value: baseCmdID},
				opcode.PushI{DataType: protocol.Type_Int32, Value: 0},
				opcode.Extend{Value: 0x80000},

				opcode.PushI{DataType: protocol.Type_Int32, Value: 1},
				opcode.Extend{Value: 0},

				opcode.PushI{DataType: protocol.Type_Int32, Value: 0x0a},
				opcode.Extend{Value: 0x2aaaaaa},

				opcode.PushI{DataType: protocol.Type_Int32, Value: 0x15},
				opcode.Extend{Value: 0x1555555},
				opcode.Call{},
			},
		}, {
			name: "Push Signed Negative No Expand",
			src:  "cmd void A(s32 a, s32 b, s32 c) {}",
			data: D(
				int32(-1),
				int32(-0x55556), // Repeating pattern of 1010
				int32(-0x2aaab), // Repeating pattern of 0101
			),
			expected: []opcode.Opcode{
				opcode.Label{Value: baseCmdID},
				opcode.PushI{DataType: protocol.Type_Int32, Value: 0xfffff},
				opcode.PushI{DataType: protocol.Type_Int32, Value: 0xaaaaa},
				opcode.PushI{DataType: protocol.Type_Int32, Value: 0xd5555},
				opcode.Call{},
			},
		}, {
			name: "Push Signed Negative One Expand",
			src:  "cmd void A(s32 a, s32 b, s32 c, s32 d) {}",
			data: D(
				int32(-0x100001),  // One bit beyond what can fit in a PushI
				int32(-0x4000001), // One bit beyond what can fit in a Extend payload

				int32(-0x2aaaaaab), // 110101010...
				int32(-0x55555556), // 101010101...
			),
			expected: []opcode.Opcode{
				opcode.Label{Value: baseCmdID},
				opcode.PushI{DataType: protocol.Type_Int32, Value: 0xfffff},
				opcode.Extend{Value: 0x03efffff},

				opcode.PushI{DataType: protocol.Type_Int32, Value: 0xffffe},
				opcode.Extend{Value: 0x03ffffff},

				opcode.PushI{DataType: protocol.Type_Int32, Value: 0xffff5},
				opcode.Extend{Value: 0x1555555},

				opcode.PushI{DataType: protocol.Type_Int32, Value: 0xfffea},
				opcode.Extend{Value: 0x2aaaaaa},
				opcode.Call{},
			},
		}, {
			name: "Push Float No Expand",
			src:  "cmd void A(f32 a, f32 b, f32 c, f32 d, f32 e, f32 f, f32 g) {}",
			data: D(
				float32(-2.0),
				float32(-1.0),
				float32(-0.5),
				float32(0),
				float32(0.5),
				float32(1.0),
				float32(2.0),
			),
			expected: []opcode.Opcode{
				opcode.Label{Value: baseCmdID},
				opcode.PushI{DataType: protocol.Type_Float, Value: 0x180},
				opcode.PushI{DataType: protocol.Type_Float, Value: 0x17f},
				opcode.PushI{DataType: protocol.Type_Float, Value: 0x17e},
				opcode.PushI{DataType: protocol.Type_Float, Value: 0x000},
				opcode.PushI{DataType: protocol.Type_Float, Value: 0x07e},
				opcode.PushI{DataType: protocol.Type_Float, Value: 0x07f},
				opcode.PushI{DataType: protocol.Type_Float, Value: 0x080},
				opcode.Call{},
			},
		}, {
			name: "Push Float Expand",
			src:  "cmd void A(f32 a, f32 b, f32 c, f32 d) {}",
			data: D(
				float32(-3),
				float32(-1.75),
				float32(1.75),
				float32(3),
			),
			expected: []opcode.Opcode{
				opcode.Label{Value: baseCmdID},
				opcode.PushI{DataType: protocol.Type_Float, Value: 0x180},
				opcode.Extend{Value: 0x400000},

				opcode.PushI{DataType: protocol.Type_Float, Value: 0x17F},
				opcode.Extend{Value: 0x600000},

				opcode.PushI{DataType: protocol.Type_Float, Value: 0x07F},
				opcode.Extend{Value: 0x600000},

				opcode.PushI{DataType: protocol.Type_Float, Value: 0x080},
				opcode.Extend{Value: 0x400000},
				opcode.Call{},
			},
		}, {
			name: "Push Double No Expand",
			src:  "cmd void A(f64 a, f64 b, f64 c, f64 d, f64 e, f64 f, f64 g) {}",
			data: D(
				float64(-2.0),
				float64(-1.0),
				float64(-0.5),
				float64(0),
				float64(0.5),
				float64(1.0),
				float64(2.0),
			),
			expected: []opcode.Opcode{
				opcode.Label{Value: baseCmdID},
				opcode.PushI{DataType: protocol.Type_Double, Value: 0xc00},
				opcode.PushI{DataType: protocol.Type_Double, Value: 0xbff},
				opcode.PushI{DataType: protocol.Type_Double, Value: 0xbfe},
				opcode.PushI{DataType: protocol.Type_Double, Value: 0x000},
				opcode.PushI{DataType: protocol.Type_Double, Value: 0x3fe},
				opcode.PushI{DataType: protocol.Type_Double, Value: 0x3ff},
				opcode.PushI{DataType: protocol.Type_Double, Value: 0x400},
				opcode.Call{},
			},
		}, {
			name: "Push Double Expand",
			src:  "cmd void A(f64 a, f64 b, f64 c, f64 d, f64 e) {}",
			data: D(
				float64(-3),
				float64(-1.75),
				float64(1.0/3.0),
				float64(1.75),
				float64(3),
			),
			expected: []opcode.Opcode{
				opcode.Label{Value: baseCmdID},
				opcode.PushI{DataType: protocol.Type_Double, Value: 0xc00},
				opcode.Extend{Value: 0x2000000},
				opcode.Extend{Value: 0x0},

				opcode.PushI{DataType: protocol.Type_Double, Value: 0xbff},
				opcode.Extend{Value: 0x3000000},
				opcode.Extend{Value: 0x0},

				opcode.PushI{DataType: protocol.Type_Double, Value: 0x3fd},
				opcode.Extend{Value: 0x1555555},
				opcode.Extend{Value: 0x1555555},

				opcode.PushI{DataType: protocol.Type_Double, Value: 0x3ff},
				opcode.Extend{Value: 0x3000000},
				opcode.Extend{Value: 0x0},

				opcode.PushI{DataType: protocol.Type_Double, Value: 0x400},
				opcode.Extend{Value: 0x2000000},
				opcode.Extend{Value: 0x0},
				opcode.Call{},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.run(log.SubTest(ctx, t))
		})
	}
}

type test struct {
	name     string
	src      string
	data     []byte
	dump     bool
	expected []opcode.Opcode
}

func (t test) run(ctx context.Context) (succeeded bool) {
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("Panic in test '%v':\n%v", t.name, r))
		}
	}()

	fmt.Printf("--- %s ---\n", t.name)

	ctx = database.Put(ctx, database.NewInMemory(ctx))

	c := &capture.GraphicsCapture{}

	processor := gapil.NewProcessor()
	processor.Loader = gapil.NewDataLoader([]byte(t.src))
	a, errs := processor.Resolve(t.name + ".api")
	if !assert.For(ctx, "Resolve").ThatSlice(errs).Equals(parse.ErrorList{}) {
		return false
	}

	settings := compiler.Settings{
		EmitExec: true,
		Plugins: []compiler.Plugin{
			replay.Plugin(nil),
		},
	}

	program, err := compiler.Compile([]*semantic.API{a}, processor.Mappings, settings)
	if !assert.For(ctx, "Compile").ThatError(err).Succeeded() {
		return false
	}

	exec := executor.New(program, false)
	env := exec.NewEnv(ctx, c)
	defer env.Dispose()

	for i, f := range a.Functions {
		cmd := &testutils.Cmd{N: f.Name(), D: t.data}
		err = env.Execute(ctx, cmd, api.CmdID(baseCmdID+i))
		if !assert.For(ctx, "Execute").ThatError(err).Succeeded() {
			return false
		}
	}

	if t.dump {
		fmt.Println(program.Dump())
	}

	payload, err := replay.Build(env)
	succeeded = assert.For(ctx, "Build").ThatError(err).Succeeded()
	if succeeded {
		got, err := opcode.Disassemble(bytes.NewReader(payload.Opcodes), device.LittleEndian)
		succeeded = assert.For(ctx, "Disassemble").ThatError(err).Succeeded()
		if succeeded {
			succeeded = assert.For(ctx, "opcodes").ThatSlice(got).Equals(t.expected)
		}
	}

	defer func() {
		if !succeeded {
			fmt.Println(program.Dump())
		}
	}()

	return true
}

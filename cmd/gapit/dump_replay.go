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

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"

	"github.com/golang/protobuf/proto"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/os/device"
	replaysrv "github.com/google/gapid/gapir/replay_service"
	"github.com/google/gapid/gapis/replay/opcode"
)

type dumpReplayVerb struct{}

func init() {
	verb := &dumpReplayVerb{}
	app.AddVerb(&app.Verb{
		Name:      "dump_replay",
		ShortHelp: "Prints textual representation of a replay payload.",
		Action:    verb,
	})
}

func (verb *dumpReplayVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "dump_replay expects the path to the payload.bin file", flags.NArg())
		return nil
	}

	path := flags.Arg(0)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	payload := replaysrv.Payload{}
	if err := proto.Unmarshal(data, &payload); err != nil {
		return err
	}

	fmt.Printf("Stack Size:           0x%x\n", payload.StackSize)
	fmt.Printf("Volatile Memory Size: 0x%x\n", payload.VolatileMemorySize)

	// TODO: Constants
	// TODO: Resources

	if err := dumpOpcodes(&payload); err != nil {
		return err
	}

	return nil
}

func dumpOpcodes(payload *replaysrv.Payload) error {
	count := len(payload.Opcodes) / 4
	fmt.Printf("Opcodes:\n")

	f := fmt.Sprintf("%%.%dd: %%v\n", int(math.Round(math.Log10(float64(count)))+0.5))
	r := endian.Reader(bytes.NewReader(payload.Opcodes), device.LittleEndian)
	idx := 0
	for {
		opcode, err := opcode.Decode(r)
		switch err {
		case nil:
			fmt.Printf(f, idx, opcode)
			idx++
		case io.EOF:
			return nil
		default:
			return err
		}
	}
}

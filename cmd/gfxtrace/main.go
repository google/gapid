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
	"io"
	"os"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/atom_pb"
	"github.com/google/gapid/gapis/capture"
	_ "github.com/google/gapid/gapis/gfxapi/all"
)

var (
	packFile   file.Path
	legacyFile file.Path
	timing     bool
)

func main() {
	app.ShortHelp = "Convert is a file format converter for the gapid system."
	app.Version = app.VersionSpec{Major: 0, Minor: 1}
	flag.Var(&packFile, "pack", "the pack file to generate")
	flag.Var(&legacyFile, "legacy", "the legacy file to generate")
	flag.BoolVar(&timing, "time", false, "time the encode and decode performanc3")
	app.Run(run)
}

func run(ctx context.Context) error {
	var (
		atoms, legacyAtoms, packAtoms *atom.List
		protos                        []atom_pb.Atom
		live                          []atom.Atom
		size                          int64
		err                           error
	)
	args := flag.Args()
	if len(args) != 1 {
		return log.Errf(ctx, nil, "Expected 1 argument, got %d", len(args))
	}
	readTime := delta(func() { atoms, size, err = readAtoms(ctx, args[0]) })
	if err != nil {
		return log.Err(ctx, err, "Unable to read source")
	}
	log.I(ctx, "Read %d atoms from %d bytes in %v", len(atoms.Atoms), size, readTime)
	atomsSummary := Summary{}.Compute(atoms.Atoms).List()

	if !legacyFile.IsEmpty() {
		d := delta(func() { err = writeAtoms(ctx, legacyFile.System(), atoms, capture.WriteLegacy) })
		if err != nil {
			return log.Err(ctx, err, "Unable write legacy")
		}
		log.I(ctx, "Wrote %v atoms to legacy file in %v", len(atoms.Atoms), d)
	}
	if !packFile.IsEmpty() {
		d := delta(func() { err = writeAtoms(ctx, packFile.System(), atoms, capture.WritePack) })
		if err != nil {
			return log.Err(ctx, err, "Unable write pack")
		}
		log.I(ctx, "Wrote %v atoms to pack file in %v", len(atoms.Atoms), d)
	}

	if !timing {
		return nil
	}
	// pre allocate a big buffer so we are not measuring alllocations
	legacyData := bytes.NewBuffer(make([]byte, 0, size*2))
	packData := bytes.NewBuffer(make([]byte, 0, size*2))

	// Lets time the pure conversions
	protos = make([]atom_pb.Atom, 0, len(atoms.Atoms)*2)
	toProtoTime := delta(func() {
		err = atom.ConvertAllTo(ctx, atoms, func(ctx context.Context, a atom_pb.Atom) error {
			protos = append(protos, a)
			return nil
		})
	})
	if err != nil {
		return log.Err(ctx, err, "Unable convert to proto")
	}
	log.I(ctx, "Live[%d]->Proto[%d] in %v", len(atoms.Atoms), len(protos), toProtoTime)

	live = make([]atom.Atom, 0, len(atoms.Atoms)*2)
	toLiveTime := delta(func() {
		err = atom.ConvertAllFrom(ctx, protos, func(a atom.Atom) {
			live = append(live, a)
		})
	})
	if err != nil {
		return log.Err(ctx, err, "Unable convert to live")
	}
	log.I(ctx, "Proto[%d]->Live[%d] in %v", len(protos), len(live), toLiveTime)
	liveSummary := Summary{}.Compute(live).List()

	// Time writing in legacy format
	toLegacyTime := delta(func() { err = capture.WriteLegacy(ctx, atoms, legacyData) })
	if err != nil {
		return log.Err(ctx, err, "Unable write legacy")
	}
	log.I(ctx, "Live[%d]->Legacy[%d] in %v", len(atoms.Atoms), legacyData.Len(), toLegacyTime)

	// Time writing in pack format
	toPackTime := delta(func() { err = capture.WritePack(ctx, atoms, packData) })
	if err != nil {
		return log.Err(ctx, err, "Unable write pack")
	}
	log.I(ctx, "Live[%d]->Pack[%d] in %v", len(atoms.Atoms), packData.Len(), toPackTime)

	// Time reading legacy format
	fromLegacyTime := delta(func() { legacyAtoms, err = capture.ReadLegacy(ctx, legacyData) })
	if err != nil {
		return log.Err(ctx, err, "Unable read legacy")
	}
	log.I(ctx, "Legacy->Live[%d] in %v", len(legacyAtoms.Atoms), fromLegacyTime)
	legacySummary := Summary{}.Compute(legacyAtoms.Atoms).List()

	// Time reading pack format
	fromPackTime := delta(func() { packAtoms, err = capture.ReadPack(ctx, packData) })
	if err != nil {
		return log.Err(ctx, err, "Unable read pack")
	}
	log.I(ctx, "Pack->Live[%d] in %v", len(packAtoms.Atoms), fromPackTime)
	packSummary := Summary{}.Compute(packAtoms.Atoms).List()

	//Summarise
	SummaryDiff(log.Enter(ctx, "Live"), atomsSummary, liveSummary)
	SummaryDiff(log.Enter(ctx, "Legacy"), atomsSummary, legacySummary)
	SummaryDiff(log.Enter(ctx, "Pack"), atomsSummary, packSummary)
	//AtomDiff(log.Enter(ctx,"Atom"), atoms.List, packAtoms.List)
	return nil
}

func readAtoms(ctx context.Context, name string) (*atom.List, int64, error) {
	// intial read into memory
	f, err := os.Open(name)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	atoms, err := capture.ReadAny(ctx, f)
	stat, _ := f.Stat()
	return atoms, stat.Size(), err
}

func writeAtoms(ctx context.Context, name string, atoms *atom.List, w func(context.Context, *atom.List, io.Writer) error) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return w(ctx, atoms, f)
}

func delta(action func()) time.Duration {
	stamp := time.Now()
	action()
	return time.Since(stamp)
}

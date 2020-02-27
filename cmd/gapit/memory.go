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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type memoryVerb MemoryFlags

func init() {
	verb := &memoryVerb{}
	app.AddVerb(&app.Verb{
		Name:      "memory",
		ShortHelp: "Prints memory metrics about a capture file",
		Action:    verb,
	})
}

func (verb *memoryVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	client, capture, err := getGapisAndLoadCapture(ctx, verb.Gapis, GapirFlags{}, flags.Arg(0), verb.CaptureFileFlags)
	if err != nil {
		return err
	}
	defer client.Close()

	if len(verb.At) == 0 {
		boxedCapture, err := client.Get(ctx, capture.Path(), nil)
		if err != nil {
			return log.Err(ctx, err, "Failed to load the capture")
		}
		verb.At = []uint64{uint64(boxedCapture.(*service.Capture).NumCommands) - 1}
	}

	boxedVal, err := client.Get(ctx, (&path.Metrics{
		Command:         capture.Command(verb.At[0], verb.At[1:]...),
		MemoryBreakdown: true,
	}).Path(), nil)
	if err != nil {
		return log.Errf(ctx, err, "Failed to load metrics")
	}

	mem := boxedVal.(*api.Metrics).MemoryBreakdown
	if mem == nil {
		return log.Errf(ctx, err, "Loaded metrics do not have memory breakdown")
	}

	allocationFlags := []*service.Constant{}
	if mem.AllocationFlagsIndex != -1 {
		boxedConstants, err := client.Get(ctx, (&path.ConstantSet{
			API:   mem.API,
			Index: mem.AllocationFlagsIndex,
		}).Path(), nil)
		if err != nil {
			return log.Errf(ctx, err, "Failed to load allocation flag names")
		}
		constants := boxedConstants.(*service.ConstantSet)
		// If not a bitfield, we can't compare it against the flags
		if constants.IsBitfield {
			allocationFlags = constants.Constants
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 4, 4, 0, ' ', 0)
	fmt.Fprintf(w, "%v memory allocations\n", len(mem.Allocations))
	sort.Slice(mem.Allocations, func(i, j int) bool {
		return mem.Allocations[i].Handle < mem.Allocations[j].Handle
	})

	for _, alloc := range mem.Allocations {
		fmt.Fprintln(w, "Name:", alloc.Name)
		fmt.Fprintf(w, "\tDevice: \t%v\n", alloc.Device)
		fmt.Fprintf(w, "\tMemory Type: \t%v\n", alloc.MemoryType)
		fmt.Fprintf(w, "\tSize: \t%v\n", alloc.Size)

		if alloc.Flags != 0 && len(allocationFlags) != 0 {
			fmt.Fprintln(w, "\tFlags:")
			for _, f := range allocationFlags {
				if (alloc.Flags & uint32(f.Value)) != 0 {
					fmt.Fprintf(w, "\t\t%v\n", f.Name)
				}
			}
		}

		if alloc.Mapping.Size != 0 {
			fmt.Fprintf(w, "\tMapped into host memory at 0x%x\n",
				alloc.Mapping.MappedAddress)
			fmt.Fprintf(w, "\t\tOffset: \t%v\n", alloc.Mapping.Offset)
			fmt.Fprintf(w, "\t\tSize: \t%v\n", alloc.Mapping.Size)
		}

		bindings := bindingSlice(alloc.Bindings)
		sort.Slice(bindings, bindings.bindingLess)
		fmt.Fprintf(w, "\t%v bindings:\n", len(bindings))
		for _, binding := range bindings {
			var typ string
			switch binding.Type.(type) {
			case *api.MemoryBinding_Buffer:
				typ = "Buffer"
			case *api.MemoryBinding_Image:
				typ = "Image"
			case *api.MemoryBinding_SparseImageBlock:
				typ = "Sparse Image Block"
			case *api.MemoryBinding_SparseImageMetadata:
				typ = "Sparse Image Metadata"
			case *api.MemoryBinding_SparseImageMipTail:
				typ = "Sparse Image Mip Tail"
			case *api.MemoryBinding_SparseOpaqueImageBlock:
				typ = "Sparse Opaque Image Block"
			case *api.MemoryBinding_SparseBufferBlock:
				typ = "Sparse Buffer Block"
			}
			fmt.Fprintf(w, "\t%v: %v\n", typ, binding.Name)

			fmt.Fprintf(w, "\t\tOffset: \t%v\n", binding.Offset)
			fmt.Fprintf(w, "\t\tSize: \t%v\n", binding.Size)

			switch val := binding.Type.(type) {
			case *api.MemoryBinding_SparseImageBlock:
				info := val.SparseImageBlock
				fmt.Fprintf(w, "\t\tBlock Offset: \t(%v, %v)\n",
					info.XOffset, info.YOffset)
				fmt.Fprintf(w, "\t\tBlock Extent: \t(%v, %v)\n",
					info.Width, info.Height)
				fmt.Fprintf(w, "\t\tMip Level: \t%v\n", info.MipLevel)
				fmt.Fprintf(w, "\t\tArray Layer: \t%v\n", info.ArrayLayer)
				fmt.Fprintf(w, "\t\tAspects: \t%v\n", strings.Trim(fmt.Sprint(info.Aspects), "[]"))
			case *api.MemoryBinding_SparseImageMetadata:
				info := val.SparseImageMetadata
				fmt.Fprintf(w, "\t\tArray Layer: \t%v\n", info.ArrayLayer)
				fmt.Fprintf(w, "\t\tMip Tail Offset: \t%v\n", info.Offset)
			case *api.MemoryBinding_SparseImageMipTail:
				info := val.SparseImageMipTail
				fmt.Fprintf(w, "\t\tArray Layer: \t%v\n", info.ArrayLayer)
				fmt.Fprintf(w, "\t\tMip Tail Offset: \t%v\n", info.Offset)
				fmt.Fprintf(w, "\t\tAspects: \t%v\n", strings.Trim(fmt.Sprint(info.Aspects), "[]"))
			case *api.MemoryBinding_SparseOpaqueImageBlock:
				fmt.Fprintf(w, "\t\tImage Memory Offset: \t%v\n",
					val.SparseOpaqueImageBlock.Offset)
			case *api.MemoryBinding_SparseBufferBlock:
				fmt.Fprintf(w, "\t\tBuffer Memory Offset: \t%v\n",
					val.SparseBufferBlock.Offset)
			}
		}

		aliases := bindings.computeAliasing()
		if len(aliases) == 0 {
			fmt.Fprintln(w, "\tNo aliased regions")
		} else {
			fmt.Fprintf(w, "\t%v aliased regions:\n", len(aliases))
			for i, a := range aliases {
				fmt.Fprintf(w, "\t%v:\n", i)
				fmt.Fprintf(w, "\t\tOffset: \t%v\n", a.offset)
				fmt.Fprintf(w, "\t\tSize: \t%v\n", a.size)
				fmt.Fprintf(w, "\t\tShared by:\n")
				for _, s := range a.sharers {
					fmt.Fprintf(w, "\t\t\t%v\n", s)
				}
			}
		}
	}
	w.Flush()
	return nil
}

type bindingSlice []*api.MemoryBinding

func (bindings bindingSlice) bindingLess(i, j int) bool {
	if bindings[i].Offset != bindings[j].Offset {
		return bindings[i].Offset < bindings[j].Offset
	}
	if bindings[i].Size != bindings[j].Size {
		return bindings[i].Size < bindings[j].Size
	}
	return bindings[i].Handle < bindings[j].Handle
}

type alias struct {
	offset uint64
	size   uint64

	sharers []uint64
}

func (bindings bindingSlice) computeAliasing() []alias {
	if len(bindings) == 0 {
		return []alias{}
	}
	startsAt := map[uint64][]uint64{}
	endsAt := map[uint64][]uint64{}
	pointSet := map[uint64]struct{}{}

	for _, b := range bindings {
		start := b.Offset
		end := start + b.Size

		s, _ := startsAt[start]
		startsAt[start] = append(s, b.Handle)
		pointSet[start] = struct{}{}

		e, _ := endsAt[end]
		endsAt[end] = append(e, b.Handle)
		pointSet[end] = struct{}{}
	}

	points := make([]uint64, 0, len(pointSet))
	for k := range pointSet {
		points = append(points, k)
	}
	sort.Slice(points, func(i, j int) bool { return points[i] < points[j] })

	aliases := []alias{}
	active := map[uint64]struct{}{}
	for i, p := range points[:len(points)-1] {
		e, _ := endsAt[p]
		for _, handle := range e {
			delete(active, handle)
		}
		s, _ := startsAt[p]
		for _, handle := range s {
			active[handle] = struct{}{}
		}

		if len(active) > 1 {
			sharers := []uint64{}
			for k := range active {
				sharers = append(sharers, k)
			}
			sort.Slice(sharers, func(i, j int) bool { return sharers[i] < sharers[j] })
			aliases = append(aliases, alias{
				offset:  p,
				size:    points[i+1] - p,
				sharers: sharers,
			})
		}
	}

	return aliases
}

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

package resolve

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/google/gapid/core/app/analytics"
	coreid "github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/memory_box"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/service/types"
)

var (
	tyPointer = reflect.TypeOf((*memory.ReflectPointer)(nil)).Elem()
	tySlice   = reflect.TypeOf((*api.Slice)(nil)).Elem()
)

// filterTypedRanges takes a list of typed ranges and
//   combines ranges that have identical bases and types, and
//   filters some highly correlated structs into smaller number.
func filterTypedRanges(ranges []*service.TypedMemoryRange) ([]*service.TypedMemoryRange, error) {
	if len(ranges) == 0 {
		return ranges, nil
	}
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].Root < ranges[j].Root {
			return true
		} else if ranges[i].Root != ranges[j].Root {
			return false
		} else if ranges[i].Type.TypeIndex < ranges[j].Type.TypeIndex {
			return true
		} else if ranges[i].Type.TypeIndex != ranges[j].Type.TypeIndex {
			return false
		} else if ranges[i].Range.Base < ranges[j].Range.Base {
			return true
		} else if ranges[i].Range.Base != ranges[j].Range.Base {
			return false
		} else if a, b := ranges[i].Write, ranges[j].Write; a != b {
			return a // Writes should win, and be first.
		} else {
			return false
		}
	})
	// Combines ranges that have identical bases and types.
	newRanges := []*service.TypedMemoryRange{ranges[0]}
	last := 0
	for i := 1; i < len(ranges); i++ {
		if newRanges[last].Root == ranges[i].Root &&
			newRanges[last].Type.TypeIndex == ranges[i].Type.TypeIndex {
			start := newRanges[last].Range.Base
			if start > ranges[i].Range.Base {
				start = ranges[i].Range.Base
			}
			end := newRanges[last].Range.Base + newRanges[last].Range.Size
			if end < ranges[i].Range.Base+ranges[i].Range.Size {
				end = ranges[i].Range.Base + ranges[i].Range.Size
			}

			if newRanges[last].Range.Size != end-start {
				if sl, ok := newRanges[last].Value.Val.(*memory_box.Value_Slice); ok {
					newRanges[last].Value = &memory_box.Value{
						Val: &memory_box.Value_Slice{
							Slice: &memory_box.Slice{
								Values: append(sl.Slice.GetValues(), ranges[i].GetValue().GetSlice().GetValues()...),
							}}}
				}
			}

			newRanges[last].Range.Base = start
			newRanges[last].Range.Size = end - start
		} else {
			last = len(newRanges)
			newRanges = append(newRanges, ranges[i])
		}
	}
	// Filter pnext pointer related structs, only keep the most specific one.
	// When seeing a pointer struct, scan all the structs having the same root,
	// and identify whether some are less precise and excessive structs caused by
	// pnext pointer type castings. If found, filter them out.
	toDelete := map[int]bool{}
	for index := 0; index < len(newRanges); index++ {
		ty, err := types.GetType(newRanges[index].Type.TypeIndex)
		if err != nil {
			return nil, err
		}
		if strings.Contains(strings.ToLower(ty.Name), "void") {
			root := newRanges[index].Root
			dups := []int{}
			for left := index - 1; left >= 0 && newRanges[left].Root == root; left-- {
				dups = append(dups, left)
			}
			for right := index + 1; right < len(newRanges) && newRanges[right].Root == root; right++ {
				dups = append(dups, right)
			}
			if len(dups) == 0 {
				continue
			}

			mostPreciseIndex := index
			mostPreciseLevel, err := pNextPrecision(newRanges[index].Type.TypeIndex)
			if err != nil {
				return nil, err
			}
			for _, i := range dups {
				preciseLevel, err := pNextPrecision(newRanges[i].Type.TypeIndex)
				if err != nil {
					return nil, err
				}
				if preciseLevel < 0 {
					continue
				} else if preciseLevel < mostPreciseLevel {
					toDelete[i] = true
				} else if preciseLevel > mostPreciseLevel {
					toDelete[mostPreciseIndex] = true
					mostPreciseIndex = i
					mostPreciseLevel = preciseLevel
				}
			}
			if skip := dups[len(dups)-1]; skip > index {
				index = skip
			}
		}
	}

	filteredRanges := []*service.TypedMemoryRange{}
	for i := 0; i < len(newRanges); i++ {
		if shouldDelete := toDelete[i]; !shouldDelete {
			filteredRanges = append(filteredRanges, newRanges[i])
		}
	}
	return filteredRanges, nil
}

// pNextPrecision returns the precision of a pnext pointer derived struct.
// The larger the number, the more precise and specific the struct is. Return -1
// for irrelevant structs.
// The ranking is based on how the pnext pointers are type casted in api files,
// a typical casting chain could be found at gapis/api/vulkan/api/device.api.
func pNextPrecision(typeIndex uint64) (int, error) {
	ty, err := types.GetType(typeIndex)
	if err != nil {
		return -1, err
	}
	tyName := strings.ToLower(ty.Name)
	if strings.Contains(tyName, "void") {
		return 0, nil
	} else if strings.HasPrefix(tyName, "vkstructuretype") {
		return 1, nil
	} else if strings.HasPrefix(tyName, "vulkanstructheader") {
		return 2, nil
	} else if strings.HasPrefix(tyName, "vk") {
		return 3, nil
	} else {
		return -1, nil
	}
}

// expandCharArrays goes through the range and converts the int value to a char value
// and attaches a more human-readable representation to char arrays/slices
func expandCharArrays(typedRanges []*service.TypedMemoryRange) ([]*service.TypedMemoryRange, error) {
	if len(typedRanges) == 0 {
		return typedRanges, nil
	}

	for _, typedRange := range typedRanges {
		memType, err := types.GetType(typedRange.Type.TypeIndex)
		if err != nil {
			return typedRanges, err
		}

		if slice := memType.GetSlice(); slice != nil && slice.GetUnderlying() == types.CharType {
			values := typedRange.Value.GetSlice().GetValues()
			typedRange.Value.GetSlice().Representation = stringFromValues(values)
			convertValueTypeToChar(values)
		} else if arr := memType.GetArray(); arr != nil && arr.GetElementType() == types.CharType {
			entries := typedRange.Value.GetArray().GetEntries()
			typedRange.Value.GetArray().Representation = stringFromValues(entries)
			convertValueTypeToChar(entries)
		}
	}
	return typedRanges, nil
}

// stringFromValues converts the int values to a string that omits the
// null terminator and is enclosed by quotation marks (")
func stringFromValues(values []*memory_box.Value) *pod.Value {
	var sb strings.Builder

	// Pre-allocate array to include quotation marks but not null terminator
	sb.Grow(len(values) + 1)
	sb.WriteString("\"")
	for _, v := range values {
		char := v.GetPod().GetUint8()

		// Skip null terminator in string representation to avoid formatting issues
		if char != 0 {
			sb.WriteString(string(char))
		}
	}
	sb.WriteString("\"")
	return &pod.Value{
		Val: &pod.Value_String_{
			String_: sb.String(),
		},
	}
}

// convertValueTypeToChar converts the int values in the array to char
func convertValueTypeToChar(values []*memory_box.Value) {
	for i, v := range values {
		values[i].GetPod().Val = &pod.Value_Char{
			Char: v.GetPod().GetUint8(),
		}
	}
}

// Memory resolves and returns the memory from the path p.
func Memory(ctx context.Context, p *path.Memory, rc *path.ResolveConfig) (*service.Memory, error) {
	ctx = SetupContext(ctx, path.FindCapture(p), rc)

	cmdIdx := p.After.Indices[0]
	fullCmdIdx := p.After.Indices

	allCmds, err := Cmds(ctx, path.FindCapture(p))
	if err != nil {
		return nil, err
	}

	if count := uint64(len(allCmds)); cmdIdx >= count {
		return nil, errPathOOB(cmdIdx, "Index", 0, count-1, p)
	}

	sd, err := SyncData(ctx, path.FindCapture(p))
	if err != nil {
		return nil, err
	}

	cmds, err := sync.MutationCmdsFor(ctx, path.FindCapture(p), sd, allCmds, api.CmdID(cmdIdx), fullCmdIdx[1:], true)
	if err != nil {
		return nil, err
	}

	defer analytics.SendTiming("resolve", "memory")(analytics.Count(len(cmds)))

	s, err := capture.NewState(ctx)
	if err != nil {
		return nil, err
	}
	err = api.ForeachCmd(ctx, cmds[:len(cmds)-1], true, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		if err := cmd.Mutate(ctx, id, s, nil, nil); err != nil {
			return fmt.Errorf("Fail to mutate command %v: %v", cmd, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	r := memory.Range{Base: p.Address, Size: p.Size}
	var reads, writes, observed memory.RangeList
	typedRanges := []*service.TypedMemoryRange{}

	s.Memory.SetOnCreate(func(poolID memory.PoolID, pool *memory.Pool) {
		if poolID == memory.PoolID(p.Pool) {
			pool.OnRead = func(rng memory.Range, root uint64, typeID uint64, apiId coreid.ID) {
				if rng.Overlaps(r) {
					interval.Merge(&reads, rng.Window(r).Span(), false)
					if p.IncludeTypes {
						value, err := memoryAsType(ctx, s, rng, pool, typeID, p, rc)
						if err != nil {
							return
						}
						typedRanges = append(typedRanges,
							&service.TypedMemoryRange{
								Type: &path.Type{
									TypeIndex: typeID,
									API:       &path.API{ID: path.NewID(apiId)},
								},
								Range: &service.MemoryRange{
									Base: rng.Base,
									Size: rng.Size,
								},
								Root:  root,
								Value: value,
								Write: false,
							},
						)
					}
				}
			}
			pool.OnWrite = func(rng memory.Range, root uint64, typeID uint64, apiId coreid.ID) {
				if rng.Overlaps(r) {
					interval.Merge(&writes, rng.Window(r).Span(), false)
					if p.IncludeTypes {
						value, err := memoryAsType(ctx, s, rng, pool, typeID, p, rc)
						if err != nil {
							return
						}
						typedRanges = append(typedRanges,
							&service.TypedMemoryRange{
								Type: &path.Type{
									TypeIndex: typeID,
									API:       &path.API{ID: path.NewID(apiId)},
								},
								Range: &service.MemoryRange{
									Base: rng.Base,
									Size: rng.Size,
								},
								Root:  root,
								Value: value,
								Write: true,
							},
						)
					}
				}
			}
		}
	})

	lastCmd := cmds[len(cmds)-1]
	err = api.MutateCmds(ctx, s, nil, nil, lastCmd)
	if err != nil {
		return nil, err
	}

	typedRanges, err = filterTypedRanges(typedRanges)
	if err != nil {
		return nil, err
	}

	typedRanges, err = expandCharArrays(typedRanges)
	if err != nil {
		return nil, err
	}

	// Check whether the requested pool was ever created.
	pool, err := s.Memory.Get(memory.PoolID(p.Pool))
	if err != nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrInvalidMemoryPool(p.Pool)}
	}

	slice := pool.Slice(r)

	if !p.ExcludeObserved {
		observed = slice.ValidRanges()
	}

	var data []byte
	if !p.ExcludeData && slice.Size() > 0 {
		data = make([]byte, slice.Size())
		if err := slice.Get(ctx, 0, data); err != nil {
			return nil, err
		}
	}

	return &service.Memory{
		Data:        data,
		Reads:       service.NewMemoryRanges(reads),
		Writes:      service.NewMemoryRanges(writes),
		Observed:    service.NewMemoryRanges(observed),
		TypedRanges: typedRanges,
	}, nil
}

// MemoryAsType resolves and returns the memory from the path p.
// This is a resolving function accessible indirectly to gapic.
func MemoryAsType(ctx context.Context, p *path.MemoryAsType, rc *path.ResolveConfig) (*memory_box.Value, error) {
	ctx = SetupContext(ctx, path.FindCapture(p), rc)

	cmdIdx := p.After.Indices[0]
	fullCmdIdx := p.After.Indices

	allCmds, err := Cmds(ctx, path.FindCapture(p))
	if err != nil {
		return nil, err
	}

	if count := uint64(len(allCmds)); cmdIdx >= count {
		return nil, errPathOOB(cmdIdx, "Index", 0, count-1, p)
	}

	sd, err := SyncData(ctx, path.FindCapture(p))
	if err != nil {
		return nil, err
	}

	cmds, err := sync.MutationCmdsFor(ctx, path.FindCapture(p), sd, allCmds, api.CmdID(cmdIdx), fullCmdIdx[1:], true)
	if err != nil {
		return nil, err
	}

	defer analytics.SendTiming("resolve", "memory")(analytics.Count(len(cmds)))

	s, err := capture.NewState(ctx)
	if err != nil {
		return nil, err
	}
	err = api.ForeachCmd(ctx, cmds, true, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		if err := cmd.Mutate(ctx, id, s, nil, nil); err != nil {
			return fmt.Errorf("Fail to mutate command %v: %v", cmd, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Check whether the requested pool was ever created.
	pool, err := s.Memory.Get(memory.PoolID(p.Pool))
	if err != nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrInvalidMemoryPool(p.Pool)}
	}

	rng := memory.Range{Base: p.Address, Size: p.Size}
	return memoryAsType(ctx, s, rng, pool, p.Type.TypeIndex, p, rc)
}

// Function memoryAsType resolves typed memory related raw data,
// and return the decoded memory.
// This is a private resolving function inside gapis,.
func memoryAsType(ctx context.Context, s *api.GlobalState, rng memory.Range, pool *memory.Pool, typeIndex uint64,
	p path.Node, rc *path.ResolveConfig) (*memory_box.Value, error) {
	sz := rng.Size
	if sz == 0 {
		sz = 0xFFFFFFFFFFFFFFFF
	}

	dec := s.MemoryDecoder(ctx, pool.Slice(memory.Range{
		Base: rng.Base,
		Size: sz,
	}))

	ty, err := types.GetType(typeIndex)
	if err != nil {
		return nil, err
	}

	nElems := 1
	elemSize := 0
	isSlice := false
	if sl, ok := ty.Ty.(*types.Type_Slice); ok {
		isSlice = true
		sliceType, err := types.GetType(sl.Slice.Underlying)
		if err != nil {
			return nil, err
		}
		elemSize, err = sliceType.Size(ctx, s.MemoryLayout)
		if err != nil {
			return nil, err
		}
		if sz == 0 {
			return nil, log.Err(ctx, nil, "Cannot have an unsized range with a slice")
		}
		nElems = int(sz / uint64(elemSize))
		ty = sliceType
	} else {
		elemSize, err = ty.Size(ctx, s.MemoryLayout)
		if err != nil {
			return nil, err
		}
	}
	vals := []*memory_box.Value{}
	for i := 0; i < nElems; i++ {
		v, err := memory_box.Box(ctx, dec, ty, p, rc)
		if err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}

	if isSlice {
		return &memory_box.Value{
			Val: &memory_box.Value_Slice{
				Slice: &memory_box.Slice{
					Values: vals,
				}}}, nil
	}

	return vals[0], nil
}

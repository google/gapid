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
	"strconv"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/slice"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/box"
	"github.com/google/gapid/gapis/service/path"
)

// StateTree resolves the specified state tree path.
func StateTree(ctx context.Context, c *path.StateTree) (*service.StateTree, error) {
	id, err := database.Store(ctx, &StateTreeResolvable{c.After.StateAfter(), c.ArrayGroupSize})
	if err != nil {
		return nil, err
	}
	return &service.StateTree{
		Root: &path.StateTreeNode{Tree: path.NewID(id)},
	}, nil
}

type stateTree struct {
	state      *gfxapi.State
	apiState   interface{}
	path       *path.State
	api        *path.API
	groupLimit uint64
}

func (t stateTree) needsSubgrouping(size uint64) bool {
	return t.groupLimit > 0 && size > t.groupLimit
}

func (t stateTree) limit(numChildren uint64) uint64 {
	if t.groupLimit > 0 && numChildren > t.groupLimit {
		numChildren = (numChildren + t.groupLimit - 1) / t.groupLimit
		if numChildren > t.groupLimit {
			numChildren = (numChildren + t.groupLimit - 1) / t.groupLimit
		}
	}
	return numChildren
}

func (t stateTree) subrange(size, idx uint64) (i, j uint64, name string) {
	if size > t.groupLimit*t.groupLimit {
		i = idx * t.groupLimit * t.groupLimit
		j = i + t.groupLimit*t.groupLimit
	} else {
		i = idx * t.groupLimit
		j = i + t.groupLimit
	}
	if j > size {
		j = size
	}
	name = fmt.Sprintf("[%d - %d]", i, j-1)
	return
}

func deref(v reflect.Value) reflect.Value {
	for (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && !v.IsNil() {
		v = v.Elem()
	}
	return v
}

// StateTreeNode resolves the specified command tree node path.
func StateTreeNode(ctx context.Context, p *path.StateTreeNode) (*service.StateTreeNode, error) {
	boxed, err := database.Resolve(ctx, p.Tree.ID())
	if err != nil {
		return nil, err
	}
	return stateTreeNode(ctx, boxed.(*stateTree), p)
}

func stateTreeNode(ctx context.Context, tree *stateTree, p *path.StateTreeNode) (*service.StateTreeNode, error) {
	name, pth, consts := "root", path.Node(tree.path), (*path.ConstantSet)(nil)
	v := deref(reflect.ValueOf(tree.apiState))

	numChildren := uint64(visibleFieldCount(v.Type()))
	syntheticOffset := uint64(0)

	for i, idx64 := range p.Indices {
		idx := int(idx64)
		if idx64 >= numChildren {
			at := &path.StateTreeNode{Tree: p.Tree, Indices: p.Indices[:i+1]}
			return nil, errPathOOB(idx64, "Index", 0, numChildren-1, at)
		}

		t := v.Type()
		switch {
		case box.IsMemorySlice(t):
			slice := box.AsMemorySlice(v)
			if size := slice.Count(); tree.needsSubgrouping(size) {
				i, j, n := tree.subrange(size, idx64)
				name = n
				v = reflect.ValueOf(slice.ISlice(i, j, tree.state.MemoryLayout))
				syntheticOffset += i
			} else {
				name = fmt.Sprint(syntheticOffset + idx64)
				pth = path.NewArrayIndex(syntheticOffset+idx64, pth)
				ptr := slice.IIndex(idx64, tree.state.MemoryLayout)
				el, err := memory.LoadPointer(ctx, ptr, tree.state.Memory, tree.state.MemoryLayout)
				if err != nil {
					return nil, err
				}
				v = reflect.ValueOf(el)
				syntheticOffset = 0
			}
		default:
			switch v.Kind() {
			case reflect.Struct:
				f, t := visibleField(v, idx)
				if cs, ok := t.Tag.Lookup("constset"); ok {
					if idx, _ := strconv.Atoi(cs); idx > 0 {
						consts = tree.api.ConstantSet(idx)
					}
				}
				name = t.Name
				pth = path.NewField(name, pth)
				v = deref(f)
			case reflect.Slice, reflect.Array:
				if size := uint64(v.Len()); tree.needsSubgrouping(size) {
					i, j, n := tree.subrange(size, idx64)
					name = n
					v = v.Slice(int(i), int(j))
					syntheticOffset += i
				} else {
					name = fmt.Sprint(syntheticOffset + idx64)
					pth = path.NewArrayIndex(syntheticOffset+idx64, pth)
					v = deref(v.Index(idx))
					syntheticOffset = 0
				}
			case reflect.Map:
				keys := v.MapKeys()
				slice.SortValues(keys, v.Type().Key())
				key := keys[idx]
				name = fmt.Sprint(key.Interface())
				pth = path.NewMapIndex(key.Interface(), pth)
				v = deref(v.MapIndex(key))
			default:
				return nil, fmt.Errorf("Cannot index type %v (%v)", v.Type(), v.Kind())
			}
		}

		t = v.Type()
		switch {
		case box.IsMemoryPointer(t):
			numChildren = 0
		case box.IsMemorySlice(t):
			numChildren = tree.limit(box.AsMemorySlice(v).Count())
		default:
			switch v.Kind() {
			case reflect.Struct:
				numChildren = uint64(visibleFieldCount(t))
			case reflect.Slice, reflect.Array:
				numChildren = tree.limit(uint64(v.Len()))
			case reflect.Map:
				numChildren = uint64(v.Len())
			default:
				numChildren = 0
			}
		}
	}

	preview, previewIsValue := stateValuePreview(v)

	return &service.StateTreeNode{
		NumChildren:    numChildren,
		Name:           name,
		ValuePath:      pth.Path(),
		Preview:        preview,
		PreviewIsValue: previewIsValue,
		Constants:      consts,
	}, nil
}

func isFieldVisible(t reflect.Type, i int) bool {
	f := t.Field(i)
	return f.PkgPath == "" && f.Tag.Get("nobox") != "true"
}

func visibleFieldCount(t reflect.Type) int {
	count := 0
	for i, c := 0, t.NumField(); i < c; i++ {
		if isFieldVisible(t, i) {
			count++
		}
	}
	return count
}

func visibleField(v reflect.Value, idx int) (reflect.Value, reflect.StructField) {
	t := v.Type()
	count := 0
	for i, c := 0, v.NumField(); i < c; i++ {
		if !isFieldVisible(t, i) {
			continue
		}
		if count == idx {
			return v.Field(i), t.Field(i)
		}
		count++
	}
	return reflect.Value{}, reflect.StructField{}
}

func stateValuePreview(v reflect.Value) (*box.Value, bool) {
	t := v.Type()
	switch {
	case box.IsMemoryPointer(t), box.IsMemorySlice(t):
		return box.NewValue(v.Interface()), true
	}

	switch v.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return box.NewValue(v.Interface()), true
	case reflect.Array, reflect.Slice:
		const maxLen = 4
		if v.Len() > maxLen {
			return box.NewValue(v.Slice(0, maxLen).Interface()), false
		}
		return box.NewValue(v.Interface()), true
	case reflect.String:
		const maxLen = 256
		runes := []rune(v.Interface().(string))
		if len(runes) > maxLen {
			return box.NewValue(string(append(runes[:maxLen-1], '…'))), false
		}
		return box.NewValue(v.Interface()), true
	case reflect.Interface, reflect.Ptr:
		if v.IsNil() {
			return box.NewValue(v.Interface()), true
		}
		return stateValuePreview(v.Elem())
	default:
		return nil, false
	}
}

// Resolve builds and returns a *StateTree for the path.StateTreeNode.
// Resolve implements the database.Resolver interface.
func (r *StateTreeResolvable) Resolve(ctx context.Context) (interface{}, error) {
	state, err := GlobalState(ctx, r.Path)
	if err != nil {
		return nil, err
	}
	c, err := capture.ResolveFromPath(ctx, r.Path.After.Capture)
	if err != nil {
		return nil, err
	}
	atomIdx := r.Path.After.Indices[0]
	if len(r.Path.After.Indices) > 1 {
		return nil, fmt.Errorf("Subcommands currently not supported") // TODO: Subcommands
	}
	api := c.Atoms[atomIdx].API()
	apiState := state.APIs[api]
	apiPath := &path.API{Id: path.NewID(id.ID(api.ID()))}
	return &stateTree{state, apiState, r.Path, apiPath, uint64(r.ArrayGroupSize)}, nil
}

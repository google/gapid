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
	"sync"

	"github.com/google/gapid/core/data/dictionary"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/math/u64"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/box"
	"github.com/google/gapid/gapis/service/path"
)

// StateTree resolves the specified state tree path.
func StateTree(ctx context.Context, c *path.StateTree, r *path.ResolveConfig) (*service.StateTree, error) {
	id, err := database.Store(ctx, &StateTreeResolvable{
		Path:           c.State,
		ArrayGroupSize: c.ArrayGroupSize,
		Config:         r,
	})
	if err != nil {
		return nil, err
	}
	return &service.StateTree{
		Root: &path.StateTreeNode{Tree: path.NewID(id)},
	}, nil
}

type stateTree struct {
	globalState *api.GlobalState
	state       interface{}
	root        *stn
	api         *path.API
	groupLimit  uint64
}

// needsSubgrouping returns true if the child count exceeds the group limit and
// grouping is desired (groupLimit > 0).
func needsSubgrouping(groupLimit, childCount uint64) bool {
	return groupLimit > 0 && childCount > groupLimit
}

// subgroupSize returns the maximum number of entries in each subgroup.
func subgroupSize(groupLimit, childCount uint64) uint64 {
	if !needsSubgrouping(groupLimit, childCount) {
		return 1
	}
	groupSize := uint64(1)
	for (childCount+groupSize-1)/groupSize > groupLimit {
		groupSize *= groupLimit
	}
	return groupSize
}

// subgroupCount returns the number of immediate children for a given group,
// taking into consideration group limits.
func subgroupCount(groupLimit, childCount uint64) uint64 {
	groupSize := subgroupSize(groupLimit, childCount)
	return (childCount + groupSize - 1) / groupSize
}

// subgroupRange returns the start and end indices (s, e) for the i'th immediate
// child for the given group. e is one greater than the last index in the
// subgroup.
func subgroupRange(groupLimit, childCount, i uint64) (s, e uint64) {
	groupSize := subgroupSize(groupLimit, childCount)
	s = i * groupSize
	e = u64.Min(s+groupSize, childCount)
	return s, e
}

func deref(v reflect.Value) reflect.Value {
	for (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && !v.IsNil() {
		v = v.Elem()
	}
	return v
}

// StateTreeNode resolves the specified command tree node path.
func StateTreeNode(ctx context.Context, p *path.StateTreeNode, r *path.ResolveConfig) (*service.StateTreeNode, error) {
	boxed, err := database.Resolve(ctx, p.Tree.ID())
	if err != nil {
		return nil, err
	}
	return stateTreeNode(ctx, boxed.(*stateTree), p)
}

// StateTreeNodeForPath returns the path to the StateTreeNode representing the
// path p.
func StateTreeNodeForPath(ctx context.Context, p *path.StateTreeNodeForPath, r *path.ResolveConfig) (*path.StateTreeNode, error) {
	boxed, err := database.Resolve(ctx, p.Tree.ID())
	if err != nil {
		return nil, err
	}
	indices, err := stateTreeNodePath(ctx, boxed.(*stateTree), p.Member.Node())
	if err != nil {
		return nil, err
	}
	return &path.StateTreeNode{
		Tree:    p.Tree,
		Indices: indices,
	}, nil
}

func stateTreeNode(ctx context.Context, tree *stateTree, p *path.StateTreeNode) (*service.StateTreeNode, error) {
	node := tree.root
	for i, idx64 := range p.Indices {
		var err error
		node, err = node.index(ctx, idx64, tree)
		switch err := err.(type) {
		case nil:
		case errIndexOOB:
			at := &path.StateTreeNode{Tree: p.Tree, Indices: p.Indices[:i+1]}
			return nil, errPathOOB(err.idx, "Index", 0, err.count-1, at)
		default:
			return nil, err
		}
	}
	return node.service(ctx, tree), nil
}

func stateTreeNodePath(ctx context.Context, tree *stateTree, p path.Node) ([]uint64, error) {
	n := tree.root
	indices := []uint64{}
	for {
		ci := n.findByPath(ctx, p, tree)
		if ci == nil {
			break
		}
		for _, i := range ci {
			n, indices = n.children[i], append(indices, i)
		}
	}
	return indices, nil
}

type errIndexOOB struct {
	idx, count uint64
}

func (e errIndexOOB) Error() string { return fmt.Sprintf("index %d out of bounds", e.idx) }

type stn struct {
	mutex          sync.Mutex
	name           string
	label          string
	value          reflect.Value
	path           path.Node
	consts         *path.ConstantSet
	children       []*stn
	isSubgroup     bool
	subgroupOffset uint64
}

func (n *stn) index(ctx context.Context, i uint64, tree *stateTree) (*stn, error) {
	n.buildChildren(ctx, tree)
	if count := uint64(len(n.children)); i >= count {
		return nil, errIndexOOB{i, count}
	}
	return n.children[i], nil
}

func (n *stn) findByPath(ctx context.Context, p path.Node, tree *stateTree) []uint64 {
	n.buildChildren(ctx, tree)
	for i, c := range n.children {
		if path.HasRoot(p, c.path) {
			return []uint64{uint64(i)}
		}
	}
	for i, c := range n.children {
		if c.isSubgroup {
			if ci := c.findByPath(ctx, p, tree); ci != nil {
				return append([]uint64{uint64(i)}, ci...)
			}
		}
	}
	return nil
}

func (n *stn) buildChildren(ctx context.Context, tree *stateTree) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.children != nil {
		return
	}

	v, t, children := n.value, n.value.Type(), []*stn{}

	dict := dictionary.From(v.Interface())

	switch {
	case dict != nil:
		for _, key := range dict.Keys() {
			child := &stn{
				name:  fmt.Sprint(key),
				value: deref(reflect.ValueOf(dict.Get(key))),
				path:  path.NewMapIndex(key, n.path),
			}
			if labeled, ok := dict.Get(key).(api.Labeled); ok {
				child.label = labeled.Label(ctx, tree.globalState)
			}
			children = append(children, child)
		}

	case box.IsMemorySlice(t):
		slice := box.AsMemorySlice(v)
		count := slice.Count()
		if needsSubgrouping(tree.groupLimit, count) {
			for i, c := uint64(0), subgroupCount(tree.groupLimit, count); i < c; i++ {
				s, e := subgroupRange(tree.groupLimit, count, i)
				children = append(children, &stn{
					name:           fmt.Sprintf("[%d - %d]", n.subgroupOffset+s, n.subgroupOffset+e-1),
					value:          reflect.ValueOf(slice.ISlice(s, e)),
					path:           path.NewSlice(s, e-1, n.path),
					isSubgroup:     true,
					subgroupOffset: n.subgroupOffset + s,
				})
			}
		} else {
			slice := slice.ISlice(0, count)
			els, err := memory.LoadSlice(ctx, slice, tree.globalState.Memory, tree.globalState.MemoryLayout)
			if err != nil {
				panic(err)
			}
			arr := reflect.ValueOf(els)
			for i := uint64(0); i < count; i++ {
				el := arr.Index(int(i))
				children = append(children, &stn{
					name:  fmt.Sprint(n.subgroupOffset + i),
					value: el,
					path:  path.NewArrayIndex(i, n.path),
				})
			}
		}

	default:
		switch v.Kind() {
		case reflect.Slice, reflect.Array:
			size := uint64(v.Len())
			if needsSubgrouping(tree.groupLimit, size) {
				for i, c := uint64(0), subgroupCount(tree.groupLimit, size); i < c; i++ {
					s, e := subgroupRange(tree.groupLimit, size, i)
					children = append(children, &stn{
						name:           fmt.Sprintf("[%d - %d]", n.subgroupOffset+s, n.subgroupOffset+e-1),
						value:          v.Slice(int(s), int(e)),
						path:           path.NewSlice(s, e-1, n.path),
						isSubgroup:     true,
						subgroupOffset: n.subgroupOffset + s,
					})
				}
			} else {
				for i := uint64(0); i < size; i++ {
					children = append(children, &stn{
						name:  fmt.Sprint(n.subgroupOffset + i),
						value: deref(v.Index(int(i))),
						path:  path.NewArrayIndex(i, n.path),
					})
				}
			}
		default:
			if isNil(v) {
				break
			}
			pp, ok := v.Interface().(api.PropertyProvider)
			if !ok {
				break
			}
			for _, p := range pp.Properties() {
				var consts *path.ConstantSet
				if p.Constants >= 0 {
					consts = tree.api.ConstantSet(p.Constants)
				}
				children = append(children, &stn{
					name:   p.Name,
					value:  deref(reflect.ValueOf(p.Get())),
					path:   path.NewField(p.Name, n.path),
					consts: consts,
				})
			}
		}
	}

	n.children = children
}

// isNil returns true if v is a nil pointer or interface, or is a type that
// implements the method:
//   IsNil() bool
// which returns true when called.
func isNil(v reflect.Value) bool {
	if (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && v.IsNil() {
		return true
	}
	if i, ok := v.Interface().(interface {
		IsNil() bool
	}); ok {
		return i.IsNil()
	}
	return false
}

func (n *stn) service(ctx context.Context, tree *stateTree) *service.StateTreeNode {
	n.buildChildren(ctx, tree)
	preview, previewIsValue := stateValuePreview(n.value)
	if preview != nil && n.value.CanInterface() {
		if labeled, ok := n.value.Interface().(api.Labeled); ok {
			preview.Label = labeled.Label(ctx, tree.globalState)
		}
	}
	return &service.StateTreeNode{
		NumChildren:    uint64(len(n.children)),
		Name:           n.name,
		Label:          n.label,
		ValuePath:      n.path.Path(),
		Preview:        preview,
		PreviewIsValue: previewIsValue,
		Constants:      n.consts,
	}
}

func isFieldVisible(f reflect.StructField) bool {
	return f.PkgPath == "" && f.Tag.Get("hidden") != "true"
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
		const maxLen = 64
		runes := []rune(v.Interface().(string))
		if len(runes) > maxLen {
			return box.NewValue(string(append(runes[:maxLen-1], '…'))), false
		}
		return box.NewValue(v.Interface()), true
	case reflect.Interface, reflect.Ptr:
		if isNil(v) {
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
	globalState, err := GlobalState(ctx, r.Path.After.GlobalStateAfter(), r.Config)
	if err != nil {
		return nil, err
	}

	rootObj, rootPath, apiID, err := state(ctx, r.Path, r.Config)
	if err != nil {
		return nil, err
	}

	apiPath := &path.API{ID: path.NewID(id.ID(apiID))}

	root := &stn{
		name:  "root",
		value: deref(reflect.ValueOf(rootObj)),
		path:  rootPath,
	}
	return &stateTree{globalState, rootObj, root, apiPath, uint64(r.ArrayGroupSize)}, nil
}

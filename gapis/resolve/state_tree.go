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
	"strconv"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/data/pod"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// StateTree resolves the specified state tree path.
func StateTree(ctx context.Context, c *path.StateTree) (*service.StateTree, error) {
	id, err := database.Store(ctx, &StateTreeResolvable{c.After.StateAfter()})
	if err != nil {
		return nil, err
	}
	return &service.StateTree{
		Root: &path.StateTreeNode{Tree: path.NewID(id)},
	}, nil
}

type stateTree struct {
	state interface{}
	path  *path.State
	api   *path.API
}

func deref(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	return v
}

type strVal struct {
	s string
	v reflect.Value
}

type strVals []strVal

func (l strVals) Len() int           { return len(l) }
func (l strVals) Less(i, j int) bool { return l[i].s < l[j].s }
func (l strVals) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

func sortMapKeys(m reflect.Value) []reflect.Value {
	keys := m.MapKeys()
	pairs := make(strVals, len(keys))
	for i, k := range keys {
		pairs[i] = strVal{fmt.Sprint(k.Interface()), k}
	}
	sort.Sort(pairs)
	sorted := make([]reflect.Value, len(keys))
	for i, v := range pairs {
		sorted[i] = v.v
	}
	return sorted
}

// StateTreeNode resolves the specified command tree node path.
func StateTreeNode(ctx context.Context, c *path.StateTreeNode) (*service.StateTreeNode, error) {
	boxed, err := database.Resolve(ctx, c.Tree.ID())
	if err != nil {
		return nil, err
	}

	stateTree := boxed.(*stateTree)

	name, pth, consts := "root", path.Node(stateTree.path), (*path.ConstantSet)(nil)
	v := deref(reflect.ValueOf(stateTree.state))

	numChildren := uint64(v.Type().NumField())

	for i, idx64 := range c.Index {
		idx := int(idx64)
		if idx64 >= numChildren {
			at := &path.StateTreeNode{Tree: c.Tree, Index: c.Index[:i+1]}
			return nil, errPathOOB(idx64, "Index", 0, numChildren-1, at)
		}

		switch v.Kind() {
		case reflect.Struct:
			t := v.Type()
			if cs, ok := t.Field(idx).Tag.Lookup("constset"); ok {
				if idx, _ := strconv.Atoi(cs); idx > 0 {
					consts = stateTree.api.ConstantSet(idx)
				}
			}
			name = t.Field(idx).Name
			pth = path.NewField(name, pth)
			v = deref(v.Field(idx))
		case reflect.Slice, reflect.Array:
			name = fmt.Sprint(idx)
			pth = path.NewArrayIndex(idx64, pth)
			v = deref(v.Index(idx))
		case reflect.Map:
			key := sortMapKeys(v)[idx]
			name = fmt.Sprint(key.Interface())
			pth = path.NewMapIndex(key.Interface(), pth)
			v = deref(v.MapIndex(key))
		default:
			return nil, fmt.Errorf("Cannot index type %v (%v)", v.Type(), v.Kind())
		}

		switch v.Kind() {
		case reflect.Struct:
			numChildren = uint64(v.NumField())
		case reflect.Slice, reflect.Array, reflect.Map:
			numChildren = uint64(v.Len())
		default:
			numChildren = 0
		}
	}

	return &service.StateTreeNode{
		NumChildren: numChildren,
		Name:        name,
		Path:        pth.Path(),
		Value:       pod.NewValue(v.Interface()),
		Constants:   consts,
	}, nil
}

// Resolve builds and returns a *StateTree for the path.StateTreeNode.
// Resolve implements the database.Resolver interface.
func (r *StateTreeResolvable) Resolve(ctx context.Context) (interface{}, error) {
	state, err := APIState(ctx, r.Path)
	if err != nil {
		return nil, err
	}
	c, err := capture.ResolveFromPath(ctx, r.Path.After.Capture)
	if err != nil {
		return nil, err
	}
	atomIdx := r.Path.After.Index[0]
	if len(r.Path.After.Index) > 1 {
		return nil, fmt.Errorf("Subcommands currently not supported") // TODO: Subcommands
	}
	api := c.Atoms[atomIdx].API()
	apiPath := &path.API{Id: path.NewID(id.ID(api.ID()))}
	return &stateTree{state, r.Path, apiPath}, nil
}

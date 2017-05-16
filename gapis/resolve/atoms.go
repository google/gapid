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

	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Atoms resolves and returns the atom list from the path p.
func Atoms(ctx context.Context, p *path.Capture) (*atom.List, error) {
	c, err := capture.ResolveFromPath(ctx, p)
	if err != nil {
		return nil, err
	}
	return atom.NewList(c.Atoms...), nil
}

// NAtoms resolves and returns the atom list from the path p, ensuring
// that the number of commands is at least N.
func NAtoms(ctx context.Context, p *path.Capture, n uint64) (*atom.List, error) {
	list, err := Atoms(ctx, p)
	if err != nil {
		return nil, err
	}
	if count := uint64(len(list.Atoms)); n > count {
		return nil, errPathOOB(n-1, "Index", 0, count-1, p.Command(n-1))
	}
	return list, nil
}

// Atom resolves and returns the atom from the path p.
func Atom(ctx context.Context, p *path.Command) (atom.Atom, error) {
	atomIdx := p.Index[0]
	if len(p.Index) > 1 {
		return nil, fmt.Errorf("Subcommands currently not supported") // TODO: Subcommands
	}
	list, err := NAtoms(ctx, p.Capture, atomIdx+1)
	if err != nil {
		return nil, err
	}
	return list.Atoms[atomIdx], nil
}

// Parameter resolves and returns the parameter from the path p.
func Parameter(ctx context.Context, p *path.Parameter) (interface{}, error) {
	obj, err := ResolveInternal(ctx, p.Parent())
	if err != nil {
		return nil, err
	}
	a := obj.(atom.Atom)
	param, err := atom.Parameter(ctx, a, p.Name)
	switch err {
	case nil:
		return param, nil
	case atom.ErrParameterNotFound:
		return nil, &service.ErrInvalidPath{
			Reason: messages.ErrParameterDoesNotExist(a.AtomName(), p.Name),
			Path:   p.Path(),
		}
	default:
		return nil, err
	}
}

// Result resolves and returns the command's result from the path p.
func Result(ctx context.Context, p *path.Result) (interface{}, error) {
	obj, err := ResolveInternal(ctx, p.Parent())
	if err != nil {
		return nil, err
	}
	a := obj.(atom.Atom)
	param, err := atom.Result(ctx, a)
	switch err {
	case nil:
		return param, nil
	case atom.ErrResultNotFound:
		return nil, &service.ErrInvalidPath{
			Reason: messages.ErrResultDoesNotExist(a.AtomName()),
			Path:   p.Path(),
		}
	default:
		return nil, err
	}
}

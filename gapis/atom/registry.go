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

package atom

import "github.com/google/gapid/gapis/gfxapi"
import "reflect"
import "sync"

var (
	registry      = map[gfxapi.API]*namespace{}
	registryMutex sync.RWMutex
)

type (
	factory   func() Atom
	namespace struct {
		factories map[string]factory
	}
)

// Register registers the atoms into the API namespace.
func Register(api gfxapi.API, atoms ...Atom) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	n, ok := registry[api]
	if !ok {
		n = &namespace{factories: map[string]factory{}}
		registry[api] = n
	}
	for _, a := range atoms {
		ty := reflect.TypeOf(a).Elem()
		n.factories[a.AtomName()] = func() Atom {
			return reflect.New(ty).Interface().(Atom)
		}
	}
}

// Create returns a newly created atom with the specified name that belongs to
// api. If the api or atom was not registered then Create returns nil.
func Create(api gfxapi.API, name string) Atom {
	registryMutex.RLock()
	defer registryMutex.RUnlock()
	if r, ok := registry[api]; ok {
		if f, ok := r.factories[name]; ok {
			return f()
		}
	}
	return nil
}

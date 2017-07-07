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

import "github.com/google/gapid/gapis/api"
import "reflect"
import "sync"

var (
	registry      = map[api.API]*namespace{}
	registryMutex sync.RWMutex
)

type (
	factory   func() api.Cmd
	namespace struct {
		factories map[string]factory
	}
)

// Register registers the commands into the API namespace.
func Register(a api.API, cmds ...api.Cmd) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	n, ok := registry[a]
	if !ok {
		n = &namespace{factories: map[string]factory{}}
		registry[a] = n
	}
	for _, c := range cmds {
		ty := reflect.TypeOf(c).Elem()
		n.factories[c.CmdName()] = func() api.Cmd {
			return reflect.New(ty).Interface().(api.Cmd)
		}
	}
}

// Create returns a newly created command with the specified name that belongs
// to api. If the api or atom was not registered then Create returns nil.
func Create(a api.API, name string) api.Cmd {
	registryMutex.RLock()
	defer registryMutex.RUnlock()
	if r, ok := registry[a]; ok {
		if f, ok := r.factories[name]; ok {
			return f()
		}
	}
	return nil
}

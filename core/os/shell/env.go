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

package shell

import (
	"fmt"
	"os"
	"strings"
)

// Env holds the environment variables for a new process.
type Env struct {
	PathListSeparator rune
	vars              []string
	keys              map[string]int
}

func splitEnvVar(v string) (key, val string) {
	sep := strings.Index(v, "=")
	if sep < 0 {
		return v, ""
	}
	return v[:sep], v[sep+1:]
}

func joinEnvVar(key, val string) string {
	if len(val) > 0 {
		return fmt.Sprintf("%v=%v", key, val)
	}
	return key
}

// NewEnv returns a copy of the current process's environment variables.
func NewEnv() *Env {
	return &Env{
		PathListSeparator: os.PathListSeparator,
		vars:              []string{},
		keys:              map[string]int{},
	}
}

// CloneEnv returns a copy of the current process's environment variables.
func CloneEnv() *Env {
	out := NewEnv()
	out.vars = os.Environ()
	for i, v := range out.vars {
		if key, val := splitEnvVar(v); len(val) > 0 {
			out.keys[strings.ToUpper(key)] = i
		}
	}
	return out
}

func (e *Env) Apply() error {
	os.Clearenv()
	for _, evar := range e.vars {
		k, v := splitEnvVar(evar)
		if err := os.Setenv(k, v); err != nil {
			return err
		}
	}
	return nil
}

// Vars returns a copy of the full list of environment variables.
func (e *Env) Vars() []string {
	if e == nil {
		return nil
	}
	out := make([]string, len(e.vars))
	copy(out, e.vars)
	return out
}

// Keys returns all of the environment variable keys.
func (e *Env) Keys() []string {
	if e == nil {
		return nil
	}
	out := make([]string, len(e.keys))
	for k := range e.keys {
		out = append(out, k)
	}
	return out
}

// Exists returns true if the environment variable with the specified key
// exists.
// The variable can be of the form 'key=value' or simply 'key'.
func (e *Env) Exists(key string) bool {
	_, existing := e.keys[strings.ToUpper(key)]
	return existing
}

// Get returns the value of an environment variable stored in the form
// 'key=value'.
// If there are no variables with the key, then an empty string is returned.
func (e *Env) Get(key string) string {
	if idx, existing := e.keys[strings.ToUpper(key)]; existing {
		_, val := splitEnvVar(e.vars[idx])
		return val
	}
	return ""
}

// Add inserts a new environment variable that is of the form "key=value".
// That is, it parses the environment variable if necessary
func (e *Env) Add(key string) {
	k, v := splitEnvVar(key)
	e.Set(k, v)
}

// Unset removes an environment variable in the form 'key=value' or 'key'.
func (e *Env) Unset(key string) *Env {
	if idx, existing := e.keys[strings.ToUpper(key)]; existing {
		copy(e.vars[idx:], e.vars[idx+1:])
		e.vars = e.vars[:len(e.vars)-1]
	}
	e.keys = map[string]int{}
	for i, v := range e.vars {
		if key, val := splitEnvVar(v); len(val) > 0 {
			e.keys[strings.ToUpper(key)] = i
		}
	}
	return e
}

// Set adds or replaces an environment variable in the form 'key=value' or
// 'key'.
func (e *Env) Set(key, value string) *Env {
	v := joinEnvVar(key, value)
	if idx, existing := e.keys[strings.ToUpper(key)]; existing {
		e.vars[idx] = v
		e.keys[strings.ToUpper(key)] = idx
	} else {
		idx = len(e.vars)
		e.vars = append(e.vars, v)
		e.keys[strings.ToUpper(key)] = idx
	}
	return e
}

// AddPathStart adds paths to the start of the environment variable with the
// specified key, using e.PathListSeparator as a delimiter.
// If there was no existing environment variable with the given key then it is
// created.
func (e *Env) AddPathStart(key string, paths ...string) *Env {
	prefix := strings.Join(paths, string(e.PathListSeparator))
	if val := e.Get(key); len(val) > 0 {
		e.Set(key, fmt.Sprintf("%v%c%v", prefix, e.PathListSeparator, e.Get(key)))
	} else {
		e.Set(key, prefix)
	}
	return e
}

// AddPathEnd adds paths  to the start of the environment variable with the
// specified key, using the os.PathListSeparator as a delimiter.
// If there was no existing environment variable with the given key then it is
// created.
func (e *Env) AddPathEnd(key string, paths ...string) *Env {
	suffix := strings.Join(paths, string(e.PathListSeparator))
	if val := e.Get(key); len(val) > 0 {
		e.Set(key, fmt.Sprintf("%v%c%v", e.Get(key), e.PathListSeparator, suffix))
	} else {
		e.Set(key, suffix)
	}
	return e
}

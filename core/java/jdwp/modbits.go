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

package jdwp

import "strings"

// ModBits represents the modifier bitflags for a class, method or field.
type ModBits int

const (
	ModPublic       = ModBits(1)
	ModPrivate      = ModBits(2)
	ModProtected    = ModBits(4)
	ModStatic       = ModBits(8)
	ModFinal        = ModBits(16)
	ModSynchronized = ModBits(32)
	ModVolatile     = ModBits(64)
	ModTransient    = ModBits(128)
	ModInterface    = ModBits(512)
	ModNative       = ModBits(256)
	ModAbstract     = ModBits(1024)
	ModStrict       = ModBits(2048)
)

func (m ModBits) String() string {
	parts := []string{}
	if m&ModPublic != 0 {
		parts = append(parts, "public")
	}
	if m&ModPrivate != 0 {
		parts = append(parts, "private")
	}
	if m&ModProtected != 0 {
		parts = append(parts, "protected")
	}
	if m&ModStatic != 0 {
		parts = append(parts, "static")
	}
	if m&ModFinal != 0 {
		parts = append(parts, "final")
	}
	if m&ModSynchronized != 0 {
		parts = append(parts, "synchronized")
	}
	if m&ModVolatile != 0 {
		parts = append(parts, "volatile")
	}
	if m&ModTransient != 0 {
		parts = append(parts, "transient")
	}
	if m&ModInterface != 0 {
		parts = append(parts, "interface")
	}
	if m&ModNative != 0 {
		parts = append(parts, "native")
	}
	if m&ModAbstract != 0 {
		parts = append(parts, "abstract")
	}
	if m&ModStrict != 0 {
		parts = append(parts, "strict")
	}

	return strings.Join(parts, " ")
}

func (m ModBits) Public() bool       { return m&ModPublic != 0 }
func (m ModBits) Private() bool      { return m&ModPrivate != 0 }
func (m ModBits) Protected() bool    { return m&ModProtected != 0 }
func (m ModBits) Static() bool       { return m&ModStatic != 0 }
func (m ModBits) Final() bool        { return m&ModFinal != 0 }
func (m ModBits) Synchronized() bool { return m&ModSynchronized != 0 }
func (m ModBits) Volatile() bool     { return m&ModVolatile != 0 }
func (m ModBits) Transient() bool    { return m&ModTransient != 0 }
func (m ModBits) Interface() bool    { return m&ModInterface != 0 }
func (m ModBits) Native() bool       { return m&ModNative != 0 }
func (m ModBits) Abstract() bool     { return m&ModAbstract != 0 }
func (m ModBits) Strict() bool       { return m&ModStrict != 0 }

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

package device

var (
	UnknownCPU = cpu("unknown", UnknownArchitecture)
	// Arm
	CortexA53 = cpu("Cortex A53", ARMv8a)
	CortexA57 = cpu("Cortex A57", ARMv8a)
	// Qualcomm
	Scorpion = cpu("Scorpion", ARMv7a)
	Krait    = cpu("Krait", ARMv7a)
	Kryo     = cpu("Kryo", ARMv8a)
	// Nvidia
	Denver = cpu("Denver", ARMv8a)
)

var cpuByName = map[string]*CPU{}

func cpu(name string, arch Architecture) *CPU {
	cpu := &CPU{
		Name:         name,
		Architecture: arch,
	}
	cpuByName[name] = cpu
	return cpu
}

// CPUByName looks up a CPU from a product name.
// If the product is not know, it returns an UnknownCPU.
func CPUByName(name string) *CPU {
	cpu, ok := cpuByName[name]
	if !ok {
		cpu = &CPU{
			Name:         name,
			Architecture: UnknownArchitecture,
		}
	}
	return cpu
}

// SameAs returns true if the two CPU objects are a match.
func (c *CPU) SameAs(o *CPU) bool {
	// If the cpu name is set, treat it as an authoratitive comparison point
	if c.Name != "" && o.Name != "" {
		return c.Name == o.Name
	}
	return c.Architecture == o.Architecture
}

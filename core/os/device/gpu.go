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
	UnknownGPU = gpu("unknown")

	Adreno320 = gpu("Adreno 320")
	Adreno330 = gpu("Adreno 330")
	Adreno418 = gpu("Adreno 418")
	Adreno420 = gpu("Adreno 420")
	Adreno430 = gpu("Adreno 430")
	Adreno530 = gpu("Adreno 530")
	Kepler    = gpu("Kepler")
)

var gpuByName = map[string]*GPU{}

func gpu(name string) *GPU {
	gpu := &GPU{
		Name: name,
	}
	gpuByName[name] = gpu
	return gpu
}

// GPUByName looks up a GPU from a product name.
// If the product is not know, it returns an UnknownGPU.
func GPUByName(name string) *GPU {
	gpu, ok := gpuByName[name]
	if !ok {
		gpu = &GPU{
			Name: name,
		}
	}
	return gpu
}

// SameAs returns true if the two gpu objects are a match.
func (g *GPU) SameAs(o *GPU) bool {
	// Name is all we have...
	return g.Name == o.Name
}

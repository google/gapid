// Copyright (C) 2018 Google Inc.
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

package status

import (
	"context"
	"runtime"
	"sync"
)

var (
	memorySnapshot runtime.MemStats
	memoryMutex    sync.RWMutex
)

// MemorySnapshot returns the memory statistics sampled from the last call to
// SnapshotMemory().
func MemorySnapshot() runtime.MemStats {
	memoryMutex.RLock()
	defer memoryMutex.RUnlock()
	return memorySnapshot
}

// SnapshotMemory takes a snapshot of the current process's memory.
func SnapshotMemory(ctx context.Context) {
	var snapshot runtime.MemStats
	runtime.ReadMemStats(&snapshot)

	memoryMutex.Lock()
	memorySnapshot = snapshot
	memoryMutex.Unlock()

	onMemorySnapshot(ctx, snapshot)
}

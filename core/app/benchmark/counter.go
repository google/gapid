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

package benchmark

import (
	"sync"
	"sync/atomic"
	"time"
)

// GlobalCounters is a singleton set of performance counters.
var GlobalCounters = NewCounters()

// Integer is a convenience function for calling GlobalCounters.Integer(name).
func Integer(name string) *IntegerCounter {
	return GlobalCounters.Integer(name)
}

// Duration is a convenience function for calling GlobalCounters.Duration(name).
func Duration(name string) *DurationCounter {
	return GlobalCounters.Duration(name)
}

// Counters represents a collection of named performance counters.
//
// Individual counters are created on retrieve if they do not exist.
// Counters of different types with the same name are disallowed.
// Users are encouraged to hang on to retrieved counters rather than
// calling these methods repeatedly, for performance reasons.
type Counters struct {
	counters map[string]interface{}
	mutex    sync.Mutex
}

// NewCounters allocates a new set of counters.
func NewCounters() *Counters {
	return &Counters{counters: map[string]interface{}{}}
}

func (m *Counters) getOrAllocate(name string, f func() interface{}) interface{} {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	counter, found := m.counters[name]
	if !found {
		counter = f()
		m.counters[name] = counter
	}
	return counter
}

// Integer returns the IntegerCounter with the given name,
// instantiating a new one if necessary.
func (m *Counters) Integer(name string) *IntegerCounter {
	return m.getOrAllocate(name, newIntegerCounter).(*IntegerCounter)
}

// Duration returns the DurationCounter with the given name,
// instantiating a new one if necessary.
func (m *Counters) Duration(name string) *DurationCounter {
	return m.getOrAllocate(name, newDurationCounter).(*DurationCounter)
}

// All retrieves a copy of the counters keyed by name.
func (m *Counters) All() map[string]interface{} {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	res := make(map[string]interface{}, len(m.counters))
	for name, counter := range m.counters {
		res[name] = counter
	}
	return res
}

// IntegerCounter is a Counter that holds an int64 value.
type IntegerCounter int64

func newIntegerCounter() interface{} {
	return new(IntegerCounter)
}

// Reset resets the counter to 0.
func (c *IntegerCounter) Reset() {
	atomic.StoreInt64((*int64)(c), 0)
}

// Get implements counter.
func (c *IntegerCounter) Get() int64 {
	return atomic.LoadInt64((*int64)(c))
}

// Set assigns v to the counter.
func (c *IntegerCounter) Set(v int64) {
	atomic.StoreInt64((*int64)(c), v)
}

// Add adds the given value to the value of this counter.
func (c *IntegerCounter) Add(i int64) {
	atomic.AddInt64((*int64)(c), i)
}

// Increment adds 1 to the value of this counter.
func (c *IntegerCounter) Increment() {
	atomic.AddInt64((*int64)(c), 1)
}

// DurationCounter is a Counter that holds an int64, exposed as a time.Duration.
type DurationCounter int64

func newDurationCounter() interface{} {
	return new(DurationCounter)
}

// Add adds the given duration to the value of this counter.
func (c *DurationCounter) Add(t time.Duration) {
	atomic.AddInt64((*int64)(c), int64(t))
}

// Get retrieves the value of this counter as a time.Duration.
func (c *DurationCounter) Get() time.Duration {
	return time.Duration(atomic.LoadInt64((*int64)(c)))
}

// Set resets the counter to t.
func (c *DurationCounter) Set(t time.Duration) {
	atomic.StoreInt64((*int64)(c), int64(t))
}

// Reset resets the duration to 0.
func (c *DurationCounter) Reset() {
	c.Set(0)
}

// Start returns the current time. It's meant to be used with Stop()
// as a way to time blocks of code and add their durations to the counter.
func (c *DurationCounter) Start() time.Time {
	return time.Now()
}

// Stop adds to the value of this counter the Duration elapsed
// since the time.Time received as argument.
func (c *DurationCounter) Stop(startTime time.Time) {
	c.Add(time.Since(startTime))
}

// Time times the call to f, adding the timed duration to the counter.
func (c *DurationCounter) Time(f func()) {
	start := c.Start()
	defer c.Stop(start)
	f()
}

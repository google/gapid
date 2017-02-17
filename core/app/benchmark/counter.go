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
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// GlobalCounters is a singleton set of performance counters.
	GlobalCounters = NewCounters()

	counterFactories = map[string]func() Counter{}
)

func init() {
	register := func(f func() Counter) {
		counterFactories[f().TypeName()] = f
	}
	register(newDurationCounter)
	register(newIntegerCounter)
	register(newStringHolder)
}

// Counters represents a collection of named performance counters.
//
// Individual counters are created on retrieve if they do not exist.
// Counters of different types with the same name are disallowed.
// Users are encouraged to hang on to retrieved counters rather than
// calling these methods repeatedly, for performance reasons.
type Counters struct {
	counters map[string]Counter
	mutex    sync.Mutex
}

type Counter interface {
	// Get retrieves the current value of the counter.
	Get() interface{}

	// Reset resets the counter to its initial value.
	Reset()

	// TypeName retrieves a string that uniquely identifies the
	// type of counter.
	TypeName() string
}

// NewCounters allocates a new set of counters.
func NewCounters() *Counters {
	return &Counters{counters: make(map[string]Counter)}
}

func (m *Counters) getOrAllocate(name string, f func() Counter) Counter {
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

// String returns the StringHolder with the given name,
// instantiating a new one if necessary.
func (m *Counters) String(name string) *StringHolder {
	return m.getOrAllocate(name, newStringHolder).(*StringHolder)
}

// Lazy returns the LazyCounter with the given name,
// instantiating a new one if necessary.
func (m *Counters) Lazy(name string) *LazyCounter {
	return m.getOrAllocate(name, func() Counter { return &LazyCounter{} }).(*LazyCounter)
}

// LazyWithFunction returns the LazyCounter with the given name,
// atomically instantiating a new one with the given
// function if necessary.
func (m *Counters) LazyWithFunction(name string, f func() Counter) *LazyCounter {
	return m.getOrAllocate(name, func() Counter {
		l := &LazyCounter{}
		l.SetFunction(f)
		return l
	}).(*LazyCounter)
}

// LazyWithCaching returns the LazyCounter with the given name,
// atomically instantiating a new one with the given
// caching function if necessary.
func (m *Counters) LazyWithCaching(name string, f func() Counter) *LazyCounter {
	return m.getOrAllocate(name, func() Counter {
		l := &LazyCounter{}
		l.SetCaching(f)
		return l
	}).(*LazyCounter)
}

// MarshalJSON implements json.Marshaler.
func (m *Counters) MarshalJSON() ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	res := map[string]interface{}{}
	for name, counter := range m.counters {
		res[name] = struct {
			Type  string
			Value Counter
		}{
			Type:  counter.TypeName(),
			Value: counter,
		}
	}
	return json.Marshal(res)
}

// AllCounters retrieves a copy of the counter map,
// keyed by name.
func (m *Counters) AllCounters() map[string]Counter {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	res := make(map[string]Counter)
	for name, counter := range m.counters {
		res[name] = counter
	}
	return res
}

// UnmarshalJSON implements json.Unmarshaler.
func (m *Counters) UnmarshalJSON(data []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	rawCounters := map[string]struct {
		Type  string
		Value json.RawMessage
	}{}
	if err := json.Unmarshal(data, &rawCounters); err != nil {
		return err
	}

	m.counters = make(map[string]Counter)
	for name, c := range rawCounters {
		factory := counterFactories[c.Type]
		if factory != nil {
			cnt := factory()
			if err := json.Unmarshal(c.Value, cnt); err != nil {
				return err
			}
			m.counters[name] = cnt
		}
	}
	return nil
}

// IntegerCounter is a Counter that holds an int64 value.
type IntegerCounter int64

func newIntegerCounter() Counter {
	return IntegerCounterOf(0)
}

// IntegerCounterOf creates a new IntegerCounter with the given duration.
func IntegerCounterOf(i int64) *IntegerCounter {
	c := new(IntegerCounter)
	*c = IntegerCounter(i)
	return c
}

// Reset implements Counter.
func (c *IntegerCounter) Reset() {
	atomic.StoreInt64((*int64)(c), 0)
}

// Get implements Counter.
func (c *IntegerCounter) Get() interface{} {
	return atomic.LoadInt64((*int64)(c))
}

// TypeName implements Counter.
func (c *IntegerCounter) TypeName() string {
	return "int"
}

// AddInt64 adds the given value to the value of this counter.
func (c *IntegerCounter) AddInt64(i int64) {
	atomic.AddInt64((*int64)(c), i)
}

// AddInt64 adds 1 to the value of this counter.
func (c *IntegerCounter) Increment() {
	atomic.AddInt64((*int64)(c), 1)
}

// GetInt64 retrieves the value of this counter as an int64.
func (c *IntegerCounter) GetInt64() int64 {
	return atomic.LoadInt64((*int64)(c))
}

func (c *IntegerCounter) SetInt64(val int64) {
	atomic.StoreInt64((*int64)(c), val)
}

// MarshalJSON implements json.Marshaller.
func (c *IntegerCounter) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", c.GetInt64())), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (c *IntegerCounter) UnmarshalJSON(data []byte) error {
	var val int64
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	c.SetInt64(val)
	return nil
}

// DurationCounter is a Counter that holds an int64, exposed as a time.Duration.
type DurationCounter int64

func newDurationCounter() Counter {
	return DurationCounterOf(0)
}

// DurationCounterOf creates a new DurationCounter with the given duration.
func DurationCounterOf(t time.Duration) *DurationCounter {
	d := new(DurationCounter)
	*d = DurationCounter(t)
	return d
}

// AddDuration adds the given duration to the value of this counter.
func (c *DurationCounter) AddDuration(t time.Duration) {
	atomic.AddInt64((*int64)(c), int64(t))
}

// GetDuration retrieves the value of this counter as a time.Duration.
func (c *DurationCounter) GetDuration() time.Duration {
	return time.Duration(atomic.LoadInt64((*int64)(c)))
}

// Reset implements Counter.
func (c *DurationCounter) Reset() {
	atomic.StoreInt64((*int64)(c), 0)
}

// Get implements Counter.
func (c *DurationCounter) Get() interface{} {
	return c.GetDuration()
}

// TypeName implements Counter.
func (c *DurationCounter) TypeName() string {
	return "duration"
}

func (c *DurationCounter) SetDuration(t time.Duration) {
	atomic.StoreInt64((*int64)(c), int64(t))
}

// Start returns the current time. It's meant to be used with Stop()
// as a way to time blocks of code and add their durations to the counter.
func (c *DurationCounter) Start() time.Time {
	return time.Now()
}

// Stop adds to the value of this counter the Duration elapsed
// since the time.Time received as argument.
func (c *DurationCounter) Stop(startTime time.Time) {
	c.AddDuration(time.Since(startTime))
}

// MarshalJSON implements json.Marshaler.
func (c *DurationCounter) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%v"`, c.GetDuration())), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (c *DurationCounter) UnmarshalJSON(data []byte) error {
	var durationString string
	if err := json.Unmarshal(data, &durationString); err != nil {
		return err
	}
	dur, err := time.ParseDuration(durationString)
	if err == nil {
		c.SetDuration(dur)
	}
	return err
}

// Holder is a Counter that holds a String.
type StringHolder struct {
	val atomic.Value
}

func newStringHolder() Counter {
	return StringHolderOf("")
}

// StringHolderOf creates a new StringHolder with the given value.
func StringHolderOf(s string) *StringHolder {
	sh := new(StringHolder)
	sh.SetString(s)
	return sh
}

// Reset implements Counter.
func (c *StringHolder) Reset() {
	c.SetString("")
}

// Get implements Counter.
func (c *StringHolder) Get() interface{} {
	return c.val.Load()
}

// GetString returns the string.
func (c *StringHolder) GetString() string {
	return c.Get().(string)
}

func (c *StringHolder) TypeName() string {
	return "string"
}

// SetString sets the string in the holder.
func (c *StringHolder) SetString(s string) {
	c.val.Store(s)
}

// MarshalJSON implements json.Marshaler.
func (c *StringHolder) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.GetString())
}

func (c *StringHolder) UnmarshalJSON(data []byte) error {
	var val string
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	c.SetString(val)
	return nil
}

// LazyCounter is a Counter that delegates Counter method calls to
// a Counter retrieved by calling a function.
type LazyCounter struct {
	fun atomic.Value
}

// Counter retrieves the Underlying counter.
func (c *LazyCounter) Counter() Counter {
	return c.fun.Load().(func() Counter)()
}

// SetFunction sets the function to retrieve the underlying counter.
// This function will be called every time one of the Counter methods
// is called.
func (c *LazyCounter) SetFunction(f func() Counter) {
	c.fun.Store(f)
}

// SetCaching sets a function to be called to retrieve the underlying
// counter. This function will be called once, the first time one of
// the Counter methods is called.
func (c *LazyCounter) SetCaching(f func() Counter) {
	var counter Counter
	c.SetFunction(func() Counter {
		if counter == nil {
			counter = f()
		}
		return counter
	})
}

// Reset implements Counter.
func (c *LazyCounter) Reset() {
	c.Counter().Reset()
}

// Get implements Counter.
func (c *LazyCounter) Get() interface{} {
	return c.Counter().Get()
}

// TypeName implements Counter.
func (c *LazyCounter) TypeName() string {
	return c.Counter().TypeName()
}

// MarshalJSON implements json.Marshaler.
func (c *LazyCounter) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Counter())
}

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

package memory

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"
	"unsafe"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/core/math/u64"
	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapis/database"
	"github.com/pkg/errors"
)

// Pool represents an unbounded and isolated memory space. Pool can be used
// to represent the application address space, or hidden GPU Pool.
//
// Pool can be sliced into smaller regions which can be read or written to.
// All writes to Pool or its slices do not actually perform binary data
// copies, but instead all writes are stored as lightweight records. Only when a
// Pool slice has Get called will any resolving, loading or copying of binary
// data occur.
type Pool struct {
	id          PoolID
	writes      poolWriteList
	allocations []allocation
	arena       arena.Arena
	OnRead      func(Range)
	OnWrite     func(Range)
}

// PoolID is an identifier of a Pool.
type PoolID uint32

// Pools contains a collection of Pools identified by PoolIDs.
type Pools struct {
	pools      map[PoolID]*Pool
	nextPoolID PoolID
	OnCreate   func(PoolID, *Pool)
}

type poolSlice struct {
	rng    Range         // The memory range of the slice.
	writes poolWriteList // The list of writes to the pool when this slice was created.
}

const (
	// ApplicationPool is the PoolID of Pool representing the application's memory
	// address space.
	ApplicationPool = PoolID(PoolNames_Application)
)

// byteSlice returns a slice into an unsafe pointer
func byteSlice(ptr unsafe.Pointer) []byte {
	return ((*[1 << 30]byte)(ptr))[:]
}

type allocation struct {
	offset uint64
	size   uint64
	data   unsafe.Pointer
}

func insertAllocation(allocations []allocation, a allocation) []allocation {
	index := sort.Search(len(allocations), func(i int) bool { return allocations[i].offset > a.offset })
	allocations = append(allocations, allocation{})
	copy(allocations[index+1:], allocations[index:])
	allocations[index] = a
	return allocations
}

// NewPools creates and returns a new Pools instance.
func NewPools(ranges interval.U64RangeList, arena arena.Arena) Pools {
	memPools := map[PoolID]*Pool{ApplicationPool: {
		id:          ApplicationPool,
		writes:      poolWriteList{},
		allocations: make([]allocation, 0, ranges.Length()),
		arena:       arena,
	}}
	memPools[ApplicationPool].ReserveRanges(ranges)
	runtime.SetFinalizer(memPools[ApplicationPool], func(p *Pool) {
		for _, v := range p.allocations {
			arena.Free(v.data)
		}
	})

	return Pools{
		pools:      memPools,
		nextPoolID: ApplicationPool + 1,
	}
}

// ReserveRanges makes a set of ranges available in the
// application pool.
func (m *Pool) ReserveRanges(ranges interval.U64RangeList) {
	if m.id != ApplicationPool {
		panic("You can only reserve ranges in the application pool")
	}
	for idx := range ranges {
		alloc := allocation{
			ranges[idx].First,
			ranges[idx].Count,
			m.arena.Allocate(uint32(ranges[idx].Count), 1),
		}
		m.allocations = insertAllocation(m.allocations, alloc)
	}
}

// AddRange makes a single allocation available to the application
// pool
func (m *Pool) AddRange(offset, count uint64) {
	if m.id != ApplicationPool {
		panic("You can only reserve ranges in the application pool")
	}

	alloc := allocation{
		offset,
		count,
		m.arena.Allocate(uint32(count), 1),
	}
	m.allocations = insertAllocation(m.allocations, alloc)
}

// RemoveRange removes a single allocation from the application pool.
// It is invalid to remove an allocation that was never added
func (m *Pool) RemoveRange(offset uint64) {
	if m.id != ApplicationPool {
		panic("You can only remove ranges from the application pool")
	}
	i := sort.Search(len(m.allocations), func(i int) bool { return m.allocations[i].offset >= offset })
	if i >= len(m.allocations) || m.allocations[i].offset != offset {
		msg := fmt.Sprintf("Removed allocation [%d] was not found in arena %+v", offset, m.allocations)
		panic(msg)
	}
	m.arena.Free(m.allocations[i].data)
	copy(m.allocations[i:], m.allocations[i+1:])
	m.allocations = m.allocations[:len(m.allocations)-1]
}

func (m *Pool) tryGetAllocation(base uint64) int {
	i := sort.Search(len(m.allocations), func(i int) bool { return m.allocations[i].offset > base })
	if i == 0 {
		return len(m.allocations) + 1
	}

	if i >= len(m.allocations) && 0 != len(m.allocations) {
		i = len(m.allocations)
	}

	return i - 1
}

func (m *Pool) mustGetAllocation(base uint64) int {
	i := sort.Search(len(m.allocations), func(i int) bool { return m.allocations[i].offset > base })
	if i == 0 {
		panic(fmt.Sprintf("Could not find allocation in pool: %+v, %+v", base, m.allocations))
	}

	if i >= len(m.allocations) && 0 != len(m.allocations) {
		i = len(m.allocations)
	}

	i--
	allocation := m.allocations[i]
	if base > allocation.offset+allocation.size {
		panic(fmt.Sprintf("Could not find allocation in pool: %+v, %+v, %+v", base, allocation, m.allocations))
	}
	return i
}

func (m *Pool) mustGetOffsetAndAllocation(rng Range) (int, uint64) {
	i := m.mustGetAllocation(rng.Base)
	allocation := m.allocations[i]
	if allocation.offset+allocation.size < rng.Base+rng.Size {
		panic(fmt.Sprintf("Slicing outside of range %+v %+v %+v \n\n---------------------------\n%+v\n----------------------\n", rng, rng.Size, allocation, m.allocations))
	}
	offsetInAlloc := rng.Base - allocation.offset
	return i, offsetInAlloc
}

func (m *Pool) StringSize(base uint64) int {
	i := m.mustGetAllocation(base)
	size := 0
	alloc := m.allocations[i]
	pos := base - alloc.offset
	data := byteSlice(alloc.data)
	for pos < alloc.size {
		size++
		if data[pos] == 0 {
			break
		}
		pos++
	}
	return size
}

var panicRead = 2

// ValidRanges returns all valid ranges in this pool that fall within
// the given range
func (m *Pool) ValidRanges(ctx context.Context, rng Range) RangeList {
	last := rng.Base + rng.Size - 1
	valid := make(RangeList, 0)
	i := m.tryGetAllocation(last)
	for last >= rng.Base {
		if i >= len(m.allocations) || i < 0 {
			break
		}
		alloc := m.allocations[i]

		begin := u64.Max(rng.Base, alloc.offset)
		end := u64.Min(rng.Base+rng.Size, alloc.offset+alloc.size)
		if end <= begin {
			break
		}

		valid = append([]Range{Range{begin, end - begin}}, valid...)
		last = begin - 1

		i--
	}

	return valid
}

// Slice returns a Data referencing the subset of the Pool range.
func (m *Pool) Slice(ctx context.Context, rng Range) Data {
	panicRead--
	if m.id == ApplicationPool {

		alloc, offsetInAlloc := m.mustGetOffsetAndAllocation(rng)
		data := make([]byte, rng.Size)
		copy(data, byteSlice(m.allocations[alloc].data)[offsetInAlloc:offsetInAlloc+rng.Size])
		b := Blob(data)
		return b
	}
	data := make([]byte, rng.Size)
	copy(data, byteSlice(m.allocations[0].data)[rng.Base:rng.Base+rng.Size])

	return Blob(data)
}

// WriteableSlice returns a slice that is only valid until the next
// Read/Write operation on this pool.
// This puts much less pressure on the garbage collector since we don't
// have to make a copy of the range.
func (m *Pool) WriteableSlice(rng Range) []byte {
	if m.id == ApplicationPool {
		alloc, offsetInAlloc := m.mustGetOffsetAndAllocation(rng)
		return byteSlice(m.allocations[alloc].data)[offsetInAlloc : offsetInAlloc+rng.Size]
	} else {
		return byteSlice(m.allocations[0].data)[rng.Base : rng.Base+rng.Size]
	}
}

var panicWrites = 2

// Write copies the data src to the address dst.
func (m *Pool) Write(ctx context.Context, dst uint64, src Data) {
	panicWrites--
	if m.id == ApplicationPool {
		alloc, offsetInAlloc := m.mustGetOffsetAndAllocation(Range{dst, src.Size()})
		src.Get(ctx, 0, byteSlice(m.allocations[alloc].data)[offsetInAlloc:offsetInAlloc+src.Size()])
		return
	}
	if m.allocations[0].size < dst+src.Size() {
		panic(fmt.Sprintf("Cannot write [ %v bytes ] %p, at offset %v in pool %+v", src.Size(), src, dst, *m))
	}

	src.Get(ctx, 0, byteSlice(m.allocations[0].data)[dst:dst+src.Size()])
}

// String returns the full history of writes performed to this pool.
func (m *Pool) String() string {
	l := make([]string, len(m.writes)+1)
	l[0] = fmt.Sprintf("Pool(%p):", m)
	for i, w := range m.writes {
		l[i+1] = fmt.Sprintf("(%d) %v <- %v", i, w.dst, w.src)
	}
	return strings.Join(l, "\n")
}

// NextPoolID returns the next free pool ID (but does not assign it).
// All existing pools in the set have pool ID which is less then this value.
func (m *Pools) NextPoolID() PoolID {
	return m.nextPoolID
}

// New creates and returns a new Pool and its id.
func (m *Pools) New(size uint64, arena arena.Arena) (id PoolID, p *Pool) {
	id, p = m.nextPoolID, &Pool{
		id:     m.nextPoolID,
		writes: poolWriteList{},
		// DONT CHECK THIS IN YET: We should probably plumb uint64 all the way through
		allocations: []allocation{allocation{0, size, arena.Allocate(uint32(size), 1)}},
		arena:       arena,
	}
	m.pools[id] = p
	m.nextPoolID++

	if m.OnCreate != nil {
		m.OnCreate(id, p)
	}
	runtime.SetFinalizer(p, func(p *Pool) {
		for _, v := range p.allocations {
			arena.Free(v.data)
		}
	})
	return
}

// NewAt creates and returns a new Pool with a specific ID, panics if it cannot
func (m *Pools) NewAt(id PoolID, size uint64, arena arena.Arena) *Pool {
	if _, ok := m.pools[id]; ok {
		panic("Could not create given pool")
	}
	p := &Pool{
		id:          id,
		writes:      poolWriteList{},
		allocations: []allocation{allocation{uint64(0), size, arena.Allocate(uint32(size), 1)}},
		arena:       arena,
	}
	m.pools[id] = p
	if id >= m.nextPoolID {
		m.nextPoolID = id + 1
	}

	if m.OnCreate != nil {
		m.OnCreate(id, p)
	}
	return p
}

// Get returns the Pool with the given id.
func (m *Pools) Get(id PoolID) (*Pool, error) {
	if p, ok := m.pools[id]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("Pool %v not found", id)
}

// MustGet returns the Pool with the given id, or panics if it's not found.
func (m *Pools) MustGet(id PoolID) *Pool {
	if p, ok := m.pools[id]; ok {
		return p
	}
	panic(fmt.Errorf("Pool %v not found", id))
}

// ApplicationPool returns the application memory pool.
func (m *Pools) ApplicationPool() *Pool {
	return m.pools[ApplicationPool]
}

// Count returns the number of pools.
func (m *Pools) Count() int {
	return len(m.pools)
}

// SetOnCreate sets the OnCreate callback and invokes it for every pool already created.
func (m *Pools) SetOnCreate(onCreate func(PoolID, *Pool)) {
	m.OnCreate = onCreate
	for i, p := range m.pools {
		onCreate(i, p)
	}
}

// String returns a string representation of all pools.
func (m *Pools) String() string {
	mem := make([]string, 0, len(m.pools))
	for i, p := range m.pools {
		mem[i] = fmt.Sprintf("    %d: %v", i, strings.Replace(p.String(), "\n", "\n      ", -1))
	}
	return strings.Join(mem, "\n")
}

func (m poolSlice) Get(ctx context.Context, offset uint64, dst []byte) error {
	orng := Range{Base: m.rng.Base + offset, Size: m.rng.Size - offset}
	i, c := interval.Intersect(&m.writes, orng.Span())
	for _, w := range m.writes[i : i+c] {
		if w.dst.First() > orng.First() {
			if err := w.src.Get(ctx, 0, dst[w.dst.First()-orng.First():]); err != nil {
				return err
			}
		} else {
			if err := w.src.Get(ctx, orng.First()-w.dst.First(), dst); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m poolSlice) ResourceID(ctx context.Context) (id.ID, error) {
	bytes := make([]byte, m.Size())
	if err := m.Get(ctx, 0, bytes); err != nil {
		return id.ID{}, err
	}
	return database.Store(ctx, bytes)
}

func (m poolSlice) Slice(rng Range) Data {
	if uint64(rng.Last()) > m.rng.Size {
		panic(fmt.Errorf("%v.Slice(%v) - out of bounds", m.String(), rng))
	}
	rng.Base += m.rng.Base
	i, c := interval.Intersect(&m.writes, rng.Span())
	return poolSlice{rng, m.writes[i : i+c]}
}

func (m poolSlice) ValidRanges() RangeList {
	valid := make(RangeList, len(m.writes))
	for i, w := range m.writes {
		s := u64.Max(w.dst.Base, m.rng.Base)
		e := u64.Min(w.dst.Base+w.dst.Size, m.rng.Base+m.rng.Size)
		valid[i] = Range{Base: s - m.rng.Base, Size: e - s}
	}
	return valid
}

func (m poolSlice) Size() uint64 {
	return m.rng.Size
}

func (m poolSlice) String() string {
	return fmt.Sprintf("Slice(%v)", m.rng)
}

func (m poolSlice) NewReader(ctx context.Context) io.Reader {
	r := &poolSliceReader{ctx: ctx, writes: m.writes, rng: m.rng}
	r.readImpl = r.prepareAndRead
	return r
}

type readFunction func([]byte) (int, error)

type poolSliceReader struct {
	ctx      context.Context
	writes   poolWriteList
	rng      Range
	readImpl readFunction
}

// Implements io.Reader
func (r *poolSliceReader) Read(p []byte) (n int, err error) {
	return r.readImpl(p)
}

// prepareAndRead determines whether we are about to read from an area covered
// by a write. If so, it obtains an io.Reader for the write and starts reading
// from it (additionally, subsequent Read() calls will skip this function, and
// go through a fast path that delegates to the Reader, until we reach the end
// of the write). If, instead, the area we're looking at is unwritten, we will
// have a fast path until the end of the unwritten area which simply fills the
// output buffers with zeros. At the end of each contiguous region (one single
// write or unwritten area), we go through this again.
func (r *poolSliceReader) prepareAndRead(p []byte) (n int, err error) {
	if r.rng.Size <= 0 {
		return r.setError(0, io.EOF)
	}
	if len(p) == 0 {
		return 0, nil
	}

	if len(r.writes) > 0 {
		w := r.writes[0]
		intersection := w.dst.Intersect(r.rng)

		if intersection.First() > r.rng.First() {
			r.readImpl = r.zeroReadFunc(intersection.First() - r.rng.First())
		} else {
			r.writes = r.writes[1:]
			slice := w.src
			if intersection != w.dst {
				slice = w.src.Slice(Range{
					Base: intersection.First() - w.dst.First(),
					Size: intersection.Size,
				})
			}
			sliceReader := slice.NewReader(r.ctx)
			r.readImpl = r.readerReadFunc(sliceReader, intersection.Size)
		}
	} else {
		r.readImpl = r.zeroReadFunc(r.rng.Size)
	}

	return r.readImpl(p)
}

// zeroReadFunc returns a read function that fills up to bytesLeft bytes
// in the buffer with zeros, after which it switches to the slow path.
func (r *poolSliceReader) zeroReadFunc(bytesLeft uint64) readFunction {
	r.rng = r.rng.TrimLeft(bytesLeft)
	return func(p []byte) (n int, err error) {
		zeroCount := min(bytesLeft, uint64(len(p)))
		for i := uint64(0); i < zeroCount; i++ {
			p[i] = 0
		}

		bytesLeft -= zeroCount
		if bytesLeft == 0 {
			r.readImpl = r.prepareAndRead
		}

		return int(zeroCount), nil
	}
}

// readerReadFunc returns a read function that reads up to bytesLeft
// bytes from srcReader after which it switches to the slow path.
func (r *poolSliceReader) readerReadFunc(srcReader io.Reader, bytesLeft uint64) readFunction {
	r.rng = r.rng.TrimLeft(bytesLeft)
	return func(p []byte) (n int, err error) {
		bytesToRead := min(bytesLeft, uint64(len(p)))

		bytesRead, err := srcReader.Read(p[:bytesToRead])
		if bytesRead == 0 && errors.Cause(err) == io.EOF {
			return r.setError(0, fmt.Errorf("Premature EOF from underlying reader"))
		}
		if err != nil && err != io.EOF {
			return r.setError(bytesRead, err)
		}

		bytesLeft -= uint64(bytesRead)
		if bytesLeft == 0 {
			r.readImpl = r.prepareAndRead
		}
		return bytesRead, nil
	}
}

// setError returns its arguments and makes subsequent Read()s return (0, err).
func (r *poolSliceReader) setError(n int, err error) (int, error) {
	r.readImpl = func([]byte) (int, error) {
		return 0, err
	}
	return n, err
}

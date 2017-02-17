/*
 * Copyright (C) 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include "scratch_allocator.h"

#include <functional>
#include <memory>
#include <tuple>
#include <unordered_map>
#include <vector>

#include <gmock/gmock.h>
#include <gtest/gtest.h>

namespace core {

namespace test {

namespace {
enum Type { TypeA, TypeB, TypeC };
struct I {
    virtual Type getType() = 0;
};
struct A : public I {
    Type getType() override { return TypeA; }
};
struct B : public I {
    Type getType() override { return TypeB; }
};
struct C : public I {
    Type getType() override { return TypeC; }
};
} // anonymous namespace

// A testing class to test the ScratchAllocator. It provides the buffer
// creating and releasing functions which are suppposed to be used to test
// scratch allocators.
class ScratchAllocatorTestBase : public ::testing::Test {
public:
    ScratchAllocatorTestBase()
        : mCreatedBuffers()
        , mLastCreatedBufferBase(nullptr)
        , mLastCreatedBufferEnd(nullptr) {}

    // A mock buffer creating function. Returns a tuple of two: 1) a pointer to
    // a buffer aligned to the given alignment value, 2) the size of the buffer
    // starting from the return buffer pointer.
    std::tuple<uint8_t*, size_t> createBuffer(size_t request_size,
                                              size_t min_heap_buffer_size,
                                              size_t alignment) {
        size_t size = request_size > min_heap_buffer_size
                          ? request_size
                          : min_heap_buffer_size;
        auto buffer = std::unique_ptr<std::vector<uint8_t>>(
            new std::vector<uint8_t>(size, 0));
        // Adjust the created buffer to fit the alignment.
        buffer->reserve(size + alignment);
        uintptr_t p = reinterpret_cast<uintptr_t>(buffer->data());
        if (uintptr_t o = p % alignment) {
            p += alignment - o;
            buffer->insert(buffer->end(), alignment - o, 0);
        }
        mLastCreatedBufferBase = reinterpret_cast<uint8_t*>(p);
        mLastCreatedBufferEnd = &*buffer->end();
        mCreatedBuffers[mLastCreatedBufferBase] = std::move(buffer);
        EXPECT_EQ(0, p % alignment) << "createBuffer: The buffer pointer to be "
                                       "is not aligned to the alignment value:"
                                    << alignment;
        return std::make_tuple(mLastCreatedBufferBase, size);
    }

    // A mock buffer releasing function.
    void freeBuffer(uint8_t* buffer) {
        EXPECT_EQ(1, mCreatedBuffers.count(buffer));
        mCreatedBuffers.erase(buffer);
    }

    // Bookkeeping of created functions.
    std::unordered_map<uint8_t*, std::unique_ptr<std::vector<uint8_t>>>
        mCreatedBuffers;
    // The base address of the last created buffer.
    uint8_t* mLastCreatedBufferBase;
    // The end address of the last created buffer.
    uint8_t* mLastCreatedBufferEnd;
};

// Templated tests.
template <typename Allocator>
class ScratchAllocatorTest : public ScratchAllocatorTestBase {};

namespace {

// A derived class template from the test target: ScratchAllocator. It exposes
// the internal members and functions for tests.
template <size_t stack_capacity>
class ScratchAllocatorForTest : public ScratchAllocator<stack_capacity> {
public:
    using BufferHeader =
        typename ScratchAllocator<stack_capacity>::BufferHeader;

    // Constructs a scratch allocator with a custom alignment for the
    // internally created buffers for tests. The underlying buffer creating and
    // releasing functions are provided by the given test instance.
    ScratchAllocatorForTest(size_t heap_buffer_size,
                            ScratchAllocatorTestBase* test,
                            size_t buffer_alignment = alignof(BufferHeader))
        : ScratchAllocator<stack_capacity>(
              [test, heap_buffer_size, buffer_alignment](size_t size) {
                  return test->createBuffer(size, heap_buffer_size,
                                            buffer_alignment);
              },
              [test](uint8_t* buffer) { return test->freeBuffer(buffer); }) {
        this->mUsableStackBufferSize =
            this->mStackBufferHeader->end - this->mStackBufferHeader->head;
    }

    // Delegates the internal allocation buffer initialization function so that
    // we can test it.
    static BufferHeader* initializeAllocationBuffer(uint8_t* buffer,
                                                    size_t size) {
        return ScratchAllocator<stack_capacity>::initializeAllocationBuffer(
            buffer, size);
    }

    // Delegates the internal try-to-allocate function so that we can test it.
    static uint8_t* tryAllocateOnBuffer(size_t size, size_t alignment,
                                        BufferHeader* buffer) {
        return ScratchAllocator<stack_capacity>::tryAllocateOnBuffer(
            size, alignment, buffer);
    }

    // Returns the first buffer (aka. stack buffer) header.
    BufferHeader* getStackBufferHeader() { return this->mStackBufferHeader; };

    // The usable memory in bytes of the stack buffer.
    size_t mUsableStackBufferSize;
};

} // anonymous namespace

TYPED_TEST_CASE_P(ScratchAllocatorTest);

TYPED_TEST_P(ScratchAllocatorTest, AlignmentOnStackBuffer) {
    // Tests whether the allocator handles the alignment on its stack buffer
    // correctly. Note that when the stack buffer size is too small, this test
    // may be allocating on the heap buffers instead of stack buffer.
    size_t alignments[] = {1, 2, 4, 8, 16, 8, 4, 2, 1};
    size_t sizes[] = {1, 2, 4, 8, 16};
    TypeParam allocator(1, this);
    for (size_t a : alignments) {
        for (size_t s : sizes) {
            uintptr_t ptr =
                reinterpret_cast<uintptr_t>(allocator.template allocate(s, a));
            EXPECT_EQ(0, ptr % a) << "allocation size: " << s << ", "
                                  << "alignment: " << a;
        }
    }
}

TYPED_TEST_P(ScratchAllocatorTest, AlignmentOnHeapBuffer) {
    // Tests whether the allocator handles the alignment on its heap buffers
    // correctly.
    size_t alloc_alignments[] = {1, 2, 4, 8, 16, 8, 4, 2, 1};
    size_t alloc_sizes[] = {1, 2, 4, 8, 16};
    size_t buf_alignments[] = {1, 3, 5, 7, 17};
    for (size_t buf_a : buf_alignments) {
        TypeParam allocator(1, this, buf_a);
        // Fill up the stack buffer so that we always test on the heap buffer.
        allocator.allocate(allocator.mUsableStackBufferSize, 1);
        for (size_t a : alloc_alignments) {
            for (size_t s : alloc_sizes) {
                uintptr_t ptr = reinterpret_cast<uintptr_t>(
                    allocator.template allocate(s, a));
                EXPECT_EQ(0, ptr % a) << "allocation size: " << s << ", "
                                      << "alignment: " << a;
            }
        }
    }
}

TYPED_TEST_P(ScratchAllocatorTest, CreateInternalBuffers) {
    // Tests on an allocator whose minimal heap buffer size is only 1 byte, so
    // that once the stack buffer is full, any allocation request with size
    // greater than 1 results into new a buffer.
    TypeParam allocator(1, this);
    // Fill up the the first buffer which should be a stack buffer.
    void* ptr =
        allocator.template allocate(allocator.mUsableStackBufferSize, 1);
    EXPECT_EQ(0, this->mCreatedBuffers.size());
    // From now on, allocate() should trigger the creation of new internal
    // buffers.
    for (size_t i = 0; i < 10; i++) {
        void* ptr = allocator.template allocate(100, 1);
        EXPECT_NE(0, this->mCreatedBuffers.size());
        // ptr should be in the range of the created buffer.
        EXPECT_TRUE(ptr > this->mLastCreatedBufferBase);
        EXPECT_TRUE(ptr < this->mLastCreatedBufferEnd);
    }
}

TYPED_TEST_P(ScratchAllocatorTest, FreeInternalBuffers) {
    // Tests that the allocator is able to release its internal buffers when it
    // goes out of scope.
    size_t buf_alignments[] = {1, 2, 4, 8, 16, 3, 5, 7, 11, 13, 17};
    for (size_t buf_a : buf_alignments) {
        TypeParam allocator(1, this, buf_a);
        allocator.template allocate(allocator.mUsableStackBufferSize + 1, 1);
        allocator.template allocate(allocator.mUsableStackBufferSize + 1, 1);
        allocator.template allocate(allocator.mUsableStackBufferSize + 1, 1);
        EXPECT_EQ(3, this->mCreatedBuffers.size());
    }
    // All the buffers should be erased by now.
    EXPECT_EQ(0, this->mCreatedBuffers.size());
}

TYPED_TEST_P(ScratchAllocatorTest, Reset) {
    // Tests that function reset() does reset the stack buffer and releases
    // all the heap buffers.
    TypeParam allocator(1024, this, 17 /* bad internal buffer alignment */);
    void* first = allocator.template allocate(1, 1);
    EXPECT_EQ(0, this->mCreatedBuffers.size());
    // The stack buffer should has been reset, a same allocation request should
    // return a same pointer.
    allocator.template reset();
    EXPECT_EQ(first, allocator.template allocate(1, 1));
    // No heap buffer should has been created by now.
    EXPECT_EQ(0, this->mCreatedBuffers.size());
    allocator.template allocate(allocator.mUsableStackBufferSize + 1, 1);
    EXPECT_EQ(1, this->mCreatedBuffers.size());
    allocator.template reset();
    EXPECT_EQ(0, this->mCreatedBuffers.size());
}

TYPED_TEST_P(ScratchAllocatorTest, Create) {
    TypeParam allocator(1024, this);
    // Test the stack buffer first.
    auto* base = allocator.template create<char>(1);
    allocator.template reset();
    auto* a = allocator.template create<char>(1);
    EXPECT_EQ(base, a);
    // Resets then tests on the new created buffers.
    allocator.reset();
    allocator.allocate(allocator.mUsableStackBufferSize, 1);
    int* b = allocator.template create<int>(1);
    int* c = allocator.template create<int>(2);
    int* d = allocator.template create<int>(3);
    EXPECT_EQ(b + 1, c);
    EXPECT_EQ(d, c + 2);
    EXPECT_TRUE(reinterpret_cast<uint8_t*>(b) > this->mLastCreatedBufferBase);
    EXPECT_TRUE(reinterpret_cast<uint8_t*>(d) < this->mLastCreatedBufferEnd);
}

// Tests function make.
TYPED_TEST_P(ScratchAllocatorTest, Make) {
    TypeParam allocator(0x1000, this);
    size_t* made_ptrs[100];
    for (size_t i = 0; i < 100; i++) {
        made_ptrs[i] = allocator.template make<size_t>(i);
    }
    for (size_t i = 0; i < 100; i++) {
        EXPECT_EQ(i, *made_ptrs[i]);
    }
}

// Tests function vector.
TYPED_TEST_P(ScratchAllocatorTest, Vector) {
    TypeParam allocator(0x1000, this);
    auto v = allocator.template vector<I*>(3);

    for (auto e : v) {
        FAIL() << "Empty vector should not interate.";
    }

    v.append(allocator.template create<A>());
    v.append(allocator.template create<B>());
    v.append(allocator.template create<C>());

    EXPECT_EQ(TypeA, v[0]->getType());
    EXPECT_EQ(TypeB, v[1]->getType());
    EXPECT_EQ(TypeC, v[2]->getType());

    Type expected[3] = {TypeA, TypeB, TypeC};
    int i = 0;
    for (auto c : v) {
        EXPECT_EQ(expected[i++], c->getType());
    }
}

// Tests function map.
TYPED_TEST_P(ScratchAllocatorTest, Map) {
    TypeParam allocator(0x1000, this);
    auto m = allocator.template map<int, I*>(3);

    for (auto e : m) {
        FAIL() << "Empty map should not interate.";
    }

    m.set(2, allocator.template create<A>());
    m.set(4, allocator.template create<B>());
    m.set(8, allocator.template create<C>());

    int keys[3] = {2, 4, 8};
    Type types[3] = {TypeA, TypeB, TypeC};
    int i = 0;
    for (auto e : m) {
        EXPECT_EQ(keys[i], e.key);
        EXPECT_EQ(types[i], e.value->getType());
        i++;
    }
}

REGISTER_TYPED_TEST_CASE_P(ScratchAllocatorTest, Reset, AlignmentOnStackBuffer,
                           AlignmentOnHeapBuffer, CreateInternalBuffers,
                           FreeInternalBuffers, Create, Make, Vector, Map);

template <size_t stack_capacity>
using SA = ScratchAllocatorForTest<stack_capacity>;
// Implementations of stack scratch allocator to test.
using AllocatorImplementations = ::testing::Types<SA<1>, SA<5>, SA<1024>>;

INSTANTIATE_TYPED_TEST_CASE_P(ScratchAllocatorTest, ScratchAllocatorTest,
                              AllocatorImplementations);

} // namespace test
}  // namespace core

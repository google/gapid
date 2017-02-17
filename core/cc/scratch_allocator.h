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

#ifndef CORE_SCRATCH_ALLOCATOR_H
#define CORE_SCRATCH_ALLOCATOR_H

#include "map.h"
#include "vector.h"

#include <array>
#include <functional>
#include <memory>
#include <tuple>
#include <utility>

namespace core {

// ScratchAllocator is a simple linear allocator that uses a stack buffer (a
// static-sized array) for allocation in priority. Internally it maintains a
// linked list of 'allocation buffers', each of which is a pre-allocated memory
// space to be reserved for allocation requests. Stack buffer is the first
// buffer in the link list to be used. New internal buffers will be created
// when none of the existing internal buffers can provide the memory for a
// coming allocation request. The user needs to provide the buffer creating and
// releasing functions when constructing this allocator. Otherwise the
// allocator won't create new internal buffers and will fail when the stack
// buffer is filled up. The buffers created via the user specified buffer
// creating function are called 'heap buffers' as contrary to the stack buffer,
// but those buffers do not have to be located on the process heap.
template <size_t stack_capacity>
class ScratchAllocator {
public:
    // The signature of the buffer creating function. The function should take
    // one argment which specifies the required memory size in bytes, and
    // returns a tuple of two: 1) a pointer to the base address of the created
    // buffer, 2) the size of the created buffer in bytes.
    using CreateBufferFunction =
        std::function<std::tuple<uint8_t*, size_t>(size_t)>;
    // The signature of the buffer releasing function. The function should take
    // one argument which is a pointer to the base address of the buffer to be
    // released.
    using FreeBufferFunction = std::function<void(uint8_t*)>;

    // Constructs a ScratchAllocator with specified buffer creating and
    // releasing functions.
    inline ScratchAllocator(CreateBufferFunction create_buffer,
                            FreeBufferFunction free_buffer);

    // Destructs the ScratchAllocator, releasing all the memory allocated for
    // the buffers created via the create buffer callback (aka. heap buffers).
    inline ~ScratchAllocator() { reset(); };

    // reset sets the head of the stack buffer to its initial value and
    // releases all the memory allocated for the buffers created via the
    // create buffer callback (aka. heap buffers).
    inline void reset();

    // allocate reserves size bytes from the allocator and returns the aligned
    // pointer to be used by the caller. The pointer returned will be aligned
    // to the specified number of bytes, even when the value of size is 0 (but
    // no memory reserved when size is 0).
    inline void* allocate(size_t size, size_t alignment);

    // create constructs count instances of T using the default constructor,
    // and returns the type T-aligned pointer to the first instance. The
    // created instances of T will be allocated by this allocator in its
    // internal buffers. When the count value is 0, a null pointer will be
    // returned and no instance will be created.
    template <typename T>
    inline T* create(size_t count = 1);

    // make constructs and returns an instance of T using the supplied
    // constructor arguments, allocates the storage of the constructed T in its
    // internal buffers, then returns a pointer to the constructed T.
    template <typename T, typename... ARGS>
    inline T* make(ARGS&&... args);

    // vector creates and returns a core::Vector with the specified maximum
    // capacity. The internal data of the vector will be allocated in this
    // allocator.
    template <typename T>
    inline Vector<T> vector(size_t capacity);

    // map creates and returns a core::Map with the specified maximum
    // capacity. The internal storage of the map will be allocated in this
    // allocator.
    template <typename K, typename V>
    inline Map<K, V> map(size_t capacity);

protected:
    // Buffer header holds the bookkeeping info of an allocation buffer.
    struct BufferHeader {
        uint8_t* base;
        uint8_t* end;
        uint8_t* head;
        BufferHeader* next;
    };

    // The first buffer header. It should be initialized with the stack buffer
    // info.
    BufferHeader* mStackBufferHeader;

    // Setup an allocation buffer with the buffer header initialized inside.
    // On a successful initialization, returns a pointer to the buffer header,
    // otherwise returns a null pointer.
    static BufferHeader* initializeAllocationBuffer(uint8_t* buffer,
                                                    size_t size);

    // Try to allocate for a memory request on a specified buffer (either a
    // stack buffer or a buffer created via buffer creating call back
    // function). On a successful allocation, returns an aligned pointer for
    // the request and advance the head record of the buffer. Otherwise returns
    // a null pointer. Alignment value can not be 0.
    static uint8_t* tryAllocateOnBuffer(size_t size, size_t alignment,
                                        BufferHeader* buffer);

private:
    ScratchAllocator() = delete;

    // The allocator's stack buffer.
    const static size_t mStackBufferSize =
        stack_capacity + sizeof(BufferHeader) + alignof(BufferHeader);
    std::array<uint8_t, mStackBufferSize> mStackBuffer;

    // The buffer creating function, set when the constructor is called. It is
    // called when this allocator requires new buffers.
    CreateBufferFunction mCreateBuffer;

    // The buffer freeing function, set when the constructor is called. It is
    // called when the internal buffers are to be released by this allocator.
    FreeBufferFunction mFreeBuffer;
};

template <size_t stack_capacity>
ScratchAllocator<stack_capacity>::ScratchAllocator(
    CreateBufferFunction create_buffer, FreeBufferFunction free_buffer)
    : mStackBufferHeader(nullptr)
    , mCreateBuffer(create_buffer)
    , mFreeBuffer(free_buffer) {
    static_assert(stack_capacity > 0,
                  "Stack Buffer capacity must be greater than 0");
    mStackBufferHeader =
        initializeAllocationBuffer(mStackBuffer.data(), mStackBufferSize);
}

template <size_t stack_capacity>
inline void ScratchAllocator<stack_capacity>::reset() {
    auto* heap_buffer = mStackBufferHeader->next;
    while (heap_buffer && mFreeBuffer) {
        auto* next_buffer = heap_buffer->next;
        mFreeBuffer(heap_buffer->base);
        heap_buffer = next_buffer;
    }
    mStackBufferHeader =
        initializeAllocationBuffer(mStackBuffer.data(), mStackBufferSize);
}

template <size_t stack_capacity>
inline void* ScratchAllocator<stack_capacity>::allocate(size_t size,
                                                        size_t alignment) {
    GAPID_ASSERT(alignment != 0);

    typename ScratchAllocator<stack_capacity>::BufferHeader*
        current_buffer_header = mStackBufferHeader;
    typename ScratchAllocator<stack_capacity>::BufferHeader*
        prev_buffer_header = nullptr;
    while (current_buffer_header) {
        if (uint8_t* ptr =
                tryAllocateOnBuffer(size, alignment, current_buffer_header)) {
            return ptr;
        }
        prev_buffer_header = current_buffer_header;
        current_buffer_header = current_buffer_header->next;
    }
    // Need new memory buffer to allocate.
    uint8_t* new_buffer = nullptr;
    size_t new_buffer_size =
        size + alignment + sizeof(BufferHeader) + alignof(BufferHeader);
    if (!mCreateBuffer || !mFreeBuffer) {
        GAPID_FATAL(
            "ScratchAllocator: Buffer creating and/or releasing "
            "functions not defined, can not create new internal buffers."
            "\nAllocation request size: 0x%x bytes, alignment: %u",
            size, alignment);
        return nullptr;
    }
    std::tie(new_buffer, new_buffer_size) = mCreateBuffer(new_buffer_size);
    auto* new_buffer_header =
        initializeAllocationBuffer(new_buffer, new_buffer_size);
    if (!new_buffer_header) {
        GAPID_FATAL(
            "ScratchAllocator: Can not initialize allocation buffer header"
            "on the new created buffer. The start address of the new "
            "created buffer: 0x%x, the size of the new created buffer: %u",
            new_buffer, new_buffer_size);
        return nullptr;
    }
    prev_buffer_header->next = new_buffer_header;
    uint8_t* ptr =
        tryAllocateOnBuffer(size, alignment, prev_buffer_header->next);
    return ptr;
}

template <size_t stack_capacity>
uint8_t* ScratchAllocator<stack_capacity>::tryAllocateOnBuffer(
    size_t size, size_t alignment,
    typename ScratchAllocator<stack_capacity>::BufferHeader* buffer) {
    GAPID_ASSERT(alignment != 0);
    uint8_t* head = buffer->head;
    uint8_t* end = buffer->end;
    uintptr_t a = static_cast<uintptr_t>(alignment);
    uintptr_t p = reinterpret_cast<uintptr_t>(head);
    if (uintptr_t o = p % a) {
        head += a - o; // Alignment
    }
    uint8_t* out = head;
    head += size;
    if (head > end) {
        return nullptr;
    } else {
        buffer->head = head;
        return out;
    }
}

template <size_t stack_capacity>
typename ScratchAllocator<stack_capacity>::BufferHeader*
ScratchAllocator<stack_capacity>::initializeAllocationBuffer(uint8_t* buffer,
                                                             size_t size) {
    using BufferHeader =
        typename ScratchAllocator<stack_capacity>::BufferHeader;
    // First, handle the alignment for BufferHeader. If it is impossible to
    // create a buffer header in the given buffer, returns a null pointer.
    uintptr_t header_loc = reinterpret_cast<uintptr_t>(buffer);
    if (uintptr_t o = header_loc % alignof(BufferHeader)) {
        header_loc += alignof(BufferHeader) - o;
    }
    uint8_t* header_ptr = reinterpret_cast<uint8_t*>(header_loc);
    if (header_ptr + sizeof(BufferHeader) > buffer + size) {
        return nullptr;
    }
    // Creates the header buffer in the aligned address.
    typename ScratchAllocator<stack_capacity>::BufferHeader* header =
        new ((void*)header_ptr) BufferHeader;
    header->base = buffer;
    header->end = buffer + size;
    header->head = header_ptr + sizeof(BufferHeader);
    header->next = nullptr;
    return header;
}

template <size_t stack_capacity>
template <typename T>
inline T* ScratchAllocator<stack_capacity>::create(size_t count /* = 1 */) {
    if (count == 0) {
        return nullptr;
    }
    void* buffer = allocate(sizeof(T) * count, alignof(T));
    T* item = reinterpret_cast<T*>(buffer);
    while (count != 0) {
        new (item++) T();
        count -= 1;
    }
    return reinterpret_cast<T*>(buffer);
}

template <size_t stack_capacity>
template <typename T, typename... ARGS>
inline T* ScratchAllocator<stack_capacity>::make(ARGS&&... args) {
    void* buffer = allocate(sizeof(T), alignof(T));
    return new (buffer) T(std::forward<ARGS>(args)...);
}

template <size_t stack_capacity>
template <typename T>
inline Vector<T> ScratchAllocator<stack_capacity>::vector(size_t capacity) {
    T* first = create<T>(capacity);
    return Vector<T>(first, 0, capacity);
}

template <size_t stack_capacity>
template <typename K, typename V>
inline Map<K, V> ScratchAllocator<stack_capacity>::map(size_t capacity) {
    typedef typename Map<K, V>::Entry T;
    T* first = create<T>(capacity);
    return Map<K, V>(first, capacity);
}

using DefaultScratchAllocator = ScratchAllocator<1024>;

}  // namespace core

#endif  // CORE_SCRATCH_ALLOCATOR_H

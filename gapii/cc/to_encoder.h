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

#ifndef GAPII_ENCODE_CONVERTERS_H
#define GAPII_ENCODE_CONVERTERS_H

#include "slice.h"

#include "core/cc/coder/memory.h"
#include "core/cc/scratch_allocator.h"

#include <stdio.h>
#include <string.h>

#include <string>
#include <unordered_map>

namespace gapii {

template<typename Out, typename In, typename Allocator>
struct EncoderConverter {};


// Passthrough: T -> T
template<typename T, typename Allocator>
struct EncoderConverter<T, T, Allocator> {
    static inline T convert(const T& in, Allocator& alloc) { return in; }
};

// Passthrough: T* -> T*
template<typename T, typename Allocator>
struct EncoderConverter<T*, T*, Allocator> {
    static inline T* convert(const T* in, Allocator& alloc) { return in; }
};

// Passthrough: void* -> coder::T__P
template<typename Out, typename Allocator>
struct EncoderConverter<Out, void*, Allocator> {
    static inline Out convert(void* in, Allocator& alloc) {
        uint32_t poolID = 0; // TODO: Support non-application pools?
        uint64_t address = reinterpret_cast<uintptr_t>(in);
        return Out(address, poolID);
    }
};

// T* -> coder::T__P
template<typename Out, typename T, typename Allocator>
struct EncoderConverter<Out, T*, Allocator> {
    static inline Out convert(T* in, Allocator& alloc) {
        return EncoderConverter<Out, void*, Allocator>::convert(in, alloc);
    }
};

// std::string -> const char*
template<typename Allocator>
struct EncoderConverter<const char*, std::string, Allocator> {
    static inline const char* convert(const std::string& in, Allocator& alloc) {
        char* buf = alloc.template create<char>(in.size() + 1);
        strncpy(buf, in.c_str(), in.size() + 1);
        return buf;
    }
};

// std::shared_ptr<T> -> coder::T*
template<typename T, typename Allocator>
struct EncoderConverter<typename T::CoderType*, std::shared_ptr<T>, Allocator> {
    typedef typename T::CoderType* Out;
    static inline Out convert(const std::shared_ptr<T>& in, Allocator& alloc) {
        return alloc.template make<typename T::CoderType>(in->encodeable(alloc));
    }
};

// T -> coder::T
template<typename T, typename Allocator>
struct EncoderConverter<typename T::CoderType, T, Allocator> {
    static inline typename T::CoderType convert(const T& in, Allocator& alloc) {
        return in.encodeable(alloc);
    }
};

// gapii::Slice<T> -> coder::T__S
template<typename Out, typename T, typename Allocator>
struct EncoderConverter< Out, gapii::Slice<T>, Allocator> {
    static inline Out convert(const gapii::Slice<T>& in, Allocator& alloc) {
        uint32_t poolID = 0; // TODO: Support non-application pools?
        uint64_t base = reinterpret_cast<uintptr_t>(in.begin());
        uint64_t root = base; // TODO: Track root.
        return Out(core::coder::memory::SliceInfo(root, base, in.count(), poolID));
    }
};

// core::StaticArray<T, N> -> coder::T
template<typename Out, typename T, int N, typename Allocator>
struct EncoderConverter< Out, core::StaticArray<T, N>, Allocator> {
    static inline Out convert(const core::StaticArray<T, N>& in, Allocator& alloc) {
        return Out(in);
    }
};

// core::StaticArray<T, N> -> core::StaticArray<T, N> (resolves ambiguous case)
template<typename T, int N, typename Allocator>
struct EncoderConverter< core::StaticArray<T, N>, core::StaticArray<T, N>, Allocator> {
    static inline core::StaticArray<T, N> convert(const core::StaticArray<T, N>& in, Allocator& alloc) {
        return in;
    }
};

// std::unordered_map<KeyIn, ValueIn> -> core::Map<KeyOut, ValueOut>
template<typename KeyOut, typename ValueOut, typename KeyIn, typename ValueIn, typename Allocator>
struct EncoderConverter< core::Map<KeyOut, ValueOut>, std::unordered_map<KeyIn, ValueIn>, Allocator> {
    typedef core::Map<KeyOut, ValueOut> Out;
    typedef std::unordered_map<KeyIn, ValueIn> In;

    static inline Out convert(const In& in, Allocator& alloc) {
        Out out = alloc.template map<KeyOut, ValueOut>(in.size());
        for (auto it : in) {
            KeyOut key = EncoderConverter<KeyOut, KeyIn, Allocator>::convert(it.first, alloc);
            ValueOut value = EncoderConverter<ValueOut, ValueIn, Allocator>::convert(it.second, alloc);
            out.set(key, value);
        }
        return out;
    }
};

// core::StaticArray<ElIn, N> -> core::StaticArray<ElOut, N>
template<typename ElOut, typename ElIn, int N, typename Allocator>
struct EncoderConverter< core::StaticArray<ElOut, N>, core::StaticArray<ElIn, N>, Allocator> {
    static inline core::StaticArray<ElOut, N> convert(const core::StaticArray<ElIn, N>& in, Allocator& alloc) {
        core::StaticArray<ElOut, N> out;
        for (size_t i = 0; i < N; ++i) {
            out[i] = EncoderConverter<ElOut, ElIn, Allocator>::convert(in[i], alloc);
        }
        return out;
    }
};

template <typename Out, typename In, typename Allocator>
inline Out toEncoder(const In& in, Allocator& alloc) {
    return EncoderConverter<Out, In, Allocator>::convert(in, alloc);
}

}  // namespace gapii

#endif // GAPII_ENCODE_CONVERTERS_H

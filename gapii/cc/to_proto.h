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

#ifndef GAPII_TO_PROTO_H
#define GAPII_TO_PROTO_H

#include "gapii/cc/shared_map.h"
#include "gapis/memory/memory_pb/memory.pb.h"

#include "core/cc/static_array.h"

#include <type_traits>
#include <unordered_map>

namespace gapii {

class ToProtoContext {
  public:
    ToProtoContext() : mSeenReferences { { nullptr, 0 } } { }

    // Returns unique ReferenceID for the given address,
    // and true when the address is seen for the first time.
    // Nullptr address is always mapped to ReferenceID 0.
    virtual std::pair<uint64_t, bool> GetReferenceID(const void* address) {
      auto it = mSeenReferences.emplace(address, mSeenReferences.size());
      return std::pair<uint64_t, bool>(it.first->second, it.second);
    }

  private:
    std::unordered_map<const void*, uint64_t> mSeenReferences;
};

// Default converter for any type which is not specialized below.
template<typename Out, typename In>
struct ProtoConverter {
    static inline void convert(Out* out, const In& in, ToProtoContext&) {
        *out = static_cast<Out>(in);
    }
};

// Converts pointers types to proto
template<typename Out>
struct ProtoConverter<Out, const void*> {
    static inline void convert(Out* out, const void* in, ToProtoContext&) {
        *out = reinterpret_cast<Out>(in);
    }
};

// Converts pointers types to proto
template<typename Out>
struct ProtoConverter<Out, void*> {
    static inline void convert(Out* out, void* in, ToProtoContext&) {
        *out = reinterpret_cast<Out>(in);
    }
};

// Converts Slice to proto
template<typename T>
struct ProtoConverter<memory_pb::Slice, gapii::Slice<T>> {
    static inline void convert(memory_pb::Slice* out, const gapii::Slice<T>& in, ToProtoContext& ctx) {
        out->set_pool(0); // TODO: Set pool id
        out->set_root(reinterpret_cast<uintptr_t>(in.begin()));
        out->set_base(reinterpret_cast<uintptr_t>(in.begin()));
        out->set_count(in.count());
    }
};

// Converts objects which implement their own toProto method
// (detected by presence of ProtoType inside the class)
template<typename T>
struct ProtoConverter<typename T::ProtoType, T> {
    static inline void convert(typename T::ProtoType* out, const T& in, ToProtoContext& ctx) {
        in.toProto(out, ctx);
    }
};

// Converts StaticArray to proto
template <typename Out, typename In, int N>
struct ProtoConverter<Out, core::StaticArray<In, N>> {
    static inline void convert(Out* out, const core::StaticArray<In, N>& in, ToProtoContext& ctx) {
        out->Reserve(N);
        for (int i = 0; i < N; i++) {
            toProto(out->Add(), in[i], ctx);
        }
    }
};

// Converts SharedMap to proto
template <typename Out, typename K, typename V>
struct ProtoConverter<Out, SharedMap<K, V>> {
    static inline void convert(Out* out, const SharedMap<K, V>& in, ToProtoContext& ctx) {
        auto ref = ctx.GetReferenceID(in.get());
        out->set_referenceid(ref.first);
        if (ref.second) {
            std::map<K, V> sorted(in.begin(), in.end());
            auto keys = out->mutable_keys();
            auto values = out->mutable_values();
            keys->Reserve(sorted.size());
            values->Reserve(sorted.size());
            for (auto it : sorted) {
                toProto(keys->Add(), it.first, ctx);
                toProto(values->Add(), it.second, ctx);
            }
            trimKeys<std::is_integral<K>::value>(keys);
        }
    }

    // Remove tailing consecutive keys to save space (only applicable for integral keys).
    template<bool IsInt, typename T>
    static typename std::enable_if<IsInt>::type trimKeys(T keys) {
        while(keys->size() >= 2 && keys->Get(keys->size() - 2) + 1 == keys->Get(keys->size() - 1)) {
            keys->RemoveLast();
        }
        if (keys->size() == 1 && keys->Get(0) == 0) {
            keys->RemoveLast();
        }
    }
    template<bool IsInt, typename T>
    static typename std::enable_if<!IsInt>::type trimKeys(T keys) {
    }
};

// Converts reference to proto
template <typename Out, typename T>
struct ProtoConverter<Out, std::shared_ptr<T>> {
    static inline void convert(Out* out, const std::shared_ptr<T>& in, ToProtoContext& ctx) {
        auto ref = ctx.GetReferenceID(in.get());
        out->set_referenceid(ref.first);
        if (ref.second) {
            toProto(out->mutable_value(), *in, ctx);
        }
    }
};

template <typename Out, typename In>
inline void toProto(Out* out, const In& in, ToProtoContext& ctx) {
    ProtoConverter<Out, In>::convert(out, in, ctx);
}

inline const std::string& toProtoString(const std::string& str) {
    return str;
}

inline const char* toProtoString(const char* str) {
    return (str != nullptr) ? str : "";
}

}  // namespace gapii

#endif // GAPII_TO_PROTO_H

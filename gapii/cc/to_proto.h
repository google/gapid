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

#include "gapil/runtime/cc/slice.h"

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
    std::pair<uint64_t, bool> GetReferenceID(const void* address) {
      auto it = mSeenReferences.emplace(address, mSeenReferences.size());
      return std::pair<uint64_t, bool>(it.first->second, it.second);
    }

    // This method is called for all serialized slices.
    template<typename T>
    void SeenSlice(const gapil::Slice<T>& s) {
      mSeenSlices.push_back(s.template as<uint8_t>());
    }

    // Return a vector of all serialized slices within this context.
    // Note that slices are recorded as byte-slices (the original type is discarded).
    const std::vector<gapil::Slice<uint8_t>>& SeenSlices() { return mSeenSlices; }

  private:
    std::unordered_map<const void*, uint64_t> mSeenReferences;
    std::vector<gapil::Slice<uint8_t>> mSeenSlices;
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
struct ProtoConverter<memory::Slice, gapil::Slice<T>> {
    static inline void convert(memory::Slice* out, const gapil::Slice<T>& in, ToProtoContext& ctx) {
        ctx.SeenSlice(in);

        auto base = reinterpret_cast<uintptr_t>(in.begin());
        if (auto p = in.pool()) {
          base -= reinterpret_cast<uintptr_t>(p->buffer);
          out->set_pool(p->id);
        }
        out->set_root(base);
        out->set_base(base);
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

// Converts gapil::String to std::string
template <>
struct ProtoConverter<std::string, gapil::String> {
    static inline void convert(std::string* out, const gapil::String& in, ToProtoContext& ctx) {
        *out = in.c_str();
    }
};

// Converts gapil::Map to proto
template <typename Out, typename K, typename V>
struct ProtoConverter<Out, gapil::Map<K, V>> {
    static inline void convert(Out* out, const gapil::Map<K, V>& in, ToProtoContext& ctx) {
        auto ref = ctx.GetReferenceID(in.instance_ptr());
        out->set_referenceid(ref.first);
        if (ref.second) {
            std::map<K, V> sorted;
            for (auto it : in) {
                sorted.emplace(std::make_pair(it.first, it.second));
            }
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
struct ProtoConverter<Out, gapil::Ref<T>> {
    static inline void convert(Out* out, const gapil::Ref<T>& in, ToProtoContext& ctx) {
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

inline const char* toProtoString(const gapil::String& str) {
    return str.c_str();
}

inline const char* toProtoString(const char* str) {
    return (str != nullptr) ? str : "";
}

}  // namespace gapii

#endif // GAPII_TO_PROTO_H

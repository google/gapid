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

#include "gapis/memory/memory_pb/memory.pb.h"

#include "core/cc/static_array.h"

#include <type_traits>
#include <unordered_map>

namespace gapii {

// ReferenceSerializer is the base class for all serializers that can be
// used to process references.
class ReferenceSerializer {
public:
    using to_proto = const std::function<::google::protobuf::Message*()>&;

    virtual uint64_t process_reference(void* address, to_proto get_message) = 0;
    virtual uint64_t process_slice(const Pool* pool, void* root, size_t bytes) = 0;
    virtual void finalize() = 0;
};

class InvalidReferences : public ReferenceSerializer {
public:
    uint64_t process_reference(void* address, to_proto get_message) override {
        GAPID_ASSERT(false && "We are trying to encode a reference where we should not");
    }
    uint64_t process_slice(const Pool* pool, void* root, size_t bytes) override {
        GAPID_ASSERT(false && "We are trying to encode a slice where we should not");
    }
    virtual void finalize() override {}
};

template<typename Out, typename In>
struct ProtoConverter {
    static inline void convert(Out* out, const In& in, ReferenceSerializer*) {
        *out = static_cast<Out>(in);
    }
};

// gapii::Slice<T> -> memory_pb::Slice*
template<typename T>
struct ProtoConverter<memory_pb::Slice*, gapii::Slice<T>> {
    static inline void convert(memory_pb::Slice* out, const gapii::Slice<T>& in, ReferenceSerializer* s) {
        uint64_t address = reinterpret_cast<uintptr_t>(in.begin());
        if (in.isApplicationPool()) {
            out->set_root(address);
            out->set_base(address);
            out->set_count(in.count());
            out->set_pool(0);
        } else {
            const Pool* pool = in.getPool();
            uint64_t identifier = s->process_slice(pool, in.begin(), in.count());
            uint64_t root = address - reinterpret_cast<uint64_t>(pool->base());
            out->set_root(root);
            out->set_base(root);
            out->set_count(in.count());
            out->set_pool(identifier);
        }
    }
};

// void* -> memory_pb::Pointer*
template<>
struct ProtoConverter<memory_pb::Pointer*, void*> {
    static inline void convert(memory_pb::Pointer* out, const void* in, ReferenceSerializer*) {
        uint32_t pool = 0; // TODO: Support non-application pools?
        uint64_t address = reinterpret_cast<uintptr_t>(in);
        out->set_pool(pool);
        out->set_address(address);
    }
};

// T* -> memory_pb::Pointer*
template<typename T>
struct ProtoConverter<memory_pb::Pointer*, T*> {
    static inline void convert(memory_pb::Pointer* out, const T* in, ReferenceSerializer* s) {
        ProtoConverter<memory_pb::Pointer*, void*>::convert(out, in, s);
    }
};

// const T& -> T::ProtoType*
template<typename T>
struct ProtoConverter<typename T::ProtoType*, T> {
    static inline void convert(typename T::ProtoType* out, const T& in, ReferenceSerializer* s) {
        in.toProto(out, s);
    }
};

// core::StaticArray<In, N> -> ::google::protobuf::RepeatedPtrField<Out>*
template <typename Out, typename In, int N>
struct ProtoConverter<::google::protobuf::RepeatedPtrField<Out>*, core::StaticArray<In, N>> {
    static inline void convert(::google::protobuf::RepeatedPtrField<Out>* out, const core::StaticArray<In, N>& in, ReferenceSerializer* s) {
        out->Reserve(N);
        for (int i = 0; i < N; i++) {
            ProtoConverter<Out*, In>::convert(out->Add(), in[i], s);
        }
    }
};

// core::StaticArray<In, N> -> ::google::protobuf::RepeatedField<Out>*
template <typename Out, typename In, int N>
struct ProtoConverter<::google::protobuf::RepeatedField<Out>*, core::StaticArray<In, N>> {
    static inline void convert(::google::protobuf::RepeatedField<Out>* out, const core::StaticArray<In, N>& in, ReferenceSerializer* s) {
        out->Reserve(N);
        for (int i = 0; i < N; i++) {
            ProtoConverter<Out, In>::convert(out->Add(), in[i], s);
        }
    }
};

template <typename EntryOut, typename KeyOut, typename KeyIn>
struct ProtoMapKeyConverter {
    static inline void convert(EntryOut* out, const KeyIn& in, ReferenceSerializer* s) {
        KeyOut key;
        ProtoConverter<KeyOut, KeyIn>::convert(&key, in, s);
        out->set_key(key);
    }
};

template <typename EntryOut, typename KeyIn>
struct ProtoMapKeyConverter<EntryOut, typename KeyIn::ProtoType, KeyIn> {
    static inline void convert(EntryOut* out, const KeyIn& in, ReferenceSerializer* s) {
        in.toProto(out->mutable_key(), s);
    }
};

template <typename EntryOut, typename ValOut, typename ValIn>
struct ProtoMapValConverter {
    static inline void convert(EntryOut* out, const ValIn& in, ReferenceSerializer* s) {
        ValOut val;
        ProtoConverter<ValOut, ValIn>::convert(&val, in, s);
        out->set_value(val);
    }
};


template <typename EntryOut, typename ValIn>
struct ProtoMapValConverter<EntryOut, memory_pb::Reference, ValIn> {
    static inline void convert(EntryOut* out, const ValIn& in, ReferenceSerializer* s) {
        out->mutable_value()->set_identifier(s->process_reference(in.get(),
            std::function<::google::protobuf::Message*()>([in, s](){ return in->toProto(s);})));
    }
};

template <typename EntryOut, typename ValIn>
struct ProtoMapValConverter<EntryOut, typename ValIn::ProtoType, ValIn> {
    static inline void convert(EntryOut* out, const ValIn& in, ReferenceSerializer* s) {
        in.toProto(out->mutable_value(), s);
    }
};

// std::unordered_map<KeyIn, ValIn> -> ::google::protobuf::RepeatedPtrField<EntryOut>*
template <typename EntryOut, typename KeyIn, typename ValIn>
struct ProtoConverter<::google::protobuf::RepeatedPtrField<EntryOut>*, std::unordered_map<KeyIn, ValIn>> {
    typedef decltype(((EntryOut*)(0))->key()) KeyOutRaw;
    typedef decltype(((EntryOut*)(0))->value()) ValOutRaw;

    typedef typename std::remove_cv<typename std::remove_reference<KeyOutRaw>::type>::type KeyOut;
    typedef typename std::remove_cv<typename std::remove_reference<ValOutRaw>::type>::type ValOut;

    static inline void convert(::google::protobuf::RepeatedPtrField<EntryOut>* out, const std::unordered_map<KeyIn, ValIn>& in, ReferenceSerializer* s) {
        out->Reserve(in.size());
        for (auto it : in) {
            auto entry = out->Add();
            ProtoMapKeyConverter<EntryOut, KeyOut, KeyIn>::convert(entry, it.first, s);
            ProtoMapValConverter<EntryOut, ValOut, ValIn>::convert(entry, it.second, s);
        }
    }
};

template <typename Out, typename In>
inline void toProto(Out out, const In& in, ReferenceSerializer* s) {
    ProtoConverter<Out, In>::convert(out, in, s);
}

template <typename T>
inline void toProtoSlice(memory_pb::Slice* out, const gapii::Slice<T>& in, ReferenceSerializer* s) {
    toProto(out, in, s);
}

template <typename T>
inline void toProtoPointer(memory_pb::Pointer* out, const T* in, ReferenceSerializer* s) {
    toProto(out, in, s);
}

inline const std::string& toProtoString(const std::string& str) {
    return str;
}

inline const char* toProtoString(const char* str) {
    return (str != nullptr) ? str : "";
}

}  // namespace gapii

#endif // GAPII_TO_PROTO_H

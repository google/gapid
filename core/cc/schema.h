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

#ifndef CORE_SCHEMA_H
#define CORE_SCHEMA_H

#include <initializer_list>
#include <memory>
#include <string>

#include "encoder.h"
#include "log.h"

namespace core {
namespace schema {

class Type {
 public:
  enum TypeTag {
    PrimitiveTag,
    StructTag,
    PointerTag,
    InterfaceTag,
    VariantTag,
    AnyTag,
    SliceTag,
    ArrayTag,
    MapTag,
  };

  virtual void encode(Encoder& e) const = 0;
};

class Field {
 public:
  Field(const std::string& declared, Type* type)
      : mDeclared(declared), mType(type) {}

  void encode(Encoder& e) const {
    mType->encode(e);
  }

  const char* name() const { return mDeclared.c_str(); }
private:
  std::string mDeclared;
  Type* mType;
};

class Entity {
 public:
  Entity() = default;

  Entity(const std::string& package,
         const std::string& name,
         const std::string& identity,
         const std::string& version,
         std::initializer_list<Field> fields)
      :
      mPackage(package),
      mName(name),
      mIdentity(identity),
      mVersion(version),
      mFields(fields) {}

  void encode(Encoder& e) const {
    e.String(mPackage);
    e.String(mIdentity);
    e.String(mVersion);
    e.Uint32(uint32_t(mFields.size()));
    for (const auto& f : mFields) {
      f.encode(e);
    }
  }
private:
  std::string mPackage;
  std::string mName;
  std::string mIdentity;
  std::string mVersion;
  std::vector<Field> mFields;
};

class Primitive : public Type {
 public:
  enum Method {
    Bool,
    Int8,
    Uint8,
    Int16,
    Uint16,
    Int32,
    Uint32,
    Int64,
    Uint64,
    Float32,
    Float64,
    String,
  };

  Method method() const { return mMethod; }

  Primitive(const std::string& name, Method method) : mName(name), mMethod(method) {}

  void encode(Encoder& e) const {
    e.Uint8(uint8_t(Type::PrimitiveTag) | (uint8_t(mMethod) << 4));
  }
 private:
  std::string mName;
  Method mMethod;
};

class Struct : public Type {
 public:
  Struct(const Entity* entity)
      : mEntity(entity) {}

  void encode(Encoder& e) const {
    e.Uint8(uint8_t(StructTag));
    e.Entity(mEntity);
  }
 private:
  const Entity* mEntity;
};

class Pointer : public Type {
 public:
  explicit Pointer(Type* type) : mType(type) {}

  void encode(Encoder& e) const {
    e.Uint8(uint8_t(PointerTag));
    mType->encode(e);
  }
 private:
  Type* mType;
};

class Interface : public Type {
 public:
  explicit Interface(const std::string& name) : mName(name) {}

  void encode(Encoder& e) const {
    e.Uint8(uint8_t(InterfaceTag));
  }
 private:
  std::string mName;
};

class Variant : public Type {
 public:
  explicit Variant(const std::string& name) : mName(name) {}

  void encode(Encoder& e) const {
    e.Uint8(uint8_t(VariantTag));
  }
 private:
  std::string mName;
};

class Any : public Type {
 public:
  Any() = default;

  void encode(Encoder& e) const {
    e.Uint8(uint8_t(AnyTag));
  }
};

class Slice : public Type {
 public:
  Slice(const std::string& alias, Type* valueType)
      : mAlias(alias), mValueType(valueType) {}
  void encode(Encoder& e) const {
    e.Uint8(uint8_t(SliceTag));
    mValueType->encode(e);
  }
 private:
  std::string mAlias;
  Type* mValueType;
};

class Array : public Type {
 public:
  Array(const std::string& alias, Type* valueType, uint32_t size)
      : mAlias(alias), mValueType(valueType), mSize(size) {}
  void encode(Encoder& e) const {
    e.Uint8(uint8_t(ArrayTag));
    e.Uint32(mSize);
    mValueType->encode(e);
  }
 private:
  std::string mAlias;
  Type* mValueType;
  uint32_t mSize;
};

class Map : public Type {
 public:
  Map(const std::string& alias, Type* keyType, Type* valueType)
      : mAlias(alias),
        mKeyType(keyType),
        mValueType(valueType) {}
  void encode(Encoder& e) const {
      e.Uint8(uint8_t(MapTag));
      mKeyType->encode(e);
      mValueType->encode(e);
  }
 private:
  std::string mAlias;
  Type* mKeyType;
  Type* mValueType;
};

}  // namespace schema
}  // namespace core
#endif

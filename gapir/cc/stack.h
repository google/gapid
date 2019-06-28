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

#ifndef GAPIR_STACK_H
#define GAPIR_STACK_H

#include <stdint.h>
#include <string.h>

#include <type_traits>
#include <vector>

#include "base_type.h"
#include "core/cc/log.h"
#include "core/cc/static_array.h"
#include "memory_manager.h"

namespace gapir {

// Strongly typed, limited size stack for the stack based virtual machine. If an
// invalid operation is called on the stack then the stack will go into an
// invalid state where each operation is no op (except print stack). From an
// invalid state the stack can't go back to a valid state again.
class Stack {
 public:
  // Representation of an unconverted value from the stack.
  typedef uint64_t BaseValue;

  // Construct a new stack with the given size and memory manager
  // The memory manager is needed to resolve constant and volatile pointers to
  // absolute pointers
  Stack(uint32_t size, const MemoryManager* memoryManager);

  // Pop the element from the top of the stack as the given type if its type is
  // matching with the given type. For pointer types convert the pointer to an
  // absolute pointer before returning it. Puts the stack into an invalid state
  // if called on an empty stack or if the requested type doesn't match with the
  // type of the value at the top of the stack.
  template <typename T>
  T pop() {
    if (!popCheck("pop")) {
      return T();
    }
    GAPID_VERBOSE("-%s pop()", mStack[mTop - 1].debugInfo(mMemoryManager));
    return PopImpl<T>::pop(this);
  }

  // Pop the static array (stored as a pointer) from the top of the stack.
  template <typename T, int N>
  core::StaticArray<T, N> pop() {
    T* ptr = pop<T*>();
    core::StaticArray<T, N> out;
    for (int i = 0; i < N; i++) {
      out[i] = ptr[i];
    }
    return out;
  }

  // Pop the volatile pointer from the top of the stack. If the top element
  // is not a volatile pointer puts the stack into and invalid state. Also,
  // puts the stack into an invalid state if called on an empty stack.
  // Use with care, this template casts a pointer to a T* with no way of
  // knowing if that is safe.
  template <typename T>
  T* popVolatile() {
    if (!popCheck("popVolatile")) {
      return nullptr;
    }
    BaseType type = getTopType();
    if (type != BaseType::VolatilePointer) {
      GAPID_WARNING("popVolatile called for type %s", baseTypeName(type));
      mValid = false;
      return nullptr;
    }
    return pop<T*>();
  }

  // Pop the constant pointer from the top of the stack. If the top element
  // is not a constant pointer puts the stack into and invalid state. Also,
  // puts the stack into an invalid state if called on an empty stack.
  // Use with care, this template casts a pointer to a const T* with no way of
  // knowing if that is safe.
  template <typename T>
  const T* popConstant() {
    if (!popCheck("popConstant")) {
      return nullptr;
    }
    BaseType type = getTopType();
    if (type != BaseType::ConstantPointer) {
      GAPID_WARNING("popConstant called for type %s", baseTypeName(type));
      mValid = false;
      return nullptr;
    }
    return pop<const T*>();
  }

  // Pop the element from the top of the stack and return its unconverted typed
  // value. Puts the stack into an invalid state if called on an empty stack.
  BaseValue popBaseValue() {
    if (!popCheck("popBaseValue")) {
      return 0;
    }
    mTop--;
    return mStack[mTop].getBaseValue();
  }

  // Push the given type and base value to the top of the stack. Put the
  // stack into invalid state if called on a full stack.
  void pushValue(BaseType type, BaseValue value) {
    if (!pushCheck("pushValue")) {
      return;
    }
    pushFrom(type, &value);
    checkTopForInvalidPointer("pushValue");
  }

  // Push the given value to the top of the stack with a type determined by the
  // type of the value given. Put the stack into invalid state if called on a
  // full stack.
  template <typename T>
  void push(T value) {
    if (!pushCheck("push")) {
      return;
    }

    mStack[mTop].set(value);
    if (!checkTopForInvalidPointer("push")) {
      return;
    }
    GAPID_VERBOSE("+%s push()", mStack[mTop].debugInfo(mMemoryManager));
    mTop++;
  }

  // Returns the type of the element at the top of the stack. Put the stack into
  // invalid state if called on an empty stack.
  BaseType getTopType();

  // Discards count amount of elements from the top of the stack. Put the stack
  // into invalid state if the stack contains fewer element then the given
  // count.
  void discard(uint32_t count);

  // Clone the n-th element from the top of the stack to the top of the stack.
  // The currently top most element is the 0th and the index is increasing with
  // going down in the stack. Put the stack into invalid state if called on a
  // full stacked or if the given index points out from the stack (greater then
  // current size minus one).
  void clone(uint32_t n);

  // Print the content of the stack to the log output. The output is only
  // written to the log output if the debug level is at least DEBUG. This
  // function will work even if the stack is not in a valid state.
  void printStack() const;

  // Returns if the stack is in a valid state or not.
  bool isValid() const { return mValid; }

  // Pop the item from the top of the stack to the given memory address. The
  // number of bytes written to the address is determined by the type of the
  // element at the top of the stack. Pointers are converted to absolute
  // pointers before writing to the address. The stack will enter an invalid
  // state if called on an empty stack. Take care if using an address on the
  // program stack. Use getTopType() to check the type of the object you will
  // pop and/or make sure the receiver is sizeof(BaseValue).
  void popTo(void* address);

  // Push a new item to the stack with the given type from the given memory
  // address. Put the stack into invalid state if it is already full before the
  // call.
  void pushFrom(BaseType type, const void* data);

  // Returns true if the stack is empty false otherwise.
  bool isEmpty() const { return mTop == 0; }

 private:
  // Check that the stack is valid and a pop is allowed (non-empty).
  bool popCheck(const char* what);
  // Check that the stack is valid and a push is allowed (non-full).
  bool pushCheck(const char* what);
  // Check that if top is a pointer type that it is valid.
  bool checkTopForInvalidPointer(const char* what);
  // Check that top is a pointer type, that it is valid and return it.
  const void* checkAndGetTopPointer(const char* what);

  // Implementation of the pop method for non pointer types.
  template <typename T>
  struct PopImpl {
    static T pop(Stack* stack) {
      stack->mTop--;
      BaseType baseType =
          TypeToBaseType<typename std::remove_cv<T>::type>::type;
      if (stack->mStack[stack->mTop].type() != baseType) {
        stack->mValid = false;
        GAPID_WARNING(
            "Pop type (%s) doesn't match with the type at the top of the stack "
            "(%s)",
            baseTypeName(baseType),
            baseTypeName(stack->mStack[stack->mTop].type()));
        return T();
      }
      return stack->mStack[stack->mTop].value<T>();
    }
  };

  // Implementation of the pop method for pointer types.
  // Use with care, this template casts a pointer to a T* with no
  // way of knowing if that is safe.
  template <typename T>
  struct PopImpl<T*> {
    static T* pop(Stack* stack) {
      stack->mTop--;
      const void* pointer = stack->checkAndGetTopPointer("pop");
      return const_cast<T*>(static_cast<const T*>(pointer));
    }
  };

  class Entry {
   public:
    const void* valuePtr() const { return &mValue; }

    template <typename T>
    const T value() const {
      static_assert(sizeof(mValue) >= sizeof(T),
                    "T is too large to be used as value");
      T t;
      if (!getTo(&t)) {
        GAPID_WARNING("Error: read stack value inappropriate type %s wanted %s",
                      baseTypeName(mType),
                      baseTypeName(TypeToBaseType<T>::type));
        return T();
      }
      return t;
    }

    const BaseType& type() const { return mType; }

    void set(bool b) {
      mType = TypeToBaseType<bool>::type;
      mValue.b = b;
    }
    void set(int8_t i8) {
      mType = TypeToBaseType<int8_t>::type;
      mValue.i8 = i8;
    }
    void set(int16_t i16) {
      mType = TypeToBaseType<int16_t>::type;
      mValue.i16 = i16;
    }
    void set(int32_t i32) {
      mType = TypeToBaseType<int32_t>::type;
      mValue.i32 = i32;
    }
    void set(int64_t i64) {
      mType = TypeToBaseType<int64_t>::type;
      mValue.i64 = i64;
    }
    void set(uint8_t u8) {
      mType = TypeToBaseType<uint8_t>::type;
      mValue.u8 = u8;
    }
    void set(uint16_t u16) {
      mType = TypeToBaseType<uint16_t>::type;
      mValue.u16 = u16;
    }
    void set(uint32_t u32) {
      mType = TypeToBaseType<uint32_t>::type;
      mValue.u32 = u32;
    }
    void set(uint64_t u64) {
      mType = TypeToBaseType<uint64_t>::type;
      mValue.u64 = u64;
    }
    void set(float f) {
      mType = TypeToBaseType<float>::type;
      mValue.f = f;
    }
    void set(double d) {
      mType = TypeToBaseType<double>::type;
      mValue.d = d;
    }

    template <typename T>
    void set(T* p) {
      mType = BaseType::AbsolutePointer;
      mValue.p = const_cast<typename std::remove_const<T>::type*>(p);
    }

    template <typename T>
    void set(T val) {
      static_assert(sizeof(T) == sizeof(mValue.u32),
                    "Enum base type is not uint32_t");
      mType = BaseType::Uint32;
      mValue.u32 = uint32_t(val);
    }

    void set(BaseType type, const void* data) {
      // Little endian assumption
      memcpy(&mValue, data, baseTypeSize(type));
      mType = type;
    }

    bool getTo(bool* b) const {
      if (mType != TypeToBaseType<bool>::type) {
        return false;
      }
      *b = mValue.b;
      return true;
    }
    bool getTo(int8_t* i8) const {
      if (mType != TypeToBaseType<int8_t>::type) {
        return false;
      }
      *i8 = mValue.i8;
      return true;
    }
    bool getTo(int16_t* i16) const {
      if (mType != TypeToBaseType<int16_t>::type) {
        return false;
      }
      *i16 = mValue.i16;
      return true;
    }
    bool getTo(int32_t* i32) const {
      if (mType != TypeToBaseType<int32_t>::type) {
        return false;
      }
      *i32 = mValue.i32;
      return true;
    }
    bool getTo(int64_t* i64) const {
      if (mType != TypeToBaseType<int64_t>::type) {
        return false;
      }
      *i64 = mValue.i64;
      return true;
    }
    bool getTo(uint8_t* u8) const {
      if (mType != TypeToBaseType<uint8_t>::type) {
        return false;
      }
      *u8 = mValue.u8;
      return true;
    }
    bool getTo(uint16_t* u16) const {
      if (mType != TypeToBaseType<uint16_t>::type) {
        return false;
      }
      *u16 = mValue.u16;
      return true;
    }
    bool getTo(uint32_t* u32) const {
      if (mType != TypeToBaseType<uint32_t>::type &&
          mType != BaseType::VolatilePointer &&
          mType != BaseType::ConstantPointer) {
        return false;
      }
      *u32 = mValue.u32;
      return true;
    }
    bool getTo(uint64_t* u64) const {
      if (mType != TypeToBaseType<uint64_t>::type) {
        return false;
      }
      *u64 = mValue.u64;
      return true;
    }
    bool getTo(float* f) const {
      if (mType != TypeToBaseType<float>::type) {
        return false;
      }
      *f = mValue.f;
      return true;
    }
    bool getTo(double* d) const {
      if (mType != TypeToBaseType<double>::type) {
        return false;
      }
      *d = mValue.d;
      return true;
    }

    template <typename T>
    bool getTo(T* val) const {
      static_assert(sizeof(T) == sizeof(mValue.u32),
                    "Enum base type is not uint32_t");
      if (mType != BaseType::Uint32) {
        return false;
      }
      *val = T(mValue.u32);
      return true;
    }

    bool getTo(const void** p) const {
      if (mType != BaseType::AbsolutePointer) {
        return false;
      }
      *p = mValue.p;
      return true;
    }

    bool getTo(void** p) const {
      if (mType != BaseType::AbsolutePointer) {
        return false;
      }
      *p = mValue.p;
      return true;
    }

    BaseValue getBaseValue() const { return mValue.bv; }

    // Return a string describing the stack entry.
    // The pointer returned is only valid until the next call to debugInfo,
    // regardless of the Entry instance. This function is not thread safe.
    const char* debugInfo(const MemoryManager* memoryManager) const;

   private:
    // Type of the element stored by this entry
    BaseType mType;

    // Union of all possible types stored on the stack for creating a unified
    // value type with getter function to access the value as a specific type
    union ValueType {
      bool b;
      int8_t i8;
      int16_t i16;
      int32_t i32;
      int64_t i64;
      uint8_t u8;
      uint16_t u16;
      uint32_t u32;
      uint64_t u64;
      float f;
      double d;
      void* p;
      BaseValue bv;
    };

    ValueType mValue;

    static_assert(sizeof(BaseValue) >= sizeof(ValueType),
                  "Stack::BaseValue is not large enough");
  };

  // Indicates if the stack is in a consistent state (true value) or not (false
  // value). The stack go into an inconsistent state after an invalid operation.
  // When mValid is false then all of the operation on the stack (expect
  // printing the stack) produce a warning message and falls back to no op (with
  // zero initialized return value where necessary). The stack can't go back
  // from an invalid state to a valid state again.
  bool mValid;

  // Contains the offset of the first empty slot in the stack from the bottom of
  // the stack. The value of it indicates the number of elements currently in
  // the stack.
  uint32_t mTop;

  // mStack stores the entries currently in the stack. The 0th element
  // corresponds to the bottom of the stack.
  std::vector<Entry> mStack;

  // Reference to the memory manager used to resolve constant and volatile
  // pointers to absolute pointers when thy are popped from the stack.
  const MemoryManager* mMemoryManager;
};

}  // namespace gapir

#endif  // GAPIR_STACK_H

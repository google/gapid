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

#include "stack.h"

#include "core/cc/log.h"

#define __STDC_FORMAT_MACROS
#include <inttypes.h>

#define DEBUG_STACK(...)

namespace gapir {

bool Stack::pushCheck(const char* what) {
  if (!mValid) {
    GAPID_WARNING("%s on invalid stack", what);
    return false;
  }

  if (mTop > mStack.size() - 1) {
    mValid = false;
    GAPID_WARNING("%s with invalid stack head, offset: %d", what, mTop);
    return false;
  }
  return true;
}

bool Stack::popCheck(const char* what) {
  if (!mValid) {
    GAPID_WARNING("%s on invalid stack", what);
    return false;
  }

  if (mTop == 0 || mTop > mStack.size()) {
    mValid = false;
    GAPID_WARNING("%s with invalid stack head, offset: %d", what, mTop);
    return false;
  }
  return true;
}

const char* Stack::Entry::debugInfo(const MemoryManager* memoryManager) const {
  static const size_t size = 256;
  static char buf[size];
  switch (mType) {
    case BaseType::Bool:
      snprintf(buf, size, "bool<%d>", value<bool>());
      break;
    case BaseType::Int8:
      snprintf(buf, size, "int8<%" PRId8 ">", value<int8_t>());
      break;
    case BaseType::Int16:
      snprintf(buf, size, "int16<%" PRId16 ">", value<int16_t>());
      break;
    case BaseType::Int32:
      snprintf(buf, size, "int32<%" PRId32 ">", value<int32_t>());
      break;
    case BaseType::Int64:
      snprintf(buf, size, "int64<%" PRId64 ">", value<int64_t>());
      break;
    case BaseType::Uint8:
      snprintf(buf, size, "uint8<%" PRIu8 ">", value<uint8_t>());
      break;
    case BaseType::Uint16:
      snprintf(buf, size, "uint16<%" PRIu16 ">", value<uint16_t>());
      break;
    case BaseType::Uint32:
      snprintf(buf, size, "uint32<%" PRIu32 ">", value<uint32_t>());
      break;
    case BaseType::Uint64:
      snprintf(buf, size, "uint64<%" PRIu64 ">", value<uint64_t>());
      break;
    case BaseType::Float:
      snprintf(buf, size, "float<%f>", value<float>());
      break;
    case BaseType::Double:
      snprintf(buf, size, "double<%f>", value<double>());
      break;
    case BaseType::AbsolutePointer: {
      const void* pointer = value<void*>();
      if (memoryManager->isNotObservedAbsoluteAddress(pointer)) {
        snprintf(buf, size, "absolute-ptr<%p> SPECIAL", value<void*>());
      } else {
        snprintf(buf, size, "absolute-ptr<%p> valid", value<void*>());
      }
      break;
    }
    case BaseType::ConstantPointer: {
      uint32_t offset = value<uint32_t>();
      const void* pointer = memoryManager->constantToAbsolute(offset);
      if (memoryManager->isConstantAddress(pointer)) {
        snprintf(buf, size, "constant-ptr<0x%x> valid (%p)", offset, pointer);
      } else {
        snprintf(buf, size, "constant-ptr<0x%x> INVALID (%p)", offset, pointer);
      }
      break;
    }
    case BaseType::VolatilePointer: {
      uint32_t offset = value<uint32_t>();
      const void* pointer = memoryManager->volatileToAbsolute(offset);
      if (memoryManager->isVolatileAddress(pointer)) {
        snprintf(buf, size, "volatile-ptr<0x%x> valid (%p)", offset, pointer);
      } else {
        snprintf(buf, size, "volatile-ptr<0x%x> INVALID (%p)", offset, pointer);
      }
      break;
    }
    default:
      snprintf(buf, size, "unknown type<%d>", int(mType));
      break;
  }
  return buf;
}

Stack::Stack(uint32_t size, const MemoryManager* memoryManager)
    : mValid(true), mTop(0), mStack(size), mMemoryManager(memoryManager) {}

void Stack::printStack() const {
  GAPID_DEBUG("Stack size: %u", mTop);
  for (uint32_t i = 0; i < mTop; ++i) {
    GAPID_DEBUG("(%d) %s", i, mStack[i].debugInfo(mMemoryManager));
  }
}

BaseType Stack::getTopType() {
  if (!mValid) {
    GAPID_WARNING("GetTopType on invalid stack");
    return BaseType::Bool;
  }

  if (mTop == 0 || mTop > mStack.size()) {
    mValid = false;
    GAPID_WARNING("GetTopType with invalid stack head: %u (size: %" PRIsize ")",
                  mTop, mStack.size());
    return BaseType::Bool;
  }

  return mStack[mTop - 1].type();
}

void Stack::pushFrom(BaseType type, const void* data) {
  if (!pushCheck("pushFrom")) {
    return;
  }

  if (data == nullptr) {
    GAPID_WARNING("pushFrom nullptr");
    mValid = false;
    return;
  }

  mStack[mTop].set(type, data);
  DEBUG_STACK("+%s pushFrom(%p)\n", mStack[mTop].debugInfo(mMemoryManager),
              data);
  mTop++;
}

void Stack::popTo(void* address) {
  if (!popCheck("popTo")) {
    return;
  }

  switch (getTopType()) {
    case BaseType::ConstantPointer:
    case BaseType::VolatilePointer: {
      void* pointer = pop<void*>();
      // Note we are copying the pointer not what is pointed to.
      memcpy(address, &pointer, sizeof(pointer));
      return;
    }
    default:
      mTop--;
      DEBUG_STACK("-%s popTo(%p)\n", mStack[mTop].debugInfo(mMemoryManager),
                  address);
      memcpy(address, mStack[mTop].valuePtr(),
             baseTypeSize(mStack[mTop].type()));
      return;
  }
}

void Stack::discard(uint32_t count) {
  if (!mValid) {
    GAPID_WARNING("Discard on invalid stack");
    return;
  }

  if (count > mTop) {
    mValid = false;
    GAPID_WARNING("Discarding more element (%u) then in the stack (%u)", count,
                  mTop);
    return;
  }

  for (uint32_t i = 0; i < count; i++) {
    DEBUG_STACK("-%s discard()\n",
                mStack[mTop - i - 1].debugInfo(mMemoryManager));
  }

  mTop -= count;
}

void Stack::clone(uint32_t n) {
  if (!mValid) {
    GAPID_WARNING("Clone on invalid stack");
    return;
  }

  if (mTop >= mStack.size()) {
    mValid = false;
    GAPID_WARNING("Cloning to full stack");
    return;
  }

  if (mTop < n + 1) {
    mValid = false;
    GAPID_WARNING("Cloning from invalid index: %u (head: %u)", n, mTop);
    return;
  }

  mStack[mTop] = mStack[mTop - n - 1];
  DEBUG_STACK("+%s clone(%d)\n", mStack[mTop].debugInfo(mMemoryManager), n);
  mTop++;
}

const void* Stack::checkAndGetTopPointer(const char* what) {
  auto type = mStack[mTop].type();
  switch (type) {
    case BaseType::AbsolutePointer: {
      return mStack[mTop].value<const void*>();
    }
    case BaseType::ConstantPointer: {
      uint32_t offset = mStack[mTop].value<uint32_t>();
      const void* pointer = mMemoryManager->constantToAbsolute(offset);
      if (!mMemoryManager->isConstantAddress(pointer)) {
        GAPID_WARNING("%s: Invalid constant address %p offset 0x%x", what,
                      pointer, offset);
        mValid = false;
        return nullptr;
      }
      return pointer;
    }
    case BaseType::VolatilePointer: {
      uint32_t offset = mStack[mTop].value<uint32_t>();
      void* pointer = mMemoryManager->volatileToAbsolute(offset);
      if (!mMemoryManager->isVolatileAddress(pointer)) {
        GAPID_WARNING("%s Invalid volatile address %p offset 0x%x", what,
                      pointer, offset);
        mValid = false;
        return nullptr;
      }
      return pointer;
    }
    default:
      GAPID_WARNING("%s top was not a pointer type: %s", what,
                    baseTypeName(type));
      mValid = false;
      return nullptr;
  }
  return nullptr;
}

bool Stack::checkTopForInvalidPointer(const char* what) {
  auto type = mStack[mTop].type();
  switch (type) {
    case BaseType::ConstantPointer:
    case BaseType::VolatilePointer: {
      checkAndGetTopPointer(what);
      return isValid();
    }
    default:
      return true;
  }
}

}  // namespace gapir

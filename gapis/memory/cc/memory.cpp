// Copyright (C) 2018 Google Inc.
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

#include "memory.h"

#include "core/cc/assert.h"
#include "core/cc/interval_list.h"
#include "core/memory/arena/cc/arena.h"

struct Data {
  typedef uint64_t interval_unit_type;

  // Interval compilance
  inline uint64_t start() const { return data_start; }
  inline uint64_t end() const { return data_end; }
  inline void adjust(uint64_t start, uint64_t end) {
    data_start = start;
    data_end = end;
  }
  inline uint64_t data_size() const { return data_end - data_start; }

  void* get() const;
  void get(void* out, uint64_t size) const;

  enum class Kind {
    BYTES,
    RESOURCE,
  };

  uint64_t pool_start;
  uint64_t pool_end;
  uint64_t data_start;
  uint64_t data_end;
  void* data;
  Kind kind;
};

void* Data::get() const {
  switch (kind) {
    case Kind::BYTES:
      return data;
    case Kind::RESOURCE:
      // TODO
      return nullptr;
    default:
      GAPID_ASSERT_MSG(false, "Unknown data kind");
      return nullptr;
  }
}

class Pool {
 public:
  void* read(core::Arena* arena, size_t base, size_t size);
  void write(core::Arena* arena, size_t base, size_t size, const void* data);
  void copy(core::Arena* arena, Pool* src_pool, size_t dst_base,
            size_t src_base, size_t size);

 private:
  core::CustomIntervalList<Data> writes_;
};

void* Pool::read(core::Arena* arena, size_t base, size_t size) {
  auto intervals = writes_.intersect(base, base + size);
  if (intervals.size() == 1) {
    auto data = intervals.begin();
    if (data->data_start == base && data->data_size() == size) {
      return data->get();
    }
  }
  uint8_t* out = reinterpret_cast<uint8_t*>(arena->allocate(size, 8));
  memset(out, 0, size);
  for (auto& data : intervals) {
    auto start = std::max(base, data.data_start);
    auto end = std::min(base + size, data.data_end);
    auto size = end - start;
    auto offset = start - base;
    data.get(out + offset, size);
  }
  return out;
}

void Pool::write(core::Arena* arena, size_t base, size_t size,
                 const void* data) {
  auto start = base;
  auto end = base + size;
  auto alloc = arena->allocate(size, 8);
  memcpy(alloc, data, size);
  writes_.merge(Data{
      .pool_start = start,
      .pool_end = end,
      .data_start = start,
      .data_end = end,
      .data = alloc,
      .kind = Data::Kind::BYTES,
  });
}

void Pool::copy(core::Arena* arena, Pool* src_pool, size_t dst_base,
                size_t src_base, size_t size) {
  auto intervals = src_pool->writes_.intersect(src_base, src_base + size);
  auto start = src_base;
  auto end = src_base + size;
  for (auto data : intervals) {
    data.data_start = std::max(data.data_start, start);
    data.data_end = std::min(data.data_end, end);
    writes_.replace(data);
  }
}

class Memory {
 public:
  Memory(core::Arena*);

  void* read(gapil_gapil_slice* sli);
  void write(gapil_gapil_slice* sli, const void* data);
  void copy(gapil_gapil_slice* dst, gapil_gapil_slice* src);

 private:
  Pool* get_pool(uint64_t id);

  core::Arena* arena_;
  std::unordered_map<uint64_t, Pool*> pools_;
};

Memory::Memory(core::Arena* a) : arena_(a) {}

void* Memory::read(gapil_slice* sli) {
  auto pool = get_pool(sli->pool);
  return pool->read(arena_, sli->base, sli->size);
}

void Memory::write(gapil_slice* sli, const void* data) {
  auto pool = get_pool(sli->pool);
  return pool->write(arena_, sli->base, sli->size, data);
}

void Memory::copy(gapil_slice* dst, gapil_slice* src) {
  auto d = get_pool(dst->pool);
  auto s = get_pool(src->pool);
  auto size = std::min(dst->size, src->size);
  d->copy(arena_, s, dst->base, src->base, size);
}

Pool* Memory::get_pool(uint64_t id) {
  auto it = pools_.find(id);
  GAPID_ASSERT_MSG(it != pools_.end(), "Pool %d does not exist", int(id));
  return it->second;
}

extern "C" {

memory* memory_create(arena* a) {
  auto arena = reinterpret_cast<core::Arena*>(a);
  return reinterpret_cast<memory*>(new Memory(arena));
}

void memory_destroy(memory* mem) {
  auto m = reinterpret_cast<Memory*>(mem);
  delete m;
}

void* memory_read(memory* mem, gapil_slice* sli) {
  auto m = reinterpret_cast<Memory*>(mem);
  return m->read(sli);
}

void memory_write(memory* mem, gapil_slice* sli, const void* data) {
  auto m = reinterpret_cast<Memory*>(mem);
  return m->write(sli, data);
}

void memory_copy(memory* mem, gapil_slice* dst, gapil_slice* src) {
  auto m = reinterpret_cast<Memory*>(mem);
  return m->copy(dst, src);
}

}  // extern "C"
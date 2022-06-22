/*
 * Copyright (C) 2022 Google Inc.
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

#pragma once
#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>
#ifdef _WIN32
#include <windows.h>
#endif
#include <functional>
#include <map>
#include <mutex>
#include <set>
#include <unordered_map>
#include <unordered_set>

#include "common.h"

namespace gapid2 {
class memory_tracker;
memory_tracker*& static_tracker();

struct range_data {
  range_data() = default;
  range_data(const range_data&) = default;
  char* src_ptr;
  char* dst_ptr;
  VkDeviceSize mapped_size;
  VkDeviceMemory mem;
  bool fast = false;
};
class memory_tracker {
 public:
  static LONG handler(_EXCEPTION_POINTERS* ExceptionInfo);

  memory_tracker() {
    static_tracker() = this;
    AddVectoredExceptionHandler(1, &handler);
  }

  bool handle_exception(char* ptr, bool read) {
    std::unique_lock<std::mutex> l(mut);
    if (!ptr || ranges.empty()) {
      return false;
    }
    auto gr = ranges.upper_bound(ptr);
    --gr;
    char* base_addr =
        reinterpret_cast<char*>(reinterpret_cast<uintptr_t>(ptr) & ~4095);

    if (gr == ranges.end() ||
        gr->second.dst_ptr + gr->second.mapped_size <= ptr) {
      return false;
    }
    auto& rng = gr->second;
    if (read) {
      DWORD old_protect = 0;
      VirtualProtect(base_addr, 4096, PAGE_READWRITE, &old_protect);
      GAPID2_ASSERT(PAGE_WRITECOPY == old_protect,
                    "Unhandled : memory with both read and write in one page");
      const uintptr_t offs = base_addr - rng.dst_ptr;
      memcpy(base_addr, rng.src_ptr + offs, 4096);
    } else {
      auto pg_rng = base_addr;
      dirty_read_pages.insert(pg_rng);
      DWORD old_protect = 0;
      VirtualProtect(base_addr, 4096, PAGE_READWRITE, &old_protect);
      // Copy from the GPU range to the Host range before writes can take place
      // in case a GPU write has happend in the mean-time.
      if (!rng.fast) {
        const uintptr_t offs = base_addr - rng.dst_ptr;
        memcpy(base_addr, rng.src_ptr + offs, 4096);
        GAPID2_ASSERT(
            PAGE_READONLY == old_protect || PAGE_READWRITE == old_protect,
            "Unhandled : memory with both read and write in one page");
      }
    }
    return true;
  }

  void* AddTrackedRange(VkDeviceMemory mem,
                        void* mapped_loc,
                        VkDeviceSize mapped_offset,
                        VkDeviceSize mapped_size, void* fast_host = nullptr) {
    std::unique_lock<std::mutex> l(mut);
    // Don't use write-watch for this.
    mapped_size = (mapped_size + 4095) & ~4095;
    void* ptr = fast_host ? fast_host : VirtualAlloc(nullptr, mapped_size, MEM_COMMIT | MEM_RESERVE, PAGE_READWRITE);
    if (!fast_host) {
      memcpy(ptr, mapped_loc, mapped_size);
    }

    uintptr_t mapped_pages = mapped_size >> 12;
    DWORD dp = 0;
    auto prot = VirtualProtect(ptr, mapped_size, PAGE_READONLY, &dp);
    GAPID2_ASSERT(prot, "VirtualProtect failed");
    auto it = ranges.insert(
        std::make_pair(reinterpret_cast<char*>(ptr), range_data()));
    it.first->second.src_ptr = reinterpret_cast<char*>(mapped_loc);
    it.first->second.dst_ptr = reinterpret_cast<char*>(ptr);
    it.first->second.mapped_size = mapped_size;
    it.first->second.mem = mem;
    it.first->second.fast = fast_host;
    total_pages += mapped_pages;
    src_ranges[mem] = &it.first->second;
    return ptr;
  }

  // Helpfully DeviceMemory can only be mapped a single time.
  void RemoveTrackedRange(VkDeviceMemory mem) {
    std::unique_lock<std::mutex> l(mut);
    if (!src_ranges.count(mem)) {
      return;
    }
    auto sr = src_ranges[mem];
    if (!sr->fast) {
      VirtualProtect(sr->dst_ptr, sr->mapped_size, PAGE_READWRITE, nullptr);
      memcpy(sr->src_ptr, sr->dst_ptr, sr->mapped_size);
    }

    ranges.erase(sr->dst_ptr);
    src_ranges.erase(mem);
  }

  void for_dirty_in_mem(VkDeviceMemory mem,
                        std::function<void(void*, VkDeviceSize)> fn) {
    std::unique_lock<std::mutex> l(mut);
    if (src_ranges.empty()) {
      return;
    }
    auto& rd = src_ranges[mem];
    auto end_ptr = (rd->dst_ptr + rd->mapped_size);
    if (dirty_read_pages.empty()) {
      return;
    }
    auto gp = dirty_read_pages.upper_bound(rd->dst_ptr);
    if (gp != dirty_read_pages.begin()) {
      gp--;
    }

    if (gp == dirty_read_pages.end()) {
      gp = dirty_read_pages.begin();
    }

    // This means we got a read for something BEFORE our memory allocation.
    if (*gp < rd->dst_ptr) {
      return;
    }
    auto start_page = *gp;
    auto end_page = static_cast<char*>(nullptr);
    auto process = [&start_page, &end_page, rd, &fn]() {
      if (!end_page) {
        return;
      }
      auto len = (end_page - start_page) + 4096;
      uintptr_t offs = (start_page - rd->dst_ptr);
      if (!rd->fast) {
        memcpy(rd->src_ptr + offs, start_page, len);
      }
      DWORD old_protect;
      VirtualProtect(start_page, len, PAGE_READONLY, &old_protect);
      GAPID2_ASSERT(old_protect == PAGE_READWRITE, "Unexpected memory flags");
      fn(rd->src_ptr + offs, len);
    };
    while (gp != dirty_read_pages.end()) {
      if (*gp > end_ptr) {
        break;
      }
      if (end_page == nullptr) {
        end_page = *gp;
      } else {
        if (*gp != end_page + 4096) {
          process();
          end_page = nullptr;
          start_page = *gp;
          continue;
        }
      }
      end_page = *gp;
      gp = dirty_read_pages.erase(gp);
    }
    process();
  }

  void AddGPUWrite(VkDeviceMemory mem, VkDeviceSize offset, VkDeviceSize size) {
    std::unique_lock<std::mutex> l(mut);
    auto& rng = src_ranges[mem];
    if (rng->fast) {
      return;
    }
    auto begin_range = reinterpret_cast<char*>(
        reinterpret_cast<uintptr_t>(rng->src_ptr + offset) >> 12);
    auto end_range = reinterpret_cast<char*>(
        reinterpret_cast<uintptr_t>(rng->src_ptr + offset + size + 4095) >> 12);

    auto range_size = end_range - begin_range;
    DWORD old_protect;
    VirtualProtect(begin_range, range_size, 0, &old_protect);
  }

  void InvalidateMappedRange(VkDeviceMemory mem,
                             VkDeviceSize offset,
                             VkDeviceSize size) {
    std::unique_lock<std::mutex> l(mut);
    auto& rng = src_ranges[mem];
    if (rng->fast) {
      return;
    }
    memcpy(rng->dst_ptr + offset, rng->src_ptr + offset, size);
  }

 private:
  std::map<char*, range_data> ranges;
  std::unordered_map<VkDeviceMemory, range_data*> src_ranges;
  std::mutex mut;
  std::set<char*> dirty_read_pages;
  uint32_t total_pages = 0;
};
}  // namespace gapid2

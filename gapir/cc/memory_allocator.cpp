/*
 * Copyright (C) 2019 Google Inc.
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

#include <assert.h>
#include <cstring>
#include <iostream>
#include <set>
#include <sstream>

#include <sys/time.h>

#include "core/cc/log.h"
#include "memory_allocator.h"

namespace gapir {

std::map<unsigned int,
         std::map<unsigned char*, MemoryAllocator::MemoryRegion>::iterator>
    MemoryAllocator::Handle::_emptyMap;

std::unique_ptr<MemoryAllocator> MemoryAllocator::create(size_t heapSize) {
  return std::unique_ptr<MemoryAllocator>(new MemoryAllocator(heapSize));
}

MemoryAllocator::MemoryAllocator(size_t heapSize)
    : heapSize_(0), heap_(nullptr), purgableHead_(0), idGenerator_(0) {
  while (heap_ == nullptr) {
    const auto overSize = heapSize + heapSize / 2;
    heap_ = new (std::nothrow) unsigned char[overSize];

    if (heap_ != nullptr) {
      delete[] heap_;
      heap_ = new (std::nothrow) unsigned char[heapSize];
    }

    if (heap_ == nullptr) {
      heapSize = heapSize / 2;
    }
  }

  purgableHead_ = heapSize_ = heapSize;
}

MemoryAllocator::~MemoryAllocator() {}

MemoryAllocator::Handle MemoryAllocator::allocateStatic(size_t size) {
  MemoryAllocator::MemoryRegion bestCandidate(
      0, staticRegionMap_.size() > 0
             ? staticRegionMap_.begin()->second.getOffset()
             : heapSize_);

  for (auto staticRegionIter = staticRegionMap_.begin();
       staticRegionIter != staticRegionMap_.end(); ++staticRegionIter) {
    const auto nextStaticRegionIter = std::next(staticRegionIter);

    const auto candidateStart = staticRegionIter->second.getOffset() +
                                staticRegionIter->second.getSize();
    const auto candidateSize = (nextStaticRegionIter != staticRegionMap_.end()
                                    ? nextStaticRegionIter->second.getOffset()
                                    : heapSize_) -
                               candidateStart;

    if (candidateSize > bestCandidate.getSize()) {
      bestCandidate =
          MemoryAllocator::MemoryRegion(candidateStart, candidateSize);
    }
  }

  if (bestCandidate.getSize() < size) {
    return MemoryAllocator::Handle();
  }

  if (bestCandidate.getOffset() == 0) {
    bestCandidate =
        MemoryAllocator::MemoryRegion(bestCandidate.getOffset(), size);
  } else {
    const size_t oversize = bestCandidate.getSize() - size;
    bestCandidate = MemoryAllocator::MemoryRegion(
        bestCandidate.getOffset() + oversize / 2, size);
  }

  auto handle = registerStaticAllocate(bestCandidate);
  purgeOrRelocateRange(bestCandidate.getOffset(),
                       bestCandidate.getOffset() + bestCandidate.getSize());

  return handle;
}

MemoryAllocator::Handle MemoryAllocator::allocatePurgable(size_t size,
                                                          bool allowRelocate) {
  std::map<unsigned char*, MemoryAllocator::MemoryRegion>::const_iterator
      closestStaticIter;
  size_t closestStaticData =
      getClosestStaticData(purgableHead_, &closestStaticIter);

  if (purgableHead_ - closestStaticData >= size) {
    auto purgableRegion =
        MemoryAllocator::MemoryRegion(purgableHead_ - size, size);

    auto handle = registerPurgableAllocate(purgableRegion);
    purgableHead_ -= size;

    return handle;
  }

  if (closestStaticIter == staticRegionMap_.end()) {
    if (allowRelocate == true && compactPurgableMemory()) {
      return allocatePurgable(size, allowRelocate);
    } else {
      return MemoryAllocator::Handle();
    }
  } else {
    purgableHead_ = closestStaticIter->second.getOffset();
    return allocatePurgable(size, allowRelocate);
  }
}

bool MemoryAllocator::resizeStaticAllocation(
    const MemoryAllocator::Handle& address, size_t size) {
  auto staticIter = staticRegionMap_.find(&(*address));
  if (staticIter != staticRegionMap_.end()) {
    auto nextStaticIter = std::next(staticIter);

    size_t ceiling = (nextStaticIter != staticRegionMap_.end()
                          ? nextStaticIter->second.getOffset()
                          : heapSize_) -
                     staticIter->second.getOffset();
    if (size > ceiling) {
      return false;
    }

    auto newStaticAllocation =
        MemoryAllocator::MemoryRegion(staticIter->second.getOffset(), size);

    registerResize(newStaticAllocation);
    purgeOrRelocateRange(
        newStaticAllocation.getOffset(),
        newStaticAllocation.getOffset() + newStaticAllocation.getSize());

    return true;
  }

  return false;
}

bool MemoryAllocator::releaseAllocation(MemoryAllocator::Handle& address) {
  if (address == nullptr) {
    auto uid = address._backing->first;
    relocationMap_.erase(uid);
    address = MemoryAllocator::Handle();
    return true;
  }

  auto staticIter = staticRegionMap_.find(&(*address));
  if (staticIter != staticRegionMap_.end()) {
    registerStaticRelease(staticIter->second);
    address = MemoryAllocator::Handle();
    return true;
  }

  auto purgableIter = purgableRegionMap_.find(&(*address));
  if (purgableIter != purgableRegionMap_.end()) {
    registerPurgableRelease(purgableIter->second);
    address = MemoryAllocator::Handle();
    return true;
  }

  return false;
}

size_t MemoryAllocator::getTotalStaticDataUsage() const {
  size_t size = 0;
  for (auto&& alloc : staticRegionMap_) {
    size += alloc.second.getSize();
  }

  return size;
}

size_t MemoryAllocator::getTotalPurgableDataUsage() const {
  size_t size = 0;
  for (auto&& alloc : purgableRegionMap_) {
    size += alloc.second.getSize();
  }

  return size;
}

void MemoryAllocator::purgeOrRelocateRange(size_t start, size_t end) {
  std::stringstream ss;
  ss << "MemoryAllocator[" << this << "]::purgeOrRelocateRange(" << start
     << ", " << end << ")";
  GAPID_DEBUG(ss.str().c_str());

  auto nextPurgableIter = purgableRegionMap_.lower_bound(heap_ + start);
  auto prevPurgableIter = nextPurgableIter != purgableRegionMap_.begin()
                              ? std::prev(nextPurgableIter)
                              : nextPurgableIter;

  std::vector<
      std::pair<MemoryAllocator::MemoryRegion, MemoryAllocator::MemoryRegion> >
      toRelocate;
  std::vector<MemoryAllocator::MemoryRegion> toPurge;

  for (auto purgableIter = prevPurgableIter;
       purgableIter != purgableRegionMap_.end() &&
       purgableIter->second.getOffset() < end;
       ++purgableIter) {
    // Check if this entry represents a purged block, and ignore it if so.
    if (purgableIter->first == nullptr &&
        purgableIter->second.getOffset() == 0 &&
        purgableIter->second.getSize() == 0) {
      continue;
    }

    // Try to allocate somewhere outside the forbidden region to relocate the
    // data.
    auto newAlloc = allocatePurgable(purgableIter->second.getSize(), false);

    // If we got somewhere to move the data, send it there.
    if (newAlloc != nullptr) {
      memmove(&(*newAlloc), purgableIter->first,
              purgableIter->second.getSize());
      toRelocate.push_back(std::make_pair(
          purgableIter->second,
          MemoryAllocator::MemoryRegion(&(*newAlloc) - heap_,
                                        purgableIter->second.getSize())));
    } else {
      // Otherwise we're going to purge this data.
      toPurge.push_back(purgableIter->second);
    }
  }

  // Do the relocations.
  for (auto&& relocate : toRelocate) {
    registerRelocate(relocate.first, relocate.second);
  }

  // Do the purges.
  for (auto&& purge : toPurge) {
    registerPurge(purge);
  }
}

bool MemoryAllocator::compactPurgableMemory() {
  std::stringstream ss;
  ss << "MemoryAllocator[" << this << "]::compactPurgableMemory()";
  GAPID_DEBUG(ss.str().c_str());

  std::map<unsigned char*, MemoryRegion> newPurgableRegionMap;
  std::map<unsigned char*, unsigned char*> relocations;

  auto newPurgableHead = heapSize_;
  bool compactedSomething = false;

  std::map<unsigned char*, MemoryAllocator::MemoryRegion>::const_iterator
      closestStaticIter;
  size_t closestStaticData =
      getClosestStaticData(newPurgableHead, &closestStaticIter);

  for (auto purgableIter = purgableRegionMap_.rbegin();
       purgableIter != purgableRegionMap_.rend(); ++purgableIter) {
    // Check if this entry represents a purged block, and just copy it across
    // verbatim it if so.
    if (purgableIter->first == nullptr &&
        purgableIter->second.getOffset() == 0 &&
        purgableIter->second.getSize() == 0) {
      newPurgableRegionMap.insert(
          std::make_pair(purgableIter->first, purgableIter->second));
      continue;
    }

    auto size = purgableIter->second.getSize();

    while (newPurgableHead - closestStaticData < size) {
      newPurgableHead = closestStaticIter->second.getOffset();
      closestStaticData =
          getClosestStaticData(newPurgableHead, &closestStaticIter);

      if (newPurgableHead == 0) {
        return compactedSomething;
      }
    }

    auto destAddress = heap_ + newPurgableHead - size;
    auto srcAddress = heap_ + purgableIter->second.getOffset();

    relocations[srcAddress] = destAddress;

    if (srcAddress != destAddress) {
      assert(destAddress > srcAddress);
      memmove(destAddress, srcAddress, size);

      auto newPurgableRegion =
          MemoryAllocator::MemoryRegion(newPurgableHead - size, size);
      newPurgableRegionMap[heap_ + newPurgableRegion.getOffset()] =
          newPurgableRegion;

      compactedSomething = true;
    } else {
      newPurgableRegionMap[heap_ + purgableIter->second.getOffset()] =
          purgableIter->second;
    }

    newPurgableHead = newPurgableHead - size;
  }

  if (compactedSomething == true) {
    std::set<unsigned int> purgedRegionIDs;

    auto oldNullRegionIter = purgableRegionMap_.find(nullptr);
    if (oldNullRegionIter != purgableRegionMap_.end()) {
      for (auto relocationMapIter = relocationMap_.begin();
           relocationMapIter != relocationMap_.end(); ++relocationMapIter) {
        if (relocationMapIter->second == oldNullRegionIter) {
          purgedRegionIDs.insert(relocationMapIter->first);
        }
      }
    }

    purgableRegionMap_ = newPurgableRegionMap;

    for (auto relocationIter = relocations.rbegin();
         relocationIter != relocations.rend(); ++relocationIter) {
      auto inverseRelocationIter =
          inverseRelocationMap_.find(relocationIter->first);
      assert(inverseRelocationIter != inverseRelocationMap_.end());

      auto uid = inverseRelocationIter->second;

      auto relocationMapIter = relocationMap_.find(uid);
      assert(relocationMapIter != relocationMap_.end());

      auto purgableIter = purgableRegionMap_.find(relocationIter->second);
      assert(purgableIter != purgableRegionMap_.end());

      relocationMapIter->second = purgableIter;

      inverseRelocationMap_.erase(inverseRelocationIter);
      inverseRelocationMap_.insert(std::make_pair(relocationIter->second, uid));
    }

    if (purgedRegionIDs.size() > 0) {
      auto newNullRegionIter = purgableRegionMap_.find(nullptr);
      assert(newNullRegionIter != purgableRegionMap_.end());

      for (auto&& purgedUID : purgedRegionIDs) {
        auto iter = relocationMap_.find(purgedUID);
        assert(iter != relocationMap_.end());

        iter->second = newNullRegionIter;
      }
    }

    purgableHead_ = newPurgableHead;
  }

  return compactedSomething;
}

size_t MemoryAllocator::getClosestStaticData(
    size_t belowOffset,
    std::map<unsigned char*, MemoryAllocator::MemoryRegion>::const_iterator*
        closestStaticIter) const {
  size_t closestStaticData = 0;
  auto nextStaticIter = staticRegionMap_.lower_bound(heap_ + belowOffset);
  auto prevStaticIter = nextStaticIter != staticRegionMap_.begin()
                            ? std::prev(nextStaticIter)
                            : staticRegionMap_.end();

  if (prevStaticIter != staticRegionMap_.end()) {
    closestStaticData =
        prevStaticIter->second.getOffset() + prevStaticIter->second.getSize();
  }

  if (closestStaticIter != nullptr) {
    *closestStaticIter = prevStaticIter;
  }

  return closestStaticData;
}

MemoryAllocator::Handle MemoryAllocator::registerStaticAllocate(
    const MemoryRegion& newRegion) {
  auto address = heap_ + newRegion.getOffset();
  auto uid = ++idGenerator_;

  auto allocIter = staticRegionMap_.insert(std::make_pair(address, newRegion));
  inverseRelocationMap_.insert(std::make_pair(address, uid));
  auto relocationIter =
      relocationMap_.insert(std::make_pair(uid, allocIter.first));

  return MemoryAllocator::Handle(relocationIter.first);
}

MemoryAllocator::Handle MemoryAllocator::registerPurgableAllocate(
    const MemoryRegion& newRegion) {
  auto address = heap_ + newRegion.getOffset();
  auto uid = ++idGenerator_;

  auto allocIter =
      purgableRegionMap_.insert(std::make_pair(address, newRegion));
  inverseRelocationMap_.insert(std::make_pair(address, uid));
  auto relocationIter =
      relocationMap_.insert(std::make_pair(uid, allocIter.first));

  return MemoryAllocator::Handle(relocationIter.first);
}

void MemoryAllocator::registerResize(const MemoryRegion& resizedRegion) {
  auto address = heap_ + resizedRegion.getOffset();

  assert(staticRegionMap_.find(address) != staticRegionMap_.end());
  staticRegionMap_[address] = resizedRegion;
}

void MemoryAllocator::registerRelocate(const MemoryRegion& from,
                                       const MemoryRegion& to) {
  assert(from.getSize() == to.getSize());

  auto fromAddress = heap_ + from.getOffset();
  auto toAddress = heap_ + to.getOffset();

  auto inverseIter = inverseRelocationMap_.find(fromAddress);
  assert(inverseIter != inverseRelocationMap_.end());

  auto uid = inverseIter->second;

  auto relocationIter = relocationMap_.find(uid);
  assert(relocationIter != relocationMap_.end());

  auto fromPurgableIter = purgableRegionMap_.find(fromAddress);
  assert(fromPurgableIter != purgableRegionMap_.end());

  auto toPurgableIter = purgableRegionMap_.find(toAddress);
  assert(toPurgableIter != purgableRegionMap_.end());

  relocationIter->second = toPurgableIter;

  inverseRelocationMap_.erase(inverseIter);
  inverseRelocationMap_.insert(std::make_pair(toAddress, uid));

  purgableRegionMap_.erase(fromPurgableIter);
}

void MemoryAllocator::registerPurge(const MemoryRegion& purge) {
  auto address = heap_ + purge.getOffset();

  auto purgableIter = purgableRegionMap_.find(address);
  assert(purgableIter != purgableRegionMap_.end());
  purgableRegionMap_.erase(purgableIter);

  auto inverseIter = inverseRelocationMap_.find(address);
  assert(inverseIter != inverseRelocationMap_.end());

  auto uid = inverseIter->second;
  inverseRelocationMap_.erase(inverseIter);

  auto purgedEntryIter = purgableRegionMap_.find(nullptr);
  if (purgedEntryIter == purgableRegionMap_.end()) {
    purgedEntryIter =
        purgableRegionMap_.insert(std::make_pair(nullptr, MemoryRegion()))
            .first;
  }
  assert(purgedEntryIter != purgableRegionMap_.end());

  auto relocationIter = relocationMap_.find(uid);
  assert(relocationIter != relocationMap_.end());
  relocationIter->second = purgedEntryIter;
}

void MemoryAllocator::registerStaticRelease(const MemoryRegion& release) {
  auto address = heap_ + release.getOffset();

  auto staticIter = staticRegionMap_.find(address);
  assert(staticIter != staticRegionMap_.end());
  staticRegionMap_.erase(staticIter);

  auto inverseIter = inverseRelocationMap_.find(address);
  assert(inverseIter != inverseRelocationMap_.end());

  auto uid = inverseIter->second;
  inverseRelocationMap_.erase(inverseIter);

  auto relocationIter = relocationMap_.find(uid);
  assert(relocationIter != relocationMap_.end());
  relocationMap_.erase(relocationIter);
}

void MemoryAllocator::registerPurgableRelease(const MemoryRegion& release) {
  auto address = heap_ + release.getOffset();

  auto purgableIter = purgableRegionMap_.find(address);
  assert(purgableIter != purgableRegionMap_.end());
  purgableRegionMap_.erase(purgableIter);

  auto inverseIter = inverseRelocationMap_.find(address);
  assert(inverseIter != inverseRelocationMap_.end());

  auto uid = inverseIter->second;
  inverseRelocationMap_.erase(inverseIter);

  auto relocationIter = relocationMap_.find(uid);
  assert(relocationIter != relocationMap_.end());
  relocationMap_.erase(relocationIter);
}

}  // namespace gapir

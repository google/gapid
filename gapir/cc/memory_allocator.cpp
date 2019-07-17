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
    MemoryAllocator::Handle::_dummyMap;
unsigned int MemoryAllocator::_idGenerator = 0;

std::unique_ptr<MemoryAllocator> MemoryAllocator::create(size_t heapSize) {
  return std::unique_ptr<MemoryAllocator>(new MemoryAllocator(heapSize));
}

MemoryAllocator::MemoryAllocator(size_t heapSize)
    : _heapSize(heapSize),
      _heap(new unsigned char[heapSize]),
      _purgableHead(heapSize) {}

MemoryAllocator::~MemoryAllocator() {}

MemoryAllocator::Handle MemoryAllocator::allocateStatic(size_t size) {
  MemoryAllocator::MemoryRegion bestCandidate(
      0, _staticRegionMap.size() > 0
             ? _staticRegionMap.begin()->second.getOffset()
             : _heapSize);

  for (auto staticRegionIter = _staticRegionMap.begin();
       staticRegionIter != _staticRegionMap.end(); ++staticRegionIter) {
    const auto nextStaticRegionIter = std::next(staticRegionIter);

    const auto candidateStart = staticRegionIter->second.getOffset() +
                                staticRegionIter->second.getSize();
    const auto candidateSize = (nextStaticRegionIter != _staticRegionMap.end()
                                    ? nextStaticRegionIter->second.getOffset()
                                    : _heapSize) -
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
      getClosestStaticData(_purgableHead, &closestStaticIter);

  if (_purgableHead - closestStaticData >= size) {
    auto purgableRegion =
        MemoryAllocator::MemoryRegion(_purgableHead - size, size);

    auto handle = registerPurgableAllocate(purgableRegion);
    _purgableHead -= size;

    return handle;
  }

  if (closestStaticIter == _staticRegionMap.end()) {
    if (allowRelocate == true && compactPurgableMemory()) {
      return allocatePurgable(size, allowRelocate);
    } else {
      return MemoryAllocator::Handle();
    }
  } else {
    _purgableHead = closestStaticIter->second.getOffset();
    return allocatePurgable(size, allowRelocate);
  }
}

bool MemoryAllocator::resizeStaticAllocation(
    const MemoryAllocator::Handle& address, size_t size) {
  auto staticIter = _staticRegionMap.find(&(*address));
  if (staticIter != _staticRegionMap.end()) {
    auto nextStaticIter = std::next(staticIter);

    size_t ceiling = (nextStaticIter != _staticRegionMap.end()
                          ? nextStaticIter->second.getOffset()
                          : _heapSize) -
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

bool MemoryAllocator::releaseAllocation(
    const MemoryAllocator::Handle& address) {
  if (address == nullptr) {
    auto uid = address._backing->first;
    _relocationMap.erase(uid);
    return true;
  }

  auto staticIter = _staticRegionMap.find(&(*address));
  if (staticIter != _staticRegionMap.end()) {
    registerStaticRelease(staticIter->second);
    return true;
  }

  auto purgableIter = _purgableRegionMap.find(&(*address));
  if (purgableIter != _purgableRegionMap.end()) {
    registerPurgableRelease(purgableIter->second);
    return true;
  }

  return false;
}

size_t MemoryAllocator::getTotalStaticDataUsage() const {
  size_t size = 0;
  for (auto&& alloc : _staticRegionMap) {
    size += alloc.second.getSize();
  }

  return size;
}

size_t MemoryAllocator::getTotalPurgableDataUsage() const {
  size_t size = 0;
  for (auto&& alloc : _purgableRegionMap) {
    size += alloc.second.getSize();
  }

  return size;
}

void MemoryAllocator::purgeOrRelocateRange(size_t start, size_t end) {
  std::stringstream ss;
  ss << "MemoryAllocator[" << this << "]::purgeOrRelocateRange(" << start
     << ", " << end << ")";
  GAPID_DEBUG(ss.str().c_str());

  auto nextPurgableIter = _purgableRegionMap.lower_bound(_heap + start);
  auto prevPurgableIter = nextPurgableIter != _purgableRegionMap.begin()
                              ? std::prev(nextPurgableIter)
                              : nextPurgableIter;

  std::vector<
      std::pair<MemoryAllocator::MemoryRegion, MemoryAllocator::MemoryRegion> >
      toRelocate;
  std::vector<MemoryAllocator::MemoryRegion> toPurge;

  for (auto purgableIter = prevPurgableIter;
       purgableIter != _purgableRegionMap.end() &&
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
          MemoryAllocator::MemoryRegion(&(*newAlloc) - _heap,
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

  auto newPurgableHead = _heapSize;
  bool compactedSomething = false;

  std::map<unsigned char*, MemoryAllocator::MemoryRegion>::const_iterator
      closestStaticIter;
  size_t closestStaticData =
      getClosestStaticData(newPurgableHead, &closestStaticIter);

  for (auto purgableIter = _purgableRegionMap.rbegin();
       purgableIter != _purgableRegionMap.rend(); ++purgableIter) {
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

    auto destAddress = _heap + newPurgableHead - size;
    auto srcAddress = _heap + purgableIter->second.getOffset();

    relocations[srcAddress] = destAddress;

    if (srcAddress != destAddress) {
      assert(destAddress > srcAddress);
      memmove(destAddress, srcAddress, size);

      auto newPurgableRegion =
          MemoryAllocator::MemoryRegion(newPurgableHead - size, size);
      newPurgableRegionMap[_heap + newPurgableRegion.getOffset()] =
          newPurgableRegion;

      compactedSomething = true;
    } else {
      newPurgableRegionMap[_heap + purgableIter->second.getOffset()] =
          purgableIter->second;
    }

    newPurgableHead = newPurgableHead - size;
  }

  if (compactedSomething == true) {
    std::set<unsigned int> purgedRegionIDs;

    auto oldNullRegionIter = _purgableRegionMap.find(nullptr);
    if (oldNullRegionIter != _purgableRegionMap.end()) {
      for (auto relocationMapIter = _relocationMap.begin();
           relocationMapIter != _relocationMap.end(); ++relocationMapIter) {
        if (relocationMapIter->second == oldNullRegionIter) {
          purgedRegionIDs.insert(relocationMapIter->first);
        }
      }
    }

    _purgableRegionMap = newPurgableRegionMap;

    for (auto relocationIter = relocations.rbegin();
         relocationIter != relocations.rend(); ++relocationIter) {
      auto inverseRelocationIter =
          _inverseRelocationMap.find(relocationIter->first);
      assert(inverseRelocationIter != _inverseRelocationMap.end());

      auto uid = inverseRelocationIter->second;

      auto relocationMapIter = _relocationMap.find(uid);
      assert(relocationMapIter != _relocationMap.end());

      auto purgableIter = _purgableRegionMap.find(relocationIter->second);
      assert(purgableIter != _purgableRegionMap.end());

      relocationMapIter->second = purgableIter;

      _inverseRelocationMap.erase(inverseRelocationIter);
      _inverseRelocationMap.insert(std::make_pair(relocationIter->second, uid));
    }

    if (purgedRegionIDs.size() > 0) {
      auto newNullRegionIter = _purgableRegionMap.find(nullptr);
      assert(newNullRegionIter != _purgableRegionMap.end());

      for (auto&& purgedUID : purgedRegionIDs) {
        auto iter = _relocationMap.find(purgedUID);
        assert(iter != _relocationMap.end());

        iter->second = newNullRegionIter;
      }
    }

    _purgableHead = newPurgableHead;
  }

  return compactedSomething;
}

size_t MemoryAllocator::getClosestStaticData(
    size_t belowOffset,
    std::map<unsigned char*, MemoryAllocator::MemoryRegion>::const_iterator*
        closestStaticIter) const {
  size_t closestStaticData = 0;
  auto nextStaticIter = _staticRegionMap.lower_bound(_heap + belowOffset);
  auto prevStaticIter = nextStaticIter != _staticRegionMap.begin()
                            ? std::prev(nextStaticIter)
                            : _staticRegionMap.end();

  if (prevStaticIter != _staticRegionMap.end()) {
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
  auto address = _heap + newRegion.getOffset();
  auto uid = ++_idGenerator;

  auto allocIter = _staticRegionMap.insert(std::make_pair(address, newRegion));
  _inverseRelocationMap.insert(std::make_pair(address, uid));
  auto relocationIter =
      _relocationMap.insert(std::make_pair(uid, allocIter.first));

  return MemoryAllocator::Handle(relocationIter.first);
}

MemoryAllocator::Handle MemoryAllocator::registerPurgableAllocate(
    const MemoryRegion& newRegion) {
  auto address = _heap + newRegion.getOffset();
  auto uid = ++_idGenerator;

  auto allocIter =
      _purgableRegionMap.insert(std::make_pair(address, newRegion));
  _inverseRelocationMap.insert(std::make_pair(address, uid));
  auto relocationIter =
      _relocationMap.insert(std::make_pair(uid, allocIter.first));

  return MemoryAllocator::Handle(relocationIter.first);
}

void MemoryAllocator::registerResize(const MemoryRegion& resizedRegion) {
  auto address = _heap + resizedRegion.getOffset();

  assert(_staticRegionMap.find(address) != _staticRegionMap.end());
  _staticRegionMap[address] = resizedRegion;
}

void MemoryAllocator::registerRelocate(const MemoryRegion& from,
                                       const MemoryRegion& to) {
  assert(from.getSize() == to.getSize());

  auto fromAddress = _heap + from.getOffset();
  auto toAddress = _heap + to.getOffset();

  auto inverseIter = _inverseRelocationMap.find(fromAddress);
  assert(inverseIter != _inverseRelocationMap.end());

  auto uid = inverseIter->second;

  auto relocationIter = _relocationMap.find(uid);
  assert(relocationIter != _relocationMap.end());

  auto fromPurgableIter = _purgableRegionMap.find(fromAddress);
  assert(fromPurgableIter != _purgableRegionMap.end());

  auto toPurgableIter = _purgableRegionMap.find(toAddress);
  assert(toPurgableIter != _purgableRegionMap.end());

  relocationIter->second = toPurgableIter;

  _inverseRelocationMap.erase(inverseIter);
  _inverseRelocationMap.insert(std::make_pair(toAddress, uid));

  _purgableRegionMap.erase(fromPurgableIter);
}

void MemoryAllocator::registerPurge(const MemoryRegion& purge) {
  auto address = _heap + purge.getOffset();

  auto purgableIter = _purgableRegionMap.find(address);
  assert(purgableIter != _purgableRegionMap.end());
  _purgableRegionMap.erase(purgableIter);

  auto inverseIter = _inverseRelocationMap.find(address);
  assert(inverseIter != _inverseRelocationMap.end());

  auto uid = inverseIter->second;
  _inverseRelocationMap.erase(inverseIter);

  auto purgedEntryIter = _purgableRegionMap.find(nullptr);
  if (purgedEntryIter == _purgableRegionMap.end()) {
    purgedEntryIter =
        _purgableRegionMap.insert(std::make_pair(nullptr, MemoryRegion()))
            .first;
  }
  assert(purgedEntryIter != _purgableRegionMap.end());

  auto relocationIter = _relocationMap.find(uid);
  assert(relocationIter != _relocationMap.end());
  relocationIter->second = purgedEntryIter;
}

void MemoryAllocator::registerStaticRelease(const MemoryRegion& release) {
  auto address = _heap + release.getOffset();

  auto staticIter = _staticRegionMap.find(address);
  assert(staticIter != _staticRegionMap.end());
  _staticRegionMap.erase(staticIter);

  auto inverseIter = _inverseRelocationMap.find(address);
  assert(inverseIter != _inverseRelocationMap.end());

  auto uid = inverseIter->second;
  _inverseRelocationMap.erase(inverseIter);

  auto relocationIter = _relocationMap.find(uid);
  assert(relocationIter != _relocationMap.end());
  _relocationMap.erase(relocationIter);
}

void MemoryAllocator::registerPurgableRelease(const MemoryRegion& release) {
  auto address = _heap + release.getOffset();

  auto purgableIter = _purgableRegionMap.find(address);
  assert(purgableIter != _purgableRegionMap.end());
  _purgableRegionMap.erase(purgableIter);

  auto inverseIter = _inverseRelocationMap.find(address);
  assert(inverseIter != _inverseRelocationMap.end());

  auto uid = inverseIter->second;
  _inverseRelocationMap.erase(inverseIter);

  auto relocationIter = _relocationMap.find(uid);
  assert(relocationIter != _relocationMap.end());
  _relocationMap.erase(relocationIter);
}

}  // namespace gapir

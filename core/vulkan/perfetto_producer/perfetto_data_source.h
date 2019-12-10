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
 *
 */

#ifndef PERFETTO_DATA_SOURCE_H__
#define PERFETTO_DATA_SOURCE_H__

#include "core/cc/recursive_spinlock.h"
#include "core/memory/arena/cc/arena.h"
#include "gapil/runtime/cc/map.h"
#include "gapil/runtime/cc/map.inc"
#include "perfetto/tracing/core/data_source_descriptor.h"
#include "perfetto/tracing/data_source.h"
#include "perfetto/tracing/tracing.h"

#include "core/vulkan/perfetto_producer/threadlocal_emitter_base.h"

namespace core {

template <typename T>
class PerfettoProducerData;

template <typename ProducerTraits>
class PerfettoProducer
    : public perfetto::DataSource<PerfettoProducer<ProducerTraits>> {
 public:
  PerfettoProducer() = default;
  PerfettoProducer(const PerfettoProducer&) = delete;
  PerfettoProducer& operator=(const PerfettoProducer&) = delete;
  PerfettoProducer(PerfettoProducer&&) = delete;
  PerfettoProducer& operator=(PerfettoProducer&&) = delete;

  static PerfettoProducerData<ProducerTraits>& Get();

  void OnSetup(const typename perfetto::DataSourceBase::SetupArgs&) override;
  void OnStart(const typename perfetto::DataSourceBase::StartArgs&) override;
  void OnStop(const typename perfetto::DataSourceBase::StopArgs&) override;

 private:
};

template <typename T>
class PerfettoProducerData {
 public:
  PerfettoProducerData();
  void RegisterEmitter(ThreadlocalEmitterBase*);
  void UnregisterEmitter(ThreadlocalEmitterBase*);
  void OnStart(const typename perfetto::DataSourceBase::StartArgs&);
  void OnStop(const typename perfetto::DataSourceBase::StopArgs&);
  void OnSetup(const typename perfetto::DataSourceBase::SetupArgs&);

 private:
  core::Arena arena_;
  core::RecursiveSpinLock emitter_lock_;
  gapil::Map<ThreadlocalEmitterBase*, bool, false> emitters_;
  bool started_ = false;
};
}  // namespace core

#define INCLUDING_DATA_SOURCE_INC__
#include "core/vulkan/perfetto_producer/perfetto_data_source.inc"
#undef INCLUDING_DATA_SOURCE_INC__

#endif  // PERFETTO_DATA_SOURCE_H__

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

#ifndef INCLUDING_DATA_SOURCE_INC__
#error "It is invalid to include this file directly. Include the header instead."
#endif

namespace core {

namespace internal {

struct PerfettoInitializer {
private:
    PerfettoInitializer();
public:
    static PerfettoInitializer& get_initializer() {
        static PerfettoInitializer pi;
        return pi;
    }
};

inline PerfettoInitializer::PerfettoInitializer() {
  perfetto::TracingInitArgs init_args;
  init_args.backends = perfetto::kSystemBackend;
  init_args.platform = perfetto::Platform::GetDefaultPlatform();
  init_args.shmem_size_hint_kb = 32 * 1024;
  init_args.shmem_page_size_hint_kb = 32;
  perfetto::Tracing::Initialize(init_args);
}
}

template<typename T>
PerfettoProducerData<T>::PerfettoProducerData() :
    emitters_(&arena_) {
  static auto& p = internal::PerfettoInitializer::get_initializer();
  (void)p;

  perfetto::DataSourceDescriptor dsd;
  dsd.set_name(T::producer_name);
  PerfettoProducer<T>::Register(dsd);
}

template<typename T>
PerfettoProducerData<T>& PerfettoProducer<T>::Get() {
    static PerfettoProducerData<T> data;
    return data;
}

template<typename T>
void PerfettoProducerData<T>::RegisterEmitter(ThreadlocalEmitterBase* v) {
    emitter_lock_.Lock();
    emitters_[v] = true;
    if (started_) {
        v->StartTracing();
    }
    emitter_lock_.Unlock();
}

template<typename T>
void PerfettoProducerData<T>::UnregisterEmitter(ThreadlocalEmitterBase* v) {
    emitter_lock_.Lock();
    emitters_.erase(v);
    emitter_lock_.Unlock();
}

template<typename T>
void PerfettoProducer<T>::OnSetup(const typename perfetto::DataSourceBase::SetupArgs& a) {
    Get().OnSetup(a);
}

template<typename T>
void PerfettoProducer<T>::OnStart(const typename perfetto::DataSourceBase::StartArgs& a) {
    Get().OnStart(a);
}

template<typename T>
void PerfettoProducer<T>::OnStop(const typename perfetto::DataSourceBase::StopArgs& a) {
    Get().OnStop(a);
}

template<typename T>
void PerfettoProducerData<T>::OnSetup(const typename perfetto::DataSourceBase::SetupArgs& a) {
    emitter_lock_.Lock();
    started_ = true;
    for (auto& e : emitters_) {
        e.first->SetupTracing(a);
    }
    emitter_lock_.Unlock();
}

template<typename T>
void PerfettoProducerData<T>::OnStart(const typename perfetto::DataSourceBase::StartArgs&) {
    emitter_lock_.Lock();
    started_ = true;
    for (auto e : emitters_) {
        e.first->StartTracing();
    }
    emitter_lock_.Unlock();
}


template<typename T>
void PerfettoProducerData<T>::OnStop(const typename perfetto::DataSourceBase::StopArgs&) {
    emitter_lock_.Lock();
    started_ = false;
    for (auto e : emitters_) {
        e.first->StopTracing();
    }
    emitter_lock_.Unlock();
}

} // namespace core
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

#ifndef __INCLUDING_PERFETTO_THREADLOCAL_EMITTER_INC__
#error "It is invalid to include this file directly. Include the header instead."
#endif
#include "core/vulkan/perfetto_producer/perfetto_data_source.h"
#include "core/cc/thread.h"
#include "core/cc/process_name.h"
#include "core/cc/timer.h"

#include "perfetto/trace/track_event/debug_annotation.pbzero.h"
#include "perfetto/trace/track_event/track_event.pbzero.h"
#include "perfetto/trace/track_event/thread_descriptor.pbzero.h"
#include "perfetto/trace/interned_data/interned_data.pbzero.h"

// Note these implementations should match map.go

namespace core {

template<typename T>
ThreadlocalEmitter<T>::ThreadlocalEmitter() :
    interned_names_(&arena_),
    interned_annotation_names_(&arena_),
    interned_categories_(&arena_),
    reset_(false) {
    thread_name_ = Thread::current().get_name();
    process_name_ = core::get_process_name();
    thread_id_ = Thread::current().id();
    process_id_ = core::get_process_id();
    PerfettoProducer<T>::Get().RegisterEmitter(this);
}

template<typename T>
ThreadlocalEmitter<T>::~ThreadlocalEmitter() {
    PerfettoProducer<T>::Get().UnregisterEmitter(this);
}

template<typename T>
uint64_t ThreadlocalEmitter<T>::InternName(
        const char* name,
        typename PerfettoProducer<T>::TraceContext::TracePacketHandle& packet,
        perfetto::protos::pbzero::InternedData** interned_data) {
    uint64_t val = interned_names_.findOrZero(name);
    if (!val) {
        val =  interned_names_.count() + 1;
        interned_names_[name] = val;
        if (!*interned_data) {
            *interned_data = packet->set_interned_data();
        }
        perfetto::protos::pbzero::EventName* event_name =
        (*interned_data)->add_event_names();
        event_name->set_iid(val);
        event_name->set_name(name);
        return interned_names_.count();
    }
    return val;
}

template<typename T>
uint64_t ThreadlocalEmitter<T>::InternAnnotationName(
        const char* name,
        typename PerfettoProducer<T>::TraceContext::TracePacketHandle& packet,
        perfetto::protos::pbzero::InternedData** interned_data) {
    uint64_t val = interned_annotation_names_.findOrZero(name);
    if (!val) {
        val =  interned_annotation_names_.count() + 1;
        interned_annotation_names_[name] = val;
        if (!*interned_data) {
            *interned_data = packet->set_interned_data();
        }
        auto* event_name = (*interned_data)->add_debug_annotation_names();
        event_name->set_iid(val);
        event_name->set_name(name);
        return interned_annotation_names_.count();
    }
    return val;
}


template<typename T>
uint64_t ThreadlocalEmitter<T>::InternCategory(
        const char* name,
        typename PerfettoProducer<T>::TraceContext::TracePacketHandle& packet,
        perfetto::protos::pbzero::InternedData** interned_data) {
        uint64_t val = interned_categories_.findOrZero(name);
    if (!val) {
        val =  interned_categories_.count() + 1;
        interned_categories_[name] = val;
        if (!*interned_data) {
            *interned_data = packet->set_interned_data();
        }
        auto* event_name = (*interned_data)->add_event_categories();
        event_name->set_iid(val);
        event_name->set_name(name);
        return interned_categories_.count();
    }
    return val;
}

template<typename T>
void ThreadlocalEmitter<T>::ResetIfNecessary() {
    if (reset_) {
        interned_annotation_names_.clear();
        interned_names_.clear();
        emitted_thread_data_ = false;
        emitted_process_data_ = false;
        reset_ = false;
    }
}

template<typename T>
void ThreadlocalEmitter<T>::EmitThreadData() {
    if (emitted_thread_data_) {
        return;
    }
    uint64_t time = perfetto::base::GetBootTimeNs().count();
    PerfettoProducer<T>::Trace(
        [this, time](typename PerfettoProducer<T>::TraceContext ctx) {
            auto packet = ctx.NewTracePacket();
            packet->set_timestamp(time);
            packet->set_incremental_state_cleared(true);
            perfetto::protos::pbzero::ThreadDescriptor* thread_descriptor =
            packet->set_thread_descriptor();
            thread_descriptor->set_pid(process_id_);
            thread_descriptor->set_tid(thread_id_);
            thread_descriptor->set_thread_name(thread_name_.c_str());
    });

    EmitProcessData();
    emitted_thread_data_ = true;
}


template<typename T>
void ThreadlocalEmitter<T>::EmitProcessData() {
    if (emitted_process_data_) {
        return;
    }
    uint64_t time = perfetto::base::GetBootTimeNs().count();
    PerfettoProducer<T>::Trace(
    [this, time](typename PerfettoProducer<T>::TraceContext ctx) {
            auto packet = ctx.NewTracePacket();
            packet->set_timestamp(time);
            perfetto::protos::pbzero::InternedData* interned_data = nullptr;
            uint64_t name_iid = InternName("process_name", packet, &interned_data);
            uint64_t annotation_name_iid = InternAnnotationName("name", packet, &interned_data);

            uint64_t category_iid = InternCategory("cat", packet, &interned_data);

            auto* track_event = packet->set_track_event();
            track_event->set_timestamp_absolute_us(time / 1000);
            track_event->add_category_iids(category_iid);

            auto* debug_annotation = track_event->add_debug_annotations();
            debug_annotation->set_name_iid(annotation_name_iid);
            debug_annotation->set_string_value(process_name_.c_str());

            auto* legacy_event = track_event->set_legacy_event();
            legacy_event->set_name_iid(name_iid);
            legacy_event->set_phase('M');
    });
    emitted_process_data_ = true;
}

template<typename T>
void ThreadlocalEmitter<T>::StartEvent(const char* name) {
    if (!enabled_) {
        return;
    }
    ResetIfNecessary();
    EmitThreadData();
    uint64_t time = perfetto::base::GetBootTimeNs().count();

    PerfettoProducer<T>::Trace(
        [this, name, time](typename PerfettoProducer<T>::TraceContext ctx) {
        auto packet = ctx.NewTracePacket();
        packet->set_timestamp(time);
        perfetto::protos::pbzero::InternedData* interned_data = nullptr;
        uint64_t name_iid = InternName(name, packet, &interned_data);

        // TODO: proper categories
        uint64_t category_iid = InternCategory("cat", packet, &interned_data);

        auto track_event = packet->set_track_event();
        track_event->set_timestamp_absolute_us(time / 1000);
        track_event->add_category_iids(category_iid);

        auto legacy_event = track_event->set_legacy_event();
        legacy_event->set_name_iid(name_iid);
        legacy_event->set_phase('B');
    });
}

template<typename T>
void ThreadlocalEmitter<T>::EndEvent() {
    if (!enabled_) {
        return;
    }
    uint64_t time = perfetto::base::GetBootTimeNs().count();
    PerfettoProducer<T>::Trace(
        [this, time](typename PerfettoProducer<T>::TraceContext ctx) {
        auto packet = ctx.NewTracePacket();
        packet->set_timestamp(time);

        perfetto::protos::pbzero::InternedData* interned_data = nullptr;
        uint64_t category_iid = InternCategory("cat", packet, &interned_data);

        auto track_event = packet->set_track_event();
        track_event->set_timestamp_absolute_us(time / 1000);
        track_event->add_category_iids(category_iid);

        auto legacy_event = track_event->set_legacy_event();
        legacy_event->set_phase('E');
    });
}

}
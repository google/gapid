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
package com.google.gapid.perfetto.models;

import com.google.common.collect.ImmutableSet;

/**
 * Data about a process in the trace.
 */
public class ProcessInfo {
  public final long upid;   // the perfetto id.
  public final long pid;    // the system id.
  public final String name;
  public final long totalDur;
  public final ImmutableSet<Long> utids;

  public ProcessInfo(long upid, long pid, String name, long totalDur, ImmutableSet<Long> utids) {
    this.upid = upid;
    this.pid = pid;
    this.name = name;
    this.totalDur = totalDur;
    this.utids = utids;
  }

  public String getDisplay() {
    return name.isEmpty() ? "[" + pid + "]" : name + " [" + pid + "]";
  }

  public static class Builder {
    private final long upid;
    private final long pid;
    private final String name;
    private long totalDur = 0;
    private final ImmutableSet.Builder<Long> utids = ImmutableSet.builder();

    public Builder(long upid, long pid, String name) {
      this.upid = upid;
      this.pid = pid;
      this.name = name;
    }

    public Builder addThread(long tid, long dur) {
      totalDur += dur;
      utids.add(tid);
      return this;
    }

    public ProcessInfo build() {
      return new ProcessInfo(upid, pid, name, totalDur, utids.build());
    }
  }
}

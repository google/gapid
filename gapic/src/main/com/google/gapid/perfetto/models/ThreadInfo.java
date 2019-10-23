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

import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;

import com.google.common.collect.ImmutableMap;
import com.google.common.collect.Maps;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Perfetto;
import com.google.gapid.perfetto.views.State;
import com.google.gapid.perfetto.views.StyleConstants;

import java.util.Map;

/**
 * Data about a thread in the trace.
 */
public class ThreadInfo {
  private static final long MIN_DUR = State.MAX_ZOOM_SPAN_NSEC / 1600;
  private static final String MAX_DEPTH_QUERY =
      "select utid, max(track_id), max(depth) + 1 from slices where dur >= " + MIN_DUR + " group by utid";
  private static final String THREAD_QUERY =
      "select utid, tid, thread.name, upid, pid, process.name, sum(dur) " +
      "from thread left join process using(upid) left join sched using (utid) " +
      "where utid != 0 group by utid";

  public final long utid;  // the perfetto id.
  public final long tid;   // the system id.
  public final long upid;  // the perfetto id.
  public final long trackId;
  public final String name;
  public final int maxDepth;
  public final long totalDur;

  public ThreadInfo(
      long utid, long tid, long upid, long trackId, String name, int maxDepth, long totalDur) {
    this.utid = utid;
    this.tid = tid;
    this.upid = upid;
    this.trackId = trackId;
    this.name = name;
    this.maxDepth = maxDepth;
    this.totalDur = totalDur;
  }

  public String getDisplay() {
    return name.isEmpty() ? "[" + tid + "]" : name + " [" + tid + "]";
  }

  public static ListenableFuture<Perfetto.Data.Builder> listThreads(Perfetto.Data.Builder data) {
    return transformAsync(maxDepth(data.qe), maxDepth ->
      transform(data.qe.query(THREAD_QUERY), res -> {
        Map<Long, ProcessInfo.Builder> procs = Maps.newHashMap();
        ImmutableMap.Builder<Long, ThreadInfo> threads = ImmutableMap.builder();
        res.forEachRow(($1, row) -> {
          long utid = row.getLong(0);
          long tid = row.getLong(1);
          String tName = row.getString(2);
          long upid = row.getLong(3);
          long pid = row.getLong(4);
          String pName = row.getString(5);
          long dur = row.getLong(6);
          TrackDepth td = maxDepth.getOrDefault(utid, TrackDepth.NULL);
          threads.put(
              utid, new ThreadInfo(utid, tid, upid, td.trackId, tName, td.depth, dur));
          procs.computeIfAbsent(upid, $2 -> new ProcessInfo.Builder(upid, pid, pName))
              .addThread(utid, dur);
        });
        data.setThreads(threads.build());

        ImmutableMap.Builder<Long, ProcessInfo> procMap = ImmutableMap.builder();
        procs.forEach((id, builder) -> procMap.put(id, builder.build()));
        data.setProcesses(procMap.build());

        return data;
      }));
  }

  private static ListenableFuture<Map<Long, TrackDepth>> maxDepth(QueryEngine qe) {
    return transform(qe.queries(MAX_DEPTH_QUERY),
        res -> res.map(row -> row.getLong(0), TrackDepth::new));
  }

  public static StyleConstants.HSL getColor(State state, long utid) {
    ThreadInfo threadInfo = state.getThreadInfo(utid);
    StyleConstants.HSL baseColor = StyleConstants.colorForThread(threadInfo);

    long upid = threadInfo.upid;
    if (state.getSelectedUpid() == -1 || state.getSelectedUtid() == -1) {
      return baseColor;
    } else if (utid == state.getSelectedUtid()) {
      return baseColor;
    } else if (upid == state.getSelectedUpid()) {
      return baseColor.adjusted(baseColor.h, baseColor.s - 20, Math.min(baseColor.l + 20,  60));
    } else {
      return StyleConstants.getGrayColor();
    }
  }

  public static Display getDisplay(Perfetto.Data data, long utid, boolean hover) {
    ThreadInfo thread = data.threads.get(utid);
    if (thread == null) {
      // fallback, should not really happen.
      return hover ? null : new Display(null, null, "??? [id: " + utid + "]", "");
    }
    String threadLabel = (hover ? "T: " : "") + thread.name + " [" + thread.tid + "]";

    ProcessInfo process = data.processes.get(thread.upid);
    if (process == null || process.name.isEmpty()) {
      return new Display(process, thread, threadLabel, "");
    }
    String processLabel = (hover ? "P: " : "") + process.name + " [" + process.pid + "]";

    return new Display(process, thread, threadLabel, processLabel);
  }

  public static class Display {
    public final ProcessInfo process;
    public final ThreadInfo thread;
    public final String title;
    public final String subTitle;

    public Display(ProcessInfo process, ThreadInfo thread, String title, String subTitle) {
      this.process = process;
      this.thread = thread;
      this.title = title;
      this.subTitle = subTitle;
    }
  }

  private static class TrackDepth {
    public static final TrackDepth NULL = new TrackDepth(-1, 0);

    public final long trackId;
    public final int depth;

    public TrackDepth(long trackId, int depth) {
      this.trackId = trackId;
      this.depth = depth;
    }

    public TrackDepth(QueryEngine.Row row) {
      this(row.getLong(1), row.getInt(2));
    }
  }
}

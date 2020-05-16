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

import static com.google.gapid.perfetto.views.StyleConstants.gradient;
import static com.google.gapid.util.MoreFutures.transform;

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
  public static final ThreadInfo EMPTY = new ThreadInfo(-1, -1, -1, -1, "", -1, -1);

  private static final long MIN_DUR = State.MAX_ZOOM_SPAN_NSEC / 1600;
  private static final String THREAD_QUERY =
      "with threads as (" +
        "select utid, tid, thread.name tname, upid, pid, process.name pname, sum(dur) dur " +
        "from thread left join process using(upid) left join sched using (utid) " +
        "where utid != 0 group by utid), " +
      "depth as (" +
        "select t.utid utid, t.id track_id, max(depth) + 1 depth " +
        "from thread_track t inner join slice s on (s.track_id = t.id) " +
        "where dur >= " + MIN_DUR + " group by t.id) " +
      "select utid, tid, tname, upid, pid, pname, dur, track_id, depth " +
      "from threads left join depth using (utid)";

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

  public StyleConstants.Gradient getColor() {
    return gradient(Long.hashCode(upid != 0 ? upid : utid));
  }

  public static ListenableFuture<Perfetto.Data.Builder> listThreads(Perfetto.Data.Builder data) {
    return transform(data.qe.query(THREAD_QUERY), res -> {
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
        long trackId = row.getLong(7, -1);
        int depth = row.getInt(8);
        threads.put(utid, new ThreadInfo(utid, tid, upid, trackId, tName, depth, dur));
        procs.computeIfAbsent(upid, $2 -> new ProcessInfo.Builder(upid, pid, pName))
            .addThread(utid, dur);
      });
      data.setThreads(threads.build());

      ImmutableMap.Builder<Long, ProcessInfo> procMap = ImmutableMap.builder();
      procs.forEach((id, builder) -> procMap.put(id, builder.build()));
      data.setProcesses(procMap.build());

      return data;
    });
  }

  public static Display getDisplay(State state, long utid, boolean hover) {
    ThreadInfo thread = state.getThreadInfo(utid);
    if (thread == null) {
      // fallback, should not really happen.
      return hover ? null : new Display(
          null, new ThreadInfo(utid, -1, 0, -1, "", 0, 0), "??? [id: " + utid + "]", "???", "", "");
    }
    String threadLabel = (hover ? "T: " : "") + thread.name + " [" + thread.tid + "]";

    ProcessInfo process = state.getProcessInfo(thread.upid);
    if (process == null || process.name.isEmpty()) {
      return new Display(process, thread, threadLabel, thread.name, "", "");
    }
    String processLabel = (hover ? "P: " : "") + process.name + " [" + process.pid + "]";

    return new Display(process, thread, threadLabel, thread.name, processLabel, process.name);
  }

  public static class Display {
    public final ProcessInfo process;
    public final ThreadInfo thread;
    public final String title;
    public final String shortTitle;
    public final String subTitle;
    public final String shortSubTitle;

    public Display(ProcessInfo process, ThreadInfo thread, String title, String shortTitle,
        String subTitle, String shortSubTitle) {
      this.process = process;
      this.thread = thread;
      this.title = title;
      this.shortTitle = shortTitle;
      this.subTitle = subTitle;
      this.shortSubTitle = shortSubTitle;
    }
  }
}

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
package com.google.gapid.models;

import static com.google.common.collect.ImmutableListMultimap.toImmutableListMultimap;
import static com.google.gapid.rpc.UiErrorCallback.error;
import static com.google.gapid.rpc.UiErrorCallback.success;
import static com.google.gapid.util.Logging.throttleLogRpcError;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static java.util.function.Function.identity;
import static java.util.logging.Level.WARNING;

import com.google.common.collect.ImmutableListMultimap;
import com.google.common.collect.ImmutableMap;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.models.CounterInfo;
import com.google.gapid.perfetto.models.GpuInfo;
import com.google.gapid.perfetto.models.ProcessInfo;
import com.google.gapid.perfetto.models.QueryEngine;
import com.google.gapid.perfetto.models.ThreadInfo;
import com.google.gapid.perfetto.models.TrackConfig;
import com.google.gapid.perfetto.models.Tracks;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiErrorCallback.ResultOrError;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.Loadable;
import com.google.gapid.views.StatusBar;

import org.eclipse.swt.widgets.Shell;

import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * Model responsible for querying a Perfetto trace.
 */
public class Perfetto extends ModelBase<Perfetto.Data, Path.Capture, Loadable.Message, Perfetto.Listener> {
  private static final Logger LOG = Logger.getLogger(Perfetto.class.getName());

  private final StatusBar status;

  public Perfetto(
      Shell shell, Analytics analytics, Client client, Capture capture, StatusBar status) {
    super(LOG, shell, analytics, client, Listener.class);
    this.status = status;

    capture.addListener(new Capture.Listener() {
      @Override
      public void onCaptureLoadingStart(boolean maintainState) {
        reset();
      }

      @Override
      public void onCaptureLoaded(Loadable.Message error) {
        if (error == null && capture.isPerfetto()) {
          load(capture.getData().path, false);
        } else {
          reset();
        }
      }
    });
  }

  @Override
  protected ListenableFuture<Data> doLoad(Path.Capture source) {
    Data.Builder data = new Data.Builder(new QueryEngine(client, source, status));
    return
        transformAsync(withStatus("Examining the trace...", examineTrace(data)), $1 ->
          transformAsync(withStatus("Querying threads...", queryThreads(data)), $2 ->
            transformAsync(withStatus("Querying GPU info...", queryGpu(data)), $3 ->
              transformAsync(withStatus("Querying counters...", queryCounters(data)), $4 ->
                transform(withStatus("Enumerating tracks...", enumerateTracks(data)), $5 ->
                  data.build())))));
  }

  private static ListenableFuture<Data.Builder> examineTrace(Data.Builder data) {
    return transformAsync(data.qe.getTraceTimeBounds(), traceTime -> {
      data.setTraceTime(traceTime);
      return transform(data.qe.getNumberOfCpus(), numCpus -> data.setNumCpus(numCpus));
    });
  }

  private static ListenableFuture<Data.Builder> queryThreads(Data.Builder data) {
    return ThreadInfo.listThreads(data);
  }

  private static ListenableFuture<Data.Builder> queryGpu(Data.Builder data) {
    return GpuInfo.listGpus(data);
  }

  private static ListenableFuture<Data.Builder> queryCounters(Data.Builder data) {
    return CounterInfo.listCounters(data);
  }

  private static ListenableFuture<Data.Builder> enumerateTracks(Data.Builder data) {
    return Tracks.enumerate(data);
  }

  private <T> ListenableFuture<T> withStatus(String msg, ListenableFuture<T> future) {
    return withStatus(Loadable.Message.loading(msg), future);
  }

  private <T> ListenableFuture<T> withStatus(Loadable.Message msg, ListenableFuture<T> future) {
    scheduleIfNotDisposed(shell, () -> {
      listeners.fire().onPerfettoLoadingStatus(msg);
    });
    return future;
  }

  @Override
  protected ResultOrError<Data, Loadable.Message> processResult(Rpc.Result<Data> result) {
    try {
      return success(result.get());
    } catch (RpcException e) {
      LOG.log(WARNING, "Failed to load System Trace", e);
      return error(Loadable.Message.error(e));
    } catch (ExecutionException e) {
      if (!shell.isDisposed()) {
        analytics.reportException(e);
        throttleLogRpcError(LOG, "Failed to load System Trace", e);
      }
      return error(Loadable.Message.error("Failed to load System Trace"));
    }
  }

  @Override
  protected void fireLoadStartEvent() {
    // Don't care about this event.
  }

  @Override
  protected void updateError(Loadable.Message error) {
    listeners.fire().onPerfettoLoaded(error);
  }

  @Override
  protected void fireLoadedEvent() {
    listeners.fire().onPerfettoLoaded(null);
  }

  public ListenableFuture<com.google.gapid.proto.perfetto.Perfetto.QueryResult> query(String sql) {
    if (!isLoaded()) {
      return Futures.immediateFailedFuture(new Exception("System Trace not loaded"));
    }
    return getData().qe.raw(sql);
  }

  public static class Data {
    public final QueryEngine qe;
    public final TimeSpan traceTime;
    public final int numCpus;
    public final ImmutableMap<Long, ProcessInfo> processes;
    public final ImmutableMap<Long, ThreadInfo> threads;
    public final GpuInfo gpu;
    public final ImmutableMap<Long, CounterInfo> counters;
    public final TrackConfig tracks;

    public Data(QueryEngine queries, TimeSpan traceTime, int numCpus,
        ImmutableMap<Long, ProcessInfo> processes, ImmutableMap<Long, ThreadInfo> threads,
        GpuInfo gpu, ImmutableMap<Long, CounterInfo> counters, TrackConfig tracks) {
      this.qe = queries;
      this.traceTime = traceTime;
      this.numCpus = numCpus;
      this.processes = processes;
      this.threads = threads;
      this.gpu = gpu;
      this.counters = counters;
      this.tracks = tracks;
    }

    public static class Builder {
      public final QueryEngine qe;
      private TimeSpan traceTime;
      private int numCpus;
      private ImmutableMap<Long, ProcessInfo> processes;
      private ImmutableMap<Long, ThreadInfo> threads;
      private GpuInfo gpu = GpuInfo.NONE;
      private ImmutableMap<Long, CounterInfo> counters;
      private ImmutableListMultimap<String, CounterInfo> countersByName;
      public final TrackConfig.Builder tracks = new TrackConfig.Builder();

      public Builder(QueryEngine qe) {
        this.qe = qe;
      }

      public TimeSpan getTraceTime() {
        return traceTime;
      }

      public Builder setTraceTime(TimeSpan traceTime) {
        this.traceTime = traceTime;
        return this;
      }

      public int getNumCpus() {
        return numCpus;
      }

      public Builder setNumCpus(int numCpus) {
        this.numCpus = numCpus;
        return this;
      }

      public ImmutableMap<Long, ProcessInfo> getProcesses() {
        return processes;
      }

      public Builder setProcesses(ImmutableMap<Long, ProcessInfo> processes) {
        this.processes = processes;
        return this;
      }

      public ImmutableMap<Long, ThreadInfo> getThreads() {
        return threads;
      }

      public Builder setThreads(ImmutableMap<Long, ThreadInfo> threads) {
        this.threads = threads;
        return this;
      }

      public GpuInfo getGpu() {
        return gpu;
      }

      public Builder setGpu(GpuInfo gpu) {
        this.gpu = gpu;
        return this;
      }

      public ImmutableMap<Long, CounterInfo> getCounters() {
        return counters;
      }

      public ImmutableListMultimap<String, CounterInfo> getCountersByName() {
        return countersByName;
      }

      public Builder setCounters(ImmutableMap<Long, CounterInfo> counters) {
        this.counters = counters;
        this.countersByName = counters.values().stream()
            .collect(toImmutableListMultimap(c -> c.name, identity()));
        return this;
      }

      public Data build() {
        return new Data(qe, traceTime, numCpus, processes, threads, gpu, counters, tracks.build());
      }
    }
  }

  public static interface Listener extends Events.Listener {
    /**
     * Event indicating progress on loading Perfetto data.
     *
     * @param msg message communicating the currently executed work.
     */
    public default void onPerfettoLoadingStatus(Loadable.Message msg) { /* empty */ }
    /**
     * Event indicating that the Perfetto trace has finished loading.
     *
     * @param error the loading error or {@code null} if loading was successful.
     */
    public default void onPerfettoLoaded(Loadable.Message error) { /* empty */ }
  }
}

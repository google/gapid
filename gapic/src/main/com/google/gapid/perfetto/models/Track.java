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

import static com.google.gapid.util.MoreFutures.logFailure;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static com.google.gapid.util.Scheduler.EXECUTOR;
import static java.util.concurrent.TimeUnit.MICROSECONDS;
import static java.util.concurrent.TimeUnit.MILLISECONDS;

import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.views.State;

import java.util.concurrent.Semaphore;
import java.util.concurrent.atomic.AtomicReference;
import java.util.logging.Logger;

// Note on multi-threading issues here:
// Because of how the window tables work, the below computeData(..) calls have to be serialized
// by track. That is, data for different requests can not be fetched in parallel. Thus, the calls
// to computeData(..) are funneled through the getDataLock semaphore. Care needs to be taken not to
// block the executor threads indefinitely, as otherwise a deadlock could occur, due to the results
// of the query futures no longer being able to be executed. Thus, the semaphore is try-acquired
// with a short timeout, followed by a slightly longer wait, before retrying.

/**
 * A {@link Track} is responsible for loading the data to be shown in the UI.
 */
public abstract class Track<D extends Track.Data> {
  private static final Logger LOG = Logger.getLogger(Track.class.getName());

  private static final long REQUEST_DELAY_MS = 50;
  private static final long ACQUIRE_TIMEOUT_MS = 5;
  private static final long ACQUIRE_RETRY_MS = 10;

  private final String trackId;

  private D data;
  private ListenableFuture<?> scheduledFuture;
  // Set to null on any thread, set to non-null only on the UI thread.
  private final AtomicReference<DataRequest> scheduledRequest =
      new AtomicReference<DataRequest>(null);
  private final Semaphore getDataLock = new Semaphore(1);
  private boolean initialized; // guarded by getDataLock

  public Track(String trackId) {
    this.trackId = trackId.replace("-", "_");
  }

  public String getId() {
    return trackId;
  }

  // on UI Thread
  public D getData(State state, Runnable repainter) {
    if (checkScheduledRequest(state) && (data == null || !data.request.satisfies(state))) {
      schedule(state, repainter);
    }
    return data;
  }

  // on UI Thread. returns true, if a new request may be scheduled.
  private boolean checkScheduledRequest(State state) {
    DataRequest scheduled = scheduledRequest.get();
    if (scheduled == null) {
      return true;
    } else if (scheduled.satisfies(state)) {
      return false;
    }

    scheduledFuture.cancel(true);
    scheduledFuture = null;
    scheduledRequest.set(null);
    return true;
  }

  // on UI Thread
  private void schedule(State state, Runnable repainter) {
    DataRequest request = DataRequest.from(state);
    scheduledRequest.set(request);
    scheduledFuture = EXECUTOR.schedule(
        () -> getData(state, request, repainter), REQUEST_DELAY_MS, MILLISECONDS);
  }

  // *not* on UI Thread
  private void getData(State state, DataRequest req, Runnable repainter) {
    try {
      if (!getDataLock.tryAcquire(ACQUIRE_TIMEOUT_MS, MILLISECONDS)) {
        logFailure(LOG, EXECUTOR.schedule(
            () -> getData(state, req, repainter), ACQUIRE_RETRY_MS, MILLISECONDS));
        return;
      }
    } catch (InterruptedException e) {
      // We were cancelled while waiting on the lock.
      scheduledRequest.compareAndSet(req, null);
      return;
    }

    if (scheduledRequest.get() != req) {
      getDataLock.release();
      return;
    }

    try {
      ListenableFuture<D> future =
          transformAsync(setup(state), $ -> computeData(state.getQueryEngine(), req));
      state.thenOnUiThread(future, newData -> update(req, newData, repainter));
      // Always unlock when the future completes/fails/is cancelled.
      future.addListener(getDataLock::release, EXECUTOR);
    } catch (RuntimeException e) {
      getDataLock.release();
      throw e;
    }
  }

  // on UI Thread
  private void update(DataRequest req, D newData, Runnable repainter) {
    if (scheduledRequest.compareAndSet(req, null)) {
      data = newData;
      scheduledFuture = null;
      repainter.run();
    }
  }

  private ListenableFuture<?> setup(State state) {
    if (initialized) {
      return Futures.immediateFuture(null);
    }
    return transform(initialize(state.getQueryEngine()), $ -> initialized = true);
  }

  protected abstract ListenableFuture<?> initialize(QueryEngine qe);
  protected abstract ListenableFuture<D> computeData(QueryEngine qe, DataRequest req);

  protected String tableName(String prefix) {
    return prefix + "_" + trackId;
  }

  public static class Data {
    public final DataRequest request;

    public Data(DataRequest request) {
      this.request = request;
    }
  }

  public static class DataRequest {
    public final TimeSpan range;
    public final long resolution;

    public DataRequest(TimeSpan range, long resolution) {
      this.range = range;
      this.resolution = resolution;
    }

    public static DataRequest from(State state) {
      TimeSpan range = state.getVisibleTime();
      return new DataRequest(range.expand(range.getDuration()), state.getResolution());
    }

    public boolean satisfies(State state) {
      return resolution == state.getResolution() && range.contains(state.getVisibleTime());
    }

    @Override
    public String toString() {
      return "Request{start: " + range.start + ", end: " + range.end + ", res: " + resolution + "}";
    }
  }

  public static class Window {
    private static final long RESOLUTION_QUANTIZE_CUTOFF = MICROSECONDS.toNanos(80);

    private static final String UPDATE_SQL = "update %s set " +
        "window_start = %d, window_dur = %d, quantum = %d where rowid = 0";
    public final long start;
    public final long end;
    public final boolean quantized;
    public final long bucketSize;

    private Window(long start, long end, boolean quantized, long bucketSize) {
      this.start = start;
      this.end = end;
      this.quantized = quantized;
      this.bucketSize = bucketSize;
    }

    public static Window compute(DataRequest request) {
      return new Window(request.range.start, request.range.end, false, 0);
    }

    public static Window compute(DataRequest request, int bucketSizePx) {
      if (request.resolution >= RESOLUTION_QUANTIZE_CUTOFF) {
        return quantized(request, bucketSizePx);
      } else {
        return compute(request);
      }
    }

    public static Window quantized(DataRequest request, int bucketSizePx) {
      long quantum = request.resolution * bucketSizePx;
      long start = (request.range.start / quantum) * quantum;
      return new Window(start, request.range.end, true, quantum);
    }

    public int getNumberOfBuckets() {
      return (int)((end - start + bucketSize - 1) / bucketSize);
    }

    public ListenableFuture<?> update(QueryEngine qe, String name) {
      return qe.query(String.format(
          UPDATE_SQL, name, start, Math.max(1, end - start), bucketSize));
    }

    @Override
    public String toString() {
      return "window{start: " + start + ", end: " + end +
          (quantized ? ", " + getNumberOfBuckets() : "") + "}";
    }
  }
}

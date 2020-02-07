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

import com.google.common.cache.Cache;
import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.util.Caches;

import java.util.List;
import java.util.concurrent.Semaphore;
import java.util.concurrent.atomic.AtomicReference;
import java.util.function.Consumer;
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

  public static final long QUANTIZE_CUT_OFF = 2000;

  private static final long REQUEST_DELAY_MS = 50;
  private static final long ACQUIRE_TIMEOUT_MS = 5;
  private static final long ACQUIRE_RETRY_MS = 10;
  private static final long PAGE_SIZE = 3600;

  private static DataCache cache = new DataCache();

  private final String trackId;

  private D data;
  private ListenableFuture<?> scheduledFuture;
  // Set to null on any thread, set to non-null only on the UI thread.
  private final AtomicReference<ScheduledRequest<D>> scheduledRequest =
      new AtomicReference<ScheduledRequest<D>>(null);
  private final Semaphore getDataLock = new Semaphore(1);
  private boolean initialized; // guarded by getDataLock

  public Track(String trackId) {
    this.trackId = trackId.replace("-", "_");
  }

  public String getId() {
    return trackId;
  }

  // on UI Thread
  public D getData(DataRequest req, OnUiThread<D> onUiThread) {
    if (checkScheduledRequest(req, onUiThread) && (data == null || !data.request.satisfies(req))) {
      schedule(req.pageAlign(), onUiThread);
    }
    return data;
  }

  // on UI Thread. returns true, if a new request may be scheduled.
  private boolean checkScheduledRequest(DataRequest req, OnUiThread<D> callback) {
    ScheduledRequest<D> scheduled = scheduledRequest.get();
    if (scheduled == null) {
      return true;
    } else if (scheduled.satisfies(req)) {
      scheduled.addCallback(callback);
      return false;
    }

    scheduledFuture.cancel(true);
    scheduledFuture = null;
    scheduledRequest.set(null);
    return true;
  }

  // on UI Thread
  private void schedule(DataRequest request, OnUiThread<D> onUiThread) {
    D newData = cache.getIfPresent(this, request);
    if (newData != null) {
      data = newData;
      return;
    }

    ScheduledRequest<D> scheduled = new ScheduledRequest<D>(request, onUiThread);
    scheduledRequest.set(scheduled);
    scheduledFuture = EXECUTOR.schedule(
        () -> query(scheduled), REQUEST_DELAY_MS, MILLISECONDS);
  }

  // *not* on UI Thread
  private void query(ScheduledRequest<D> scheduled) {
    try {
      if (!getDataLock.tryAcquire(ACQUIRE_TIMEOUT_MS, MILLISECONDS)) {
        logFailure(LOG, EXECUTOR.schedule(
            () -> query(scheduled), ACQUIRE_RETRY_MS, MILLISECONDS));
        return;
      }
    } catch (InterruptedException e) {
      // We were cancelled while waiting on the lock.
      scheduledRequest.compareAndSet(scheduled, null);
      return;
    }

    if (scheduledRequest.get() != scheduled) {
      getDataLock.release();
      return;
    }

    try {
      ListenableFuture<D> future = transformAsync(setup(), $ -> computeData(scheduled.request));
      scheduled.scheduleCallbacks(future, newData -> update(scheduled, newData));
      // Always unlock when the future completes/fails/is cancelled.
      future.addListener(getDataLock::release, EXECUTOR);
    } catch (RuntimeException e) {
      getDataLock.release();
      throw e;
    }
  }

  // on UI Thread
  private void update(ScheduledRequest<D> scheduled, D newData) {
    cache.put(this, scheduled.request, newData);
    if (scheduledRequest.compareAndSet(scheduled, null)) {
      data = newData;
      scheduledFuture = null;
    }
  }

  private ListenableFuture<?> setup() {
    if (initialized) {
      return Futures.immediateFuture(null);
    }
    return transform(initialize(), $ -> initialized = true);
  }

  protected abstract ListenableFuture<?> initialize();
  protected abstract ListenableFuture<D> computeData(DataRequest req);

  protected String tableName(String prefix) {
    return prefix + "_" + trackId;
  }

  public static interface OnUiThread<T> {
    /**
     * Runs the consumer with the result of the given future on the UI thread.
     */
    public void onUiThread(ListenableFuture<T> future, Consumer<T> callback);
    public void repaint();
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

    public DataRequest pageAlign() {
      return new DataRequest(range.align(PAGE_SIZE * resolution), resolution);
    }

    public boolean satisfies(DataRequest other) {
      return resolution == other.resolution && range.contains(other.range);
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

  public abstract static class WithQueryEngine<D extends Track.Data> extends Track<D> {
    protected final QueryEngine qe;

    public WithQueryEngine(QueryEngine qe, String trackId) {
      super(trackId);
      this.qe = qe;
    }
  }

  private static class ScheduledRequest<D extends Track.Data> {
    public final DataRequest request;
    private final List<OnUiThread<D>> callbacks;

    public ScheduledRequest(DataRequest request, OnUiThread<D> callback) {
      this.request = request;
      this.callbacks = Lists.newArrayList(callback);
    }

    public boolean satisfies(DataRequest req) {
      return request.satisfies(req);
    }

    // Only on UI thread.
    public void addCallback(OnUiThread<D> callback) {
      callbacks.add(callback);
    }

    // Not on UI thread.
    public void scheduleCallbacks(ListenableFuture<D> future, Consumer<D> update) {
      // callbacks.get(0) is safe since we only ever append to the list.
      callbacks.get(0).onUiThread(future, data -> {
        update.accept(data);
        for (OnUiThread<D> callback : callbacks) {
          callback.repaint();
        }
      });
    }
  }

  private static class DataCache {
    private final Cache<Key, Object> dataCache = Caches.softCache();

    public DataCache() {
    }

    @SuppressWarnings("unchecked")
    public <D extends Track.Data> D getIfPresent(Track<D> track, DataRequest req) {
      return (D)dataCache.getIfPresent(new Key(track, req));
    }

    public <D extends Track.Data> void put(Track<D> track, DataRequest req, D data) {
      dataCache.put(new Key(track, req), data);
    }

    private static class Key {
      private final Track<?> track;
      private final long resolution;
      private final long start;
      private final long end;
      private final int h;

      public Key(Track<?> track, DataRequest req) {
        this.track = track;
        this.resolution = req.resolution;
        this.start = req.range.start;
        this.end = req.range.end;
        this.h = ((track.hashCode() * 31 + Long.hashCode(resolution)) * 31 +
            Long.hashCode(start)) + Long.hashCode(end);
      }

      @Override
      public int hashCode() {
        return h;
      }

      @Override
      public boolean equals(Object obj) {
        if (obj == this) {
          return true;
        } else if (!(obj instanceof Key)) {
          return false;
        }
        Key o = (Key)obj;
        return track == o.track && resolution == o.resolution && start == o.start && end == o.end;
      }
    }
  }
}

/*
 * Copyright (C) 2017 Google Inc.
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
package com.google.gapid.util;

import static com.google.gapid.util.Scheduler.EXECUTOR;

import com.google.common.cache.Cache;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;

import java.util.function.Function;
import java.util.function.Predicate;

public class FutureCache<K, V> {
  private final Cache<K, V> cache;
  private final Function<K, ListenableFuture<V>> fetcher;
  private final Predicate<V> shouldCache;

  public FutureCache(
      Cache<K, V> cache, Function<K, ListenableFuture<V>> fetcher, Predicate<V> shouldCache) {
    this.cache = cache;
    this.fetcher = fetcher;
    this.shouldCache = shouldCache;
  }

  public static <K, V> FutureCache<K, V> softCache(
      Function<K, ListenableFuture<V>> fetcher, Predicate<V> shouldCache) {
    return new FutureCache<K, V>(Caches.softCache(), fetcher, shouldCache);
  }

  public static <K, V> FutureCache<K, V> hardCache(
      Function<K, ListenableFuture<V>> fetcher, Predicate<V> shouldCache) {
    return new FutureCache<K, V>(Caches.hardCache(), fetcher, shouldCache);
  }

  public ListenableFuture<V> get(K key) {
    // Look up the value in the cache using the executor.
    ListenableFuture<V> cacheLookUp = EXECUTOR.submit(() -> cache.getIfPresent(key));
    return MoreFutures.transformAsync(cacheLookUp, fromCache -> {
      if (fromCache != null) {
        return Futures.immediateFuture(fromCache);
      }

      return MoreFutures.transform(fetcher.apply(key), value -> {
        if (shouldCache.test(value)) {
          cache.put(key, value);
        }
        return value;
      });
    });
  }

  public V getIfPresent(K key) {
    return cache.getIfPresent(key);
  }

  public void clear() {
    cache.invalidateAll();
  }
}


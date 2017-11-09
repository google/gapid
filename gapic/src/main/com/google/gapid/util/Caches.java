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

import com.google.common.cache.Cache;
import com.google.common.cache.CacheBuilder;
import com.google.common.util.concurrent.UncheckedExecutionException;

import java.util.concurrent.Callable;
import java.util.concurrent.ExecutionException;

/**
 * Utilities for {@link Cache} instances.
 */
public class Caches {
  private Caches() {
  }

  public static <K, V> Cache<K, V> softCache() {
    return CacheBuilder.newBuilder().softValues().build();
  }

  public static <K, V> Cache<K, V> hardCache() {
    return CacheBuilder.newBuilder().build();
  }

  /**
   * Calls and returns the result of {@link Cache#get(Object, Callable)}, where the loader
   * {@link Callable} is guaranteed not to throw a checked exception. Unchecked exceptions are
   * still propagated via the {@link UncheckedExecutionException}.
   */
  public static <K, V> V getUnchecked(Cache<K, V> cache, K key, Callable<V> loader) {
    try {
      return cache.get(key, loader);
    } catch (ExecutionException e) {
      throw new AssertionError(e);
    }
  }
}

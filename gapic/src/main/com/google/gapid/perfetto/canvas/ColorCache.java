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
package com.google.gapid.perfetto.canvas;

import com.google.common.cache.Cache;
import com.google.common.cache.CacheBuilder;

import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.Device;
import org.eclipse.swt.graphics.RGBA;

import java.util.concurrent.ExecutionException;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Caches {@link Color} instances.
 */
public class ColorCache {
  private static final Logger LOG = Logger.getLogger(ColorCache.class.getName());

  private final Device device;
  private final Cache<RGBA, Color> cache = CacheBuilder.newBuilder()
      .maximumSize(1000)
      .recordStats()
      .<RGBA, Color>removalListener(e -> e.getValue().dispose())
      .build();

  public ColorCache(Device device) {
    this.device = device;
  }

  public Color get(RGBA rgba) {
    try {
      return cache.get(rgba, () -> new Color(device, rgba));
    } catch (ExecutionException e) {
      throw new RuntimeException(e.getCause());
    }
  }

  public void dispose() {
    LOG.log(Level.FINE, "Color cache stats: {0}", cache.stats());
    cache.invalidateAll();
  }
}

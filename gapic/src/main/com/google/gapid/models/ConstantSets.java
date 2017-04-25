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
package com.google.gapid.models;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.FutureCache;
import com.google.gapid.util.Paths;

import java.util.List;

public class ConstantSets {
  private final FutureCache<Path.ConstantSet, Service.ConstantSet> cache;

  public ConstantSets(Client client) {
    this.cache = FutureCache.hardCache(
        path -> Futures.transform(client.get(Paths.any(path)), Service.Value::getConstantSet),
        result -> result.getConstantsCount() != 0);
  }

  public ListenableFuture<Service.ConstantSet> loadConstants(Path.ConstantSet path) {
    return cache.get(path);
  }

  public ListenableFuture<List<Service.ConstantSet>> loadConstants(Service.Command cmd) {
    List<ListenableFuture<Service.ConstantSet>> sets = Lists.newArrayList();
    for (Service.Parameter param : cmd.getParametersList()) {
      if (param.hasConstants()) {
        sets.add(cache.get(param.getConstants()));
      }
    }
    return Futures.allAsList(sets);
  }

  public Service.ConstantSet getConstants(Path.ConstantSet path) {
    return cache.getIfPresent(path);
  }
}

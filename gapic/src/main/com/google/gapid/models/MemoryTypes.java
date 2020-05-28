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

import static com.google.gapid.util.MoreFutures.combine;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static com.google.gapid.util.Paths.memoryAsType;
import static com.google.gapid.util.Paths.type;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Memory.StructNode;
import com.google.gapid.models.Memory.StructObservation;
import com.google.gapid.proto.device.Device.Instance;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.proto.service.types.TypeInfo;
import com.google.gapid.server.Client;
import com.google.gapid.util.FutureCache;

import java.util.List;
import java.util.function.Function;
import java.util.function.Predicate;
import java.util.logging.Logger;

public class MemoryTypes {
  protected static final Logger LOG = Logger.getLogger(MemoryTypes.class.getName());

  protected final FutureCache<Path.Type, TypeInfo.Type> cache;
  protected final ConstantSets constants;
  private final Client client;

  public MemoryTypes(Client client, Devices devices, ConstantSets constants) {
    this.constants = constants;
    this.client = client;

    Function<Path.Type, ListenableFuture<TypeInfo.Type>> fetcher = path -> {
      return transform(client.get(type(path), devices.getReplayDevicePath()),
          Service.Value::getType);
    };
    // TODO: Currently cache all. Please think about a good cache predicate condition.
    Predicate<TypeInfo.Type> shouldCache = result -> {
      return true;
    };
    this.cache = FutureCache.hardCache(fetcher, shouldCache);

    devices.addListener(new Devices.Listener() {
      @Override
      public void onReplayDeviceChanged(Instance dev) {
        cache.clear();
      }
    });
  }

  /**
   * From a type path, Load the direct type and possible children types, save them to the cache.
   * Make sure the whole type tree is loaded.
   */
  public ListenableFuture<Void> loadTypes(Path.Type path) {
    return transformAsync(cache.get(path), intermediate -> {
      switch (intermediate.getTyCase()) {
        case SLICE:
          return loadTypes(type(intermediate.getSlice().getUnderlying(), path.getAPI()));
        case STRUCT:
          List<ListenableFuture<Void>> structChildrenTypes = Lists.newArrayList();
          for (TypeInfo.StructField childField : intermediate.getStruct().getFieldsList()) {
            structChildrenTypes.add(loadTypes(type(childField.getType(), path.getAPI())));
          }
          return combine(structChildrenTypes, $ -> null);
        case ARRAY:
          return loadTypes(type(intermediate.getArray().getElementType(), path.getAPI()));
        case PSEUDONYM:
          return loadTypes(type(intermediate.getPseudonym().getUnderlying(), path.getAPI()));
        case ENUM:
          return transform(constants.loadConstants(intermediate.getEnum()), $ -> null);
        default:
          return Futures.immediateFuture(null);
      }
    });
  }

  /**
   * Get the decoded TypeInfo.Type from cache. Call this function after relavent data is already
   * loaded into cache through method loadTypes.
   */
  public TypeInfo.Type getType(Path.Type path) {
    return cache.getIfPresent(path);
  }

  /**
   * Request the server to decode the lightweight StructObservation, return a new StructNode
   * containing relevant information.
   */
  public ListenableFuture<StructNode> loadStructNode(StructObservation structOb) {
    return transform(loadTypes(structOb.getRange().getType()), $ ->
        new StructNode(structOb.getRange().getType().getAPI(),
            getType(structOb.getRange().getType()),
            structOb.getRange().getValue(),
            structOb.range.getRoot(),
            this));
  }

  /**
   * Request and decode for all the struct observations in an array.
   */
  public ListenableFuture<List<StructNode>> loadStructNodes(StructObservation[] structObs) {
    List<ListenableFuture<StructNode>> nodes = Lists.newArrayList();
    for (StructObservation structOb : structObs) {
      nodes.add(loadStructNode(structOb));
    }
    return Futures.allAsList(nodes);
  }
}

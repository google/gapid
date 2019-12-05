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

import static com.google.gapid.util.Paths.constantSet;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.core.pod.Pod;
import com.google.gapid.proto.device.Device.Instance;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.box.Box;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.proto.service.types.TypeInfo;
import com.google.gapid.server.Client;
import com.google.gapid.util.FutureCache;
import com.google.gapid.util.MoreFutures;
import com.google.gapid.util.Pods;

import java.util.List;

public class ConstantSets {
  protected final FutureCache<Path.ConstantSet, Service.ConstantSet> cache;

  public ConstantSets(Client client, Devices devices) {
    this.cache = FutureCache.hardCache(
        path -> MoreFutures.transform(client.get(constantSet(path), devices.getReplayDevicePath()),
            Service.Value::getConstantSet),
        result -> result.getConstantsCount() != 0);

    devices.addListener(new Devices.Listener() {
      @Override
      public void onReplayDeviceChanged(Instance dev) {
        cache.clear();
      }
    });
  }

  public ListenableFuture<Service.ConstantSet> loadConstants(Path.ConstantSet path) {
    return cache.get(path);
  }

  public ListenableFuture<List<Service.ConstantSet>> loadConstants(API.Command cmd) {
    List<ListenableFuture<Service.ConstantSet>> sets = Lists.newArrayList();
    for (API.Parameter param : cmd.getParametersList()) {
      if (param.hasConstants()) {
        sets.add(cache.get(param.getConstants()));
      }
    }
    return Futures.allAsList(sets);
  }

  public ListenableFuture<Service.ConstantSet> loadConstants(Service.StateTreeNode node) {
    if (!node.hasConstants()) {
      return Futures.immediateFuture(null);
    }
    return loadConstants(node.getConstants());
  }

  public ListenableFuture<Service.ConstantSet> loadConstants(TypeInfo.EnumType enumType) {
    if (!enumType.hasConstants() || !enumType.getConstants().hasAPI()) {
      return Futures.immediateFuture(null);
    }
    return loadConstants(enumType.getConstants());
  }

  public Service.ConstantSet getConstants(Path.ConstantSet path) {
    return cache.getIfPresent(path);
  }

  public static Service.Constant find(Service.ConstantSet constants, Box.Value value) {
    if (value.getValCase() != Box.Value.ValCase.POD) {
      return Service.Constant.getDefaultInstance();
    }
    return find(constants, value.getPod());
  }

  public static Service.Constant find(Service.ConstantSet constants, Pod.Value value) {
    if (!Pods.mayBeConstant(value)) {
      return Service.Constant.getDefaultInstance();
    }
    long numValue = Pods.getConstant(value);
    for (Service.Constant constant : constants.getConstantsList()) {
      if (constant.getValue() == numValue) {
        return constant;
      }
    }
    return Service.Constant.getDefaultInstance();
  }
}

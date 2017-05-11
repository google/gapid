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
package com.google.gapid.server;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.service.GapidGrpc;
import com.google.gapid.proto.service.Service;
import com.google.gapid.util.FutureCache;

/**
 * A caching {@link GapidClientGrpc}.
 */
public class GapidClientCache extends GapidClientGrpc {
  private final FutureCache<Service.GetRequest, Service.GetResponse> getCache;
  private final FutureCache<Service.FollowRequest, Service.FollowResponse> followCache;

  public GapidClientCache(GapidGrpc.GapidFutureStub client, GapidGrpc.GapidStub stub) {
    super(client, stub);
    this.getCache = FutureCache.softCache(
        client::get, result -> result.getResCase() == Service.GetResponse.ResCase.VALUE);
    this.followCache = FutureCache.softCache(
        client::follow, result -> result.getResCase() == Service.FollowResponse.ResCase.PATH);
  }

  @Override
  public ListenableFuture<Service.GetResponse> get(Service.GetRequest request) {
    return getCache.get(request);
  }

  @Override
  public ListenableFuture<Service.FollowResponse> follow(Service.FollowRequest request) {
    return followCache.get(request);
  }
}

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

import static com.google.gapid.util.Scheduler.EXECUTOR;

import com.google.common.cache.Cache;
import com.google.common.cache.CacheBuilder;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.service.GapidGrpc;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.FollowRequest;
import com.google.gapid.proto.service.Service.FollowResponse;
import com.google.gapid.proto.service.Service.GetRequest;
import com.google.gapid.proto.service.Service.GetResponse;

/**
 * A caching {@link GapidClientGrpc}.
 */
public class GapidClientCache extends GapidClientGrpc {
  private final RpcCache<Service.GetRequest, Service.GetResponse> getCache;
  private final RpcCache<Service.FollowRequest, Service.FollowResponse> followCache;

  public GapidClientCache(GapidGrpc.GapidFutureStub client) {
    super(client);
    this.getCache = new RpcCache<Service.GetRequest, Service.GetResponse>() {
      @Override
      protected ListenableFuture<GetResponse> fetch(GetRequest key) {
        return client.get(key);
      }

      @Override
      protected boolean isSuccessful(GetResponse result) {
        return result.getResCase() == Service.GetResponse.ResCase.VALUE;
      }
    };
    this.followCache = new RpcCache<Service.FollowRequest, Service.FollowResponse>() {
      @Override
      protected ListenableFuture<FollowResponse> fetch(FollowRequest request) {
        return client.follow(request);
      }

      @Override
      protected boolean isSuccessful(FollowResponse result) {
        return result.getResCase() == Service.FollowResponse.ResCase.PATH;
      }
    };
  }

  @Override
  public ListenableFuture<Service.GetResponse> get(Service.GetRequest request) {
    return getCache.get(request);
  }

  @Override
  public ListenableFuture<Service.FollowResponse> follow(Service.FollowRequest request) {
    return followCache.get(request);
  }

  private abstract static class RpcCache<K, V> {
    private final Cache<K, V> cache = CacheBuilder.newBuilder().softValues().build();

    public RpcCache() {
    }

    public ListenableFuture<V> get(final K request) {
      // Look up the value in the cache using the executor.
      ListenableFuture<V> cacheLookUp = EXECUTOR.submit(() -> cache.getIfPresent(request));
      return Futures.transformAsync(cacheLookUp, fromCache -> {
        if (fromCache != null) {
          return Futures.immediateFuture(fromCache);
        }
        return Futures.transform(fetch(request), fromServer -> {
          if (isSuccessful(fromServer)) {
            cache.put(request, fromServer);
          }
          return fromServer;
        });
      });
    }

    protected abstract ListenableFuture<V> fetch(K request);

    protected abstract boolean isSuccessful(V result);
  }
}

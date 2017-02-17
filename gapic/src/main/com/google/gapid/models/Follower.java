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

import static com.google.gapid.util.Paths.findState;
import static java.util.logging.Level.WARNING;

import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpclib.rpccore.Rpc;
import com.google.gapid.rpclib.rpccore.Rpc.Result;
import com.google.gapid.rpclib.rpccore.RpcException;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.Events.ListenerCollection;
import com.google.gapid.util.UiCallback;

import org.eclipse.swt.widgets.Shell;

import java.util.Objects;
import java.util.concurrent.ExecutionException;
import java.util.logging.Level;
import java.util.logging.Logger;

public class Follower {
  protected static final Logger LOG = Logger.getLogger(Follower.class.getName());
  private static final int FOLLOW_TIMEOUT_MS = 1000;

  private final Shell shell;
  private final Client client;
  private final ListenerCollection<Listener> listeners = Events.listeners(Listener.class);
  protected Path.Any lastFollowCacheRequest, lastFollowCacheResult;
  protected ListenableFuture<Path.Any> lastFollowCacheFuture = Futures.immediateFuture(null);

  public Follower(Shell shell, Client client) {
    this.shell = shell;
    this.client = client;
  }

  public void prepareFollow(Path.Any path) {
    if (path == null || Objects.equals(path, lastFollowCacheRequest)) {
      return;
    }

    lastFollowCacheRequest = path;
    lastFollowCacheFuture.cancel(true);

    // Assumes the client caches follow requests. We simply hold a reference to the last returned
    // path (via the future), to keep it from being evicted from the soft reference cache.
    lastFollowCacheFuture = client.follow(path);

    /*
    Futures.addCallback(lastFollowCacheFuture, new FutureCallback<Paths.Any>() {
      @Override
      public void onSuccess(Paths.Any result) {
        LOG.log(FINE, "Follow result: " + path + " -> " + result);
      }
      @Override
      public void onFailure(Throwable t) {
        LOG.log(FINE, "Follow failure:", t);
      }
    });
    */
  }

  public void follow(Path.Any path) {
    if (path == null) {
      return;
    }

    long started = System.currentTimeMillis();
    Rpc.listen(client.follow(path), new UiCallback<Path.Any, Path.Any>(shell, LOG) {
      @Override
      protected Path.Any onRpcThread(Result<Path.Any> result) {
        try {
          return result.get();
        } catch (RpcException | ExecutionException e) {
          // We ignore errors on follow (likely just means we couldn't follow).
          LOG.log(Level.FINE, "Follow failure:", e);
          return null;
        }
      }

      @Override
      protected void onUiThread(Path.Any result) {
        if (result == null) {
          return;
        }

        long duration = System.currentTimeMillis() - started;
        if (duration > FOLLOW_TIMEOUT_MS) {
          // We took too long to compute the follow path. It's unlikely the user still
          // expects us to actually follow the link.
          LOG.log(WARNING, "We took too long (" + duration + ") to follow " + path);
        } else {
          handleFollowResult(result);
        }
      }
    });
  }

  protected void handleFollowResult(Path.Any path) {
    switch (path.getPathCase()) {
      case MEMORY:
        listeners.fire().onMemoryFollowed(path.getMemory());
        break;
      case FIELD:
      case ARRAY_INDEX:
      case MAP_INDEX:
        if (findState(path) != null) {
          listeners.fire().onStateFollowed(path);
        } else {
          LOG.log(WARNING, "Unknown follow path result: " + path);
        }
        break;
      case STATE:
        listeners.fire().onStateFollowed(path);
        break;
      default:
        LOG.log(WARNING, "Unknown follow path result: " + path);
    }
  }

  public void gotoMemory(Path.Memory memoryPath) {
    listeners.fire().onMemoryFollowed(memoryPath);
  }

  public void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public void removeListener(Listener listener) {
    listeners.removeListener(listener);
  }

  @SuppressWarnings("unused")
  public static interface Listener extends Events.Listener {
    public default void onStateFollowed(Path.Any path) { /* empty */ }
    public default void onMemoryFollowed(Path.Memory path)  { /* empty */ }
  }
}

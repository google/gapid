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

import static com.google.gapid.util.Paths.lastCommand;
import static java.util.logging.Level.FINE;
import static java.util.logging.Level.WARNING;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.util.concurrent.FutureCallback;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.server.Client;
import com.google.gapid.server.Client.PathNotFollowableException;
import com.google.gapid.util.Events;
import com.google.gapid.util.Events.ListenerCollection;
import com.google.gapid.util.Flags;
import com.google.gapid.util.Flags.Flag;
import com.google.gapid.util.MoreFutures;
import com.google.gapid.util.ObjectStore;
import com.google.gapid.util.Paths;

import org.eclipse.swt.widgets.Shell;

import java.util.List;
import java.util.Map;
import java.util.concurrent.ExecutionException;
import java.util.function.Consumer;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Model handling link following throughout the UI.
 */
public class Follower {
  public static final Flag<Boolean> logFollowRequests =
      Flags.value("logFollowRequests", false, "Whether to log follow prefetch requests.", true);

  public static final String RESULT_NAME = "ϟ__RESULT__ϟ";

  protected static final Logger LOG = Logger.getLogger(Follower.class.getName());
  private static final int FOLLOW_TIMEOUT_MS = 1000;

  private final Shell shell;
  private final Client client;
  private final ListenerCollection<Listener> listeners = Events.listeners(Listener.class);

  public Follower(Shell shell, Client client) {
    this.shell = shell;
    this.client = client;
  }

  /**
   * Prefetches all the follow paths for the given command tree node.
   */
  public Prefetcher<String> prepare(CommandStream.Node node, Runnable onResult) {
    if (node.getData() == null || node.getCommand() == null) {
      return nullPrefetcher();
    }

    Path.Command path = lastCommand(node.getData().getCommands());
    API.Command command = node.getCommand();

    LazyMap<String, Path.Any> paths = new LazyMap<String, Path.Any>();
    List<ListenableFuture<Path.Any>> futures = Lists.newArrayList();
    for (API.Parameter p : command.getParametersList()) {
      Path.Any follow = Paths.commandField(path, p.getName());
      ListenableFuture<Path.Any> future = client.follow(follow, node.device);
      MoreFutures.addCallback(future, callback(follow, v -> paths.put(p.getName(), v), onResult));
      futures.add(future);
    }

    if (command.hasResult()) {
      Path.Any follow = Paths.commandResult(path);
      ListenableFuture<Path.Any> future = client.follow(follow, node.device);
      MoreFutures.addCallback(future, callback(follow, v -> paths.put(RESULT_NAME, v), onResult));
      futures.add(future);
    }

    return new Prefetcher<String>() {
      @Override
      public Path.Any canFollow(String follow) {
        return paths.get(follow);
      }

      @Override
      public void cancel() {
        futures.forEach(f -> f.cancel(true));
      }
    };
  }

  /**
   * Prefetches the follow path for the given API state node.
   */
  public Prefetcher<Void> prepare(ApiState.Node node, Runnable onResult) {
    if (node.getData() == null || !node.getData().hasValuePath()) {
      return nullPrefetcher();
    }

    Path.Any path = node.getData().getValuePath();

    ObjectStore<Path.Any> result = ObjectStore.create();
    ListenableFuture<Path.Any> future = client.follow(path, node.device);
    MoreFutures.addCallback(future, callback(path, v -> {
      synchronized(result) {
        result.update(v);
      }
    }, onResult));

    return new Prefetcher<Void>() {
      @Override
      public Path.Any canFollow(Void ignored) {
        return result.get();
      }

      @Override
      public void cancel() {
        future.cancel(true);
      }
    };
  }

  private static FutureCallback<Path.Any> callback(
      Path.Any follow, Consumer<Path.Any> store, Runnable onResult) {
    return new FutureCallback<Path.Any>() {
      @Override
      public void onSuccess(Path.Any result) {
        store.accept(result);
        onResult.run();

        if (logFollowRequests.get()) {
          LOG.log(FINE, "Follow result: {0} -> {1}", new Object[] { follow, result });
        }
      }

      @Override
      public void onFailure(Throwable t) {
        if (t instanceof PathNotFollowableException) {
          onResult.run();

          if (logFollowRequests.get()) {
            LOG.log(FINE, "Path {0} not followable", follow);
          }
        } else if (logFollowRequests.get()) {
          LOG.log(FINE, "Follow failure:", t);
        }
      }
    };
  }

  @SuppressWarnings("unchecked")
  public static <T> Prefetcher<T> nullPrefetcher() {
    return (Prefetcher<T>)Prefetcher.NULL;
  }

  /**
   * Requests to follow the given path and update the UI selection.
   */
  public void follow(Path.Any path, Path.Device device) {
    if (path == null) {
      return;
    }

    long started = System.currentTimeMillis();
    Rpc.listen(client.follow(path, device), new UiCallback<Path.Any, Path.Any>(shell, LOG) {
      @Override
      protected Path.Any onRpcThread(Rpc.Result<Path.Any> result) {
        try {
          return result.get();
        } catch (RpcException | ExecutionException e) {
          // We ignore errors on follow.
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
          onFollow(result);
        }
      }
    });
  }

  /**
   * Update the UI with the given follow path result.
   */
  public void onFollow(Path.Any path) {
    switch (path.getPathCase()) {
      case MEMORY:
        listeners.fire().onMemoryFollowed(path.getMemory());
        break;
      case FIELD:
      case ARRAY_INDEX:
      case MAP_INDEX:
        if (Paths.contains(path, n -> n instanceof Path.State || n instanceof Path.GlobalState)) {
          listeners.fire().onStateFollowed(path);
        } else {
          LOG.log(WARNING, "Unknown follow path result: " + path);
        }
        break;
      case STATE:
        listeners.fire().onStateFollowed(path);
        break;
      case RESOURCE_DATA:
        listeners.fire().onResourceFollowed(path.getResourceData());
        break;
      case FRAMEBUFFER_ATTACHMENT:
        listeners.fire().onFramebufferAttachmentFollowed(path.getFramebufferAttachment());
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
    /**
     * Event indicating that a link with the given path to the API state was followed.
     */
    public default void onStateFollowed(Path.Any path) { /* empty */ }

    /**
     * Event indicating that a link with the given path to a memory region was followed.
     */
    public default void onMemoryFollowed(Path.Memory path)  { /* empty */ }

    /**
     * Event indicating that a link to the given resource was followed.
     */
    public default void onResourceFollowed(Path.ResourceData path) { /* empty */ }

    /**
     * Event indicating that a link with the given path to a framebuffer attachment was followed.
     */
    public default void onFramebufferAttachmentFollowed(Path.FramebufferAttachment path) {}
  }

  public static interface Prefetcher<K> {
    public static final Prefetcher<?> NULL = new Prefetcher<Object>() {
      @Override
      public Path.Any canFollow(Object key) {
        return null;
      }

      @Override
      public void cancel() {
        // No-op.
      }
    };

    /**
     * Returns {@code null} if the item cannot be followed, or the known follow path.
     */
    public Path.Any canFollow(K key);

    public void cancel();
  }

  /**
   * Map that synchronizes access and only allocates backing storage once non-empty.
   */
  private static class LazyMap<K, V> {
    private Map<K, V> map;

    public LazyMap() {
    }

    public void put(K key, V value) {
      synchronized (this) {
        if (map == null) {
          map = Maps.newHashMap();
        }
        map.put(key, value);
      }
    }

    public V get(K key) {
      synchronized (this) {
        return (map == null) ? null : map.get(key);
      }
    }
  }
}

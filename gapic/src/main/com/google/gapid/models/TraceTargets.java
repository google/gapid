/*
 * Copyright (C) 2018 Google Inc.
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

import static com.google.gapid.rpc.UiErrorCallback.error;
import static com.google.gapid.rpc.UiErrorCallback.success;
import static com.google.gapid.util.Logging.throttleLogRpcError;
import static com.google.gapid.widgets.Widgets.submitIfNotDisposed;
import static java.util.logging.Level.WARNING;

import com.google.common.base.Preconditions;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.Rpc.Result;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.rpc.UiErrorCallback.ResultOrError;
import com.google.gapid.server.Client;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.Events;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.MoreFutures;

import org.eclipse.swt.internal.DPIUtil;
import org.eclipse.swt.widgets.Shell;

import java.util.concurrent.ExecutionException;
import java.util.function.Supplier;
import java.util.logging.Logger;

/**
 * Model responsible for loading the trace target tree of a device.
 */
public class TraceTargets
    extends ModelBase<TraceTargets.Node, Void, Loadable.Message, TraceTargets.Listener> {
  protected static final Logger LOG = Logger.getLogger(TraceTargets.class.getName());
  private static final String ROOT_URI = "";

  private final Path.Device device;
  private final float density;

  public TraceTargets(Shell shell, Analytics analytics, Client client, Path.Device device) {
    super(LOG, shell, analytics, client, Listener.class);
    this.device = device;
    this.density = DPIUtil.getDeviceZoom() / 100.0f;
  }

  @Override
  protected ListenableFuture<Node> doLoad(Void ignored) {
    return MoreFutures.transform(client.getTraceTargetTreeNode(device, ROOT_URI, density), Node::new);
  }

  @Override
  protected ResultOrError<Node, Loadable.Message> processResult(Result<Node> result) {
    try {
      return success(result.get());
    } catch (DataUnavailableException e) {
      return error(Loadable.Message.info(e));
    } catch (RpcException e) {
      LOG.log(WARNING, "Failed to load the trace target root", e);
      return error(Loadable.Message.error(e));
    } catch (ExecutionException e) {
      if (!shell.isDisposed()) {
        throttleLogRpcError(LOG, "Failed to load the trace target root", e);
      }
      return error(Loadable.Message.error("Failed to load the trace target tree"));
    }
  }

  @Override
  protected void fireLoadStartEvent() {
    // Don't care about this event.
  }

  @Override
  protected void updateError(Loadable.Message error) {
    listeners.fire().onTreeRootLoaded(error);
  }

  @Override
  protected void fireLoadedEvent() {
    listeners.fire().onTreeRootLoaded(null);
  }

  public void load() {
    if (!isLoaded()) {
      load(null, true);
    }
  }

  public ListenableFuture<Node> load(Node node) {
    return node.load(shell, () -> client.getTraceTargetTreeNode(device, node.getUri(), density));
  }

  public void load(Node node, Runnable callback) {
    ListenableFuture<Node> future = load(node);
    if (future != null) {
      Rpc.listen(future, new UiCallback<Node, Node>(shell, LOG) {
        @Override
        protected Node onRpcThread(Rpc.Result<Node> result)
            throws RpcException, ExecutionException {
          return result.get();
        }

        @Override
        protected void onUiThread(Node result) {
          callback.run();
        }
      });
    }
  }

  public static class Node {
    private final Node parent;
    private final String uri;
    private final int depth;
    private Node[] children;
    private Service.TraceTargetTreeNode data;
    private ListenableFuture<Node> loadFuture;

    public Node(Service.TraceTargetTreeNode data) {
      this(null, data.getUri());
      this.data = data;
    }

    public Node(Node parent, String uri) {
      this.parent = parent;
      this.uri = uri;
      this.depth = (parent == null) ? 0 : parent.depth + 1;
    }

    public Node getParent() {
      return parent;
    }

    public String getUri() {
      return uri;
    }

    public int getDepth() {
      return depth;
    }

    public int getChildCount() {
      return (data == null) ? 0 : data.getChildrenUrisCount();
    }

    public Node getChild(int child) {
      return getOrCreateChildren()[child];
    }

    public Node[] getChildren() {
      return getOrCreateChildren().clone();
    }

    private Node[] getOrCreateChildren() {
      if (children == null) {
        Preconditions.checkState(data != null, "Querying children before loaded");
        children = new Node[data.getChildrenUrisCount()];
        for (int i = 0; i < children.length; i++) {
          children[i] = new Node(this, data.getChildrenUris(i));
        }
      }
      return children;
    }

    public Service.TraceTargetTreeNode getData() {
      return data;
    }

    public boolean isTraceable() {
      return data != null && !data.getTraceUri().isEmpty();
    }

    public Target getTraceTarget() {
      return isTraceable() ? new Target(data.getTraceUri(), data.getFriendlyApplication()) : null;
    }

    public ListenableFuture<Node> load(
        Shell shell, Supplier<ListenableFuture<Service.TraceTargetTreeNode>> loader) {
      if (data != null) {
        // Already loaded.
        return null;
      } else if (loadFuture != null && !loadFuture.isCancelled()) {
        return loadFuture;
      }

      return loadFuture = MoreFutures.transformAsync(loader.get(), newData ->
          submitIfNotDisposed(shell, () -> {
            data = newData;
            loadFuture = null; // Don't hang on to listeners.
            return Node.this;
          }));
    }

    @Override
    public boolean equals(Object obj) {
      if (obj == this) {
        return true;
      } else if (!(obj instanceof Node)) {
        return false;
      }
      Node n = (Node)obj;
      return uri.equals(n.uri) && parent.equals(n.parent);
    }

    @Override
    public int hashCode() {
      return ((parent == null) ? 0 : parent.hashCode()) * 31 + uri.hashCode();
    }

    @Override
    public String toString() {
      return parent + "/" + uri + (data == null ? "" : " " + data.getName());
    }
  }

  public static class Target {
    public final String url;
    public final String friendlyName;

    public Target(String url, String friendlyName) {
      this.url = url;
      this.friendlyName = friendlyName;
    }
  }

  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that the trace target tree root has finished loading.
     *
     * @param error the loading error or {@code null} if loading was successful.
     */
    public default void onTreeRootLoaded(Loadable.Message error) { /* empty */ }
  }
}

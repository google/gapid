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

import static com.google.gapid.rpc.UiErrorCallback.error;
import static com.google.gapid.rpc.UiErrorCallback.success;
import static com.google.gapid.util.Logging.throttleLogRpcError;
import static com.google.gapid.util.Paths.stateTree;
import static com.google.gapid.widgets.Widgets.submitIfNotDisposed;
import static java.util.logging.Level.WARNING;

import com.google.common.base.Preconditions;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.AtomStream.AtomIndex;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.rpc.UiErrorCallback.ResultOrError;
import com.google.gapid.server.Client;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.Events;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.ObjectStore;
import com.google.gapid.util.Paths;

import org.eclipse.swt.widgets.Shell;

import java.util.concurrent.ExecutionException;
import java.util.function.Supplier;
import java.util.logging.Logger;

/**
 * Model managing the API state object of the currently selected command.
 */
public class ApiState
    extends ModelBase.ForPath<ApiState.Node, Loadable.Message, ApiState.Listener> {
  protected static final Logger LOG = Logger.getLogger(ApiState.class.getName());

  private final ConstantSets constants;
  private final ObjectStore<Path.Any> selection = ObjectStore.create();

  public ApiState(
      Shell shell, Client client, Follower follower, AtomStream atoms, ApiContext contexts, ConstantSets constants) {
    super(LOG, shell, client, Listener.class);
    this.constants = constants;

    atoms.addListener(new AtomStream.Listener() {
      @Override
      public void onAtomsSelected(AtomIndex index) {
        load(stateTree(index, contexts.getSelectedContext()), false);
      }
    });
    follower.addListener(new Follower.Listener() {
      @Override
      public void onStateFollowed(Path.Any path) {
        selectPath(path, true);
      }
    });
  }

  @Override
  protected ListenableFuture<Node> doLoad(Path.Any path) {
    return Futures.transformAsync(client.get(path),
        tree -> Futures.transform(client.get(Paths.toAny(tree.getStateTree().getRoot())),
            val -> new RootNode(
                tree.getStateTree().getRoot().getTree(), val.getStateTreeNode())));
  }

  @Override
  protected ResultOrError<Node, Loadable.Message> processResult(Rpc.Result<Node> result) {
    try {
      return success(result.get());
    } catch (DataUnavailableException e) {
      return error(Loadable.Message.info(e));
    } catch (RpcException e) {
      LOG.log(WARNING, "Failed to load the API state", e);
      return error(Loadable.Message.error(e));
    } catch (ExecutionException e) {
      if (!shell.isDisposed()) {
        throttleLogRpcError(LOG, "Failed to load the API state", e);
      }
      return error(Loadable.Message.error("Failed to load the state"));
    }
  }

  @Override
  protected void updateError(Loadable.Message error) {
    listeners.fire().onStateLoaded(error);
  }

  @Override
  protected void fireLoadStartEvent() {
    listeners.fire().onStateLoadingStart();
  }

  @Override
  protected void fireLoadedEvent() {
    listeners.fire().onStateLoaded(null);
  }

  public ListenableFuture<Node> load(Node node) {
    return node.load(shell, () -> Futures.transformAsync(
        client.get(Paths.toAny(node.getPath(Path.StateTreeNode.newBuilder()))),
        value -> Futures.transform(constants.loadConstants(value.getStateTreeNode()),
            ignore -> new NodeData(value.getStateTreeNode()))));
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

  public Path.Any getSelectedPath() {
    return selection.get();
  }

  public ListenableFuture<Path.StateTreeNode> getResolvedSelectedPath() {
   return resolve(selection.get());
  }

  public void selectPath(Path.Any path, boolean force) {
    if (selection.update(path) || force) {
      listeners.fire().onStateSelected(path);
    }
  }

  public ListenableFuture<Path.StateTreeNode> resolve(Path.Any path) {
    if (path == null || !isLoaded()) {
      return Futures.immediateFuture(Path.StateTreeNode.getDefaultInstance());
    } else if (path.getPathCase() == Path.Any.PathCase.STATE_TREE_NODE) {
      return Futures.immediateFuture(path.getStateTreeNode());
    }

    return Futures.transform(client.get(Paths.stateTree(((RootNode)getData()).tree, path)),
        value -> value.getPath().getStateTreeNode());
  }

  public static class Node {
    private final Node parent;
    private final int index;
    private Node[] children;
    private Service.StateTreeNode data;
    private ListenableFuture<Node> loadFuture;

    public Node(Service.StateTreeNode data) {
      this(null, 0);
      this.data = data;
    }

    public Node(Node parent, int index) {
      this.parent = parent;
      this.index = index;
    }

    public Node getParent() {
      return parent;
    }

    public int getChildCount() {
      return (data == null) ? 0 : (int)data.getNumChildren();
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
        children = new Node[(int)data.getNumChildren()];
        for (int i = 0; i < children.length; i++) {
          children[i] = new Node(this, i);
        }
      }
      return children;
    }

    public Service.StateTreeNode getData() {
      return data;
    }

    public Path.StateTreeNode.Builder getPath(Path.StateTreeNode.Builder path) {
      return parent.getPath(path).addIndices(index);
    }

    public ListenableFuture<Node> load(Shell shell, Supplier<ListenableFuture<NodeData>> loader) {
      if (data != null) {
        // Already loaded.
        return null;
      } else if (loadFuture != null && !loadFuture.isCancelled()) {
        return loadFuture;
      }

      return loadFuture = Futures.transformAsync(loader.get(), newData ->
          submitIfNotDisposed(shell, () -> {
            data = newData.data;
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
      return index == n.index && parent.equals(n.parent);
    }

    @Override
    public int hashCode() {
      return parent.hashCode() * 31 + index;
    }

    @Override
    public String toString() {
      return parent.toString() + "/" + index + (data == null ? "" : " " + data.getName());
    }
  }

  private static class RootNode extends Node {
    public final Path.ID tree;

    public RootNode(Path.ID tree, Service.StateTreeNode data) {
      super(data);
      this.tree = tree;
    }

    @Override
    public Path.StateTreeNode.Builder getPath(Path.StateTreeNode.Builder path) {
      return path.setTree(tree);
    }

    @Override
    public String toString() {
      return "Root";
    }

    @Override
    public boolean equals(Object obj) {
      if (obj == this) {
        return true;
      } else if (!(obj instanceof RootNode)) {
        return false;
      }
      return tree.equals(((RootNode)obj).tree);
    }

    @Override
    public int hashCode() {
      return tree.hashCode();
    }
  }

  private static class NodeData {
    public final Service.StateTreeNode data;

    public NodeData(Service.StateTreeNode data) {
      this.data = data;
    }
  }

  @SuppressWarnings("unused")
  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that the state is currently being loaded.
     */
    public default void onStateLoadingStart()  { /* empty */ }

    /**
     * Event indicating that the state has finished loading.
     *
     * @param error the loading error or {@code null} if loading was successful.
     */
    public default void onStateLoaded(Loadable.Message error) { /* empty */ }

    /**
     * Event indicating that the portion of the state that is selected has changed.
     */
    public default void onStateSelected(Path.Any path) { /* empty */ }
  }
}

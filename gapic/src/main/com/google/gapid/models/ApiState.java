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

import static com.google.gapid.util.Paths.stateAfter;

import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.AtomStream.AtomIndex;
import com.google.gapid.proto.service.Service.StateTreeNode;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpclib.futures.FutureController;
import com.google.gapid.rpclib.futures.SingleInFlight;
import com.google.gapid.rpclib.rpccore.Rpc;
import com.google.gapid.rpclib.rpccore.Rpc.Result;
import com.google.gapid.rpclib.rpccore.RpcException;
import com.google.gapid.server.Client;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.Events;
import com.google.gapid.util.Events.ListenerCollection;
import com.google.gapid.util.PathStore;
import com.google.gapid.util.Paths;
import com.google.gapid.util.UiCallback;
import com.google.gapid.util.UiErrorCallback;

import org.eclipse.swt.widgets.Shell;

import java.util.concurrent.ExecutionException;
import java.util.function.Supplier;
import java.util.logging.Logger;

/**
 * Model managing the API state object of the currently selected command.
 */
public class ApiState {
  protected static final Logger LOG = Logger.getLogger(ApiState.class.getName());

  private final Shell shell;
  private final Client client;
  private final ConstantSets constants;
  private final ListenerCollection<Listener> listeners = Events.listeners(Listener.class);
  private final FutureController rpcController = new SingleInFlight();
  private final PathStore statePath = new PathStore();
  //private final PathStore selection = new PathStore();
  private RootNode root;

  public ApiState(
      Shell shell, Client client, Follower follower, AtomStream atoms, ConstantSets constants) {
    this.shell = shell;
    this.client = client;
    this.constants = constants;

    atoms.addListener(new AtomStream.Listener() {
      @Override
      public void onAtomsSelected(AtomIndex index) {
        loadState(index);
      }
    });
    follower.addListener(new Follower.Listener() {
      @Override
      public void onStateFollowed(Path.Any path) {
        //selectPath(path, true);
      }
    });
  }

  protected void loadState(AtomIndex index) {
    if (statePath.updateIfNotNull(stateAfter(index))) {
      // we are making a request for a new state, this means our current state is old and irrelevant
      root = null;
      listeners.fire().onStateLoadingStart();
      Rpc.listen(Futures.transformAsync(client.get(statePath.getPath()),
          tree -> Futures.transform(client.get(Paths.any(tree.getStateTree().getRoot())),
              val -> new RootNode(
                  tree.getStateTree().getRoot().getTree(), val.getStateTreeNode()))),
          rpcController,
          new UiErrorCallback<RootNode, RootNode, DataUnavailableException>(shell, LOG) {
        @Override
        protected ResultOrError<RootNode, DataUnavailableException> onRpcThread(
            Rpc.Result<RootNode> result) throws RpcException, ExecutionException {
          try {
            return success(result.get());
          } catch (DataUnavailableException e) {
            return error(e);
          }
        }

        @Override
        protected void onUiThreadSuccess(RootNode result) {
          update(result);
        }

        @Override
        protected void onUiThreadError(DataUnavailableException error) {
          update(error);
        }
      });
    }
  }

  protected void update(RootNode newRoot) {
    root = newRoot;
    listeners.fire().onStateLoaded(null);
  }

  protected void update(DataUnavailableException error) {
    listeners.fire().onStateLoaded(error);
  }

  public boolean isLoaded() {
    return root != null;
  }

  public Node getRoot() {
    return root;
  }

  public ListenableFuture<Node> load(Node node) {
    return node.load(() -> Futures.transformAsync(
        client.get(Paths.any(node.getPath(Path.StateTreeNode.newBuilder()))),
        value -> Futures.transform(constants.loadConstants(value.getStateTreeNode()),
            ignore -> new NodeData(value.getStateTreeNode()))));
  }

  public void load(Node node, Runnable callback) {
    ListenableFuture<Node> future = load(node);
    if (future != null) {
      Rpc.listen(future, new UiCallback<Node, Node>(shell, LOG) {
        @Override
        protected Node onRpcThread(Result<Node> result) throws RpcException, ExecutionException {
          return result.get();
        }

        @Override
        protected void onUiThread(Node result) {
          callback.run();
        }
      });
    }
  }

  /*

  public Path.Any getSelectedPath() {
    return selection.getPath();
  }

  public void selectPath(Path.Any path, boolean force) {
    if (selection.update(path) || force) {
      listeners.fire().onStateSelected(path);
    }
  }
  */

  public void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public void removeListener(Listener listener) {
    listeners.removeListener(listener);
  }

  public static class Node {
    private final Node parent;
    private final int index;
    private Node[] children;
    private StateTreeNode data;
    private ListenableFuture<Node> loadFuture;

    public Node(StateTreeNode data) {
      this(null, 0);
      this.data = data;
      this.children = new Node[(int)data.getNumChildren()];
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
      Node node = children[child];
      if (node == null) {
        node = children[child] = new Node(this, child);
      }
      return node;
    }

    public StateTreeNode getData() {
      return data;
    }

    public Path.StateTreeNode.Builder getPath(Path.StateTreeNode.Builder path) {
      return parent.getPath(path).addIndex(index);
    }

    public ListenableFuture<Node> load(Supplier<ListenableFuture<NodeData>> loader) {
      if (data != null) {
        // Already loaded.
        return null;
      } else if (loadFuture != null) {
        return loadFuture;
      }

      return loadFuture = Futures.transform(loader.get(), newData -> {
        data = newData.data;
        children = new Node[(int)data.getNumChildren()];
        loadFuture = null; // Don't hang on to listeners.
        return Node.this;
      });
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
      return index + (data == null ? "" : " " + data.getName());
    }
  }

  private static class RootNode extends Node {
    public final Path.ID tree;

    public RootNode(Path.ID tree, StateTreeNode data) {
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
    public final StateTreeNode data;

    public NodeData(StateTreeNode data) {
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
    public default void onStateLoaded(DataUnavailableException error) { /* empty */ }

    /**
     * Event indicating that the portion of the state that is selected has changed.
     */
    public default void onStateSelected(Path.Any path) { /* empty */ }
  }
}

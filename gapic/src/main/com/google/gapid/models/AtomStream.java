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

import static com.google.gapid.proto.service.memory.Memory.PoolNames.Application_VALUE;
import static com.google.gapid.util.Paths.commandTree;
import static com.google.gapid.util.Paths.lastCommand;
import static com.google.gapid.util.Paths.observationsAfter;
import static com.google.gapid.widgets.Widgets.submitIfNotDisposed;
import static java.util.logging.Level.FINE;

import com.google.common.base.Objects;
import com.google.common.base.Preconditions;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.ApiContext.FilteringContext;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.Paths;
import com.google.gapid.util.Ranges;

import org.eclipse.swt.widgets.Shell;

import java.util.concurrent.ExecutionException;
import java.util.function.Consumer;
import java.util.function.Supplier;
import java.util.logging.Logger;

/**
 * Model containing the API commands (atoms) of the capture.
 */
public class AtomStream extends ModelBase.ForPath<AtomStream.Node, Void, AtomStream.Listener>
    implements ApiContext.Listener, Capture.Listener {
  protected static final Logger LOG = Logger.getLogger(AtomStream.class.getName());

  private final Capture capture;
  private final ApiContext context;
  private final ConstantSets constants;
  private AtomIndex selection;

  public AtomStream(
      Shell shell, Client client, Capture capture, ApiContext context, ConstantSets constants) {
    super(LOG, shell, client, Listener.class);
    this.capture = capture;
    this.context = context;
    this.constants = constants;

    capture.addListener(this);
    context.addListener(this);
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    if (!maintainState) {
      selection = null;
    }
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error == null && selection != null) {
      selection = selection.withCapture(capture.getData());
      if (isLoaded()) {
        resolve(selection.getCommand(), node -> selectAtoms(selection.withNode(node), true));
      }
    }
  }

  @Override
  public void onContextsLoaded() {
    onContextSelected(context.getSelectedContext());
  }

  @Override
  public void onContextSelected(FilteringContext ctx) {
    if (selection != null && selection.getNode() != null) {
      // Clear the node, so the selection will be re-resolved once the context has updated.
      selection = selection.withNode(null);
    }
    load(commandTree(capture.getData(), ctx), false);
  }

  @Override
  protected ListenableFuture<Node> doLoad(Path.Any path) {
    return Futures.transformAsync(client.get(path),
        tree -> Futures.transform(client.get(Paths.toAny(tree.getCommandTree().getRoot())),
            val -> new RootNode(
                tree.getCommandTree().getRoot().getTree(), val.getCommandTreeNode())));
  }

  public ListenableFuture<Node> load(Node node) {
    return node.load(shell, () -> Futures.transformAsync(
        client.get(Paths.toAny(node.getPath(Path.CommandTreeNode.newBuilder()))), v1 -> {
          Service.CommandTreeNode data = v1.getCommandTreeNode();
          if (data.getGroup().isEmpty() && data.hasCommands()) {
            return Futures.transform(
                loadCommand(lastCommand(data.getCommands())), cmd -> new NodeData(data, cmd));
          }
          return Futures.immediateFuture(new NodeData(data, null));
        }));
  }

  public ListenableFuture<API.Command> loadCommand(Path.Command path) {
    return Futures.transformAsync(client.get(Paths.toAny(path)), value ->
        Futures.transform(constants.loadConstants(value.getCommand()), ignore ->
            value.getCommand()));
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

  @Override
  protected void fireLoadStartEvent() {
    listeners.fire().onAtomsLoadingStart();
  }

  @Override
  protected void fireLoadedEvent() {
    listeners.fire().onAtomsLoaded();
    if (selection != null) {
      selectAtoms(selection, true);
    }
  }

  public AtomIndex getSelectedAtoms() {
    return (selection != null && selection.getNode() != null) ? selection : null;
  }

  public void selectAtoms(AtomIndex index, boolean force) {
    if (!force && Objects.equal(selection, index)) {
      return;
    } else if (!isLoaded()) {
      this.selection = index;
      return;
    }

    RootNode root = (RootNode)getData();
    if (index.getNode() == null) {
      resolve(index.getCommand(), node -> selectAtoms(index.withNode(node), force));
    } else if (!index.getNode().getTree().equals(root.tree)) {
      // TODO
      throw new UnsupportedOperationException("This is not yet supported, needs API clarification");
    } else {
      selection = index;
      listeners.fire().onAtomsSelected(selection);
    }
  }

  private void resolve(Path.Command command, Consumer<Path.CommandTreeNode> cb) {
    Rpc.listen(client.get(commandTree(((RootNode)getData()).tree, command)),
        new UiCallback<Service.Value, Path.CommandTreeNode>(shell, LOG) {
      @Override
      protected Path.CommandTreeNode onRpcThread(Rpc.Result<Service.Value> result)
          throws RpcException, ExecutionException {
        Service.Value value = result.get();
        LOG.log(FINE, "Resolved selection to {0}", value);
        return value.getPath().getCommandTreeNode();
      }

      @Override
      protected void onUiThread(Path.CommandTreeNode result) {
        cb.accept(result);
      }
    });
  }

  public ListenableFuture<Observation[]> getObservations(AtomIndex index) {
    return Futures.transform(client.get(observationsAfter(index, Application_VALUE)), v -> {
      Service.Memory mem = v.getMemory();
      Observation[] obs = new Observation[mem.getReadsCount() + mem.getWritesCount()];
      int idx = 0;
      for (Service.MemoryRange read : mem.getReadsList()) {
        obs[idx++] = new Observation(index, true, read);
      }
      for (Service.MemoryRange write : mem.getWritesList()) {
        obs[idx++] = new Observation(index, false, write);
      }
      return obs;
    });
  }

  /**
   * Read or write memory observation at a specific command.
   */
  public static class Observation {
    public static final Observation NULL_OBSERVATION = new Observation(null, false, null) {
      @Override
      public String toString() {
        return Messages.SELECT_OBSERVATION;
      }

      @Override
      public boolean contains(long address) {
        return false;
      }
    };

    private final AtomIndex index;
    private final boolean read;
    private final Service.MemoryRange range;

    public Observation(AtomIndex index, boolean read, Service.MemoryRange range) {
      this.index = index;
      this.read = read;
      this.range = range;
    }

    public Path.Memory getPath() {
      return Paths.memoryAfter(index, Application_VALUE, range).getMemory();
    }

    public boolean contains(long address) {
      return Ranges.contains(range, address);
    }

    @Override
    public String toString() {
      long base = range.getBase(), count = range.getSize();
      return (read ? "Read " : "Write ") + count + " byte" + (count == 1 ? "" : "s") +
          String.format(" at 0x%016x", base);
    }
  }

  /**
   * An index into the atom stream, representing a specific "point in time" in the trace.
   */
  public static class AtomIndex implements Comparable<AtomIndex> {
    private final Path.Command command;
    private final Path.CommandTreeNode node;
    private final boolean group;

    private AtomIndex(Path.Command command, Path.CommandTreeNode node, boolean group) {
      this.command = command;
      this.node = node;
      this.group = group;
    }

    /**
     * Create an index pointing to the given command and node.
     */
    public static AtomIndex forNode(Path.Command command, Path.CommandTreeNode node) {
      return new AtomIndex(command, node, false);
    }

    /**
     * Create an index pointing to the given command, without knowing the tree node.
     * The tree nodes is then resolved when it is needed.
     */
    public static AtomIndex forCommand(Path.Command command) {
      return new AtomIndex(command, null, false);
    }

    /**
     * Same as {@link #forCommand}, except that group selection is to be preferred when
     * resolving to a tree node.
     */
    public static AtomIndex forGroup(Path.Command command) {
      return new AtomIndex(command, null, true);
    }

    public AtomIndex withNode(Path.CommandTreeNode newNode) {
      return new AtomIndex(command, newNode, group);
    }

    public AtomIndex withCapture(Path.Capture capture) {
      return new AtomIndex(command.toBuilder().setCapture(capture).build(), null, group);
    }

    public Path.Command getCommand() {
      return command;
    }

    public Path.CommandTreeNode getNode() {
      return node;
    }

    public boolean isGroup() {
      return group;
    }

    @Override
    public String toString() {
      return command.getIndicesList().toString();
    }

    @Override
    public int hashCode() {
      return command.getIndicesList().hashCode();
    }

    @Override
    public boolean equals(Object obj) {
      if (obj == this) {
        return true;
      } else if (!(obj instanceof AtomIndex)) {
        return false;
      }
      return command.getIndicesList().equals(((AtomIndex)obj).command.getIndicesList());
    }

    @Override
    public int compareTo(AtomIndex o) {
      return Paths.compare(command, o.command);
    }
  }

  public static class Node {
    private final Node parent;
    private final int index;
    private Node[] children;
    private Service.CommandTreeNode data;
    private API.Command command;
    private ListenableFuture<Node> loadFuture;

    public Node(Service.CommandTreeNode data) {
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

    public boolean isLastChild() {
      return parent == null || (parent.getChildCount() - 1 == index);
    }

    public Service.CommandTreeNode getData() {
      return data;
    }

    public API.Command getCommand() {
      return command;
    }

    public Path.CommandTreeNode.Builder getPath(Path.CommandTreeNode.Builder path) {
      return parent.getPath(path).addIndices(index);
    }

    public AtomIndex getIndex() {
      return (data == null) ? null : AtomIndex.forNode(data.getRepresentation(),
          getPath(Path.CommandTreeNode.newBuilder()).build());
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
          command = newData.command;
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
      return parent + "/" + index +
          (data == null ? "" : " " + data.getGroup() + data.getCommands().getToList());
    }
  }

  private static class RootNode extends Node {
    public final Path.ID tree;

    public RootNode(Path.ID tree, Service.CommandTreeNode data) {
      super(data);
      this.tree = tree;
    }

    @Override
    public Path.CommandTreeNode.Builder getPath(Path.CommandTreeNode.Builder path) {
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
    public final Service.CommandTreeNode data;
    public final API.Command command;

    public NodeData(Service.CommandTreeNode data, API.Command command) {
      this.data = data;
      this.command = command;
    }
  }

  public interface Listener extends Events.Listener {
    /**
     * Event indicating that the tree root has changed and is being loaded.
     */
    public default void onAtomsLoadingStart() { /* emtpy */ }

    /**
     * Event indicating that the tree root has finished loading.
     */
    public default void onAtomsLoaded() { /* empty */ }

    /**
     * Event indicating that the currently selected command range has changed.
     */
    @SuppressWarnings("unused")
    public default void onAtomsSelected(AtomIndex selection) { /* empty */ }
  }
}

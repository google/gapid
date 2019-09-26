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
import static com.google.gapid.util.Paths.command;
import static com.google.gapid.util.Paths.commandTree;
import static com.google.gapid.util.Paths.lastCommand;
import static com.google.gapid.util.Paths.observationsAfter;
import static com.google.gapid.widgets.Widgets.submitIfNotDisposed;
import static java.util.logging.Level.FINE;

import com.google.common.base.Objects;
import com.google.common.base.Preconditions;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.SettableFuture;
import com.google.gapid.models.ApiContext.FilteringContext;
import com.google.gapid.proto.device.Device.Instance;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.MoreFutures;
import com.google.gapid.util.Paths;

import org.eclipse.swt.widgets.Shell;

import java.util.concurrent.ExecutionException;
import java.util.function.Consumer;
import java.util.function.Supplier;
import java.util.logging.Logger;

/**
 * Model containing the API commands of the capture.
 */
public class CommandStream
    extends DeviceDependentModel.ForPath<CommandStream.Node, Void, CommandStream.Listener>
    implements ApiContext.Listener, Capture.Listener, Devices.Listener {
  protected static final Logger LOG = Logger.getLogger(CommandStream.class.getName());

  private final Capture capture;
  private final ApiContext context;
  private final ConstantSets constants;
  private CommandIndex selection;

  public CommandStream(Shell shell, Analytics analytics, Client client, Capture capture,
      Devices devices, ApiContext context, ConstantSets constants) {
    super(LOG, shell, analytics, client, Listener.class, devices);
    this.capture = capture;
    this.context = context;
    this.constants = constants;

    capture.addListener(this);
    devices.addListener(this);
    context.addListener(this);
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    if (!maintainState) {
      selection = null;
    }
    reset();
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error == null && selection != null) {
      selection = selection.withCapture(capture.getData().path);
      if (isLoaded()) {
        resolve(selection.getCommand(), node -> selectCommands(selection.withNode(node), true));
      }
    }
  }

  @Override
  public void onReplayDeviceChanged(Instance dev) {
    if (selection != null && selection.getNode() != null) {
      // Clear the node, so the selection will be re-resolved once the context has updated.
      selection = selection.withNode(null);
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
    load(commandTree(capture.getData().path, ctx), false);
  }

  @Override
  protected ListenableFuture<Node> doLoad(Path.Any path, Path.Device device) {
    return MoreFutures.transformAsync(client.get(path, device),
        tree -> MoreFutures.transform(client.get(commandTree(tree.getCommandTree().getRoot()), device),
            val -> new RootNode(
                device, tree.getCommandTree().getRoot().getTree(), val.getCommandTreeNode())));
  }

  public ListenableFuture<Node> load(Node node) {
    return node.load(shell, () -> MoreFutures.transformAsync(
        client.get(commandTree(node.getPath(Path.CommandTreeNode.newBuilder())), node.device),
        v1 -> {
          Service.CommandTreeNode data = v1.getCommandTreeNode();
          if (data.getGroup().isEmpty() && data.hasCommands()) {
            return MoreFutures.transform(loadCommand(lastCommand(data.getCommands()), node.device),
                cmd -> new NodeData(data, cmd));
          }
          return Futures.immediateFuture(new NodeData(data, null));
        }));
  }

  public ListenableFuture<API.Command> loadCommand(Path.Command path, Path.Device device) {
    return MoreFutures.transformAsync(client.get(command(path), device), value ->
        MoreFutures.transform(constants.loadConstants(value.getCommand()), ignore ->
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

  public ListenableFuture<Service.FindResponse> search(
      CommandStream.Node parent, String text, boolean regex) {
    SettableFuture<Service.FindResponse> result = SettableFuture.create();
    client.streamSearch(searchRequest(parent, text, regex), result::set);
    return result;
  }

  private static Service.FindRequest searchRequest(
      CommandStream.Node parent, String text, boolean regex) {
    return Service.FindRequest.newBuilder()
        .setCommandTreeNode(parent.getPath(Path.CommandTreeNode.newBuilder()))
        .setText(text)
        .setIsRegex(regex)
        .setMaxItems(1)
        .setWrap(true)
        .setConfig(Path.ResolveConfig.newBuilder()
            .setReplayDevice(parent.device))
        .build();
  }

  @Override
  protected void fireLoadStartEvent() {
    listeners.fire().onCommandsLoadingStart();
  }

  @Override
  protected void fireLoadedEvent() {
    listeners.fire().onCommandsLoaded();
    if (selection != null) {
      selectCommands(selection, true);
    }
  }

  public CommandIndex getSelectedCommands() {
    return (selection != null && selection.getNode() != null) ? selection : null;
  }

  public void selectCommands(CommandIndex index, boolean force) {
    if (!force && Objects.equal(selection, index)) {
      return;
    } else if (!isLoaded()) {
      this.selection = index;
      return;
    }

    RootNode root = (RootNode)getData();
    if (index.getNode() == null) {
      resolve(index.getCommand(), node -> selectCommands(index.withNode(node), force));
    } else if (!index.getNode().getTree().equals(root.tree)) {
      // TODO
      throw new UnsupportedOperationException("This is not yet supported, needs API clarification");
    } else {
      selection = index;
      listeners.fire().onCommandsSelected(selection);
    }
  }

  private void resolve(Path.Command command, Consumer<Path.CommandTreeNode> cb) {
    RootNode root = (RootNode)getData();
    Rpc.listen(client.get(commandTree(root.tree, command), root.device),
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

  public ListenableFuture<Service.Memory> getMemory(Path.Device device, CommandIndex index) {
    return MoreFutures.transform(
        client.get(observationsAfter(index, Application_VALUE), device), v -> {
          return v.getMemory();
        });
  }

  /**
   * An index into the command stream, representing a specific "point in time" in the trace.
   */
  public static class CommandIndex implements Comparable<CommandIndex> {
    private final Path.Command command;
    private final Path.CommandTreeNode node;
    private final boolean group;

    private CommandIndex(Path.Command command, Path.CommandTreeNode node, boolean group) {
      this.command = command;
      this.node = node;
      this.group = group;
    }

    /**
     * Create an index pointing to the given command and node.
     */
    public static CommandIndex forNode(Path.Command command, Path.CommandTreeNode node) {
      return new CommandIndex(command, node, false);
    }

    /**
     * Create an index pointing to the given command, without knowing the tree node.
     * The tree nodes is then resolved when it is needed.
     */
    public static CommandIndex forCommand(Path.Command command) {
      return new CommandIndex(command, null, false);
    }

    /**
     * Same as {@link #forCommand}, except that group selection is to be preferred when
     * resolving to a tree node.
     */
    public static CommandIndex forGroup(Path.Command command) {
      return new CommandIndex(command, null, true);
    }

    public CommandIndex withNode(Path.CommandTreeNode newNode) {
      return new CommandIndex(command, newNode, group);
    }

    public CommandIndex withCapture(Path.Capture capture) {
      return new CommandIndex(command.toBuilder().setCapture(capture).build(), null, group);
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
      } else if (!(obj instanceof CommandIndex)) {
        return false;
      }
      return command.getIndicesList().equals(((CommandIndex)obj).command.getIndicesList());
    }

    @Override
    public int compareTo(CommandIndex o) {
      return Paths.compare(command, o.command);
    }
  }

  public static class Node extends DeviceDependentModel.Data {
    private final Node parent;
    private final int index;
    private Node[] children;
    private Service.CommandTreeNode data;
    private API.Command command;
    private ListenableFuture<Node> loadFuture;

    public Node(Path.Device device, Service.CommandTreeNode data) {
      super(device);
      this.parent = null;
      this.index = 0;
      this.data = data;
    }

    public Node(Node parent, int index) {
      super(parent.device);
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

    public CommandIndex getIndex() {
      return (data == null) ? null : CommandIndex.forNode(data.getRepresentation(),
          getPath(Path.CommandTreeNode.newBuilder()).build());
    }

    public ListenableFuture<Node> load(Shell shell, Supplier<ListenableFuture<NodeData>> loader) {
      if (data != null) {
        // Already loaded.
        return null;
      } else if (loadFuture != null && !loadFuture.isCancelled()) {
        return loadFuture;
      }

      return loadFuture = MoreFutures.transformAsync(loader.get(), newData ->
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

    public RootNode(Path.Device device, Path.ID tree, Service.CommandTreeNode data) {
      super(device, data);
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
      RootNode n = (RootNode)obj;
      return device.equals(n.device) && tree.equals(n.tree);
    }

    @Override
    public int hashCode() {
      return device.hashCode() * 31 + tree.hashCode();
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
    public default void onCommandsLoadingStart() { /* emtpy */ }

    /**
     * Event indicating that the tree root has finished loading.
     */
    public default void onCommandsLoaded() { /* empty */ }

    /**
     * Event indicating that the currently selected command range has changed.
     */
    @SuppressWarnings("unused")
    public default void onCommandsSelected(CommandIndex selection) { /* empty */ }
  }
}

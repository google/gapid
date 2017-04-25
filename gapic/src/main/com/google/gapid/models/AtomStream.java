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

import static com.google.gapid.util.Paths.any;
import static com.google.gapid.util.Paths.commandTree;

import com.google.common.base.Objects;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.ApiContext.FilteringContext;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.Command;
import com.google.gapid.proto.service.Service.CommandTreeNode;
import com.google.gapid.proto.service.Service.Value;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpclib.futures.FutureController;
import com.google.gapid.rpclib.futures.SingleInFlight;
import com.google.gapid.rpclib.rpccore.Rpc;
import com.google.gapid.rpclib.rpccore.Rpc.Result;
import com.google.gapid.rpclib.rpccore.RpcException;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.Events.ListenerCollection;
import com.google.gapid.util.Messages;
import com.google.gapid.util.Paths;
import com.google.gapid.util.UiCallback;

import org.eclipse.swt.widgets.Shell;

import java.util.concurrent.ExecutionException;
import java.util.function.Consumer;
import java.util.function.Supplier;
import java.util.logging.Logger;

/**
 * Model containing the API commands (atoms) of the capture.
 */
public class AtomStream implements ApiContext.Listener {
  private static final Logger LOG = Logger.getLogger(AtomStream.class.getName());

  private final Shell shell;
  private final Client client;
  private final Capture capture;
  private final ApiContext context;
  private final ConstantSets constants;
  private final FutureController rpcController = new SingleInFlight();
  private final ListenerCollection<Listener> listeners = Events.listeners(Listener.class);
  private RootNode root;
  private AtomIndex selection;

  public AtomStream(
      Shell shell, Client client, Capture capture, ApiContext context, ConstantSets constants) {
    this.shell = shell;
    this.client = client;
    this.capture = capture;
    this.context = context;
    this.constants = constants;

    context.addListener(this);
  }

  @Override
  public void onContextsLoaded() {
    onContextSelected(context.getSelectedContext());
  }

  @Override
  public void onContextSelected(FilteringContext ctx) {
    root = null;
    listeners.fire().onAtomsLoadingStart();
    Rpc.listen(Futures.transformAsync(client.get(commandTree(capture.getCapture(), ctx)),
        tree -> Futures.transform(client.get(Paths.any(tree.getCommandTree().getRoot())),
            val -> new RootNode(
                tree.getCommandTree().getRoot().getTree(), val.getCommandTreeNode()))),
        rpcController, new UiCallback<RootNode, RootNode>(shell, LOG) {
      @Override
      protected RootNode onRpcThread(Result<RootNode> result)
          throws RpcException, ExecutionException {
        return result.get();
      }

      @Override
      protected void onUiThread(RootNode result) {
        update(result);
      }
    });
  }

  protected void update(RootNode newRoot) {
    root = newRoot;
    listeners.fire().onAtomsLoaded();
  }

  public boolean isLoaded() {
    return root != null;
  }

  public Node getRoot() {
    return root;
  }

  public ListenableFuture<Node> load(Node node) {
    return node.load(() -> Futures.transformAsync(
        client.get(any(node.getPath(Path.CommandTreeNode.newBuilder()))), v1 -> {
          CommandTreeNode data = v1.getCommandTreeNode();
          if (data.getGroup().isEmpty() && data.hasCommand()) {
            return Futures.transformAsync(client.get(any(data.getCommand())), v2 -> {
              Service.Command cmd = v2.getCommand();
              return Futures.transform(constants.loadConstants(cmd),
                  ignore -> new NodeData(data, v2.getCommand()));
            });
          }
          return Futures.immediateFuture(new NodeData(data, null));
        }));
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
  @Override
  protected void reset(boolean maintainState) {
    super.reset(maintainState);
    if (!maintainState) {
      selection = null;
    }
  }

  @Override
  protected Path.Any getPath(Path.Capture capturePath) {
    return Path.Any.newBuilder()
        .setCommands(Path.Commands.newBuilder()
            .setCapture(capturePath))
        .build();
  }

  @Override
  protected AtomList unbox(Value value) throws IOException {
    return Client.decode(value.getObject());
  }

  @Override
  protected void fireLoadEvent() {
    listeners.fire().onAtomsLoaded();
    if (selection != null) {
      listeners.fire().onAtomsSelected(selection);
    }
  }

  @Override
  public void onContextSelected(FilteringContext ctx) {
    if (selection != null && !ctx.contains(selection)) {
      if (ctx.contains(last(selection))) {
        selectAtoms(last(selection), 1, false);
      } else {
        selectAtoms(ctx.findClosest(selection), false);
      }
    }
  }

  public int getAtomCount() {
    return getData().getAtoms().length;
  }

  public Atom getAtom(long index) {
    return getData().get(index);
  }

  /**
   * @return the index of the first command of the frame that contains the given command.
   *
  public int getStartOfFrame(long index) {
    Atom[] atoms = getData().getAtoms();
    for (int i = (int)index; i > 0; i--) {
      if (atoms[i - 1].isEndOfFrame()) {
        return i;
      }
    }
    return 0;
  }

  /**
   * @return the index of the last command of the frame that contains the given command.
   *
  public int getEndOfFrame(long index) {
    Atom[] atoms = getData().getAtoms();
    for (int i = (int)index; i < atoms.length; i++) {
      if (atoms[i].isEndOfFrame()) {
        return i;
      }
    }
    return atoms.length - 1;
  }
  */

  public AtomIndex getSelectedAtoms() {
    return selection;
  }

  public void selectAtoms(AtomIndex index, boolean force) {
    if (!force && Objects.equal(selection, index)) {
      return;
    }

    if (index.getNode() == null) {
      resolve(index.getCommand(),
          (node) -> selectAtoms(new AtomIndex(index.getCommand(), node), force));
    } else if (!index.getNode().getTree().equals(root.tree)) {
      // TODO
      throw new UnsupportedOperationException("This is not yet supported, needs API clarification");
    } else {
      selection = index;
      listeners.fire().onAtomsSelected(selection);
    }
  }

  private void resolve(Path.Command command, Consumer<Path.CommandTreeNode> cb) {
    Rpc.listen(client.get(commandTree(root.tree, command)),
        new UiCallback<Service.Value, Path.CommandTreeNode>(shell, LOG) {
      @Override
      protected Path.CommandTreeNode onRpcThread(Result<Value> result)
          throws RpcException, ExecutionException {
        return result.get().getPath().getCommandTreeNode();
      }

      @Override
      protected void onUiThread(Path.CommandTreeNode result) {
        cb.accept(result);
      }
    });
  }

  /*
  public void selectAtoms(long from, long count, boolean force) {
    selectAtoms(commands(from, count), force);
  }

  public void selectAtoms(CommandRange range, boolean force) {
    if (force || !Objects.equal(selection, range)) {
      selection = range;
      context.selectContextContaining(range);
      listeners.fire().onAtomsSelected(selection);
    }
  }

  public Atom getFirstSelectedAtom() {
    return (selection == null || getData() == null) ? null : getData().get(first(selection));
  }

  public Atom getLastSelectedAtom() {
    return (selection == null || getData() == null) ? null : getData().get(last(selection));
  }

  /**
   * @return the path to the last draw command within the current selection or {@code null}.
   *
  public Path.Command getLastSelectedDrawCall() {
    if (selection == null || getData() == null) {
      return null;
    }

    FilteringContext selectedContext = context.getSelectedContext();
    for (long index = last(selection); index >= first(selection); index--) {
      if (selectedContext.contains(index) && getData().get(index).isDrawCall()) {
        return Path.Command.newBuilder()
            .setCommands(getPath().getCommands())
            .setIndex(index)
            .build();
      }
    }
    return null;
  }
  */

  public TypedObservation[] getObservations(AtomIndex index) {
    /*
    Atom atom = getAtom(index);
    if (atom.getObservationCount() == 0) {
      return NO_OBSERVATIONS;
    }

    Observations obs = atom.getObservations();
    TypedObservation[] result = new TypedObservation[obs.getReads().length + obs.getWrites().length];
    for (int i = 0; i < obs.getReads().length; i++) {
      result[i] = new TypedObservation(index, true, obs.getReads()[i]);
    }
    for (int i = 0, j = obs.getReads().length; i < obs.getWrites().length; i++, j++) {
      result[j] = new TypedObservation(index, false, obs.getWrites()[i]);
    }
    return result;
    */
    return NO_OBSERVATIONS;
  }

  private static final TypedObservation[] NO_OBSERVATIONS = new TypedObservation[0];

  /**
   * Read or write memory observation at a specific command.
   */
  public static class TypedObservation {
    public static final TypedObservation NULL_OBSERVATION = new TypedObservation(null, false) {
      @Override
      public String toString() {
        return Messages.SELECT_OBSERVATION;
      }
    };

    private final AtomIndex index;
    private final boolean read;

    public TypedObservation(AtomIndex index, boolean read) {
      this.index = index;
      this.read = read;
    }

    public Path.Memory getPath() {
      return Paths.memoryAfter(index, 0, null).getMemory();
    }

    public boolean contains(long address) {
      return false;//observation.getRange().contains(address);
    }

    @Override
    public String toString() {
      long base = 0/*observation.getRange().getBase()*/, count = 0;//observation.getRange().getSize();
      return (read ? "Read " : "Write ") + count + " byte" + (count == 1 ? "" : "s") +
          String.format(" at 0x%016x", base);
    }
  }

  public void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public void removeListener(Listener listener) {
    listeners.removeListener(listener);
  }

  /**
   * TODO
   */
  public static class AtomIndex implements Comparable<AtomIndex> {
    private final Path.Command command;
    private final Path.CommandTreeNode node;

    public AtomIndex(Path.Command command, Path.CommandTreeNode node) {
      this.command = command;
      this.node = node;
    }

    public Path.Command getCommand() {
      return command;
    }

    public Path.CommandTreeNode getNode() {
      return node;
    }

    @Override
    public String toString() {
      return command.getIndexList().toString();
    }

    @Override
    public int hashCode() {
      return command.getIndexList().hashCode();
    }

    @Override
    public boolean equals(Object obj) {
      if (obj == this) {
        return true;
      } else if (!(obj instanceof AtomIndex)) {
        return false;
      }
      return command.getIndexList().equals(((AtomIndex)obj).command.getIndexList());
    }

    @Override
    public int compareTo(AtomIndex o) {
      for (int i = 0; i < command.getIndexCount(); i++) {
        if (i >= o.command.getIndexCount()) {
          return 1;
        }
        int r = Long.compare(command.getIndex(i), o.command.getIndex(i));
        if (r != 0) {
          return r;
        }
      }
      return (command.getIndexCount() == o.command.getIndexCount()) ? 0 : -1;
    }
  }

  public static class Node {
    private final Node parent;
    private final int index;
    private Node[] children;
    private CommandTreeNode data;
    private Command command;
    private ListenableFuture<Node> loadFuture;

    public Node(CommandTreeNode data) {
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

    public CommandTreeNode getData() {
      return data;
    }

    public Command getCommand() {
      return command;
    }

    public Path.CommandTreeNode.Builder getPath(Path.CommandTreeNode.Builder path) {
      return parent.getPath(path).addIndex(index);
    }

    public AtomIndex getIndex() {
      return (data == null) ? null : new AtomIndex(data.getCommand(),
          getPath(Path.CommandTreeNode.newBuilder()).build());
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
        command = newData.command;
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
      return index + (data == null ? "" : " " + data.getGroup() + data.getCommand().getIndexList());
    }
  }

  private static class RootNode extends Node {
    public final Path.ID tree;

    public RootNode(Path.ID tree, CommandTreeNode data) {
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
    public final CommandTreeNode data;
    public final Command command;

    public NodeData(CommandTreeNode data, Command command) {
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

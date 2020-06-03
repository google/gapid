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
package com.google.gapid.views;

import static com.google.gapid.image.Images.noAlpha;
import static com.google.gapid.models.ImagesModel.THUMB_SIZE;
import static com.google.gapid.util.Colors.getRandomColor;
import static com.google.gapid.util.Colors.lerp;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Paths.lastCommand;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.CommandStream.Node;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Models;
import com.google.gapid.models.Profile;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.util.Events;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.MoreFutures;
import com.google.gapid.util.SelectionHandler;
import com.google.gapid.views.Formatter.StylingString;
import com.google.gapid.widgets.LinkifiedTreeWithImages;
import com.google.gapid.widgets.LoadableImage;
import com.google.gapid.widgets.LoadableImageWidget;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.SearchBox;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.TreePath;
import org.eclipse.jface.viewers.TreeViewerColumn;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.RGBA;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Menu;
import org.eclipse.swt.widgets.Shell;

import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * API command view displaying the commands with their hierarchy grouping in a tree.
 */
public class CommandTree extends Composite
    implements Tab, Capture.Listener, CommandStream.Listener, Profile.Listener {
  protected static final Logger LOG = Logger.getLogger(CommandTree.class.getName());

  private final Models models;
  private final LoadablePanel<Tree> loading;
  protected final Tree tree;
  private final SelectionHandler<Control> selectionHandler;
  private final SingleInFlight searchController = new SingleInFlight();

  public CommandTree(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new GridLayout(1, false));

    SearchBox search = new SearchBox(this, false);
    loading = LoadablePanel.create(this, widgets, p -> new Tree(p, models, widgets));
    tree = loading.getContents();

    search.setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));
    loading.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    models.capture.addListener(this);
    models.commands.addListener(this);
    models.profile.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.commands.removeListener(this);
      models.profile.removeListener(this);
    });

    search.addListener(Events.Search, e -> search(e.text, (e.detail & Events.REGEX) != 0));

    selectionHandler = new SelectionHandler<Control>(LOG, tree.getControl()) {
      @Override
      protected void updateModel(Event e) {
        models.analytics.postInteraction(View.Commands, ClientAction.Select);
        CommandStream.Node node = tree.getSelection();
        if (node != null) {
          CommandIndex index = node.getIndex();
          if (index == null) {
            models.commands.load(node, () -> models.commands.selectCommands(node.getIndex(), false));
          } else {
            models.commands.selectCommands(index, false);
          }
        }
      }
    };

    Menu popup = new Menu(tree.getControl());
    Widgets.createMenuItem(popup, "&Edit", SWT.MOD1 + 'E', e -> {
      CommandStream.Node node = tree.getSelection();
      if (node != null && node.getData() != null && node.getCommand() != null) {
        widgets.editor.showEditPopup(getShell(), lastCommand(node.getData().getCommands()),
            node.getCommand(), node.device);
      }
    });
    tree.setPopupMenu(popup, node ->
        node.getData() != null && node.getCommand() != null &&
        CommandEditor.shouldShowEditPopup(node.getCommand()));

    tree.registerAsCopySource(widgets.copypaste, node -> {
      models.analytics.postInteraction(View.Commands, ClientAction.Copy);
      Service.CommandTreeNode data = node.getData();
      if (data == null) {
        // Copy before loaded. Not ideal, but this is unlikely.
        return new String[] { "Loading..." };
      }

      StringBuilder result = new StringBuilder();
      if (data.getGroup().isEmpty() && data.hasCommands()) {
        result.append(data.getCommands().getTo(0)).append(": ");
        API.Command cmd = node.getCommand();
        if (cmd == null) {
          // Copy before loaded. Not ideal, but this is unlikely.
          result.append("Loading...");
        } else {
          result.append(Formatter.toString(cmd, models.constants::getConstants));
        }
      } else {
        result.append(data.getCommands().getFrom(0)).append(": ").append(data.getGroup());
      }
      return new String[] { result.toString() };
    }, true);
  }

  private void search(String text, boolean regex) {
    models.analytics.postInteraction(View.Commands, ClientAction.Search);
    CommandStream.Node parent = models.commands.getData();
    if (parent != null && !text.isEmpty()) {
      CommandStream.Node selection = tree.getSelection();
      if (selection != null) {
        parent = selection;
      }
      searchController.start().listen(
          MoreFutures.transformAsync(models.commands.search(parent, text, regex),
              r -> getTreePath(models.commands.getData(), Lists.newArrayList(),
                  r.getCommandTreeNode().getIndicesList().iterator())),
          new UiCallback<TreePath, TreePath>(tree, LOG) {
        @Override
        protected TreePath onRpcThread(Rpc.Result<TreePath> result)
            throws RpcException, ExecutionException {
          return result.get();
        }

        @Override
        protected void onUiThread(TreePath result) {
          select(result);
        }
      });
    }
  }

  protected void select(TreePath path) {
    models.commands.selectCommands(((CommandStream.Node)path.getLastSegment()).getIndex(), true);
  }

  @Override
  public Control getControl() {
    return this;
  }

  @Override
  public void reinitialize() {
    updateTree(false);
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    updateTree(true);
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error != null) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
  }

  @Override
  public void onCommandsLoaded() {
    updateTree(false);
  }

  @Override
  public void onCommandsSelected(CommandIndex index) {
    selectionHandler.updateSelectionFromModel(() -> getTreePath(index).get(), tree::setSelection);
  }

  @Override
  public void onProfileLoaded(Loadable.Message error) {
    tree.refresh();
  }

  private void updateTree(boolean assumeLoading) {
    if (assumeLoading || !models.commands.isLoaded()) {
      loading.startLoading();
      tree.setInput(null);
      return;
    }

    loading.stopLoading();
    tree.setInput(models.commands.getData());
    if (models.commands.getSelectedCommands() != null) {
      onCommandsSelected(models.commands.getSelectedCommands());
    }
  }

  private ListenableFuture<TreePath> getTreePath(CommandIndex index) {
    CommandStream.Node root = models.commands.getData();
    ListenableFuture<TreePath> result = getTreePath(root, Lists.newArrayList(root),
        index.getNode().getIndicesList().iterator());
    if (index.isGroup()) {
      // Find the deepest group/node in the path that is not the last child of its parent.
      result = MoreFutures.transform(result, path -> {
        while (path.getSegmentCount() > 0) {
          CommandStream.Node node = (CommandStream.Node)path.getLastSegment();
          if (!node.isLastChild()) {
            break;
          }
          path = path.getParentPath();
        }
        return path;
      });
    }
    return result;
  }

  private ListenableFuture<TreePath> getTreePath(
      CommandStream.Node node, List<Object> path, Iterator<Long> indices) {
    ListenableFuture<CommandStream.Node> load = models.commands.load(node);
    if (!indices.hasNext()) {
      TreePath result = new TreePath(path.toArray());
      // Ensure the last node in the path is loaded.
      return (load == null) ? Futures.immediateFuture(result) :
          MoreFutures.transform(load, ignored -> result);
    }
    return (load == null) ? getTreePathForLoadedNode(node, path, indices) :
        MoreFutures.transformAsync(load, loaded -> getTreePathForLoadedNode(loaded, path, indices));
  }

  private ListenableFuture<TreePath> getTreePathForLoadedNode(
      CommandStream.Node node, List<Object> path, Iterator<Long> indices) {
    int index = indices.next().intValue();

    CommandStream.Node child = node.getChild(index);
    path.add(child);
    return getTreePath(child, path, indices);
  }

  private static class Tree extends LinkifiedTreeWithImages<CommandStream.Node, String> {
    private static final float COLOR_INTENSITY = 0.15f;
    private static final int DURATION_WIDTH = 95;

    protected final Models models;
    private final Widgets widgets;
    private final Map<Long, Color> threadBackgroundColors = Maps.newHashMap();

    public Tree(Composite parent, Models models, Widgets widgets) {
      super(parent, SWT.H_SCROLL | SWT.V_SCROLL | SWT.MULTI, widgets);
      this.models = models;
      this.widgets = widgets;

      addColumn("GPU Time", false);
      addColumn("Wall Time", true);
    }

    private void addColumn(String title, boolean wallTime) {
      TreeViewerColumn column = addColumn(title, node -> {
        Service.CommandTreeNode data = node.getData();
        if (data == null) {
          return "";
        } else if (!models.profile.isLoaded()) {
          return "Profiling...";
        } else {
          Profile.Duration duration = models.profile.getData().getDuration(data.getCommands());
          return wallTime ? duration.formatWallTime() : duration.formatGpuTime();
        }
      }, DURATION_WIDTH);
      column.getColumn().setAlignment(SWT.RIGHT);
    }

    public void refresh() {
      refresher.refresh();
    }

    @Override
    protected ContentProvider<Node> createContentProvider() {
      return new ContentProvider<CommandStream.Node>() {
        @Override
        protected boolean hasChildNodes(CommandStream.Node element) {
          return element.getChildCount() > 0;
        }

        @Override
        protected CommandStream.Node[] getChildNodes(CommandStream.Node node) {
          return node.getChildren();
        }

        @Override
        protected CommandStream.Node getParentNode(CommandStream.Node child) {
          return child.getParent();
        }

        @Override
        protected boolean isLoaded(CommandStream.Node element) {
          return element.getData() != null;
        }

        @Override
        protected void load(CommandStream.Node node, Runnable callback) {
          models.commands.load(node, callback);
        }
      };
    }

    @Override
    protected <S extends StylingString> S format(
        CommandStream.Node element, S string, Follower.Prefetcher<String> follower) {
      Service.CommandTreeNode data = element.getData();
      if (data == null) {
        string.append("Loading...", string.structureStyle());
      } else {
        if (data.getGroup().isEmpty() && data.hasCommands()) {
          string.append(Formatter.lastIndex(data.getCommands()) + ": ", string.defaultStyle());
          API.Command cmd = element.getCommand();
          if (cmd == null) {
            string.append("Loading...", string.structureStyle());
          } else {
            Formatter.format(cmd, models.constants::getConstants, follower::canFollow,
                string, string.identifierStyle());
          }
        } else {
          string.append(Formatter.firstIndex(data.getCommands()) + ": ", string.defaultStyle());
          string.append(data.getGroup(), string.labelStyle());
          long count = data.getNumCommands();
          string.append(
              " (" + count + " command" + (count != 1 ? "s" : "") + ")", string.structureStyle());
        }
      }
      return string;
    }

    @Override
    protected Color getBackgroundColor(CommandStream.Node node) {
      API.Command cmd = node.getCommand();
      if (cmd == null) {
        return null;
      }

      long threadId = cmd.getThread();
      Color color = threadBackgroundColors.get(threadId);
      if (color == null) {
        Control control = getControl();
        RGBA bg = control.getBackground().getRGBA();
        color = new Color(control.getDisplay(),
            lerp(getRandomColor(getColorIndex(threadId)), bg.rgb, COLOR_INTENSITY), bg.alpha);
        threadBackgroundColors.put(threadId, color);
      }
      return color;
    }

    private static int getColorIndex(long threadId) {
      // TODO: The index should be the i'th thread in use by the capture, not a hash of the
      // thread ID. This requires using the list of threads exposed by the service.Capture.
      int hash = (int)(threadId ^ (threadId >>> 32));
      hash = hash ^ (hash >>> 16);
      hash = hash ^ (hash >>> 8);
      return hash & 0xff;
    }

    @Override
    protected boolean shouldShowImage(CommandStream.Node node) {
      return models.images.isReady() &&
          node.getData() != null && !node.getData().getGroup().isEmpty();
    }

    @Override
    protected ListenableFuture<ImageData> loadImage(CommandStream.Node node, int size) {
      return noAlpha(models.images.getThumbnail(
          node.getPath(Path.CommandTreeNode.newBuilder()).build(), size, i -> { /*noop*/ }));
    }

    @Override
    protected void createImagePopupContents(Shell shell, CommandStream.Node node) {
      LoadableImageWidget.forImage(
          shell, LoadableImage.newBuilder(widgets.loading)
              .forImageData(loadImage(node, THUMB_SIZE))
              .onErrorShowErrorIcon(widgets.theme))
      .withImageEventListener(new LoadableImage.Listener() {
        @Override
        public void onLoaded(boolean success) {
          if (success) {
            Widgets.ifNotDisposed(shell,() -> {
              Point oldSize = shell.getSize();
              Point newSize = shell.computeSize(SWT.DEFAULT, SWT.DEFAULT);
              shell.setSize(newSize);
              if (oldSize.y != newSize.y) {
                Point location = shell.getLocation();
                location.y += (oldSize.y - newSize.y) / 2;
                shell.setLocation(location);
              }
            });
          }
        }
      });
    }

    @Override
    protected Follower.Prefetcher<String> prepareFollower(CommandStream.Node node, Runnable cb) {
      return models.follower.prepare(node, cb);
    }

    @Override
    protected void follow(Path.Any path) {
      models.follower.onFollow(path);
    }

    @Override
    public void reset() {
      super.reset();
      for (Color color : threadBackgroundColors.values()) {
        color.dispose();
      }
      threadBackgroundColors.clear();
    }
  }
}

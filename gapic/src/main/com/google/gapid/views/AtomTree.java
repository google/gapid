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
import static com.google.gapid.models.Follower.nullPrefetcher;
import static com.google.gapid.models.Thumbnails.THUMB_SIZE;
import static com.google.gapid.util.Colors.getRandomColor;
import static com.google.gapid.util.Colors.lerp;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Paths.lastCommand;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.SettableFuture;
import com.google.gapid.models.ApiContext;
import com.google.gapid.models.ApiContext.FilteringContext;
import com.google.gapid.models.AtomStream;
import com.google.gapid.models.AtomStream.AtomIndex;
import com.google.gapid.models.AtomStream.Node;
import com.google.gapid.models.Capture;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Models;
import com.google.gapid.models.Thumbnails;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.SelectionHandler;
import com.google.gapid.views.Formatter.StylingString;
import com.google.gapid.widgets.LinkifiedTreeWithImages;
import com.google.gapid.widgets.LoadableImage;
import com.google.gapid.widgets.LoadableImageWidget;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.SearchBox;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.TreePath;
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
 * API command (atom) view displaying the commands with their hierarchy grouping in a tree.
 */
public class AtomTree extends Composite implements Tab, Capture.Listener, AtomStream.Listener,
    ApiContext.Listener, Thumbnails.Listener {
  protected static final Logger LOG = Logger.getLogger(AtomTree.class.getName());

  private final Client client;
  private final Models models;
  private final LoadablePanel<CommandTree> loading;
  protected final CommandTree tree;
  private final SelectionHandler<Control> selectionHandler;
  private final SingleInFlight searchController = new SingleInFlight();

  public AtomTree(Composite parent, Client client, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.client = client;
    this.models = models;

    setLayout(new GridLayout(1, false));

    SearchBox search = new SearchBox(this, false);
    loading = LoadablePanel.create(this, widgets, p -> new CommandTree(p, models, widgets));
    tree = loading.getContents();

    search.setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));
    loading.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    models.capture.addListener(this);
    models.atoms.addListener(this);
    models.contexts.addListener(this);
    models.thumbs.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.atoms.removeListener(this);
      models.contexts.removeListener(this);
      models.thumbs.removeListener(this);
    });

    search.addListener(Events.Search, e -> search(e.text, (e.detail & Events.REGEX) != 0));

    selectionHandler = new SelectionHandler<Control>(LOG, tree.getControl()) {
      @Override
      protected void updateModel(Event e) {
        AtomStream.Node node = tree.getSelection();
        if (node != null) {
          AtomIndex index = node.getIndex();
          if (index == null) {
            models.atoms.load(node, () -> models.atoms.selectAtoms(node.getIndex(), false));
          } else {
            models.atoms.selectAtoms(index, false);
          }
        }
      }
    };

    Menu popup = new Menu(tree.getControl());
    Widgets.createMenuItem(popup, "&Edit", SWT.MOD1 + 'E', e -> {
      AtomStream.Node node = tree.getSelection();
      if (node != null && node.getData() != null && node.getCommand() != null) {
        widgets.editor.showEditPopup(getShell(), lastCommand(node.getData().getCommands()),
            node.getCommand());
      }
    });
    tree.setPopupMenu(popup, node ->
        node.getData() != null && node.getCommand() != null &&
        AtomEditor.shouldShowEditPopup(node.getCommand()));

    tree.registerAsCopySource(widgets.copypaste, node -> {
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
    AtomStream.Node parent = models.atoms.getData();
    if (parent != null && !text.isEmpty()) {
      AtomStream.Node selection = tree.getSelection();
      if (selection != null) {
        parent = selection;
      }
      searchController.start().listen(
          Futures.transformAsync(search(searchRequest(parent, text, regex)),
              r -> getTreePath(models.atoms.getData(), Lists.newArrayList(),
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

  private static Service.FindRequest searchRequest(
      AtomStream.Node parent, String text, boolean regex) {
    return Service.FindRequest.newBuilder()
        .setCommandTreeNode(parent.getPath(Path.CommandTreeNode.newBuilder()))
        .setText(text)
        .setIsRegex(regex)
        .setMaxItems(1)
        .setWrap(true)
        .build();
  }

  private ListenableFuture<Service.FindResponse> search(Service.FindRequest request) {
    SettableFuture<Service.FindResponse> result = SettableFuture.create();
    client.streamSearch(request, result::set);
    return result;
  }

  protected void select(TreePath path) {
    models.atoms.selectAtoms(((AtomStream.Node)path.getLastSegment()).getIndex(), true);
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
  public void onAtomsLoaded() {
    updateTree(false);
  }

  @Override
  public void onAtomsSelected(AtomIndex index) {
    selectionHandler.updateSelectionFromModel(() -> getTreePath(index).get(), tree::setSelection);
  }

  @Override
  public void onContextsLoaded() {
    updateTree(false);
  }

  @Override
  public void onContextSelected(FilteringContext context) {
    updateTree(false);
  }

  @Override
  public void onThumbnailsChanged() {
    tree.refreshImages();
  }

  private void updateTree(boolean assumeLoading) {
    if (assumeLoading || !models.atoms.isLoaded()) {
      loading.startLoading();
      tree.setInput(null);
      return;
    }

    loading.stopLoading();
    tree.setInput(models.atoms.getData());
    if (models.atoms.getSelectedAtoms() != null) {
      onAtomsSelected(models.atoms.getSelectedAtoms());
    }
  }

  private ListenableFuture<TreePath> getTreePath(AtomIndex index) {
    AtomStream.Node root = models.atoms.getData();
    ListenableFuture<TreePath> result = getTreePath(root, Lists.newArrayList(root),
        index.getNode().getIndicesList().iterator());
    if (index.isGroup()) {
      // Find the deepest group/node in the path that is not the last child of its parent.
      result = Futures.transform(result, path -> {
        while (path.getSegmentCount() > 0) {
          AtomStream.Node node = (AtomStream.Node)path.getLastSegment();
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
      AtomStream.Node node, List<Object> path, Iterator<Long> indices) {
    ListenableFuture<AtomStream.Node> load = models.atoms.load(node);
    if (!indices.hasNext()) {
      TreePath result = new TreePath(path.toArray());
      // Ensure the last node in the path is loaded.
      return (load == null) ? Futures.immediateFuture(result) :
          Futures.transform(load, ignored -> result);
    }
    return (load == null) ? getTreePathForLoadedNode(node, path, indices) :
        Futures.transformAsync(load, loaded -> getTreePathForLoadedNode(loaded, path, indices));
  }

  private ListenableFuture<TreePath> getTreePathForLoadedNode(
      AtomStream.Node node, List<Object> path, Iterator<Long> indices) {
    int index = indices.next().intValue();

    AtomStream.Node child = node.getChild(index);
    path.add(child);
    return getTreePath(child, path, indices);
  }

  private static class CommandTree extends LinkifiedTreeWithImages<AtomStream.Node, String> {
    private static final float COLOR_INTENSITY = 0.15f;

    protected final Models models;
    private final Widgets widgets;
    private final Map<Long, Color> threadBackgroundColors = Maps.newHashMap();

    public CommandTree(Composite parent, Models models, Widgets widgets) {
      super(parent, SWT.H_SCROLL | SWT.V_SCROLL, widgets);
      this.models = models;
      this.widgets = widgets;
    }

    @Override
    protected ContentProvider<Node> createContentProvider() {
      return new ContentProvider<AtomStream.Node>() {
        @Override
        protected boolean hasChildNodes(AtomStream.Node element) {
          return element.getChildCount() > 0;
        }

        @Override
        protected AtomStream.Node[] getChildNodes(AtomStream.Node node) {
          return node.getChildren();
        }

        @Override
        protected AtomStream.Node getParentNode(AtomStream.Node child) {
          return child.getParent();
        }

        @Override
        protected boolean isLoaded(AtomStream.Node element) {
          return element.getData() != null;
        }

        @Override
        protected void load(AtomStream.Node node, Runnable callback) {
          models.atoms.load(node, callback);
        }
      };
    }

    @Override
    protected <S extends StylingString> S format(
        AtomStream.Node element, S string, Follower.Prefetcher<String> follower) {
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
    protected Color getBackgroundColor(AtomStream.Node node) {
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
    protected boolean shouldShowImage(AtomStream.Node node) {
      return models.thumbs.isReady() &&
          node.getData() != null && !node.getData().getGroup().isEmpty();
    }

    @Override
    protected ListenableFuture<ImageData> loadImage(AtomStream.Node node, int size) {
      return noAlpha(models.thumbs.getThumbnail(
          node.getPath(Path.CommandTreeNode.newBuilder()).build(), size, i -> { /*noop*/ }));
    }

    @Override
    protected void createImagePopupContents(Shell shell, AtomStream.Node node) {
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
    protected Follower.Prefetcher<String> prepareFollower(AtomStream.Node node, Runnable cb) {
      return (node.getData() == null || node.getCommand() == null) ? nullPrefetcher() :
          models.follower.prepare(lastCommand(node.getData().getCommands()), node.getCommand(), cb);
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

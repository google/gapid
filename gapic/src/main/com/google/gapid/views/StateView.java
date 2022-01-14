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

import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.util.Strings.stripQuotes;
import static java.util.Arrays.stream;
import static java.util.logging.Level.WARNING;
import static java.util.stream.Collectors.toList;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.ApiState;
import com.google.gapid.models.ApiState.Node;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.MoreFutures;
import com.google.gapid.util.Paths;
import com.google.gapid.util.SelectionHandler;
import com.google.gapid.views.Formatter.StylingString;
import com.google.gapid.widgets.LinkifiedTree;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.TextViewer;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.TreePath;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Menu;

import java.util.Iterator;
import java.util.List;
import java.util.Objects;
import java.util.concurrent.ExecutionException;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * View that displays the API state as a tree.
 */
public class StateView extends Composite
    implements Tab, Capture.Listener, CommandStream.Listener, ApiState.Listener {
  private static final Logger LOG = Logger.getLogger(StateView.class.getName());

  protected final Models models;
  private final LoadablePanel<StateTree> loading;
  protected final StateTree tree;
  private final SelectionHandler<Control> selectionHandler;
  protected List<Path.Any> scheduledExpandedPaths;
  protected Point scheduledScrollPos;

  public StateView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new FillLayout(SWT.VERTICAL));

    loading = LoadablePanel.create(this, widgets, panel -> new StateTree(panel, models, widgets));
    tree = loading.getContents();

    models.capture.addListener(this);
    models.commands.addListener(this);
    models.state.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.commands.removeListener(this);
      models.state.removeListener(this);
    });

    selectionHandler = new SelectionHandler<Control>(LOG, tree.getControl()) {
      @Override
      protected void updateModel(Event e) {
        ApiState.Node selection = tree.getSelection();
        if (selection != null) {
          Service.StateTreeNode node = selection.getData();
          if (node != null) {
            models.state.selectPath(node.getValuePath(), false);
          }
        }
      }
    };

    Menu popup = new Menu(tree.getControl());
    Widgets.createMenuItem(popup, "&View Details", SWT.MOD1 + 'D', e -> {
      ApiState.Node node = tree.getSelection();
      if (node != null) {
        Service.StateTreeNode data = node.getData();

        if (data == null) {
          // Data not loaded yet, this shouldn't happen (see MenuDetect handler). Ignore.
          LOG.log(Level.WARNING, "Impossible popup requested and ignored.");
          return;
        } else if (!data.hasPreview()) {
          // No value at this node, this shouldn't happen, either. Ignore.
          LOG.log(Level.WARNING, "Value-less popup requested and ignored: {0}", data);
          return;
        }

        TextViewer.showViewTextPopup(getShell(), widgets, data.getName() + ":",
            MoreFutures.transform(models.state.loadValue(node),
                // Formatter.toString(<string value>) formats the string as "<string>". This makes
                // sense in all but this context, so we simply strip them here rather than
                // complicate the formatting code.
                v -> stripQuotes(Formatter.toString(
                    v, models.constants.getConstants(data.getConstants()), false))));
      }
    });
    tree.setPopupMenu(popup, StateView::canShowPopup);

    tree.registerAsCopySource(widgets.copypaste, node -> {
      Service.StateTreeNode data = node.getData();
      if (data == null) {
        // Copy before loaded. Not ideal, but this is unlikely.
        return new String[] { "Loading..." };
      } else if (!data.hasPreview()) {
        return new String[] { data.getName() };
      }

      String text = Formatter.toString(data.getPreview(),
          models.constants.getConstants(data.getConstants()), data.getPreviewIsValue());
      return new String[] { data.getName(), text };
    }, true);
  }

  @Override
  public Control getControl() {
    return this;
  }

  @Override
  public void reinitialize() {
    onCaptureLoadingStart(false);
    if (models.capture.isLoaded() && models.commands.isLoaded()) {
      onCommandsLoaded();
      if (models.commands.getSelectedCommands() != null) {
        onStateLoadingStart();
        if (models.state.isLoaded()) {
          onStateLoaded(null);
        }
      }
    }
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    loading.showMessage(Info, Messages.LOADING_CAPTURE);
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error != null) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
  }

  @Override
  public void onCommandsLoaded() {
    if (!models.commands.isLoaded()) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    } else if (models.commands.getSelectedCommands() == null) {
      loading.showMessage(Info, Messages.SELECT_COMMAND);
    }
  }

  @Override
  public void onCommandsSelected(CommandIndex path) {
    if (path == null) {
      loading.showMessage(Info, Messages.SELECT_COMMAND);
    }
  }

  @Override
  public void onStateLoadingStart() {
    loading.startLoading();
  }

  @Override
  public void onStateLoaded(Loadable.Message error) {
    if (error != null) {
      loading.showMessage(error);
      return;
    }

    loading.stopLoading();
    if (scheduledExpandedPaths == null) {
      scheduledExpandedPaths = getExpandedPaths();
      scheduledScrollPos = tree.getScrollPos();
    }
    tree.setInput(models.state.getData());
    updateExpansionState(scheduledExpandedPaths, scheduledExpandedPaths.size());
  }

  @Override
  public void onStateSelected(Path.Any path) {
    if (!models.state.isLoaded()) {
      return; // Once loaded, we'll call ourselves again.
    }
    updateSelectionState();
  }

  private void updateSelectionState() {
    ApiState.Node root = models.state.getData();
    selectionHandler.updateSelectionFromModel(
        () -> MoreFutures.transformAsync(models.state.getResolvedSelectedPath(),
            nodePath -> getTreePath(root, nodePath)).get(),
        path -> {
          tree.setSelection(path);
          tree.setExpandedState(path, true);
        });
  }

  private List<Path.Any> getExpandedPaths() {
    return stream(tree.getExpandedElements())
        .map(n -> ((ApiState.Node)n).getData())
        .filter(Objects::nonNull)
        .map(Service.StateTreeNode::getValuePath)
        .collect(toList());
  }

  protected void updateExpansionState(List<Path.Any> paths, int retry) {
    ApiState.Node root = models.state.getData();
    Path.GlobalState rootPath = Paths.findGlobalState(root.getData().getValuePath());
    List<ListenableFuture<TreePath>> futures = Lists.newArrayList();
    for (Path.Any path : paths) {
      Path.Any reparented = Paths.reparent(path, rootPath);
      if (reparented == null) {
        LOG.log(WARNING, "Unable to reparent path {0}", path);
        continue;
      }
      futures.add(MoreFutures.transformAsync(models.state.resolve(reparented),
          nodePath -> getTreePath(root, nodePath)));
    }

    Rpc.listen(Futures.allAsList(futures), new UiCallback<List<TreePath>, TreePath[]>(tree, LOG) {
      @Override
      protected TreePath[] onRpcThread(Rpc.Result<List<TreePath>> result)
          throws RpcException, ExecutionException {
        List<TreePath> list = result.get();
        return list.toArray(new TreePath[list.size()]);
      }

      @Override
      protected void onUiThread(TreePath[] treePaths) {
        // Only apply the UI update if nothing else has already pulled the state out from under us.
        // TODO: cancel the futures when this happens to avoid some wasted work.
        if (root == models.state.getData()) {
          setExpanded(treePaths, paths, retry);
        }
      }
    });
  }

  protected void setExpanded(TreePath[] treePaths, List<Path.Any> paths, int retry) {
    for (TreePath path : treePaths) {
      tree.setExpandedState(path, true);
      if (!tree.getExpandedState(path)) {
        if (retry > 0) {
          updateExpansionState(paths, retry - 1);
          return;
        }
      }
    }
    tree.scrollTo(scheduledScrollPos);

    scheduledExpandedPaths = null;
    scheduledScrollPos = null;

    Path.Any selection = models.state.getSelectedPath();
    if (selection == null) {
      tree.setSelection(null);
    } else {
      updateSelectionState();
    }
  }

  private ListenableFuture<TreePath> getTreePath(ApiState.Node root, Path.StateTreeNode nodePath) {
    return getTreePath(root, Lists.newArrayList(), nodePath.getIndicesList().iterator());
  }

  private ListenableFuture<TreePath> getTreePath(
      ApiState.Node node, List<Object> path, Iterator<Long> indices) {
    ListenableFuture<ApiState.Node> load = models.state.load(node);
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
      ApiState.Node node, List<Object> path, Iterator<Long> indices) {
    ApiState.Node child = node.getChild(indices.next().intValue());
    path.add(child);
    return getTreePath(child, path, indices);
  }

  private static boolean canShowPopup(ApiState.Node node) {
    Service.StateTreeNode data = node.getData();
    return data != null && data.hasPreview();
  }

  private static class StateTree extends LinkifiedTree<ApiState.Node, Void> {
    protected final Models models;

    public StateTree(Composite parent, Models models, Widgets widgets) {
      super(parent, SWT.H_SCROLL | SWT.V_SCROLL | SWT.MULTI, widgets);
      this.models = models;
    }

    @Override
    protected ContentProvider<Node> createContentProvider() {
      return new ContentProvider<ApiState.Node>() {
        @Override
        protected boolean hasChildNodes(ApiState.Node element) {
          return element.getChildCount() > 0;
        }

        @Override
        protected ApiState.Node[] getChildNodes(ApiState.Node node) {
          return node.getChildren();
        }

        @Override
        protected ApiState.Node getParentNode(ApiState.Node child) {
          return child.getParent();
        }

        @Override
        protected boolean isLoaded(ApiState.Node element) {
          return element.getData() != null;
        }

        @Override
        protected boolean isDefaultExpanded(Node element) {
          return false;
        }

        @Override
        protected void load(ApiState.Node node, Runnable callback) {
          models.state.load(node, callback);
        }
      };
    }

    @Override
    protected <S extends StylingString> S format(
        ApiState.Node node, S string, Follower.Prefetcher<Void> follower) {
      Service.StateTreeNode data = node.getData();
      if (data == null) {
        string.append("Loading...", string.structureStyle());
      } else {
        string.append(data.getName(), string.defaultStyle());
        if (!data.getLabel().isEmpty()) {
          string.append(" " + data.getLabel(), string.labelStyle());
        }
        if (data.hasPreview()) {
          string.append(": ", string.structureStyle());
          Path.Any follow = follower.canFollow(null);
          string.startLink(follow);
          Formatter.format(data.getPreview(), models.constants.getConstants(data.getConstants()),
              data.getPreviewIsValue(), string,
              (follow == null) ? string.defaultStyle() : string.linkStyle());
          string.endLink();
        }
      }
      return string;
    }

    @Override
    protected Follower.Prefetcher<Void> prepareFollower(ApiState.Node node, Runnable callback) {
      return models.follower.prepare(node, callback);
    }

    @Override
    protected void follow(Path.Any path) {
      models.follower.onFollow(path);
    }
  }
}

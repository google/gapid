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

import static com.google.gapid.models.Follower.nullPrefetcher;
import static com.google.gapid.util.GeoUtils.center;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.widgets.Widgets.createTreeForViewer;
import static com.google.gapid.widgets.Widgets.createTreeViewer;
import static java.util.Arrays.stream;
import static java.util.logging.Level.WARNING;
import static java.util.stream.Collectors.toList;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.ApiState;
import com.google.gapid.models.AtomStream;
import com.google.gapid.models.AtomStream.AtomIndex;
import com.google.gapid.models.Capture;
import com.google.gapid.models.ConstantSets;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.server.Client;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.MouseAdapter;
import com.google.gapid.util.Paths;
import com.google.gapid.util.SelectionHandler;
import com.google.gapid.views.Formatter.StylingString;
import com.google.gapid.widgets.CopySources;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.MeasuringViewLabelProvider;
import com.google.gapid.widgets.TextViewer;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.ILazyTreeContentProvider;
import org.eclipse.jface.viewers.TreePath;
import org.eclipse.jface.viewers.TreeSelection;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.events.MouseEvent;
import org.eclipse.swt.events.SelectionEvent;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Display;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Menu;
import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeItem;
import org.eclipse.swt.widgets.Widget;

import java.util.Iterator;
import java.util.List;
import java.util.concurrent.ExecutionException;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * View that displays the API state as a tree.
 */
public class StateView extends Composite
    implements Tab, Capture.Listener, AtomStream.Listener, ApiState.Listener {
  private static final Logger LOG = Logger.getLogger(StateView.class.getName());

  private final Models models;
  private final LoadablePanel<Tree> loading;
  protected final TreeViewer viewer;
  private final SelectionHandler<Tree> selectionHandler;
  protected List<Path.Any> scheduledExpandedPaths;
  protected Point scheduledScrollPos;

  public StateView(Composite parent, Client client, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new FillLayout(SWT.VERTICAL));

    loading = LoadablePanel.create(this, widgets,
        panel -> createTreeForViewer(panel, SWT.H_SCROLL | SWT.V_SCROLL | SWT.VIRTUAL | SWT.MULTI));
    Tree tree = loading.getContents();
    viewer = createTreeViewer(tree);
    viewer.setContentProvider(new StateContentProvider(models.state, viewer));
    ViewLabelProvider labelProvider =
        new ViewLabelProvider(viewer, models.constants, widgets.theme);
    viewer.setLabelProvider(labelProvider);

    models.capture.addListener(this);
    models.atoms.addListener(this);
    models.state.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.atoms.removeListener(this);
      models.state.removeListener(this);
    });

    selectionHandler = new SelectionHandler<Tree>(LOG, tree) {
      @Override
      protected void updateModel(Event e) {
        Object selection = (tree.getSelectionCount() > 0) ? tree.getSelection()[0].getData() : null;
        if (selection instanceof ApiState.Node) {
          Service.StateTreeNode node = ((ApiState.Node)selection).getData();
          if (node != null) {
            models.state.selectPath(node.getValuePath(), false);
          }
        }
      }
    };

    Widgets.Refresher treeRefresher = Widgets.withAsyncRefresh(viewer);
    MouseAdapter mouseHandler = new MouseAdapter() {
      // TODO - dedupe with code in AtomTree.
      private TreeItem lastHovered;
      private Follower.Prefetcher<Void> lastPrefetcher = nullPrefetcher();

      @Override
      public void mouseMove(MouseEvent e) {
        updateHover(e.x, e.y);
      }

      @Override
      public void mouseScrolled(MouseEvent e) {
        updateHover(e.x, e.y);
      }

      @Override
      public void widgetSelected(SelectionEvent e) {
        // Scrollbar was moved / mouse wheel caused scrolling. This is required for systems with
        // a touchpad with scrolling inertia, where the view keeps scrolling long after the mouse
        // wheel event has been processed.
        Display disp = getDisplay();
        Point mouse = disp.map(null, tree, disp.getCursorLocation());
        updateHover(mouse.x, mouse.y);
      }

      @Override
      public void mouseDown(MouseEvent e) {
        Point location = new Point(e.x, e.y);
        Path.Any follow = (Path.Any)labelProvider.getFollow(location);
        if (follow != null) {
          models.follower.onFollow(follow);
        }
      }

      @Override
      public void mouseExit(MouseEvent e) {
        hoverItem(null);
      }

      private void updateHover(int x, int y) {
        TreeItem item = tree.getItem(new Point(x, y));
        // When hovering over the far left of deep items, getItem returns null. Let's check a few
        // more places to the right.
        if (item == null) {
          for (int testX = x + 20; item == null && testX < 300; testX += 20) {
            item = tree.getItem(new Point(testX, y));
          }
        }

        if (item != null && (item.getData() instanceof ApiState.Node)) {
          hoverItem(item);
          Path.Any follow = (Path.Any)labelProvider.getFollow(new Point(x, y));
          setCursor((follow == null) ? null : getDisplay().getSystemCursor(SWT.CURSOR_HAND));
        } else {
          hoverItem(null);
        }
      }

      private void hoverItem(TreeItem item) {
        if (item != lastHovered) {
          lastHovered = item;
          lastPrefetcher.cancel();

          ApiState.Node node = (item == null) ? null : (ApiState.Node)item.getData();
          // TODO: if still loading, once loaded should update the hover data.
          if (node == null || node.getData() == null || !node.getData().hasValuePath()) {
            lastPrefetcher = nullPrefetcher();
          } else {
            lastPrefetcher = models.follower.prepare(node.getData().getValuePath(),
                () -> Widgets.scheduleIfNotDisposed(tree, treeRefresher::refresh));
          }

          labelProvider.setHoveredItem(lastHovered, lastPrefetcher);
          treeRefresher.refresh();
        }
      }
    };
    tree.addMouseListener(mouseHandler);
    tree.addMouseTrackListener(mouseHandler);
    tree.addMouseMoveListener(mouseHandler);
    tree.addMouseWheelListener(mouseHandler);
    tree.getVerticalBar().addSelectionListener(mouseHandler);

    Menu popup = new Menu(tree);
    Widgets.createMenuItem(popup, "&View Details", SWT.MOD1 + 'D', e -> {
      TreeItem item = (tree.getSelectionCount() > 0) ? tree.getSelection()[0] : null;
      if (item != null && (item.getData() instanceof ApiState.Node)) {
        Service.StateTreeNode data = ((ApiState.Node)item.getData()).getData();

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
            Futures.transform(client.get(data.getValuePath()),
                v -> Formatter.toString(
                    v.getBox(),  models.constants.getConstants(data.getConstants()), false)));
      }
    });
    tree.setMenu(popup);
    tree.addListener(SWT.MenuDetect, e -> {
      if (!canShowPopup(tree.getItem(tree.toControl(e.x, e.y)))) {
        e.doit = false;
      }
    });

    CopySources.registerTreeAsCopySource(widgets.copypaste, viewer, object -> {
      if (object instanceof ApiState.Node) {
        Service.StateTreeNode node = ((ApiState.Node)object).getData();
        if (node == null) {
          // Copy before loaded. Not ideal, but this is unlikely.
          return new String[] { "Loading..." };
        } else if (!node.hasPreview()) {
          return new String[] { node.getName() };
        }

        String text = Formatter.toString(node.getPreview(),
            models.constants.getConstants(node.getConstants()), node.getPreviewIsValue());
        return new String[] { node.getName(), text };
      }

      return new String[] { String.valueOf(object) };
    }, true);
  }

  @Override
  public Control getControl() {
    return this;
  }

  @Override
  public void reinitialize() {
    onCaptureLoadingStart(false);
    if (models.capture.isLoaded() && models.atoms.isLoaded()) {
      onAtomsLoaded();
      if (models.atoms.getSelectedAtoms() != null) {
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
  public void onAtomsLoaded() {
    if (!models.atoms.isLoaded()) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    } else if (models.atoms.getSelectedAtoms() == null) {
      loading.showMessage(Info, Messages.SELECT_ATOM);
    }
  }

  @Override
  public void onAtomsSelected(AtomIndex path) {
    if (path == null) {
      loading.showMessage(Info, Messages.SELECT_ATOM);
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
      TreeItem topItem = viewer.getTree().getTopItem();
      scheduledScrollPos = (topItem == null) ? null : center(topItem.getBounds());
    }
    viewer.setInput(models.state.getData());
    updateExpansionState(scheduledExpandedPaths, scheduledExpandedPaths.size());

    Path.Any selection = models.state.getSelectedPath();
    if (selection == null) {
      viewer.setSelection(
          new TreeSelection(new TreePath(new Object[] { viewer.getInput() })), true);
    } else {
      updateSelectionState(false);
    }
  }

  @Override
  public void onStateSelected(Path.Any path) {
    if (!models.state.isLoaded()) {
      return; // Once loaded, we'll call ourselves again.
    }
    updateSelectionState(true);
  }

  private void updateSelectionState(boolean show) {
    ApiState.Node root = models.state.getData();
    selectionHandler.updateSelectionFromModel(
        () -> Futures.transformAsync(
            models.state.getResolvedSelectedPath(), nodePath -> getTreePath(root, nodePath)).get(),
        selection -> {
          viewer.refresh();
          viewer.setSelection((selection.getSegmentCount() == 0) ?
              TreeSelection.EMPTY : new TreeSelection(selection), show);
          if (show) {
            viewer.setExpandedState(selection, true);
          }
        });
  }

  private List<Path.Any> getExpandedPaths() {
    return stream(viewer.getExpandedElements())
        .map(element -> ((ApiState.Node)element).getData())
        .filter(data -> data != null)
        .map(data -> data.getValuePath())
        .collect(toList());
  }

  protected void updateExpansionState(List<Path.Any> paths, int retry) {
    Path.State state = models.state.getSource().getStateTree().getState();
    ApiState.Node root = models.state.getData();
    List<ListenableFuture<TreePath>> futures = Lists.newArrayList();
    for (Path.Any path : paths) {
      Path.Any reparented = Paths.reparent(path, state);
      if (reparented == null) {
        LOG.log(WARNING, "Unable to reparent path {0}", path);
        continue;
      }
      futures.add(Futures.transformAsync(models.state.resolve(reparented),
          nodePath -> getTreePath(root, nodePath)));
    }

    Rpc.listen(Futures.allAsList(futures),
        new UiCallback<List<TreePath>, TreePath[]>(viewer.getTree(), LOG) {
      @Override
      protected TreePath[] onRpcThread(Rpc.Result<List<TreePath>> result)
          throws RpcException, ExecutionException {
        List<TreePath> list = result.get();
        return list.toArray(new TreePath[list.size()]);
      }

      @Override
      protected void onUiThread(TreePath[] treePaths) {
        setExpanded(treePaths, paths, retry);
      }
    });
  }

  protected void setExpanded(TreePath[] treePaths, List<Path.Any> paths, int retry) {
    viewer.refresh();
    for (TreePath path : treePaths) {
      viewer.setExpandedState(path, true);
      if (!viewer.getExpandedState(path)) {
        if (retry > 0) {
          updateExpansionState(paths, retry - 1);
          return;
        }
      }
    }

    if (scheduledScrollPos != null) {
      TreeItem topItem = viewer.getTree().getItem(scheduledScrollPos);
      if (topItem != null) {
        viewer.getTree().setTopItem(topItem);
      }
    }

    scheduledExpandedPaths = null;
    scheduledScrollPos = null;
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
          Futures.transform(load, ignored -> result);
    }
    return (load == null) ? getTreePathForLoadedNode(node, path, indices) :
        Futures.transformAsync(load, loaded -> getTreePathForLoadedNode(loaded, path, indices));
  }

  private ListenableFuture<TreePath> getTreePathForLoadedNode(
      ApiState.Node node, List<Object> path, Iterator<Long> indices) {
    ApiState.Node child = node.getChild(indices.next().intValue());
    path.add(child);
    return getTreePath(child, path, indices);
  }

  private static boolean canShowPopup(TreeItem item) {
    if (item == null || !(item.getData() instanceof ApiState.Node)) {
      return false;
    }
    Service.StateTreeNode data = ((ApiState.Node)item.getData()).getData();
    return data != null && data.hasPreview();
  }

  /**
   * Content provider for the state tree.
   */
  private static class StateContentProvider implements ILazyTreeContentProvider {
    private final ApiState state;
    private final TreeViewer viewer;
    private final Widgets.Refresher refresher;

    public StateContentProvider(ApiState state, TreeViewer viewer) {
      this.state = state;
      this.viewer = viewer;
      this.refresher = Widgets.withAsyncRefresh(viewer);
    }

    @Override
    public void updateChildCount(Object element, int currentChildCount) {
      viewer.setChildCount(element, ((ApiState.Node)element).getChildCount());
    }

    @Override
    public void updateElement(Object parent, int index) {
      ApiState.Node child = ((ApiState.Node)parent).getChild(index);
      state.load(child, refresher::refresh);
      viewer.replace(parent, index, child);
      viewer.setHasChildren(child, child.getChildCount() > 0);
    }

    @Override
    public Object getParent(Object element) {
      return ((ApiState.Node)element).getParent();
    }
  }

  /**
   * Label provider for the state tree.
   */
  private static class ViewLabelProvider extends MeasuringViewLabelProvider {
    private final ConstantSets constants;
    private TreeItem hoveredItem;
    private Follower.Prefetcher<Void> follower;

    public ViewLabelProvider(TreeViewer viewer, ConstantSets constants, Theme theme) {
      super(viewer, theme);
      this.constants = constants;
    }

    public void setHoveredItem(TreeItem hoveredItem, Follower.Prefetcher<Void> follower) {
      this.hoveredItem = hoveredItem;
      this.follower = follower;
    }

    @Override
    protected <S extends StylingString> S format(Widget item, Object element, S string) {
      Service.StateTreeNode data = ((ApiState.Node)element).getData();
      if (data == null) {
        string.append("Loading...", string.structureStyle());
      } else {
        string.append(data.getName(), string.defaultStyle());
        if (data.hasPreview()) {
         string.append(": ", string.structureStyle());
         Path.Any follow = getFollower(item).canFollow(null);
         string.startLink(follow);
         Formatter.format(data.getPreview(), constants.getConstants(data.getConstants()),
             data.getPreviewIsValue(), string,
             (follow == null) ? string.defaultStyle() : string.linkStyle());
         string.endLink();
        }
      }
      return string;
    }

    private Follower.Prefetcher<Void> getFollower(Widget item) {
      return (item == hoveredItem) ? follower : nullPrefetcher();
    }

    @Override
    protected boolean isFollowable(Object element) {
      ApiState.Node node = (ApiState.Node)element;
      return node.getData() != null && node.getData().hasValuePath();
    }
  }
}

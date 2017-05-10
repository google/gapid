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
import static com.google.gapid.models.Thumbnails.THUMB_SIZE;
import static com.google.gapid.util.GeoUtils.right;
import static com.google.gapid.util.GeoUtils.vertCenter;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.widgets.Widgets.createTreeForViewer;
import static com.google.gapid.widgets.Widgets.createTreeViewer;
import static com.google.gapid.widgets.Widgets.ifNotDisposed;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.Server.GapisInitException;
import com.google.gapid.models.ApiContext;
import com.google.gapid.models.ApiContext.FilteringContext;
import com.google.gapid.models.AtomStream;
import com.google.gapid.models.AtomStream.AtomIndex;
import com.google.gapid.models.Capture;
import com.google.gapid.models.ConstantSets;
import com.google.gapid.models.Models;
import com.google.gapid.models.Thumbnails;
import com.google.gapid.proto.service.Service.Command;
import com.google.gapid.proto.service.Service.CommandTreeNode;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.util.Messages;
import com.google.gapid.util.MouseAdapter;
import com.google.gapid.util.Scheduler;
import com.google.gapid.util.SelectionHandler;
import com.google.gapid.views.Formatter.StylingString;
import com.google.gapid.widgets.Balloon;
import com.google.gapid.widgets.CopySources;
import com.google.gapid.widgets.LoadableImage;
import com.google.gapid.widgets.LoadableImageWidget;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.LoadingIndicator;
import com.google.gapid.widgets.MeasuringViewLabelProvider;
import com.google.gapid.widgets.SearchBox;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.VisibilityTrackingTreeViewer;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.ILazyTreeContentProvider;
import org.eclipse.jface.viewers.TreePath;
import org.eclipse.jface.viewers.TreeSelection;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.events.MouseEvent;
import org.eclipse.swt.events.SelectionEvent;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Display;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeItem;

import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import java.util.logging.Logger;

/**
 * API command (atom) view displaying the commands with their hierarchy grouping in a tree.
 */
public class AtomTree extends Composite implements Tab, Capture.Listener, AtomStream.Listener,
    ApiContext.Listener, Thumbnails.Listener {
  protected static final Logger LOG = Logger.getLogger(AtomTree.class.getName());
  private static final int PREVIEW_HOVER_DELAY_MS = 500;

  private final Models models;
  private final LoadablePanel<Tree> loading;
  private final TreeViewer viewer;
  private final ImageProvider imageProvider;
  private final SelectionHandler<Tree> selectionHandler;

  public AtomTree(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new GridLayout(1, false));

    SearchBox search = new SearchBox(this);
    loading = new LoadablePanel<Tree>(this, widgets,
        p -> createTreeForViewer(p, SWT.H_SCROLL | SWT.V_SCROLL | SWT.VIRTUAL));
    Tree tree = loading.getContents();
    viewer = createTreeViewer(tree);
    imageProvider = new ImageProvider(models.thumbs, viewer, widgets.loading);
    viewer.setContentProvider(new AtomContentProvider(models.atoms, viewer));
    ViewLabelProvider labelProvider = new ViewLabelProvider(
        viewer, models.constants, widgets.theme, imageProvider);
    viewer.setLabelProvider(labelProvider);

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
      imageProvider.reset();
    });

    //search.addListener(Events.Search, e -> search(e.text, (e.detail & Events.REGEX) != 0));

    selectionHandler = new SelectionHandler<Tree>(LOG, tree) {
      @Override
      protected void updateModel(Event e) {
        Object selection = (tree.getSelectionCount() > 0) ? tree.getSelection()[0].getData() : null;
        if (selection instanceof AtomStream.Node) {
          AtomStream.Node node = (AtomStream.Node)selection;
          AtomIndex index = node.getIndex();
          if (index == null) {
            models.atoms.load(node, () -> models.atoms.selectAtoms(node.getIndex(), false));
          } else {
            models.atoms.selectAtoms(index, false);
          }
        }
      }
    };

    /*
    Menu popup = new Menu(tree);
    Widgets.createMenuItem(popup, "&Edit", SWT.MOD1 + 'E', e -> {
      TreeItem item = (tree.getSelectionCount() > 0) ? tree.getSelection()[0] : null;
      if (item != null && (item.getData() instanceof AtomNode)) {
        widgets.editor.showEditPopup(getShell(), ((AtomNode)item.getData()).index);
      }
    });
    tree.setMenu(popup);
    tree.addListener(SWT.MenuDetect, e -> {
      TreeItem item = tree.getItem(tree.toControl(e.x, e.y));
      if (item == null || !(item.getData() instanceof AtomNode)) {
        e.doit = false;
      }
    });
    */

    MouseAdapter mouseHandler = new MouseAdapter() {
      private Future<?> lastScheduledFuture = Futures.immediateFuture(null);
      private TreeItem lastHoveredItem;
      private Balloon lastShownBalloon;

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
        Object follow = /*TODO(CanFollow)*/labelProvider.getFollow(location);
        if (follow != null) {
          //models.follower.follow(getFollowPath(tree.getItem(location), follow.getPath()));
        }
      }

      private void updateHover(int x, int y) {
        TreeItem item = tree.getItem(new Point(x, y));
        if (item != null && (item.getData() instanceof AtomStream.Node) &&
            item.getImage() != null && item.getImageBounds(0).contains(x, y)) {
          hover(item);
        } else {
          hover(null);

          Object follow = /*TODO(CanFollow)*/labelProvider.getFollow(new Point(x, y));
          setCursor((follow == null) ? null : getDisplay().getSystemCursor(SWT.CURSOR_HAND));
          if (follow != null) {
            //models.follower.prepareFollow(getFollowPath(item, follow.getPath()));
          }
        }
      }

      /*
      private Path.Any getFollowPath(TreeItem item, Pathway path) {
        AtomNode atom = (AtomNode)item.getData();
        String field = ((com.google.gapid.service.snippets.FieldPath)path).getName();
        for (int i = 0; i < atom.atom.getFieldCount(); i++) {
          String actualField = atom.atom.getFieldInfo(i).getName();
          if (actualField.equalsIgnoreCase(field)) {
            return Paths.atomField(models.atoms.getPath(), atom.index, actualField);
          }
        }
        LOG.log(Level.WARNING, "Field " + path + " not found in atom " + atom.atom);
        return null;
      }
      */

      private void hover(TreeItem item) {
        if (item != lastHoveredItem) {
          lastScheduledFuture.cancel(true);
          lastHoveredItem = item;
          if (item != null) {
            lastScheduledFuture = Scheduler.EXECUTOR.schedule(() ->
              Widgets.scheduleIfNotDisposed(
                  tree, () -> showBalloon(item, (AtomStream.Node)item.getData())),
              PREVIEW_HOVER_DELAY_MS, TimeUnit.MILLISECONDS);
          }
          if (lastShownBalloon != null) {
            lastShownBalloon.close();
          }
        }
      }

      private void showBalloon(TreeItem item, AtomStream.Node node) {
        if (lastShownBalloon != null) {
          lastShownBalloon.close();
        }
        Rectangle bounds = item.getImageBounds(0);
        lastShownBalloon = Balloon.createAndShow(tree, shell -> {
          LoadableImageWidget.forImageData(shell, loadImage(node), widgets.loading)
              .withImageEventListener(new LoadableImage.Listener() {
                @Override
                public void onLoaded(boolean success) {
                  if (success) {
                    Widgets.ifNotDisposed(shell,
                        () -> shell.setSize(shell.computeSize(SWT.DEFAULT, SWT.DEFAULT)));
                  }
                }
              });
        }, new Point(right(bounds) + 2, vertCenter(bounds) - THUMB_SIZE / 2));
      }

      private ListenableFuture<ImageData> loadImage(AtomStream.Node node) {
        return noAlpha(models.thumbs.getThumbnail(
            node.getPath(Path.CommandTreeNode.newBuilder()).build(), THUMB_SIZE));
      }
    };
    tree.addMouseListener(mouseHandler);
    tree.addMouseMoveListener(mouseHandler);
    tree.addMouseWheelListener(mouseHandler);
    tree.getVerticalBar().addSelectionListener(mouseHandler);

    CopySources.registerTreeAsCopySource(widgets.copypaste, viewer, object -> {
      if (object instanceof AtomStream.Node) {
        AtomStream.Node node = (AtomStream.Node)object;
        CommandTreeNode data = node.getData();
        if (data == null) {
          // Copy before loaded. Not ideal, but this is unlikely.
          return new String[] { "Loading..." };
        }

        StringBuilder result = new StringBuilder();
        if (data.getGroup().isEmpty() && data.hasCommands()) {
          result.append(data.getCommands().getTo(0)).append(": ");
          Command cmd = node.getCommand();
          if (cmd == null) {
            // Copy before loaded. Not ideal, but this is unlikely.
            result.append("Loading...");
          } else {
            result.append(Formatter.toString(cmd, models.constants::getConstants));
          }
        } else {
          result.append(data.getCommands().getFrom(0)).append(": ")
              .append(data.getGroup()); // TODO add counts
        }
        return new String[] { result.toString() };
      }
      return new String[] { String.valueOf(object) };
    });
  }

  /*
  private void search(String text, boolean regex) {
    if (root != null && !text.isEmpty()) {
      FilteredGroup parent = root;
      Object start = null;
      if (viewer.getTree().getSelectionCount() >= 1) {
        start = viewer.getTree().getSelection()[0].getData();
        if (start instanceof FilteredGroup) {
          parent = ((FilteredGroup)start).parent;
        } else if (start instanceof AtomNode) {
          parent = ((AtomNode)start).parent;
        } else {
          return;
        }
      }

      Pattern pattern = SearchBox.getPattern(text, regex);
      CommandRange range = parent.search(pattern, start, start instanceof AtomNode, true);
      if (range == null && (parent != root || start != null)) {
        range = root.search(pattern, null, false, false);
      }

      if (range != null) {
        models.atoms.selectAtoms(range, true);
      }
    }
  }
  */

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
  public void onCaptureLoaded(GapisInitException error) {
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
    selectionHandler.updateSelectionFromModel(
        () -> getTreePath(index.getNode()).get(),
        selection -> viewer.setSelection(new TreeSelection(selection), true));
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
    imageProvider.reset();
    viewer.refresh();
  }

  private void updateTree(boolean assumeLoading) {
    imageProvider.reset();

    if (assumeLoading || !models.atoms.isLoaded()) {
      loading.startLoading();
      viewer.setInput(null);
      return;
    }

    loading.stopLoading();
    viewer.setInput(models.atoms.getData());
    viewer.getTree().setSelection(viewer.getTree().getItem(0));
    viewer.getTree().showSelection();

    /*
    if (models.atoms.getSelectedAtoms() != null) {
      onAtomsSelected(models.atoms.getSelectedAtoms());
    }
    */
  }

  private ListenableFuture<TreePath> getTreePath(Path.CommandTreeNode path) {
    return getTreePath(
        models.atoms.getData(), Lists.newArrayList(), path.getIndexList().iterator());
  }

  private ListenableFuture<TreePath> getTreePath(
      AtomStream.Node node, List<Object> path, Iterator<Long> indices) {
    if (!indices.hasNext()) {
      return Futures.immediateFuture(new TreePath(path.toArray()));
    }
    ListenableFuture<AtomStream.Node> load = models.atoms.load(node);
    return (load == null) ? getTreePathForLoadedNode(node, path, indices) :
      Futures.transformAsync(load, loaded -> getTreePathForLoadedNode(loaded, path, indices));
  }

  private ListenableFuture<TreePath> getTreePathForLoadedNode(
      AtomStream.Node node, List<Object> path, Iterator<Long> indices) {
    AtomStream.Node child = node.getChild(indices.next().intValue());
    path.add(child);
    return getTreePath(child, path, indices);
  }

  /**
   * Content provider for the command tree.
   */
  private static class AtomContentProvider implements ILazyTreeContentProvider {
    private final AtomStream atoms;
    private final TreeViewer viewer;
    private final Widgets.Refresher refresher;

    public AtomContentProvider(AtomStream atoms, TreeViewer viewer) {
      this.atoms = atoms;
      this.viewer = viewer;
      this.refresher = Widgets.withAsyncRefresh(viewer);
    }

    @Override
    public void updateChildCount(Object element, int currentChildCount) {
      viewer.setChildCount(element, ((AtomStream.Node)element).getChildCount());
    }

    @Override
    public void updateElement(Object parent, int index) {
      AtomStream.Node child = ((AtomStream.Node)parent).getChild(index);
      atoms.load(child, refresher::refresh);
      viewer.replace(parent, index, child);
      viewer.setHasChildren(child, child.getChildCount() > 0);
    }

    @Override
    public Object getParent(Object element) {
      return ((AtomStream.Node)element).getParent();
    }
  }

  /**
   * Image provider for the command tree. Groups that represent frames or draw calls will have
   * a thumbnail preview of the framebuffer in the tree.
   */
  private static class ImageProvider implements LoadingIndicator.Repaintable {
    private static final int PREVIEW_SIZE = 18;

    private final Thumbnails thumbs;
    private final TreeViewer viewer;
    private final LoadingIndicator loading;
    private final Map<AtomStream.Node, LoadableImage> images = Maps.newIdentityHashMap();

    public ImageProvider(Thumbnails thumbs, TreeViewer viewer, LoadingIndicator loading) {
      this.thumbs = thumbs;
      this.viewer = viewer;
      this.loading = loading;
    }

    public void load(AtomStream.Node group) {
      LoadableImage image = getLoadableImage(group);
      if (image != null) {
        image.load();
      }
    }

    public void unload(AtomStream.Node group) {
      LoadableImage image = getLoadableImage(group);
      if (image != null) {
        image.unload();
      }
    }

    public Image getImage(AtomStream.Node group) {
      LoadableImage image = getLoadableImage(group);
      return (image == null) ? null : image.getImage();
    }

    private LoadableImage getLoadableImage(AtomStream.Node group) {
      LoadableImage image = images.get(group);
      if (image == null) {
        if (!shouldShowImage(group) || !thumbs.isReady()) {
          return null;
        }

        image = LoadableImage.forSmallImageData(
            viewer.getTree(), () -> loadImage(group), loading, this);
        images.put(group, image);
      }
      return image;
    }

    @Override
    public void repaint() {
      ifNotDisposed(viewer.getControl(), viewer::refresh);
    }

    private static boolean shouldShowImage(AtomStream.Node node) {
      return node.getData() != null && !node.getData().getGroup().isEmpty();
    }

    private ListenableFuture<ImageData> loadImage(AtomStream.Node node) {
      return noAlpha(thumbs.getThumbnail(node.getPath(Path.CommandTreeNode.newBuilder()).build(), PREVIEW_SIZE));
    }

    public void reset() {
      for (LoadableImage image : images.values()) {
        image.dispose();
      }
      images.clear();
    }
  }

  /**
   * Label provider for the command tree.
   */
  private static class ViewLabelProvider extends MeasuringViewLabelProvider
      implements VisibilityTrackingTreeViewer.Listener {
    private final ConstantSets constants;
    private final ImageProvider imageProvider;

    public ViewLabelProvider(
        TreeViewer viewer, ConstantSets constants, Theme theme, ImageProvider imageProvider) {
      super(viewer, theme);
      this.constants = constants;
      this.imageProvider = imageProvider;
    }

    @Override
    protected <S extends StylingString> S format(Object element, S string) {
      CommandTreeNode data = ((AtomStream.Node)element).getData();
      if (data == null) {
        string.append("Loading...", string.structureStyle());
      } else {
        if (data.getGroup().isEmpty() && data.hasCommands()) {
          string.append(Formatter.lastIndex(data.getCommands()) + ": ", string.defaultStyle());
          Command cmd = ((AtomStream.Node)element).getCommand();
          if (cmd == null) {
            string.append("Loading...", string.structureStyle());
          } else {
            Formatter.format(cmd, constants::getConstants, string, string.identifierStyle());
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
    protected Image getImage(Object element) {
      Image result = null;
      if (element instanceof AtomStream.Node) {
        result = imageProvider.getImage((AtomStream.Node)element);
      }
      return result;
    }

    @Override
    protected boolean isFollowable(Object element) {
      //return element instanceof AtomNode;
      return false;
    }

    @Override
    public void onShow(TreeItem item) {
      Object element = item.getData();
      if (element instanceof AtomStream.Node) {
        imageProvider.load((AtomStream.Node)element);
      }
    }

    @Override
    public void onHide(TreeItem item) {
      Object element = item.getData();
      if (element instanceof AtomStream.Node) {
        imageProvider.unload((AtomStream.Node)element);
      }
    }
  }
}

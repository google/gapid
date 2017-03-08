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
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Ranges.count;
import static com.google.gapid.util.Ranges.first;
import static com.google.gapid.util.Ranges.last;
import static com.google.gapid.widgets.Widgets.createTree;
import static com.google.gapid.widgets.Widgets.expandOnDoubleClick;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;

import com.google.common.collect.Maps;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.Server.GapisInitException;
import com.google.gapid.models.ApiContext;
import com.google.gapid.models.ApiContext.FilteringContext;
import com.google.gapid.models.AtomHierarchies;
import com.google.gapid.models.AtomHierarchies.AtomNode;
import com.google.gapid.models.AtomHierarchies.FilteredGroup;
import com.google.gapid.models.AtomStream;
import com.google.gapid.models.Capture;
import com.google.gapid.models.Models;
import com.google.gapid.models.Thumbnails;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.CommandGroup;
import com.google.gapid.proto.service.Service.CommandRange;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.service.atom.Atom;
import com.google.gapid.service.atom.DynamicAtom;
import com.google.gapid.service.snippets.CanFollow;
import com.google.gapid.service.snippets.Pathway;
import com.google.gapid.util.Events;
import com.google.gapid.util.Messages;
import com.google.gapid.util.MouseAdapter;
import com.google.gapid.util.Paths;
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
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.ILazyTreeContentProvider;
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
import org.eclipse.swt.widgets.Display;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Menu;
import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeItem;

import java.util.Map;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import java.util.logging.Level;
import java.util.logging.Logger;
import java.util.regex.Pattern;

public class AtomTree extends Composite implements Capture.Listener, AtomStream.Listener,
    ApiContext.Listener, AtomHierarchies.Listener, Thumbnails.Listener {
  protected static final Logger LOG = Logger.getLogger(AtomTree.class.getName());
  private static final int PREVIEW_HOVER_DELAY_MS = 500;

  private final Models models;
  private final LoadablePanel<Tree> loading;
  private final TreeViewer viewer;
  private final ImageProvider imageProvider;
  private final SelectionHandler<Tree> selectionHandler;
  private FilteredGroup root;

  public AtomTree(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new GridLayout(1, false));

    SearchBox search = new SearchBox(this);
    loading = new LoadablePanel<Tree>(this, widgets,
        loadingParent -> createTree(loadingParent, SWT.H_SCROLL | SWT.V_SCROLL | SWT.VIRTUAL));
    Tree tree = loading.getContents();
    tree.setLinesVisible(true);
    viewer = new TreeViewer(tree);
    imageProvider = new ImageProvider(models.thumbs, viewer, widgets.loading);
    viewer.setUseHashlookup(true);
    viewer.setContentProvider(new AtomContentProvider(viewer));
    ViewLabelProvider labelProvider = new ViewLabelProvider(viewer, widgets.theme, imageProvider);
    viewer.setLabelProvider(labelProvider);
    expandOnDoubleClick(viewer);

    search.setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));
    loading.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    models.capture.addListener(this);
    models.atoms.addListener(this);
    models.contexts.addListener(this);
    models.hierarchies.addListener(this);
    models.thumbs.addListener(this);

    search.addListener(Events.Search, e -> search(e.text, (e.detail & Events.REGEX) != 0));

    selectionHandler = new SelectionHandler<Tree>(LOG, tree) {
      @Override
      protected void updateModel(Event e) {
        Object selection = (tree.getSelectionCount() > 0) ? tree.getSelection()[0].getData() : null;
        if (selection instanceof FilteredGroup) {
          models.atoms.selectAtoms(((FilteredGroup)selection).group.getRange(), false);
        } else if (selection instanceof AtomNode) {
          models.atoms.selectAtoms(((AtomNode)selection).index, 1, false);
        }
      }
    };

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
        CanFollow follow = (CanFollow)labelProvider.getFollow(location);
        if (follow != null) {
          models.follower.follow(getFollowPath(tree.getItem(location), follow.getPath()));
        }
      }

      private void updateHover(int x, int y) {
        TreeItem item = tree.getItem(new Point(x, y));
        if (item != null && (item.getData() instanceof FilteredGroup) &&
            item.getImage() != null && item.getImageBounds(0).contains(x, y)) {
          hover(item);
        } else {
          hover(null);

          CanFollow follow = (CanFollow)labelProvider.getFollow(new Point(x, y));
          setCursor((follow == null) ? null : getDisplay().getSystemCursor(SWT.CURSOR_HAND));
          if (follow != null) {
            models.follower.prepareFollow(getFollowPath(item, follow.getPath()));
          }
        }
      }

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

      private void hover(TreeItem item) {
        if (item != lastHoveredItem) {
          lastScheduledFuture.cancel(true);
          lastHoveredItem = item;
          if (item != null) {
            lastScheduledFuture = Scheduler.EXECUTOR.schedule(() ->
              Widgets.scheduleIfNotDisposed(
                  tree, () -> showBalloon(item, (FilteredGroup)item.getData())),
              PREVIEW_HOVER_DELAY_MS, TimeUnit.MILLISECONDS);
          }
          if (lastShownBalloon != null) {
            lastShownBalloon.close();
          }
        }
      }

      private void showBalloon(TreeItem item, FilteredGroup group) {
        if (lastShownBalloon != null) {
          lastShownBalloon.close();
        }
        Rectangle bounds = item.getImageBounds(0);
        lastShownBalloon = Balloon.createAndShow(tree, shell -> {
          LoadableImageWidget.forImageData(shell, loadImage(group), widgets.loading);
        }, new Point(bounds.x + bounds.width + 2, bounds.y + bounds.height / 2 - THUMB_SIZE / 2));
      }

      private ListenableFuture<ImageData> loadImage(FilteredGroup group) {
        return noAlpha(models.thumbs.getThumbnail(group.getIndexOfLastLeaf(), THUMB_SIZE));
      }
    };
    tree.addMouseListener(mouseHandler);
    tree.addMouseMoveListener(mouseHandler);
    tree.addMouseWheelListener(mouseHandler);
    tree.getVerticalBar().addSelectionListener(mouseHandler);

    CopySources.registerTreeAsCopySource(widgets.copypaste, viewer, object -> {
      if (object instanceof FilteredGroup) {
        Service.CommandGroup group = ((FilteredGroup)object).group;
        CommandRange range = group.getRange();
        return new String[] { group.getName(), "(" + first(range) + " - " + last(range) + ")" };
      } else if (object instanceof AtomNode) {
        AtomNode node = (AtomNode)object;
        return new String[] { node.index + ":", Formatter.toString((DynamicAtom)node.atom) };
      }
      return new String[] { String.valueOf(object) };
    });
  }

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

  @Override
  public void dispose() {
    imageProvider.reset();
    super.dispose();
  }

  @Override
  public void onCaptureLoadingStart() {
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
  public void onAtomsSelected(CommandRange range) {
    FilteredGroup group = root;
    selectionHandler.updateSelectionFromModel(
        () -> (group == null) ? null : group.getTreePathTo(range),
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
  public void onHierarchiesLoaded() {
    updateTree(false);
  }

  @Override
  public void onThumnailsChanged() {
    imageProvider.reset();
    viewer.refresh();
  }

  private void updateTree(boolean assumeLoading) {
    imageProvider.reset();
    root = null;

    if (assumeLoading || !models.atoms.isLoaded() || !models.contexts.isLoaded() ||
        !models.hierarchies.isLoaded()) {
      loading.startLoading();
      viewer.setInput(null);
      return;
    }

    loading.stopLoading();
    root = models.hierarchies.getHierarchy(
        models.atoms.getData(), models.contexts.getSelectedContext());
    viewer.setInput(root);
    viewer.getTree().setSelection(viewer.getTree().getItem(0));
    viewer.getTree().showSelection();
  }

  private static class AtomContentProvider implements ILazyTreeContentProvider {
    private final TreeViewer viewer;

    public AtomContentProvider(TreeViewer viewer) {
      this.viewer = viewer;
    }

    @Override
    public void updateChildCount(Object element, int currentChildCount) {
      FilteredGroup group = (FilteredGroup)element;
      viewer.setChildCount(element, group.getChildCount());
    }

    @Override
    public void updateElement(Object parent, int index) {
      FilteredGroup group = (FilteredGroup)parent;
      Object child = group.getChild(index);
      viewer.replace(parent, index, child);
      viewer.setHasChildren(child, child instanceof FilteredGroup);
    }

    @Override
    public Object getParent(Object element) {
      return null;
    }
  }

  private static class ImageProvider implements LoadingIndicator.Repaintable {
    private static final int PREVIEW_SIZE = 18;

    private final Thumbnails thumbs;
    private final TreeViewer viewer;
    private final LoadingIndicator loading;
    private final Map<FilteredGroup, LoadableImage> images = Maps.newIdentityHashMap();

    public ImageProvider(Thumbnails thumbs, TreeViewer viewer, LoadingIndicator loading) {
      this.thumbs = thumbs;
      this.viewer = viewer;
      this.loading = loading;
    }

    public Image getImage(FilteredGroup group) {
      LoadableImage image = images.get(group);
      if (image == null) {
        if (!shouldShowImage(group) || !thumbs.isReady()) {
          return null;
        }

        image = LoadableImage.forSmallImageData(
            viewer.getTree(), () -> loadImage(group), loading, this);
        images.put(group, image);
      }
      return image.getImage();
    }

    public void onPaint(FilteredGroup group) {
      LoadableImage image = images.get(group);
      if (image != null) {
        image.load();
      }
    }

    @Override
    public void repaint() {
      scheduleIfNotDisposed(viewer.getTree(), viewer::refresh);
    }

    private static boolean shouldShowImage(FilteredGroup group) {
      Atom atom = group.getLastLeaf();
      return atom != null && (atom.isDrawCall() || atom.isEndOfFrame());
    }

    private ListenableFuture<ImageData> loadImage(FilteredGroup group) {
      long index = group.getIndexOfLastDrawCall();
      if (index < 0) {
        index = group.getIndexOfLastLeaf();
      }
      return noAlpha(thumbs.getThumbnail(index, PREVIEW_SIZE));
    }

    public void reset() {
      for (LoadableImage image : images.values()) {
        image.dispose();
      }
      images.clear();
    }
  }

  private static class ViewLabelProvider extends MeasuringViewLabelProvider {
    private final ImageProvider imageProvider;

    public ViewLabelProvider(TreeViewer viewer, Theme theme, ImageProvider imageProvider) {
      super(viewer, theme);
      this.imageProvider = imageProvider;
    }

    @Override
    protected <S extends StylingString> S format(Object element, S string) {
      if (element instanceof FilteredGroup) {
        CommandGroup group = ((FilteredGroup)element).group;
        string.append(first(group.getRange()) + ": ", string.defaultStyle());
        string.append(group.getName(), string.labelStyle());
        long count = count(group.getRange());
        string.append(
            " (" + count + " Command" + (count != 1 ? "s" : "") + ")", string.structureStyle());
      } else if (element instanceof AtomNode) {
        AtomNode atom = (AtomNode)element;
        string.append(atom.index + ": ", string.defaultStyle());
        Formatter.format((DynamicAtom)atom.atom, string, string.identifierStyle());
      }
      return string;
    }

    @Override
    protected Image getImage(Object element) {
      Image result = null;
      if (element instanceof FilteredGroup) {
        result = imageProvider.getImage((FilteredGroup)element);
      }
      return result;
    }

    @Override
    protected boolean isFollowable(Object element) {
      return element instanceof AtomNode;
    }

    @Override
    protected void paint(Event event, Object element) {
      if (element instanceof FilteredGroup) {
        imageProvider.onPaint((FilteredGroup)element);
      }
      super.paint(event, element);
    }
  }
}

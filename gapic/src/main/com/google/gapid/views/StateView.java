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
import static com.google.gapid.widgets.Widgets.createTreeForViewer;
import static com.google.gapid.widgets.Widgets.createTreeViewer;

import com.google.gapid.Server.GapisInitException;
import com.google.gapid.models.ApiState;
import com.google.gapid.models.AtomStream;
import com.google.gapid.models.AtomStream.AtomIndex;
import com.google.gapid.models.Capture;
import com.google.gapid.models.ConstantSets;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service.StateTreeNode;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.Messages;
import com.google.gapid.views.Formatter.StylingString;
import com.google.gapid.widgets.CopySources;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.MeasuringViewLabelProvider;
import com.google.gapid.widgets.TextViewer;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.ILazyTreeContentProvider;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Menu;
import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeItem;

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
  private final TreeViewer viewer;
  //private final SelectionHandler<Tree> selectionHandler;

  public StateView(Composite parent, Models models, Widgets widgets) {
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

    /*
    selectionHandler = new SelectionHandler<Tree>(LOG, tree) {
      @Override
      protected void updateModel(Event e) {
        // Do nothing.
      }
    };
    */

    tree.addListener(SWT.MouseMove, e -> {
      Point location = new Point(e.x, e.y);
      //CanFollow follow = (CanFollow)labelProvider.getFollow(location);
      Object follow = null;
      setCursor((follow == null) ? null : e.display.getSystemCursor(SWT.CURSOR_HAND));

      if (follow != null) {
        //models.follower.prepareFollow(getFollowPath(location));
      }
    });
    tree.addListener(SWT.MouseDown, e -> {
      Point location = new Point(e.x, e.y);
      if (labelProvider.getFollow(location) != null) {
        //models.follower.follow(getFollowPath(location));
      }
    });

    Menu popup = new Menu(tree);
    Widgets.createMenuItem(popup, "&View Text", SWT.MOD1 + 'T', e -> {
      TreeItem item = (tree.getSelectionCount() > 0) ? tree.getSelection()[0] : null;
      if (item != null && (item.getData() instanceof ApiState.Node)) {
        StateTreeNode data = ((ApiState.Node)item.getData()).getData();

        if (data == null) {
          // Data not loaded yet, this shouldn't happen (see MenuDetect handler). Ignore.
          LOG.log(Level.WARNING, "Impossible popup requested and ignored.");
          return;
        } else if (!data.hasPreview()) {
          // No value at this node, this shouldn't happen, either. Ignore.
          LOG.log(Level.WARNING, "Value-less popup requested and ignored: {0}", data);
          return;
        }

        String text = Formatter.toString(
            data.getPreview(), models.constants.getConstants(data.getConstants()));
        TextViewer.showViewTextPopup(getShell(), data.getName() + ":", text);
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
        StateTreeNode node = ((ApiState.Node)object).getData();
        if (node == null) {
          // Copy before loaded. Not ideal, but this is unlikely.
          return new String[] { "Loading..." };
        } else if (!node.hasPreview()) {
          return new String[] { node.getName() };
        }

        String text = Formatter.toString(
            node.getPreview(), models.constants.getConstants(node.getConstants()));
        return new String[] { node.getName(), text };
      }

      return new String[] { String.valueOf(object) };
    });
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
  public void onCaptureLoaded(GapisInitException error) {
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
  public void onStateLoaded(DataUnavailableException error) {
    if (error != null) {
      loading.showMessage(Error, error.getMessage());
      return;
    }

    loading.stopLoading();
    viewer.setInput(models.state.getRoot());

    /*
    Path.Any selection = models.state.getSelectedPath();
    if (selection == null) {
      viewer.setSelection(new TreeSelection(new TreePath(new Object[] { viewer.getInput() })), true);
    } else {
      onStateSelected(selection);
    }
    */
  }

  /*
  @Override
  public void onStateSelected(Path.Any path) {
    Element root = (Element)viewer.getInput();
    if (root == null) {
      return;
    }

    selectionHandler.updateSelectionFromModel(() -> {
      Element element = root;
      SnippetObject[] selection = getStatePath(path);
      List<Element> segments = Lists.newArrayList();
      for (int i = 0; i < selection.length; i++) {
        element = element.findChild(selection[i]);
        if (element == null) {
          break; // Didn't find child at current level. Give up.
        }
        segments.add(element);
      }
      return segments.isEmpty() ? null : new TreePath(segments.toArray());
    }, selection -> {
      viewer.setSelection(new TreeSelection(selection), true);
      viewer.setExpandedState(selection, true);
    });
  }
  */

  /*
  private Path.Any getFollowPath(Point location) {
    TreeItem item = viewer.getTree().getItem(location);
    return (item == null) ? null : getFollowPath(item).build();
  }

  private Paths.PathBuilder getFollowPath(TreeItem item) {
    if (item == null) {
      return new Paths.PathBuilder.State(models.state.getPath().getState());
    }

    Paths.PathBuilder parent = getFollowPath(item.getParentItem());
    if (parent == null || parent == Paths.PathBuilder.INVALID_BUILDER) {
      return Paths.PathBuilder.INVALID_BUILDER;
    }

    Element element = (Element)item.getData();
    Object obj = element.key.value.getObject();
    if (element.isMapKey) {
      return parent.map(pod(obj, element.key.type));
    } else if (obj instanceof Long) {
      return parent.array((Long)obj);
    } else if (obj instanceof String) {
      return parent.field((String)obj);
    } else {
      LOG.log(WARNING, "Unexpected object type " + obj);
      return Paths.PathBuilder.INVALID_BUILDER;
    }
  }

  private static class Element {
     private static boolean isMemorySliceInfo(Dynamic d) {
      return d.getFieldCount() == 1 && d.getFieldValue(0) instanceof MemorySliceInfo;
    }

    public CanFollow canFollow() {
      return CanFollow.fromSnippets(value.value.getSnippets());
    }
  }
  */

  private static boolean canShowPopup(TreeItem item) {
    if (item == null || !(item.getData() instanceof ApiState.Node)) {
      return false;
    }
    StateTreeNode data = ((ApiState.Node)item.getData()).getData();
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

    public ViewLabelProvider(TreeViewer viewer, ConstantSets constants, Theme theme) {
      super(viewer, theme);
      this.constants = constants;
    }

    @Override
    protected <S extends StylingString> S format(Object element, S string) {
      StateTreeNode data = ((ApiState.Node)element).getData();
      if (data == null) {
        string.append("Loading...", string.structureStyle());
      } else {
        string.append(data.getName(), string.defaultStyle());
        if (data.hasPreview()) {
         string.append(": ", string.structureStyle());
         Formatter.format(data.getPreview(), constants.getConstants(data.getConstants()),
             string, string.defaultStyle());
        }
      }
      return string;
    }

    @Override
    protected boolean isFollowable(Object element) {
      /*
      if (!(element instanceof Element)) {
        return false;
      }
      Element e = (Element)element;
      return e.isLeaf() && e.canFollow() != null;
      */
      return false;
    }
  }
}

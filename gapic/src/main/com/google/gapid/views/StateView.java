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
import static com.google.gapid.util.Pods.pod;
import static com.google.gapid.util.Pods.unpod;
import static com.google.gapid.widgets.Widgets.createTreeForViewer;
import static com.google.gapid.widgets.Widgets.createTreeViewer;
import static java.util.logging.Level.WARNING;

import com.google.common.collect.Iterables;
import com.google.common.collect.Lists;
import com.google.gapid.Server.GapisInitException;
import com.google.gapid.models.ApiState;
import com.google.gapid.models.AtomStream;
import com.google.gapid.models.Capture;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service.CommandRange;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpclib.schema.Array;
import com.google.gapid.rpclib.schema.Dynamic;
import com.google.gapid.rpclib.schema.Field;
import com.google.gapid.rpclib.schema.Map;
import com.google.gapid.rpclib.schema.Method;
import com.google.gapid.rpclib.schema.Primitive;
import com.google.gapid.rpclib.schema.Slice;
import com.google.gapid.rpclib.schema.Type;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.service.memory.MemorySliceInfo;
import com.google.gapid.service.snippets.CanFollow;
import com.google.gapid.service.snippets.KindredSnippets;
import com.google.gapid.service.snippets.SnippetObject;
import com.google.gapid.util.Messages;
import com.google.gapid.util.Paths;
import com.google.gapid.util.SelectionHandler;
import com.google.gapid.views.Formatter.Style;
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
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Menu;
import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeItem;

import java.util.List;
import java.util.Objects;
import java.util.logging.Logger;

/**
 * View that displays the API state as a tree.
 */
public class StateView extends Composite
    implements Tab, Capture.Listener, AtomStream.Listener, ApiState.Listener {
  private static final Logger LOG = Logger.getLogger(StateView.class.getName());
  private static final TypedValue ROOT_TYPE = new TypedValue(null, SnippetObject.symbol("state"));

  private final Models models;
  private final LoadablePanel<Tree> loading;
  private final TreeViewer viewer;
  private final SelectionHandler<Tree> selectionHandler;

  public StateView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new FillLayout(SWT.VERTICAL));

    loading = LoadablePanel.create(this, widgets,
        panel -> createTreeForViewer(panel, SWT.H_SCROLL | SWT.V_SCROLL | SWT.VIRTUAL | SWT.MULTI));
    Tree tree = loading.getContents();
    viewer = createTreeViewer(tree);
    viewer.setContentProvider(new StateContentProvider(viewer));
    ViewLabelProvider labelProvider = new ViewLabelProvider(viewer, widgets.theme);
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
        // Do nothing.
      }
    };

    tree.addListener(SWT.MouseMove, e -> {
      Point location = new Point(e.x, e.y);
      CanFollow follow = (CanFollow)labelProvider.getFollow(location);
      setCursor((follow == null) ? null : e.display.getSystemCursor(SWT.CURSOR_HAND));

      if (follow != null) {
        models.follower.prepareFollow(getFollowPath(location));
      }
    });
    tree.addListener(SWT.MouseDown, e -> {
      Point location = new Point(e.x, e.y);
      if (labelProvider.getFollow(location) != null) {
        models.follower.follow(getFollowPath(location));
      }
    });

    Menu popup = new Menu(tree);
    Widgets.createMenuItem(popup, "&View Text", SWT.MOD1 + 'T', e -> {
      TreeItem item = (tree.getSelectionCount() > 0) ? tree.getSelection()[0] : null;
      if (item != null && (item.getData() instanceof Element)) {
        Element element = (Element)item.getData();

        String text = Formatter.toString(element.value.value, element.value.type);
        String title = Formatter.toString(element.key.value, element.key.type) + ":";

        TextViewer.showViewTextPopup(getShell(), title, text);
      }
    });
    tree.setMenu(popup);
    tree.addListener(SWT.MenuDetect, e -> {
      TreeItem item = tree.getItem(tree.toControl(e.x, e.y));
      if (item == null || !(item.getData() instanceof Element) ||
          !((Element)item.getData()).requiresPopup()) {
        e.doit = false;
      }
    });

    CopySources.registerTreeAsCopySource(widgets.copypaste, viewer, object -> {
      if (object instanceof Element) {
        Element node = (Element)object;
        String key = "";
        if (node.key.type != null) {
          key = Formatter.toString(node.key.value, node.key.type);
        } else {
          key = String.valueOf(node.key.value.getObject());
        }

        if (!node.isLeaf() || node.value == null || node.value.value == null ||
            node.value.value.getObject() == null) {
          return new String[] { key };
        }

        return new String[] { key, Formatter.toString(node.value.value, node.value.type) };
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
        if (models.state.getState() != null) {
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
  public void onAtomsSelected(CommandRange path) {
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
    viewer.setInput(new Element(ROOT_TYPE, new TypedValue(
        null, SnippetObject.root(models.state.getState(), getSnippets(models.state)))));

    Path.Any selection = models.state.getSelectedPath();
    if (selection == null) {
      viewer.setSelection(new TreeSelection(new TreePath(new Object[] { viewer.getInput() })), true);
    } else {
      onStateSelected(selection);
    }
  }

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

  private static KindredSnippets[] getSnippets(ApiState state) {
    return KindredSnippets.fromMetadata(state.getState().klass().entity().getMetadata());
  }

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

  private static SnippetObject[] getStatePath(Path.Any path) {
    List<SnippetObject> result = Lists.newLinkedList();
    Paths.visit(path, new Paths.Visitor() {
      @Override
      public void array(Path.ArrayIndex array) {
        result.add(0, SnippetObject.symbol(array.getIndex()));
      }

      @Override
      public void map(Path.MapIndex map) {
        result.add(0, SnippetObject.symbol(unpod(map.getPod())));
      }

      @Override
      public void field(Path.Field field) {
        result.add(0, SnippetObject.symbol(field.getName()));
      }
    });
    return result.toArray(new SnippetObject[result.size()]);
  }

  /**
   * A {@link SnippetObject value} with a {@link Type} that is displayed in the state tree.
   */
  private static class TypedValue {
    public final Type type;
    public final SnippetObject value;

    public TypedValue(Type type, SnippetObject value) {
      this.type = type;
      this.value = value;
    }
  }

  /**
   * A single element in the state tree consisting of a name (such as GL_DEPTH_TEST) and its current
   * value (e.g. GL_TRUE or GL_FLASE).
   */
  private static class Element {
    public final TypedValue key;
    public final TypedValue value;
    public final boolean isMapKey;
    private final Element[] children;

    public Element(TypedValue key, TypedValue value) {
      this(key, value, false);
    }

    public Element(TypedValue key, TypedValue value, boolean isMapKey) {
      this.key = key;
      this.value = value;
      this.isMapKey = isMapKey;
      this.children = new Element[getChildCount(value)];
    }

    private static int getChildCount(TypedValue value) {
      Object underlying = value.value.getObject();
      if (value.value.getObject() instanceof Dynamic) {
        Dynamic d = (Dynamic)value.value.getObject();
        // Don't create child Nodes for MemorySliceInfo, as they are shown simply as inline values.
        return isMemorySliceInfo(d) ? 0 : d.getFieldCount();
      } else if (value.type instanceof Map) {
        return ((java.util.Map<?, ?>)underlying).size();
      } else if (underlying instanceof Object[]) {
        return ((Object[])underlying).length;
      } else if (underlying instanceof byte[]) {
        return ((byte[])underlying).length;
      } else {
        return 0;
      }
    }

    private static boolean isMemorySliceInfo(Dynamic d) {
      return d.getFieldCount() == 1 && d.getFieldValue(0) instanceof MemorySliceInfo;
    }

    public int getChildCount() {
      return children.length;
    }

    public boolean isLeaf() {
      return children.length == 0;
    }

    public CanFollow canFollow() {
      return CanFollow.fromSnippets(value.value.getSnippets());
    }

    public boolean requiresPopup() {
      return value.type instanceof Primitive &&
          ((Primitive)value.type).getMethod() == Method.String;
    }

    public Element getChild(int index) {
      if (children[index] == null) {
        Object underlying = value.value.getObject();
        if (value.value.getObject() instanceof Dynamic) {
          Field field = ((Dynamic)underlying).getFieldInfo(index);
          SnippetObject fieldObj = value.value.field((Dynamic)underlying, index);
          children[index] = new Element(
              new TypedValue(null, SnippetObject.symbol(field.getDeclared())),
              new TypedValue(field.getType(), fieldObj));
        } else if (value.type instanceof Map) {
          java.util.Map.Entry<?, ?> entry =
              Iterables.get(((java.util.Map<?, ?>)underlying).entrySet(), index);
          Map map = (Map)value.type;
          Type keyType = map.getKeyType(), valueType = map.getValueType();
          children[index] = new Element(new TypedValue(keyType, value.value.key(entry)),
              new TypedValue(valueType, value.value.elem(entry)), true);
        } else if (underlying instanceof Object[]) {
          Type valueType = (value.type instanceof Slice) ?
              ((Slice)value.type).getValueType() : ((Array)value.type).getValueType();
          children[index] = new Element(new TypedValue(null, value.value.elem(index)),
              new TypedValue(valueType, value.value.elem(((Object[])underlying)[index])));
        } else if (underlying instanceof byte[]) {
          Type valueType = (value.type instanceof Slice) ?
              ((Slice)value.type).getValueType() : ((Array)value.type).getValueType();
          children[index] = new Element(new TypedValue(null, value.value.elem(index)),
              new TypedValue(valueType, value.value.elem(((byte[])underlying)[index])));
        } else {
          return null;
        }
      }
      return children[index];
    }

    public Element findChild(Object searchKey) {
      for (int i = 0; i < getChildCount(); i++) {
        Element child = getChild(i);
        if (child != null && Objects.equals(searchKey, child.key.value)) {
          return child;
        }
      }
      return null;
    }
  }

  /**
   * Content provider for the state tree.
   */
  private static class StateContentProvider implements ILazyTreeContentProvider {
    private final TreeViewer viewer;

    public StateContentProvider(TreeViewer viewer) {
      this.viewer = viewer;
    }

    @Override
    public void updateChildCount(Object element, int currentChildCount) {
      viewer.setChildCount(element, ((Element)element).getChildCount());
    }

    @Override
    public void updateElement(Object parent, int index) {
      Element element = ((Element)parent).getChild(index);
      if (element != null) {
        viewer.replace(parent, index, element);
        viewer.setChildCount(element, element.getChildCount());
      }
    }

    @Override
    public Object getParent(Object element) {
      return null;
    }
  }

  /**
   * Label provider for the state tree.
   */
  private static class ViewLabelProvider extends MeasuringViewLabelProvider {
    public ViewLabelProvider(TreeViewer viewer, Theme theme) {
      super(viewer, theme);
    }

    @Override
    protected <S extends StylingString> S format(Object element, S string) {
      if (element instanceof Element) {
        format((Element)element, string);
      }
      return string;
    }

    @Override
    protected boolean isFollowable(Object element) {
      if (!(element instanceof Element)) {
        return false;
      }
      Element e = (Element)element;
      return e.isLeaf() && e.canFollow() != null;
    }

    private static <S extends StylingString> void format(Element element, S string) {
      if (element.key.type != null) {
        Formatter.format(element.key.value, element.key.type, string, string.defaultStyle());
      } else {
        string.append(String.valueOf(element.key.value.getObject()), string.defaultStyle());
      }
      if (element.isLeaf()
          /*TODO || (!expanded && node.canBeRenderedAsLeaf())) &&
                node.value != null && node.value.value != null*/) {
        string.append(": ", string.structureStyle());
        if (element.value.value.getObject() != null) {
          CanFollow follow = element.canFollow();
          Style style = (follow != null) ? string.linkStyle() : string.defaultStyle();
          string.startLink(follow);
          Formatter.format(element.value.value, element.value.type, string, style);
          string.endLink();
        } else {
          string.append("null", string.defaultStyle());
        }
      }
    }
  }
}

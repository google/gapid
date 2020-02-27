/*
 * Copyright (C) 2019 Google Inc.
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
package com.google.gapid.widgets;

import com.google.common.collect.Lists;
import com.google.common.collect.Sets;
import com.google.gapid.models.Analytics;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.util.Events;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.graphics.Region;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Canvas;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Layout;
import org.eclipse.swt.widgets.Shell;

import java.util.Arrays;
import java.util.List;
import java.util.ListIterator;
import java.util.Objects;
import java.util.Set;
import java.util.function.Function;

public class TabComposite extends Composite {
  private static final int TITLE_WIDTH = 50;
  private static final int SEP_HEIGHT = 2;
  private static final int BAR_MARGIN = 6;
  private static final int TAB_MARGIN = 10;
  private static final int FOLDER_MARGIN = 5; // needs to be odd.
  private static final int ICON_SIZE = 24;
  private static final int MIN_WIDTH = 50;
  private static final int MIN_HEIGHT = 75;

  protected final Theme theme;
  private final Group group;
  private final Events.ListenerCollection<Listener> listeners = Events.listeners(Listener.class);
  private Folder maximizedFolder = null;
  protected Hover hovered = Hover.NONE;
  private Hover mouseDown = Hover.NONE;
  protected Dragger dragger = null;

  public TabComposite(Composite parent, Theme theme, boolean horizontal) {
    super(parent, SWT.BORDER | SWT.DOUBLE_BUFFERED);
    this.theme = theme;
    this.group = horizontal ? new HorizontalGroup(1) : new VerticalGroup(1);

    setLayout(new Layout() {
      @Override
      protected Point computeSize(Composite composite, int wHint, int hHint, boolean flushCache) {
        return layoutComputeSize(wHint, hHint, flushCache);
      }

      @Override
      protected void layout(Composite composite, boolean flushCache) {
        layoutLayout();
      }

      @Override
      protected boolean flushCache(Control control) {
        return true;
      }
    });

    addListener(SWT.Paint, e -> {
      getElement().draw(e.gc, false, false);
    });

    addListener(SWT.MouseDown, e -> {
      if (e.button == 1) {
        mouseDown = hovered;
        switch (mouseDown.type) {
          case Button:
            if (maximizedFolder == null) {
              mouseDown.folder.maximized = true;
              maximizedFolder = mouseDown.folder;
            } else {
              mouseDown.folder.maximized = false;
              maximizedFolder = null;
            }
            requestLayout();
            break;
          case Tab:
            if (mouseDown.folder.updateCurrent(mouseDown.tab.control)) {
              listeners.fire().onTabShown(mouseDown.tab.info);
            }
            break;
          default:
            // Do nothing.
        }
      }
    });

    addListener(SWT.MouseMove, e -> {
      switch (mouseDown.type) {
        case Tab:
          if (dragger == null) {
            dragger = new Dragger(theme, getShell(), getDisplay().map(this, null, getClientArea()),
                theme.tabFolderPlaceholderFill(), mouseDown);
            setCursor(getDisplay().getSystemCursor(SWT.CURSOR_SIZEALL));
            mouseDown.folder.redrawBar();
          }
          dragger.shell.setLocation(getDisplay().getCursorLocation());

          getElement().redrawBar(dragger.location.x, dragger.location.y, e.x, e.y);
          dragger.location.x = e.x;
          dragger.location.y = e.y;

          Hover current = group.find(group, 0, e.x, e.y, false);
          if (current.isFolder()) {
            Location location = getLocation(current.folder, e.x, e.y);
            if (location != null) {
              dragger.overlay.show(location.highlight(current.folder));
              dragger.shell.setActive();
            } else {
              dragger.overlay.hide();
            }
          } else {
            dragger.overlay.hide();
          }
          break;
        case Separator:
          mouseDown.group.moveSeparator(mouseDown.index, e.x, e.y);
          setRedraw(false);
          try {
            layoutLayout();
          } finally {
            setRedraw(true);
          }
          update();
          break;
        default:
          updateHover(getElement().find(group, 0, e.x, e.y, false));
      }
    });

    addListener(SWT.MouseUp, e -> {
      mouseDown = Hover.NONE;
      if (dragger != null) {
        Hover src = dragger.tab;
        Hover dst = getElement().find(group, 0, e.x, e.y, true);

        dragger.close();
        setCursor(null);
        dragger = null;

        switch (dst.type) {
          case Tab:
            listeners.fire().onTabMoved(src.tab.info);
            if (dst.tab == null) {
              src.folder.removeTab(src.tab);
              dst.folder.addTab(src.tab);
              src.folder.redrawBar();
              if (src.folder != dst.folder) {
                dst.folder.redrawBar();
              }
            } else if (src.folder != dst.folder) {
              src.folder.removeTab(src.tab);
              dst.folder.addTab(src.tab, dst.tab);
              src.folder.redrawBar();
              dst.folder.redrawBar();
            } else {
              dst.folder.moveTab(src.tab, dst.tab);
              src.folder.redrawBar();
            }
            group.merge();
            dst.folder.updateCurrent(src.tab.control);
            break;
          case Folder:
            Location location = getLocation(dst.folder, e.x, e.y);
            if (location != null) {
              Folder newFolder = dst.group.newSubFolder(location, dst.index);
              src.folder.removeTab(src.tab);
              newFolder.addTab(src.tab);
              src.folder.redrawBar();
              newFolder.redrawBar();
              group.merge();
              requestLayout();
            }
            break;
          default:
            // Do nothing.
        }
      }
      updateHover(getElement().find(group, 0, e.x, e.y, false));
    });

    addListener(SWT.MouseExit, e -> updateHover(Hover.NONE));
  }

  private static Location getLocation(Folder folder, int x, int y) {
    switch (3 * (x - folder.x) / folder.w) {
      case 0: return Location.Left;
      case 1:
        switch (3 * (y - folder.y) / folder.h) {
          case 0: return Location.Top;
          case 2: return Location.Bottom;
          default: return null;
        }
      case 2: return Location.Right;
      default: return null;
    }
  }

  private void updateHover(Hover newHover) {
    if (!hovered.isTab()) {
      if (newHover.isTab()) {
        newHover.folder.redrawBar();
      }
    } else if (!newHover.isTab()) {
      hovered.folder.redrawBar();
    } else if (hovered.folder != newHover.folder) {
      hovered.folder.redrawBar();
      newHover.folder.redrawBar();
    } else {
      newHover.folder.redrawBar();
    }
    hovered = newHover;

    if (hovered.isSeparator()) {
      setCursor(getDisplay().getSystemCursor(hovered.cursor));
    } else {
      setCursor(null);
    }
  }

  public Group getRoot() {
    return group;
  }

  public boolean showTab(Object id) {
    return group.showTab(id);
  }

  public void addTabToFirstFolder(TabInfo info) {
    group.addTabToFirstFolder(info);
  }

  public void addTabToLargestFolder(TabInfo info) {
    group.addTabToLargestFolder(info);
  }

  public boolean disposeTab(Object id) {
    if (group.disposeTab(id)) {
      group.merge();
      return true;
    }
    return false;
  }

  public void visit(Visitor visitor) {
    group.visit(visitor);
  }

  public void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public void removeListener(Listener listener) {
    listeners.removeListener(listener);
  }

  protected Point layoutComputeSize(int wHint, int hHint, boolean flushCache) {
    int w = 0, h = 0;
    for (Control child : getChildren()) {
      Point size = child.computeSize(wHint, hHint, flushCache);
      w = Math.max(size.x, w);
      h = Math.max(size.y, h);
    }
    return new Point((wHint == SWT.DEFAULT) ? w : wHint, (hHint == SWT.DEFAULT) ? h : hHint);
  }

  protected void layoutLayout() {
    Rectangle size = getClientArea();
    Set<Control> controls = Sets.newIdentityHashSet();
    controls.addAll(Arrays.asList(getChildren()));
    getElement().setBounds(controls, 0, 0, size.width, size.height);
    for (Control child : controls) {
      child.setVisible(false);
    }
  }

  private Element getElement() {
    return (maximizedFolder == null) ? group : maximizedFolder;
  }

  /**
   * Information about a single tab in a folder.
   */
  public static class TabInfo {
    public final Object id;
    public final Analytics.View view;
    public final String label;
    public final Function<Composite, Control> contentFactory;

    public TabInfo(
        Object id, View view, String label, Function<Composite, Control> contentFactory) {
      this.id = id;
      this.view = view;
      this.label = label;
      this.contentFactory = contentFactory;
    }
  }

  @SuppressWarnings("unused")
  public interface Visitor {
    public default void group(boolean horizontal, int weight) { /* do nothing */ }
    public default void endGroup() { /* do nothing */ }
    public default void folder(int weight) { /* do nothing */ }
    public default void tab(TabInfo tab) { /* do nothing */ }
    public default void endFolder() { /* do nothing */ }
  }

  private abstract class Element {
    protected int x, y, w, h;
    protected int weight = -1;

    public Element() {
    }

    public abstract boolean showTab(Object id);
    public abstract void addTabToFirstFolder(TabInfo tab);
    public abstract void addTabToLargestFolder(TabInfo tab);
    public abstract boolean disposeTab(Object id);

    public abstract void setBounds(Set<Control> controls, int x, int y, int w, int h);

    protected void setBounds(int x, int y, int w, int h) {
      this.x = x;
      this.y = y;
      this.w = w;
      this.h = h;
    }

    protected boolean contains(int ex, int ey) {
      return ex >= x && ex < x + w && ey >= y && ey < y + h;
    }

    protected void draw(GC gc, boolean hBorder, boolean vBorder) {
      if (hBorder) {
        gc.setForeground(theme.tabFolderLine());
        gc.drawLine(x, y - FOLDER_MARGIN / 2 - 1, x + w, y - FOLDER_MARGIN / 2 - 1);
      }
      if (vBorder) {
        gc.setForeground(theme.tabFolderLine());
        gc.drawLine(x - FOLDER_MARGIN / 2 - 1, y, x - FOLDER_MARGIN / 2 - 1, y + h);
      }

      draw(gc);
    }

    protected abstract void draw(GC gc);

    protected abstract Hover find(Group parent, int index, int mx, int my, boolean includeTrailing);
    protected abstract void redrawBar(int x1, int y1, int x2, int y2);

    protected abstract MergeState merge();

    protected abstract void visit(Visitor visitor);
  }

  public abstract class Group extends Element {
    protected final List<Element> children = Lists.newArrayList();
    private final boolean childHBorder, childVBorder;

    public Group(int weight, boolean childHBorder, boolean childVBorder) {
      this.weight = weight;
      this.childHBorder = childHBorder;
      this.childVBorder = childVBorder;
    }

    public Folder newFolder(int folderWeight) {
      Folder folder = new Folder(folderWeight);
      children.add(folder);
      return folder;
    }

    public Group newGroup(int groupWeight) {
      Group result = createGroup(groupWeight);
      children.add(result);
      return result;
    }

    @Override
    public boolean showTab(Object id) {
      for (Element child : children) {
        if (child.showTab(id)) {
          return true;
        }
      }
      return false;
    }

    @Override
    public void addTabToFirstFolder(TabInfo tab) {
      Element firstChild = children.get(0);
      if (firstChild instanceof Folder) {
        ((Folder)firstChild).newTab(tab);
      } else {
        firstChild.weight /= 2;
        Folder folder = new Folder(firstChild.weight);
        children.add(0, folder);
        folder.newTab(tab);
      }
    }

    @Override
    public void addTabToLargestFolder(TabInfo tab) {
      int max = 0;
      for (int i = 1; i < children.size(); i++) {
        if (children.get(i).weight > children.get(max).weight) {
          max = i;
        }
      }
      children.get(max).addTabToLargestFolder(tab);
    }

    @Override
    public boolean disposeTab(Object id) {
      for (Element child : children) {
        if (child.disposeTab(id)) {
          return true;
        }
      }
      return false;
    }

    protected abstract Group createGroup(int newWeight);

    protected abstract Folder newSubFolder(Location location, int index);

    @Override
    public void setBounds(Set<Control> controls, int x, int y, int w, int h) {
      setBounds(x, y, w, h);

      if (children.size() == 1) {
        children.get(0).setBounds(controls, x, y, w, h);
      } else {
        int weightSum = 0;
        for (Element child : children) {
          if (child.weight <= 0) {
            weightSum = -1;
            break;
          }
          weightSum += child.weight;
        }

        setChildBounds(controls, weightSum);
      }
    }

    protected abstract void setChildBounds(Set<Control> controls, int weightTotal);

    @Override
    protected void draw(GC gc) {
      for (int i = 0; i < children.size(); i++) {
        children.get(i).draw(gc, childHBorder && i > 0, childVBorder && i > 0);
      }
    }

    @Override
    protected void redrawBar(int x1, int y1, int x2, int y2) {
      boolean firstDone = x1 < 0 || y1 < 0, secondDone = x2 < 0 || y2 < 0;
      if (firstDone && secondDone) {
        return;
      }

      for (Element child : children) {
        boolean first = child.contains(x1, y1);
        boolean second = child.contains(x2, y2);
        if (first && second) {
          child.redrawBar(x1, y1, x2, y2);
          return;
        } else if (first) {
          child.redrawBar(x1, y1, -1, -1);
          if (secondDone) {
            return;
          }
          firstDone = true;
        } else if (second) {
          child.redrawBar(x2, y2, -1, -1);
          if (firstDone) {
            return;
          }
          secondDone = true;
        }
      }
    }

    @Override
    protected Hover find(Group parent, int index, int mx, int my, boolean includeTrailing) {
      for (int i = 0; i < children.size(); i++) {
        Element child = children.get(i);
        if (mx >= child.x && mx < child.x + child.w &&
            my >= child.y && my < child.y + child.h) {
          return child.find(this, i, mx, my, includeTrailing);
        }
      }
      return Hover.NONE;
    }

    protected abstract void moveSeparator(int index, int sx, int sy);

    @Override
    protected MergeState merge() {
      for (ListIterator<Element> it = children.listIterator(); it.hasNext(); ) {
        Element current = it.next();
        MergeState state = current.merge();
        if (state == MergeState.REMOVE) {
          it.remove();
        } else if (state == MergeState.DO_NOTHING) {
          // Do nothing.
        } else {
          if (state.replacement instanceof Folder) {
            state.replacement.weight = current.weight;
            it.set(state.replacement);
          } else {
            // The current child (C) is a group where it's only child is also a group (G). Thus,
            // C is superfluous and can be removed. However, G can not just become our child, since
            // it has the same horizontal vs. vertical layout as us, while our children must have
            // the opposite from us. This does mean, however, that G, too, is superfluous and all
            // it's children - our great-grand-children - can just become our children.
            it.remove(); // Has to be done before we add any new children.
            int totalWeight = 0;
            for (Element child : ((Group)state.replacement).children) {
              it.add(child);
              totalWeight += child.weight;
            }
            for (Element child : ((Group)state.replacement).children) {
              child.weight = (int)((child.weight * current.weight) / (double)totalWeight);
            }
          }
        }
      }

      switch (children.size()) {
        case 0: return MergeState.REMOVE;
        case 1: return MergeState.replace(children.get(0));
        default: return MergeState.DO_NOTHING;
      }
    }
  }

  private static enum Location {
    Left, Right, Top, Bottom;

    public Rectangle highlight(Folder f) {
      switch (this) {
        case Left:   return new Rectangle(f.x, f.y, f.w / 3, f.h);
        case Right:  return new Rectangle(f.x + 2 * f.w / 3, f.y, f.w / 3, f.h);
        case Top:    return new Rectangle(f.x, f.y, f.w, f.h / 3);
        case Bottom: return new Rectangle(f.x, f.y + 2 * f.h / 3, f.w, f.h / 3);
        default: throw new AssertionError();
      }
    }
  }

  private class HorizontalGroup extends Group {
    public HorizontalGroup(int weight) {
      super(weight, false, true);
    }

    @Override
    protected void setChildBounds(Set<Control> controls, int weightTotal) {
      int cw = w - (children.size() - 1) * FOLDER_MARGIN;
      if (weightTotal <= 0) {
        int fw = cw / children.size();
        int rem = cw % children.size();
        for (int i = 0, fx = x; i < children.size(); i++, rem--) {
          int add = rem > 0 ? 1 : 0;
          children.get(i).setBounds(controls, fx, y, fw + add, h);
          children.get(i).weight = fw;
          fx += fw + add + FOLDER_MARGIN;
        }
      } else {
        int diff = 0;
        if (weightTotal != cw) {
          for (Element child : children) {
            child.weight = (int)((child.weight * cw) / (double)weightTotal);
            diff += child.weight;
          }
          diff = cw - diff;
          if (diff >= children.size()) {
            for (Element child : children) {
              child.weight++;
            }
            diff -= children.size();
          }
        }
        for (int i = 0, fx = x; i < children.size(); i++, diff--) {
          int add = diff > 0 ? 1 : 0;
          int nw = children.get(i).weight + add;
          children.get(i).setBounds(controls, fx, y, nw, h);
          fx += nw + FOLDER_MARGIN;
        }
      }
    }

    @Override
    protected Group createGroup(int newWeight) {
      return new VerticalGroup(newWeight);
    }

    @Override
    protected Folder newSubFolder(Location location, int index) {
      switch (location) {
        case Left: {
          int nw = children.get(index).weight /= 2;
          Folder folder = new Folder(nw);
          children.add(index, folder);
          return folder;
        }
        case Right: {
          int nw = children.get(index).weight /= 2;
          Folder folder = new Folder(nw);
          children.add(index + 1, folder);
          return folder;
        }
        case Top: {
          Element old = children.get(index);
          Group g = createGroup(old.weight);
          children.set(index, g);
          Folder folder = g.newFolder(old.weight);
          g.children.add(old);
          return folder;
        }
        case Bottom:{
          Element old = children.get(index);
          Group g = createGroup(old.weight);
          children.set(index, g);
          g.children.add(old);
          return g.newFolder(old.weight);
        }
        default:
          throw new AssertionError();
      }
    }

    @Override
    protected Hover find(Group parent, int index, int mx, int my, boolean includeTrailing) {
      Hover result = super.find(parent, index, mx, my, includeTrailing);
      if (result == Hover.NONE && children.size() > 1) {
        Element before = children.get(0);
        for (int i = 1; i < children.size(); i++) {
          Element now = children.get(i);
          if (mx >= before.x + before.w && mx < now.x) {
            return Hover.separator(this, i, SWT.CURSOR_SIZEWE);
          }
          before = now;
        }
      }
      return result;
    }

    @Override
    protected void moveSeparator(int index, int sx, int sy) {
      Element before = children.get(index - 1);
      Element after = children.get(index);
      int newBeforeW = sx - FOLDER_MARGIN / 2 - before.x;
      int newAfterW = after.x + after.w - sx - FOLDER_MARGIN / 2 - 1;
      if (newBeforeW >= MIN_WIDTH && newAfterW >= MIN_WIDTH) {
        before.weight = newBeforeW;
        after.weight = newAfterW;
      }
    }

    @Override
    protected void visit(Visitor visitor) {
      visitor.group(true, weight);
      for (Element child : children) {
        child.visit(visitor);
      }
      visitor.endGroup();
    }
  }

  private class VerticalGroup extends Group {
    public VerticalGroup(int weight) {
      super(weight, true, false);
    }

    @Override
    protected void setChildBounds(Set<Control> controls, int weightTotal) {
      int ch = h - (children.size() - 1) * FOLDER_MARGIN;
      if (weightTotal <= 0) {
        int fh = ch / children.size();
        int rem = ch % children.size();
        for (int i = 0, fy = y; i < children.size(); i++, rem--) {
          int add = rem > 0 ? 1 : 0;
          children.get(i).setBounds(controls, x, fy, w, fh + add);
          children.get(i).weight = fh;
          fy += fh + add + FOLDER_MARGIN;
        }
      } else {
        int diff = 0;
        if (weightTotal != ch) {
          for (Element child : children) {
            child.weight = (int)((child.weight * ch) / (double)weightTotal);
            diff += child.weight;
          }
          diff = ch - diff;
          if (diff >= children.size()) {
            for (Element child : children) {
              child.weight++;
            }
            diff -= children.size();
          }
        }
        for (int i = 0, fy = y; i < children.size(); i++, diff--) {
          int add = diff > 0 ? 1 : 0;
          int nh = children.get(i).weight + add;
          children.get(i).setBounds(controls, x, fy, w, nh);
          fy += nh + FOLDER_MARGIN;
        }
      }
    }

    @Override
    protected Group createGroup(int newWeight) {
      return new HorizontalGroup(newWeight);
    }

    @Override
    protected Folder newSubFolder(Location location, int index) {
      switch (location) {
        case Left: {
          Element old = children.get(index);
          Group g = createGroup(old.weight);
          children.set(index, g);
          Folder folder = g.newFolder(old.weight);
          g.children.add(old);
          return folder;
        }
        case Right:{
          Element old = children.get(index);
          Group g = createGroup(old.weight);
          children.set(index, g);
          g.children.add(old);
          return g.newFolder(old.weight);
        }
        case Top: {
          int nw = children.get(index).weight /= 2;
          Folder folder = new Folder(nw);
          children.add(index, folder);
          return folder;
        }
        case Bottom: {
          int nw = children.get(index).weight /= 2;
          Folder folder = new Folder(nw);
          children.add(index + 1, folder);
          return folder;
        }
        default:
          throw new AssertionError();
      }
    }


    @Override
    protected Hover find(Group parent, int index, int mx, int my, boolean includeTrailing) {
      Hover result = super.find(parent, index, mx, my, includeTrailing);
      if (result == Hover.NONE && children.size() > 1) {
        Element before = children.get(0);
        for (int i = 1; i < children.size(); i++) {
          Element now = children.get(i);
          if (my >= before.y + before.h && my < now.y) {
            return Hover.separator(this, i, SWT.CURSOR_SIZENS);
          }
          before = now;
        }
      }
      return result;
    }

    @Override
    protected void moveSeparator(int index, int sx, int sy) {
      Element before = children.get(index - 1);
      Element after = children.get(index);
      int newBeforeH = sy - FOLDER_MARGIN / 2 - before.y;
      int newAfterH = after.y + after.h - sy - FOLDER_MARGIN / 2 - 1;
      if (newBeforeH >= MIN_HEIGHT && newAfterH >= MIN_HEIGHT) {
        before.weight = newBeforeH;
        after.weight = newAfterH;
      }
    }

    @Override
    protected void visit(Visitor visitor) {
      visitor.group(false, weight);
      for (Element child : children) {
        child.visit(visitor);
      }
      visitor.endGroup();
    }
  }

  public class Folder extends Element {
    private final List<Tab> tabs = Lists.newArrayList();
    private int titleHeight;
    private Control current;
    protected boolean maximized;

    public Folder(int weight) {
      this.weight = weight;
    }

    public void newTab(TabInfo info) {
      GC gc = new GC(TabComposite.this);
      gc.setFont(theme.selectedTabTitleFont());
      Point size = gc.textExtent(info.label);
      gc.dispose();

      addTab(new Tab(info, info.contentFactory.apply(TabComposite.this), size));
    }

    @Override
    public boolean showTab(Object id) {
      for (Tab tab : tabs) {
        if (Objects.equals(tab.info.id, id)) {
          updateCurrent(tab.control);
          return true;
        }
      }
      return false;
    }

    @Override
    public void addTabToFirstFolder(TabInfo tab) {
      newTab(tab);
    }

    @Override
    public void addTabToLargestFolder(TabInfo tab) {
      newTab(tab);
    }

    @Override
    public boolean disposeTab(Object id) {
      for (Tab tab : tabs) {
        if (Objects.equals(tab.info.id, id)) {
          removeTab(tab);
          tab.control.dispose();
          return true;
        }
      }
      return false;
    }

    @Override
    public void setBounds(Set<Control> controls, int x, int y, int w, int h) {
      redrawBar(); // redraw the old area

      setBounds(x, y, w, h);
      this.titleHeight = getMaxTitleHeight();

      int barH = BAR_MARGIN + titleHeight + BAR_MARGIN + SEP_HEIGHT + BAR_MARGIN;
      for (Tab tab : tabs) {
        tab.control.setBounds(x, y + barH, w, h - barH);
        tab.control.setVisible(tab.control == current);
        controls.remove(tab.control);
      }

      redrawBar(); // redraw the new area
    }

    private int getMaxTitleHeight() {
      int height = 0;
      for (Tab tab : tabs) {
        height = Math.max(height, tab.titleSize.y);
      }
      return height;
    }

    protected void addTab(Tab tab) {
      tabs.add(tab);
      if (current == null) {
        current = tab.control;
      }
    }

    protected void addTab(Tab tab, Tab before) {
      for (int i = 0; i < tabs.size(); i++) {
        if (tabs.get(i) == before) {
          tabs.add(i, tab);
          return;
        }
      }
    }

    protected void moveTab(Tab from, Tab to) {
      for (int i = 0; i < tabs.size(); i++) {
        if (tabs.get(i) == to) {
          tabs.remove(from);
          tabs.add(i, from);
          return;
        }
      }
    }

    protected void removeTab(Tab tab) {
      tabs.remove(tab);
      if (current == tab.control) {
        current = tabs.isEmpty() ? null : tabs.get(0).control;
        requestLayout();
      }
      redrawBar();
    }

    protected boolean updateCurrent(Control newCurrent) {
      if (current != newCurrent) {
        current = newCurrent;
        requestLayout();
        redrawBar();
        return true;
      }
      return false;
    }

    @Override
    protected void redrawBar(int x1, int y1, int x2, int y2) {
      int tabH = BAR_MARGIN + titleHeight + BAR_MARGIN;
      if ((x1 >= x && x1 < x + w && y1 >= y && y1 < y + tabH) ||
          (x2 >= x && x2 < x + w && y2 >= y && y2 < y + tabH)) {
        redrawBar();
      }
    }

    void redrawBar() {
      redraw(x, y, w, BAR_MARGIN + titleHeight + BAR_MARGIN + SEP_HEIGHT + BAR_MARGIN, false);
    }

    @Override
    protected void draw(GC gc) {
      int tabH = BAR_MARGIN + titleHeight + BAR_MARGIN;

      gc.setBackground(theme.tabBackgound());
      gc.fillRectangle(x, y, w, tabH + SEP_HEIGHT);

      gc.setForeground(theme.tabFolderLine());
      gc.drawLine(x, y + tabH + SEP_HEIGHT - 1, x + w, y + tabH + SEP_HEIGHT - 1);

      gc.setClipping(x, y, w - ICON_SIZE, h);

      int tabX = x;
      for (Tab tab : tabs) {
        int tabW = TAB_MARGIN + Math.max(TITLE_WIDTH, tab.titleSize.x) + TAB_MARGIN;

        if (dragger != null) {
          if (dragger.tab.tab == tab) {
            continue;
          } else if (dragger.contains(tabX, y, tabW, tabH)) {
            drawPlaceholder(gc, tabX, y, tabW);
            tabX += tabW;
          }
        }

        if (hovered != null && tab == hovered.tab) {
          gc.setBackground(theme.tabFolderHovered());
          gc.fillRectangle(tabX, y, tabW, tabH + 1);
        } else if (tab.control == current) {
          gc.setBackground(theme.tabFolderSelected());
          gc.fillRectangle(tabX, y, tabW, tabH + 1);
        }

        int dx = (tabW - tab.titleSize.x) / 2;
        gc.setForeground(theme.tabTitle());
        if (tab.control == current) {
          gc.setBackground(theme.tabFolderLineSelected());
          gc.fillRectangle(tabX, y + tabH, tabW, SEP_HEIGHT);
          gc.setFont(theme.selectedTabTitleFont());
          gc.drawText(tab.info.label, tabX + dx, y + BAR_MARGIN, SWT.DRAW_TRANSPARENT);
          gc.setFont(null);
        } else {
          gc.drawText(tab.info.label, tabX + dx, y + BAR_MARGIN, SWT.DRAW_TRANSPARENT);
        }

        tabX += tabW;
      }

      if (dragger != null &&
          dragger.location.x >= tabX && dragger.location.x < x + w &&
          dragger.location.y >= y && dragger.location.y < y + tabH) {
        int tabW = Math.max(TITLE_WIDTH, dragger.tab.tab.titleSize.x);
        drawPlaceholder(gc, tabX, y, tabW);
      }

      gc.setClipping((Rectangle)null);

      gc.drawImage(maximized ? theme.fullscreenExit() : theme.fullscreen(),
          x + w - ICON_SIZE, y + (tabH - ICON_SIZE) / 2);
    }

    private void drawPlaceholder(GC gc, int px, int py, int pw) {
      gc.setBackground(theme.tabFolderPlaceholderFill());
      gc.setForeground(theme.tabFolderPlaceholderStroke());
      gc.fillRectangle(px, py, pw, 2 * BAR_MARGIN + titleHeight + 1);
      gc.drawRectangle(px, py, pw, 2 * BAR_MARGIN + titleHeight + 1);
    }

    @Override
    protected Hover find(Group parent, int index, int mx, int my, boolean includeTrailing) {
      int tabH = BAR_MARGIN + titleHeight + BAR_MARGIN;
      if (mx < x || mx >= x + w || my < y || my >= y + h) {
        return Hover.NONE;
      } else if (my >= y + tabH) {
        return Hover.folder(parent, index, this);
      }

      if (mx >= x + w - ICON_SIZE) {
        return Hover.button(this);
      }

      int tabX = x;
      for (Tab tab : tabs) {
        int tabW = TAB_MARGIN + Math.max(TITLE_WIDTH, tab.titleSize.x) + TAB_MARGIN;
        if (mx >= tabX && mx < tabX + tabW) {
          return Hover.tab(this, tab);
        }
        tabX += tabW;
      }
      return includeTrailing ? Hover.tab(this, null) : Hover.NONE;
    }

    @Override
    protected MergeState merge() {
      return tabs.isEmpty() ? MergeState.REMOVE : MergeState.DO_NOTHING;
    }

    @Override
    protected void visit(Visitor visitor) {
      visitor.folder(weight);
      for (Tab tab : tabs) {
        visitor.tab(tab.info);
      }
      visitor.endFolder();
    }
  }

  public static interface Listener extends Events.Listener {
    public void onTabShown(TabInfo tab);
    public void onTabMoved(TabInfo tab);
  }

  private static class Tab {
    public final TabInfo info;
    public final Control control;
    public final Point titleSize;

    public Tab(TabInfo info, Control control, Point titleSize) {
      this.info = info;
      this.control = control;
      this.titleSize = titleSize;
    }
  }

  private static class Hover {
    public static final Hover NONE = new Hover(Type.None, null, 0, 0, null, null);

    public final Type type;
    public final Group group;
    public final int index;
    public final int cursor;
    public final Folder folder;
    public final Tab tab;

    private Hover(Type type, Group group, int index, int cursor, Folder folder, Tab tab) {
      this.type = type;
      this.group = group;
      this.index = index;
      this.cursor = cursor;
      this.folder = folder;
      this.tab = tab;
    }

    public static Hover separator(Group group, int index, int cursor) {
      return new Hover(Type.Separator, group, index, cursor, null, null);
    }

    public static Hover button(Folder folder) {
      return new Hover(Type.Button, null, 0, 0, folder, null);
    }

    public static Hover folder(Group parent, int index, Folder folder) {
      return new Hover(Type.Folder, parent, index, 0, folder, null);
    }

    public static Hover tab(Folder folder, Tab tab) {
      return new Hover(Type.Tab, null, 0, 0, folder, tab);
    }

    public boolean isSeparator() {
      return type == Type.Separator;
    }

    public boolean isFolder() {
      return type == Type.Folder;
    }

    public boolean isTab() {
      return type == Type.Tab;
    }

    public static enum Type {
      None, Separator, Button, Folder, Tab;
    }
  }

  private static class MergeState {
    public static final MergeState DO_NOTHING = new MergeState(null);
    public static final MergeState REMOVE = new MergeState(null);

    public final Element replacement;

    private MergeState(Element replacement) {
      this.replacement = replacement;
    }

    public static MergeState replace(Element replacement) {
      return new MergeState(replacement);
    }
  }

  private static class Overlay {
    private final Shell shell;
    private Region region = null;

    public Overlay(Shell parent, Rectangle bounds, Color bg) {
      this.shell = new Shell(parent, SWT.NO_TRIM | SWT.MODELESS | SWT.NO_FOCUS | SWT.ON_TOP);

      shell.setBounds(bounds);
      shell.setEnabled(false);
      shell.setAlpha(128);
      shell.setBackground(bg);
    }

    public void hide() {
      shell.setVisible(false);
    }

    public void show(Rectangle highlight) {
      if (region != null) {
        region.dispose();
      }

      region = new Region();
      region.add(highlight);
      shell.setRegion(region);
      shell.setEnabled(false);
      shell.setVisible(true);
    }

    public void close() {
      shell.dispose();
      if (region != null) {
        region.dispose();
      }
    }
  }

  private static class Dragger {
    public final Overlay overlay;
    public final Shell shell;
    public final Hover tab;
    public final Point location = new Point(-1, -1);

    public Dragger(Theme theme, Shell parent, Rectangle bounds, Color bg, Hover tab) {
      this.overlay = new Overlay(parent, bounds, bg);
      this.shell = new Shell(parent, SWT.NO_TRIM | SWT.MODELESS | SWT.NO_FOCUS | SWT.ON_TOP);
      this.tab = tab;

      shell.setLayout(new FillLayout());
      shell.setSize(tab.tab.titleSize.x + 2 * TAB_MARGIN, tab.tab.titleSize.y + 2 * BAR_MARGIN);
      shell.setEnabled(false);

      Canvas canvas = new Canvas(shell, SWT.NONE);
      canvas.addListener(SWT.Paint, e -> {
        e.gc.setFont(theme.selectedTabTitleFont());
        e.gc.drawText(tab.tab.info.label, TAB_MARGIN, BAR_MARGIN, SWT.DRAW_TRANSPARENT);
      });

      shell.setVisible(true);
    }

    public boolean contains(int x, int y, int w, int h) {
      return location.x >= 0 && location.y >= 0 &&
          x <= location.x && x + w > location.x &&
          y <= location.y && y + h > location.y;
    }

    public void close() {
      overlay.close();
      shell.dispose();
    }
  }
}

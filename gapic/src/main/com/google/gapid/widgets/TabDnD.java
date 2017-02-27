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
package com.google.gapid.widgets;

import com.google.gapid.util.Events;
import com.google.gapid.util.Events.ListenerCollection;

import org.eclipse.jface.util.LocalSelectionTransfer;
import org.eclipse.jface.viewers.ISelection;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.CTabFolder;
import org.eclipse.swt.custom.CTabItem;
import org.eclipse.swt.dnd.DND;
import org.eclipse.swt.dnd.DragSource;
import org.eclipse.swt.dnd.DragSourceAdapter;
import org.eclipse.swt.dnd.DragSourceEvent;
import org.eclipse.swt.dnd.DropTarget;
import org.eclipse.swt.dnd.DropTargetAdapter;
import org.eclipse.swt.dnd.DropTargetEvent;
import org.eclipse.swt.dnd.Transfer;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.widgets.Control;

/**
 * Utilities for dragging tabs, e.g. in a {@link TabArea}.
 */
public class TabDnD {
  protected static final ListenerCollection<Listener> listeners = Events.listeners(Listener.class);

  private TabDnD() {
  }

  public static CTabFolder withMovableTabs(CTabFolder folder) {
    TabDragSource.create(folder);
    TabDropTarget.create(folder);
    return folder;
  }

  public static void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public static void removeListener(Listener listener) {
    listeners.removeListener(listener);
  }

  public static CTabItem moveTab(CTabItem item, CTabFolder newFolder, int newIndex) {
    CTabFolder folder = item.getParent();
    CTabItem newItem = new CTabItem(
        newFolder, SWT.NONE, (newIndex < 0) ? newFolder.getItemCount() : newIndex);
    newItem.setText(item.getText());
    Control contents = item.getControl();
    if (item.getParent() != newFolder) {
      contents.setParent(newFolder);
    }
    newItem.setControl(contents);
    listeners.fire().itemCopied(item, newItem);
    item.dispose();
    listeners.fire().onTabMoved(folder, item, newFolder, newItem);
    return newItem;
  }

  /**
   * A {@link CTabFolder} containing tabs that can be dragged away.
   */
  private static class TabDragSource extends DragSourceAdapter {
    private final CTabFolder folder;

    public TabDragSource(CTabFolder folder) {
      this.folder = folder;
    }

    public static void create(CTabFolder folder) {
      DragSource ds = new DragSource(folder, DND.DROP_MOVE);
      ds.setTransfer(new Transfer[] { LocalSelectionTransfer.getTransfer() });
      ds.addDragListener(new TabDragSource(folder));
    }

    @Override
    public void dragStart(DragSourceEvent event) {
      Point p = new Point(event.x, event.y);
      CTabItem item = folder.getItem(p);
      if (item != null && p.y < folder.getTabHeight()) {
        LocalSelectionTransfer.getTransfer().setSelection(new TabSelection(item));
      } else {
        event.doit = false;
      }
    }
  }

  /**
   * A {@link CTabFolder} where tabs can be dragged to.
   */
  private static class TabDropTarget extends DropTargetAdapter {
    private final CTabFolder folder;

    public TabDropTarget(CTabFolder folder) {
      this.folder = folder;
    }

    public static void create(CTabFolder folder) {
      DropTarget dt = new DropTarget(folder, DND.DROP_MOVE);
      dt.setTransfer(new Transfer[] { LocalSelectionTransfer.getTransfer() });
      dt.addDropListener(new TabDropTarget(folder));
    }

    @Override
    public void dropAccept(DropTargetEvent event) {
      TabSelection selection = (TabSelection)LocalSelectionTransfer.getTransfer().getSelection();
      if (!moveItem(folder, selection.item, folder.getItem(folder.toControl(event.x, event.y)))) {
        event.detail = DND.DROP_NONE;
      }
    }

    private static boolean moveItem(CTabFolder folder, CTabItem source, CTabItem target) {
      if (source == target) {
        return false; // Moving to same place.
      }
      boolean sameFolder = (source.getParent() == folder);
      if (sameFolder && target == null && source == folder.getItem(folder.getItemCount() - 1)) {
        return false; // Moving to same place.
      }

      int index = (target == null) ? folder.getItemCount() : folder.indexOf(target);
      if (sameFolder && index > folder.indexOf(source) && index < folder.getItemCount()) {
        index++;
      }

      folder.setSelection(moveTab(source, folder, index));
      return true;
    }
  }

  /**
   * A {@link ISelection} containing a tab that is currently being dragged.
   */
  private static class TabSelection implements ISelection {
    public final CTabItem item;

    public TabSelection(CTabItem item) {
      this.item = item;
    }

    @Override
    public boolean isEmpty() {
      return false;
    }
  }

  @SuppressWarnings("unused")
  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that the given source item had to be copied (e.g. because it has been moved
     * to a new folder - reparenting).
     */
    public default void itemCopied(CTabItem source, CTabItem target) { /* empty */ }

    /**
     * Event indicating that the given item was moved from one folder to another.
     */
    public default void onTabMoved(CTabFolder sourceFolder, CTabItem oldItem,
        CTabFolder destFolder, CTabItem newItem) { /* empty */ }
  }
}

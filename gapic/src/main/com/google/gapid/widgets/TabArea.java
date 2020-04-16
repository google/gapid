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

import com.google.common.collect.Lists;
import com.google.gapid.models.Analytics;
import com.google.gapid.proto.service.Service.ClientAction;

import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.CTabFolder;
import org.eclipse.swt.custom.CTabItem;
import org.eclipse.swt.widgets.Composite;

import java.util.Arrays;
import java.util.LinkedList;
import java.util.List;

/**
 * Manages {@link CTabItem tabs} in three {@link CTabFolder tab areas}. Allows the user to drag
 * tabs between areas and minimzing/maximzing each area. The three areas are horizontally laid out
 * with movable dividers.
 */
public class TabArea extends TabComposite {
  public TabArea(Composite parent, Analytics analytics, Theme theme, Persistance persistance) {
    super(parent, theme, false);

    restore(getRoot(), persistance.restore());

    TabComposite.Listener listener = new TabComposite.Listener() {
      @Override
      public void onTabShown(TabInfo tab) {
        analytics.postInteraction(tab.view, ClientAction.Show);
      }

      @Override
      public void onTabMoved(TabInfo tab) {
        analytics.postInteraction(tab.view, ClientAction.Move);
      }
    };
    addListener(listener);

    addListener(SWT.Dispose, e -> {
      removeListener(listener);
      InfoCollector ic = new InfoCollector();
      visit(ic);
      persistance.store(ic.getFolderInfos());
    });
  }

  private static void restore(Group group, FolderInfo[] folderInfos) {
    for (FolderInfo folderInfo : folderInfos) {
      if (folderInfo.tabs != null) {
        Folder folder = group.newFolder(folderInfo.weight);
        for (TabInfo tabInfo : folderInfo.tabs) {
          folder.newTab(tabInfo);
        }
      }
      if (folderInfo.children != null) {
        restore(group.newGroup(folderInfo.weight), folderInfo.children);
      }
    }
  }

  /**
   * Responsible for remembering the order and location of tabs.
   */
  public static interface Persistance {
    /**
     * @return the previously stored tab information. Has to be length 3 (left, center, right).
     */
    public FolderInfo[] restore();

    /**
     * @param folders the tab information to store. Always length 3 (left, center, right).
     */
    public void store(FolderInfo[] folders);
  }

  /**
   * Size and containing tabs information of a folder.
   */
  public static class FolderInfo {
    public final FolderInfo[] children;
    public final TabInfo[] tabs;
    public final int weight;

    public FolderInfo(FolderInfo[] children, int weight) {
      this.children = children;
      this.tabs = null;
      this.weight = weight;
    }

    public FolderInfo(TabInfo[] tabs, int weight) {
      this.children = null;
      this.tabs = tabs;
      this.weight = weight;
    }

    public FolderInfo(List<TabInfo> tabs, int weight) {
      this(tabs.toArray(new TabInfo[tabs.size()]), weight);
    }

    public FolderInfo addToFirst(TabInfo[] newTabs) {
      if (tabs != null) {
        // There's only one folder, add them here.
        TabInfo[] t = Arrays.copyOf(tabs, tabs.length + newTabs.length);
        System.arraycopy(newTabs, 0, t, tabs.length, newTabs.length);
        return new FolderInfo(t, weight);
      } else if (children[0].tabs != null) {
        // The first child is a folder, add them there.
        FolderInfo[] t = Arrays.copyOf(children, children.length);
        t[0] = children[0].addToFirst(newTabs);
        return new FolderInfo(t, weight);
      } else {
        // Create a new folder and make it our first child.
        FolderInfo[] t = new FolderInfo[children.length + 1];
        t[0] = new FolderInfo(newTabs, weight);
        System.arraycopy(children, 0, t, 1, children.length);
        return new FolderInfo(t, weight * 2);
      }
    }

    public FolderInfo addToLargest(TabInfo[] newTabs) {
      if (tabs != null) {
        TabInfo[] t = Arrays.copyOf(tabs, tabs.length + newTabs.length);
        System.arraycopy(newTabs, 0, t, tabs.length, newTabs.length);
        return new FolderInfo(t, weight);
      } else {
        int max = 0;
        for (int i = 1; i < children.length; i++) {
          if (children[i].weight > children[max].weight) {
            max = i;
          }
        }
        FolderInfo[] t = Arrays.copyOf(children, children.length);
        t[max] = t[max].addToLargest(newTabs);
        return new FolderInfo(t, weight);
      }
    }
  }

  private static class InfoCollector implements Visitor {
    private final LinkedList<GroupBuilder> groupStack = Lists.newLinkedList();
    private FolderBuilder folder;
    private FolderInfo result;

    public InfoCollector() {
    }

    public FolderInfo[] getFolderInfos() {
      return result.children;
    }

    @Override
    public void group(boolean horizontal, int weight) {
      groupStack.add(new GroupBuilder(weight));
    }

    @Override
    public void endGroup() {
      GroupBuilder group = groupStack.removeLast();
      if (groupStack.isEmpty()) {
        result = group.build();
      } else {
        groupStack.getLast().addChild(group.build());
      }
    }

    @Override
    public void folder(int weight) {
      folder = new FolderBuilder(weight);
    }

    @Override
    public void tab(TabInfo tab) {
      folder.addTab(tab);
    }

    @Override
    public void endFolder() {
      groupStack.getLast().addChild(folder.build());
    }

    class GroupBuilder {
      private final List<FolderInfo> children = Lists.newArrayList();
      private final int weight;

      public GroupBuilder(int weight) {
        this.weight = weight;
      }

      public void addChild(FolderInfo child) {
        children.add(child);
      }

      public FolderInfo build() {
        return new FolderInfo(children.toArray(new FolderInfo[children.size()]), weight);
      }
    }

    class FolderBuilder {
      private final List<TabInfo> tabs = Lists.newArrayList();
      private final int weight;

      public FolderBuilder(int weight) {
        this.weight = weight;
      }

      public void addTab(TabInfo tab) {
        tabs.add(tab);
      }

      public FolderInfo build() {
        return new FolderInfo(tabs.toArray(new TabInfo[tabs.size()]), weight);
      }
    }
  }
}

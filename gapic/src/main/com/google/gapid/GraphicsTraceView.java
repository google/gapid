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
package com.google.gapid;

import static java.util.stream.Collectors.toList;

import com.google.common.base.Splitter;
import com.google.common.collect.Lists;
import com.google.common.collect.Sets;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.SettingsProto;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.views.CommandTree;
import com.google.gapid.views.ContextSelector;
import com.google.gapid.views.FramebufferView;
import com.google.gapid.views.GeometryView;
import com.google.gapid.views.LogView;
import com.google.gapid.views.MemoryView;
import com.google.gapid.views.PipelineView;
import com.google.gapid.views.ReportView;
import com.google.gapid.views.ShaderView;
import com.google.gapid.views.StateView;
import com.google.gapid.views.Tab;
import com.google.gapid.views.TextureView;
import com.google.gapid.views.ThumbnailScrubber;
import com.google.gapid.widgets.TabArea;
import com.google.gapid.widgets.TabArea.FolderInfo;
import com.google.gapid.widgets.TabArea.Persistance;
import com.google.gapid.widgets.TabComposite.TabInfo;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.action.Action;
import org.eclipse.jface.action.IAction;
import org.eclipse.jface.action.MenuManager;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;

import java.util.Arrays;
import java.util.Iterator;
import java.util.List;
import java.util.Set;
import java.util.function.Consumer;
import java.util.function.Function;

/**
 * Main view shown when a graphics trace is loaded.
 */
public class GraphicsTraceView extends Composite implements MainWindow.MainView {
  private final Models models;
  private final Widgets widgets;
  protected final Set<MainTab.Type> hiddenTabs;

  protected TabArea tabs;

  public GraphicsTraceView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.widgets = widgets;
    this.hiddenTabs = getHiddenTabs(models.settings);

    setLayout(new GridLayout(1, false));

    new ContextSelector(this, models)
        .setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));

    tabs = new TabArea(this, models.analytics, widgets.theme, new Persistance() {
      @Override
      public void store(TabArea.FolderInfo[] folders) {
        MainTab.store(models, folders);
      }

      @Override
      public TabArea.FolderInfo[] restore() {
        return MainTab.getFolders(models, widgets, hiddenTabs);
      }
    });

    tabs.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    models.follower.addListener(new Follower.Listener() {
      @Override
      public void onMemoryFollowed(Path.Memory path) {
        tabs.showTab(MainTab.Type.Memory);
      }

      @Override
      public void onStateFollowed(Path.Any path) {
        tabs.showTab(MainTab.Type.ApiState);
      }
    });
  }

  private static Set<MainTab.Type> getHiddenTabs(Settings settings) {
    Set<MainTab.Type> hiddenTabs = Sets.newHashSet();
    for (String hidden : settings.tabs().getHiddenList()) {
      try {
        hiddenTabs.add(MainTab.Type.valueOf(hidden));
      } catch (IllegalArgumentException e) {
        // Ignore invalid tab names in the settings file.
      }
    }
    return hiddenTabs;
  }

  @Override
  public void updateViewMenu(MenuManager manager) {
    manager.removeAll();
    manager.add(createViewTabsMenu());
  }

  private MenuManager createViewTabsMenu() {
    MenuManager manager = new MenuManager("&Tabs");
    for (MainTab.Type type : MainTab.Type.values()) {
      Action action = type.createAction(shown -> {
        models.analytics.postInteraction(
            type.view, shown ? ClientAction.Enable : ClientAction.Disable);
        if (shown) {
          TabInfo tabInfo = new MainTab(type, parent -> {
            Tab tab = type.factory.create(parent, models, widgets);
            tab.reinitialize();
            return tab.getControl();
          });
          if (type.top) {
            tabs.addTabToFirstFolder(tabInfo);
          } else {
            tabs.addTabToLargestFolder(tabInfo);
          }
          tabs.showTab(type);
          hiddenTabs.remove(type);
        } else {
          tabs.disposeTab(type);
          hiddenTabs.add(type);
        }
        models.settings.writeTabs()
            .clearHidden()
            .addAllHidden(hiddenTabs.stream().map(MainTab.Type::name).collect(toList()));
      });
      action.setChecked(!hiddenTabs.contains(type));
      manager.add(action);
    }
    return manager;
  }

  /**
   * Information about the tabs to be shown in the main window.
   */
  private static class MainTab extends TabInfo {
    public MainTab(Type type, Function<Composite, Control> contentFactory) {
      super(type, type.view, type.label, contentFactory);
    }

    /**
     * Deserializes the {@link FolderInfo FolderInfos} by parsing the settings strings.
     * The "weights" and "tabs" string are simple CSV lists and values are popped off the lists
     * according to the "structs" string. The "struct" string is a semi-colon separated list of
     * a recursive structure. Each element in the list starts either with a 'g' for group, or
     * 'f' for folder. The remainder of the string is an integer representing the number of
     * recursive children in case of groups, or the number of tabs in case of folders.
     */
    public static FolderInfo[] getFolders(Models models, Widgets widgets, Set<Type> hidden) {
      SettingsProto.TabsOrBuilder sTabs = models.settings.tabs();
      Set<Type> allTabs = Sets.newLinkedHashSet(Arrays.asList(Type.values()));
      allTabs.removeAll(hidden);
      Iterator<String> structs = Splitter.on(';')
          .trimResults()
          .omitEmptyStrings()
          .split(sTabs.getStructure())
          .iterator();
      Iterator<Integer> weights = sTabs.getWeightsList().iterator();
      Iterator<String> tabs = sTabs.getTabsList().iterator();

      FolderInfo root = parse(models, widgets, structs, weights, tabs, allTabs);
      if (structs.hasNext()) {
        root = null;
      }
      if (root == null || root.children == null) {
        return getDefaultFolderInfo(models, widgets, hidden);
      }

      if (!allTabs.isEmpty()) {
        TabInfo[] tabsToAdd = new TabInfo[allTabs.size()];
        int i = 0;
        for (Type tab : allTabs) {
          tabsToAdd[i++] = new MainTab(tab,
              parent -> tab.factory.create(parent, models, widgets).getControl());
        }
        root = root.addToLargest(tabsToAdd);
      }
      return root.children;
    }

    private static FolderInfo parse(Models models, Widgets widgets, Iterator<String> structs,
        Iterator<Integer> weights, Iterator<String> tabs, Set<Type> left) {
      if (!structs.hasNext() || !weights.hasNext()) {
        return null;
      }
      String struct = structs.next(); // struct is non-empty (see splitter above)
      int weight = weights.next();

      int count = 0;
      try {
        count = Integer.parseInt(struct.substring(1));
      } catch (NumberFormatException e) {
        return null;
      }
      if (count <= 0) {
        return null;
      }

      switch (struct.charAt(0)) {
        case 'g':
          FolderInfo[] folders = new FolderInfo[count];
          for (int i = 0; i < folders.length; i++) {
            folders[i] = parse(models, widgets, structs, weights, tabs, left);
            if (folders[i] == null) {
              return null;
            }
          }
          return new FolderInfo(folders, weight);
        case 'f':
          List<TabInfo> children = getTabs(tabs, count, left, models, widgets);
          return (children.isEmpty()) ? null :
              new FolderInfo(children.toArray(new TabInfo[children.size()]), weight);
        default:
          return null;
      }
    }

    private static FolderInfo[] getDefaultFolderInfo(
        Models models, Widgets widgets, Set<Type> hidden) {
      Set<Type> allTabs = Sets.newLinkedHashSet(Arrays.asList(Type.values()));
      allTabs.removeAll(hidden);
      boolean hasFilmStrip = allTabs.remove(Type.Filmstrip);
      List<FolderInfo> folders = Lists.newArrayList();
      if (allTabs.contains(Type.ApiCalls)) {
        folders.add(new FolderInfo(new TabInfo[] {
            new MainTab(Type.ApiCalls,
                parent -> Type.ApiCalls.factory.create(parent, models, widgets).getControl())
        }, 1));
        allTabs.remove(Type.ApiCalls);
      }
      List<TabInfo> center = Lists.newArrayList();
      for (Iterator<Type> it = allTabs.iterator(); it.hasNext(); ) {
        Type type = it.next();
        if (type == Type.Memory || type == Type.ApiState) {
          continue;
        }
        center.add(new MainTab(
            type, parent -> type.factory.create(parent, models, widgets).getControl()));
        it.remove();
      }
      if (!center.isEmpty()) {
        folders.add(new FolderInfo(center.toArray(new TabInfo[center.size()]), 3));
      }
      if (!allTabs.isEmpty()) {
        TabInfo[] right = new TabInfo[allTabs.size()];
        if (allTabs.contains(Type.ApiState)) {
          right[0] = new MainTab(Type.ApiState,
              parent -> Type.ApiState.factory.create(parent, models, widgets).getControl());
        }
        if (allTabs.contains(Type.Memory)) {
          right[right.length - 1] = new MainTab(Type.Memory,
              parent -> Type.Memory.factory.create(parent, models, widgets).getControl());
        }
        folders.add(new FolderInfo(right, 1));
      }

      FolderInfo[] result = folders.toArray(new FolderInfo[folders.size()]);
      if (hasFilmStrip) {
        result = new FolderInfo[] {
            new FolderInfo(new TabInfo[] {
                new MainTab(Type.Filmstrip,
                    parent -> Type.Filmstrip.factory.create(parent, models, widgets).getControl()),
            }, 1),
            new FolderInfo(result, 4),
        };
      } else {
        result = new FolderInfo[] { new FolderInfo(result, 1) };
      }
      return result;
    }

    /**
     * Serializes the {@link FolderInfo FolderInfos} into the setting strings.
     * {@see #getFolders(Models, Widgets, Set)}
     */
    public static void store(Models models, FolderInfo[] folders) {
      List<Integer> weights = Lists.newArrayList(-1);
      StringBuilder structure = new StringBuilder().append('g').append(folders.length).append(';');
      List<String> tabs = Lists.newArrayList();
      for (FolderInfo folder : folders) {
        flatten(folder, weights, structure, tabs);
      }
      models.settings.writeTabs()
          .setStructure(structure.toString())
          .clearTabs().addAllTabs(tabs)
          .clearWeights().addAllWeights(weights);
    }

    private static void flatten(
        FolderInfo folder, List<Integer> weights, StringBuilder structure, List<String> tabs) {
      weights.add(folder.weight);
      if (folder.children != null) {
        structure.append('g').append(folder.children.length).append(';');
        for (FolderInfo child : folder.children) {
          flatten(child, weights, structure, tabs);
        }
      }
      if (folder.tabs != null) {
        structure.append('f').append(folder.tabs.length).append(';');
        for (TabInfo tab : folder.tabs) {
          tabs.add(((Type)tab.id).name());
        }
      }
    }

    private static List<TabInfo> getTabs(
       Iterator<String> names, int count, Set<Type> left, Models models, Widgets widgets) {

      List<TabInfo> result = Lists.newArrayList();
      for (int i = 0; i < count && names.hasNext(); i++) {
        try {
          Type type = Type.valueOf(names.next());
          if (left.remove(type)) {
            result.add(new MainTab(type,
                parent -> type.factory.create(parent, models, widgets).getControl()));
          }
        } catch (IllegalArgumentException e) {
          // Ignore incorrect names in the properties.
        }
      }
      return result;
    }

    /**
     * Information about the available tabs.
     */
    public static enum Type {
      Filmstrip(View.FilmStrip, "Filmstrip", true, ThumbnailScrubber::new),

      ApiCalls(View.Commands, "Commands", false, CommandTree::new),

      Framebuffer(View.Framebuffer, "Framebuffer", false, FramebufferView::new),
      Pipeline(View.Pipeline, "Pipeline", false, PipelineView::new),
      Textures(View.Textures, "Textures", false, TextureView::new),
      Geometry(View.Geometry, "Geometry", false, GeometryView::new),
      Shaders(View.Shaders, "Shaders", false, ShaderView::new),
      Report(View.Report, "Report", false, ReportView::new),
      Log(View.Log, "Log", false, (p, m, w) -> new LogView(p, w)),

      ApiState(View.State, "State", false, StateView::new),
      Memory(View.Memory, "Memory", false, MemoryView::new);

      public final View view;
      public final String label;
      public final boolean top;
      public final TabFactory factory;

      private Type(View view, String label, boolean top, TabFactory factory) {
        this.view = view;
        this.label = label;
        this.top = top;
        this.factory = factory;
      }

      public Action createAction(Consumer<Boolean> listener) {
        Action action = new Action(null, IAction.AS_CHECK_BOX) {
          @Override
          public void run() {
            listener.accept(isChecked());
          }
        };
        action.setChecked(true);
        action.setText(label);
        return action;
      }
    }

    /**
     * Factory to create the UI components of a tab.
     */
    public static interface TabFactory {
      public Tab create(Composite parent, Models models, Widgets widgets);
    }
  }
}

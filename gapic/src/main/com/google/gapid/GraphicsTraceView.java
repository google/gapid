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
import com.google.common.collect.ListMultimap;
import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.MultimapBuilder;
import com.google.common.collect.Sets;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Models;
import com.google.gapid.models.Resources;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.SettingsProto;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.util.Experimental;
import com.google.gapid.views.CommandTree;
import com.google.gapid.views.DeviceDialog;
import com.google.gapid.views.FramebufferView;
import com.google.gapid.views.GeometryView;
import com.google.gapid.views.LogView;
import com.google.gapid.views.MemoryView;
import com.google.gapid.views.PerformanceView;
import com.google.gapid.views.PipelineView;
import com.google.gapid.views.ProfileView;
import com.google.gapid.views.ReportView;
import com.google.gapid.views.ShaderList;
import com.google.gapid.views.ShaderView;
import com.google.gapid.views.StateView;
import com.google.gapid.views.Tab;
import com.google.gapid.views.TextureList;
import com.google.gapid.views.TextureView;
import com.google.gapid.widgets.TabArea;
import com.google.gapid.widgets.TabArea.FolderInfo;
import com.google.gapid.widgets.TabArea.Persistance;
import com.google.gapid.widgets.TabComposite;
import com.google.gapid.widgets.TabComposite.TabContent;
import com.google.gapid.widgets.TabComposite.TabInfo;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.action.Action;
import org.eclipse.jface.action.IAction;
import org.eclipse.jface.action.MenuManager;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;

import java.util.Arrays;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.function.Consumer;
import java.util.function.Function;

/**
 * Main view shown when a graphics trace is loaded.
 */
public class GraphicsTraceView extends Composite
    implements MainWindow.MainView, Resources.Listener, Follower.Listener {
  private final Models models;
  private final Widgets widgets;
  private final Map<MainTab.Type, Action> typeActions;
  protected final Set<MainTab.Type> hiddenTabs;

  protected final TabArea tabs;

  public GraphicsTraceView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.widgets = widgets;
    this.typeActions = Maps.newHashMap();
    this.hiddenTabs = getHiddenTabs(models.settings);

    new DeviceDialog(this, models, widgets);

    setLayout(new GridLayout(1, false));

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

    TabComposite.Listener listener = new TabComposite.Listener() {
      @Override
      public void onTabCreated(TabInfo tab) {
        if (tab.id instanceof MainTab.Type) {
          syncTabMenuItem((MainTab.Type)tab.id, true);
        }
      }

      @Override
      public void onTabClosed(TabInfo tab) {
        if (tab.id instanceof MainTab.Type) {
          syncTabMenuItem((MainTab.Type)tab.id, false);
        }
      }

      @Override
      public void onTabPinned(TabInfo tab) {
        if (tab.id instanceof MainTab.Type) {
          syncTabMenuItem((MainTab.Type)tab.id, false);
        }
      }
    };
    tabs.addListener(listener);

    models.resources.addListener(this);
    models.follower.addListener(this);
    addListener(SWT.Dispose, e -> {
      tabs.removeListener(listener);
      models.resources.removeListener(this);
      models.follower.removeListener(this);
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

  @Override
  public void onShaderSelected(Service.Resource shader) {
    if (shader != null) {
      showTab(MainTab.Type.ShaderView);
    }
  }

  @Override
  public void onTextureSelected(Service.Resource texture) {
    if (texture != null) {
      showTab(MainTab.Type.TextureView);
    }
  }

  @Override
  public void onMemoryFollowed(Path.Memory path) {
    tabs.showTab(MainTab.Type.Memory);
  }

  @Override
  public void onStateFollowed(Path.Any path) {
    tabs.showTab(MainTab.Type.ApiState);
  }

  @Override
  public void onFramebufferAttachmentFollowed(Path.FramebufferAttachment path) {
    tabs.showTab(MainTab.Type.Framebuffer);
  }

  private void showTab(MainTab.Type type) {
    if (!tabs.showTab(type)) {
      TabInfo tabInfo = new MainTab(type, parent -> {
        Tab tab = type.factory.create(parent, models, widgets);
        tab.reinitialize();
        return tab;
      });
      if (type.position == MainTab.DefaultPosition.Top) {
        tabs.addTabToFirstFolder(tabInfo);
      } else {
        tabs.addTabToLargestFolder(tabInfo);
      }
      tabs.showTab(type);
    }
  }

  private MenuManager createViewTabsMenu() {
    MenuManager manager = new MenuManager("&Tabs");
    for (MainTab.Type type : MainTab.Type.values()) {
      // TODO(b/188416598): Improve report quality and enable the report tab again.
      if (type == MainTab.Type.Report && !Experimental.enableUnstableFeatures(models.settings)) {
        continue;
      }
      Action action = type.createAction(shown -> {
        if (shown) {
          showTab(type);
        } else {
          tabs.disposeTab(type);
        }
      });
      action.setChecked(!hiddenTabs.contains(type));
      manager.add(action);
      typeActions.put(type, action);
    }
    return manager;
  }

  protected void syncTabMenuItem(MainTab.Type type, boolean shown) {
    Action action = typeActions.get(type);
    if (action != null) {
      action.setChecked(shown);
    }
    if (hiddenTabs.contains(type) == shown) {
      if (shown) {
        hiddenTabs.remove(type);
      } else {
        hiddenTabs.add(type);
      }
      models.settings.writeTabs()
          .clearHidden()
          .addAllHidden(hiddenTabs.stream().map(MainTab.Type::name).collect(toList()));
    }
  }

  /**
   * Information about the tabs to be shown in the main window.
   */
  private static class MainTab extends TabInfo {
    public MainTab(Type type, Function<Composite, TabContent> contentFactory) {
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
      // TODO(b/188416598): Improve report quality and enable the report tab again.
      if (!Experimental.enableUnstableFeatures(models.settings)) {
        allTabs.remove(MainTab.Type.Report);
      }
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
        List<TabInfo> toAddToLargest = Lists.newArrayList();
        List<TabInfo> toAddToTop = Lists.newArrayList();
        for (Type tab : allTabs) {
          (tab.position == DefaultPosition.Top ? toAddToTop : toAddToLargest).add(
              new MainTab(tab, parent -> tab.factory.create(parent, models, widgets)));
        }
        if (!toAddToLargest.isEmpty()) {
          root = root.addToLargest(toAddToLargest.toArray(new TabInfo[toAddToLargest.size()]));
        }
        if (!toAddToTop.isEmpty()) {
          root = root.addToFirst(toAddToTop.toArray(new TabInfo[toAddToTop.size()]));
        }
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
      ListMultimap<DefaultPosition, TabInfo> toAdd =
          MultimapBuilder.enumKeys(DefaultPosition.class).arrayListValues().build();
      for (Type type : Type.values()) {
        if (!hidden.contains(type)) {
          toAdd.put(type.position, new MainTab(
              type, parent -> type.factory.create(parent, models, widgets)));
        }
      }

      List<FolderInfo> bottom = Lists.newArrayList();
      if (toAdd.containsKey(DefaultPosition.Left)) {
        bottom.add(new FolderInfo(toAdd.get(DefaultPosition.Left), 1));
      }
      if (toAdd.containsKey(DefaultPosition.Center)) {
        bottom.add(new FolderInfo(toAdd.get(DefaultPosition.Center), 3));
      }
      if (toAdd.containsKey(DefaultPosition.Right)) {
        bottom.add(new FolderInfo(toAdd.get(DefaultPosition.Right), 1));
      }
      FolderInfo[] result = bottom.toArray(new FolderInfo[bottom.size()]);

      if (toAdd.containsKey(DefaultPosition.Top)) {
        result = new FolderInfo[] {
            new FolderInfo(toAdd.get(DefaultPosition.Top), 1),
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
            result.add(new MainTab(type, parent -> type.factory.create(parent, models, widgets)));
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
      Profile(View.Profile, "Profile", DefaultPosition.Top, ProfileView::new),

      ApiCalls(View.Commands, "Commands", DefaultPosition.Left, CommandTree::new),

      Framebuffer(View.Framebuffer, "Framebuffer", DefaultPosition.Center, FramebufferView::new),
      Pipeline(View.Pipeline, "Pipeline", DefaultPosition.Center, PipelineView::new),
      Textures(View.Textures, "Textures", DefaultPosition.Center, TextureList::new),
      Geometry(View.Geometry, "Geometry", DefaultPosition.Center, GeometryView::new),
      Shaders(View.Shaders, "Shaders", DefaultPosition.Center, ShaderList::new),
      Performance(View.Performance, "Performance(Experimental)", DefaultPosition.Center, PerformanceView::new),
      Report(View.Report, "Report", DefaultPosition.Center, ReportView::new),
      Log(View.Log, "Log", DefaultPosition.Center, (p, m, w) -> new LogView(p, w)),

      TextureView(View.TextureView, "Texture", DefaultPosition.Right, TextureView::new),
      ShaderView(View.ShaderView, "Shader", DefaultPosition.Right, ShaderView::new),
      ApiState(View.State, "State", DefaultPosition.Right, StateView::new),
      Memory(View.Memory, "Memory", DefaultPosition.Right, MemoryView::new);

      public final View view;
      public final String label;
      public final DefaultPosition position;
      public final TabFactory factory;

      private Type(View view, String label,DefaultPosition position, TabFactory factory) {
        this.view = view;
        this.label = label;
        this.position = position;
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

    public static enum DefaultPosition {
      Top, Left, Center, Right;
    }

    /**
     * Factory to create the UI components of a tab.
     */
    public static interface TabFactory {
      public Tab create(Composite parent, Models models, Widgets widgets);
    }
  }
}

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

import com.google.common.collect.Lists;
import com.google.common.collect.Sets;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.views.CommandTree;
import com.google.gapid.views.ContextSelector;
import com.google.gapid.views.FramebufferView;
import com.google.gapid.views.GeometryView;
import com.google.gapid.views.LogView;
import com.google.gapid.views.MemoryView;
import com.google.gapid.views.ReportView;
import com.google.gapid.views.ShaderView;
import com.google.gapid.views.StateView;
import com.google.gapid.views.Tab;
import com.google.gapid.views.TextureView;
import com.google.gapid.views.ThumbnailScrubber;
import com.google.gapid.widgets.FixedTopSplitter;
import com.google.gapid.widgets.TabArea;
import com.google.gapid.widgets.TabArea.FolderInfo;
import com.google.gapid.widgets.TabArea.Persistance;
import com.google.gapid.widgets.TabArea.TabInfo;
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
import java.util.List;
import java.util.Set;
import java.util.function.Consumer;
import java.util.function.Function;

/**
 * Main view shown when a graphics trace is loaded.
 */
public class GraphicsTraceView extends Composite {
  private final Models models;
  private final Widgets widgets;
  protected final Set<MainTab.Type> hiddenTabs;

  private final FixedTopSplitter splitter;
  protected TabArea tabs;

  public GraphicsTraceView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.widgets = widgets;
    this.hiddenTabs = getHiddenTabs(models.settings);

    setLayout(new GridLayout(1, false));

    new ContextSelector(this, models)
        .setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));
    splitter = new FixedTopSplitter(this, models.settings.splitterTopHeight) {
      @Override
      protected Control createTopControl() {
        return new ThumbnailScrubber(this, models, widgets);
      }

      @Override
      protected Control createBottomControl() {
        tabs = new TabArea(this, new Persistance() {
          @Override
          public void store(FolderInfo[] folders) {
            MainTab.store(models, folders);
          }

          @Override
          public FolderInfo[] restore() {
            return MainTab.getFolders(models, widgets, hiddenTabs);
          }
        }, models.analytics);
        tabs.setLeftVisible(!models.settings.hideLeft);
        tabs.setRightVisible(!models.settings.hideRight);
        return tabs;
      }
    };
    splitter.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    splitter.setTopVisible(!models.settings.hideScrubber);

    splitter.addListener(SWT.Dispose, e -> {
      models.settings.splitterTopHeight = splitter.getTopHeight();
    });

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
    for (String hidden : settings.hiddenTabs) {
      try {
        hiddenTabs.add(MainTab.Type.valueOf(hidden));
      } catch (IllegalArgumentException e) {
        // Ignore invalid tab names in the settings file.
      }
    }
    return hiddenTabs;
  }

  public void updateViewMenu(MenuManager manager) {
    manager.removeAll();

    Action viewScrubber = MainWindow.MenuItems.ViewThumbnails.createCheckbox(show -> {
      if (splitter != null) {
        models.analytics.postInteraction(
            View.FilmStrip, show ? ClientAction.Enable : ClientAction.Disable);
        splitter.setTopVisible(show);
        models.settings.hideScrubber = !show;
      }
    });
    viewScrubber.setChecked(!models.settings.hideScrubber);
    Action viewLeft = MainWindow.MenuItems.ViewLeft.createCheckbox(show -> {
      if (tabs != null) {
        models.analytics.postInteraction(
            View.LeftTabs, show ? ClientAction.Enable : ClientAction.Disable);
        tabs.setLeftVisible(show);
        models.settings.hideLeft = !show;
      }
    });
    viewLeft.setChecked(!models.settings.hideLeft);
    Action viewRight = MainWindow.MenuItems.ViewRight.createCheckbox(show -> {
      if (tabs != null) {
        models.analytics.postInteraction(
            View.RightTabs, show ? ClientAction.Enable : ClientAction.Disable);
        tabs.setRightVisible(show);
        models.settings.hideRight = !show;
      }
    });
    viewRight.setChecked(!models.settings.hideRight);

    manager.add(viewScrubber);
    manager.add(viewLeft);
    manager.add(viewRight);
    manager.add(createViewTabsMenu());
  }

  private MenuManager createViewTabsMenu() {
    MenuManager manager = new MenuManager("&Tabs");
    for (MainTab.Type type : MainTab.Type.values()) {
      Action action = type.createAction(shown -> {
        models.analytics.postInteraction(
            type.view, shown ? ClientAction.Enable : ClientAction.Disable);
        if (shown) {
          tabs.addNewTabToCenter(new MainTab(type, parent -> {
            Tab tab = type.factory.create(parent, models, widgets);
            tab.reinitialize();
            return tab.getControl();
          }));
          tabs.showTab(type);
          hiddenTabs.remove(type);
        } else {
          tabs.removeTab(type);
          hiddenTabs.add(type);
        }
        models.settings.hiddenTabs =
            hiddenTabs.stream().map(MainTab.Type::name).toArray(n -> new String[n]);
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

    public static FolderInfo[] getFolders(Models models, Widgets widgets, Set<Type> hidden) {
      Set<Type> allTabs = Sets.newLinkedHashSet(Arrays.asList(Type.values()));
      allTabs.removeAll(hidden);
      List<TabInfo> left = getTabs(models.settings.leftTabs, allTabs, models, widgets);
      List<TabInfo> center = getTabs(models.settings.centerTabs, allTabs, models, widgets);
      List<TabInfo> right = getTabs(models.settings.rightTabs, allTabs, models, widgets);

      for (Type missing : allTabs) {
        switch (missing.defaultLocation) {
          case Left:
            left.add(new MainTab(missing,
                parent -> missing.factory.create(parent, models, widgets).getControl()));
            break;
          case Center:
            center.add(new MainTab(missing,
                parent -> missing.factory.create(parent, models, widgets).getControl()));
            break;
          case Right:
            right.add(new MainTab(missing,
                parent -> missing.factory.create(parent, models, widgets).getControl()));
            break;
          default:
            throw new AssertionError();
        }
      }

      double[] weights = models.settings.tabWeights;
      return new FolderInfo[] {
          new FolderInfo(false, left.toArray(new TabInfo[left.size()]), weights[0]),
          new FolderInfo(false, center.toArray(new TabInfo[center.size()]), weights[1]),
          new FolderInfo(false, right.toArray(new TabInfo[right.size()]), weights[2]),
      };
    }

    public static void store(Models models, FolderInfo[] folders) {
      models.settings.leftTabs = getNames(folders[0].tabs);
      models.settings.centerTabs = getNames(folders[1].tabs);
      models.settings.rightTabs = getNames(folders[2].tabs);
      models.settings.tabWeights = getWeights(folders);
    }

    private static String[] getNames(TabInfo[] tabs) {
      return Arrays.stream(tabs).map(tab -> ((Type)tab.id).name()).toArray(len -> new String[len]);
    }

    private static double[] getWeights(FolderInfo[] folders) {
      return new double[] { folders[0].weight, folders[1].weight, folders[2].weight };
    }

    private static List<TabInfo> getTabs(
        String[] names, Set<Type> left, Models models, Widgets widgets) {

      List<TabInfo> result = Lists.newArrayList();
      for (String name : names) {
        try {
          Type type = Type.valueOf(name);
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
     * Possible tab locations.
     */
    public static enum Location {
      Left, Center, Right;
    }

    /**
     * Information about the available tabs.
     */
    public static enum Type {
      ApiCalls(Location.Left, View.Commands, "Commands", CommandTree::new),

      Framebuffer(Location.Center, View.Framebuffer, "Framebuffer", FramebufferView::new),
      Textures(Location.Center, View.Textures, "Textures", TextureView::new),
      Geometry(Location.Center, View.Geometry, "Geometry", GeometryView::new),
      Shaders(Location.Center, View.Shaders, "Shaders", ShaderView::new),
      Report(Location.Center, View.Report, "Report", ReportView::new),
      Log(Location.Center, View.Log, "Log", (p, m, w) -> new LogView(p, w)),

      ApiState(Location.Right, View.State, "State", StateView::new),
      Memory(Location.Right, View.Memory, "Memory", MemoryView::new);

      public final Location defaultLocation;
      public final View view;
      public final String label;
      public final TabFactory factory;

      private Type(Location defaultLocation, View view, String label, TabFactory factory) {
        this.defaultLocation = defaultLocation;
        this.view = view;
        this.label = label;
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

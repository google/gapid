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
package com.google.gapid;

import static com.google.gapid.views.TracerDialog.showOpenTraceDialog;
import static com.google.gapid.views.TracerDialog.showTracingDialog;
import static com.google.gapid.views.WelcomeDialog.showWelcomeDialog;
import static com.google.gapid.widgets.AboutDialog.showAbout;
import static com.google.gapid.widgets.AboutDialog.showHelp;
import static com.google.gapid.widgets.GotoAtom.showGotoAtomDialog;
import static com.google.gapid.widgets.GotoMemory.showGotoMemoryDialog;
import static com.google.gapid.widgets.Licenses.showLicensesDialog;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.Sets;
import com.google.gapid.models.AtomStream;
import com.google.gapid.models.Capture;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service.CommandRange;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.MacApplication;
import com.google.gapid.util.Messages;
import com.google.gapid.util.OS;
import com.google.gapid.views.AtomTree;
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
import com.google.gapid.widgets.CopyPaste;
import com.google.gapid.widgets.TabArea;
import com.google.gapid.widgets.TabArea.FolderInfo;
import com.google.gapid.widgets.TabArea.Persistance;
import com.google.gapid.widgets.TabArea.TabInfo;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.action.Action;
import org.eclipse.jface.action.IAction;
import org.eclipse.jface.action.MenuManager;
import org.eclipse.jface.window.ApplicationWindow;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.SashForm;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Shell;

import java.util.Arrays;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.function.Consumer;
import java.util.function.Function;

/**
 * The main {@link ApplicationWindow} containing all of the UI components.
 */
public class MainWindow extends ApplicationWindow {
  protected final Client client;
  protected final ModelsAndWidgets maw;
  protected Action gotoAtom, gotoMemory;
  protected Action viewScrubber, viewLeft, viewRight;
  protected final Map<MainTab.Type, Action> viewTabs = Maps.newHashMap();
  protected final Set<MainTab.Type> hiddenTabs = Sets.newHashSet();
  protected Action editCopy;
  private Control scrubber;
  protected TabArea tabs;

  public MainWindow(Client client, ModelsAndWidgets maw) {
    super(null);
    this.client = client;
    this.maw = maw;

    addMenuBar();
    setBlockOnOpen(true);
  }

  @Override
  protected void configureShell(Shell shell) {
    maw.init(shell);
    viewScrubber.setChecked(!models().settings.hideScrubber);
    viewLeft.setChecked(!models().settings.hideLeft);
    viewRight.setChecked(!models().settings.hideRight);
    for (String hidden : models().settings.hiddenTabs) {
      try {
        MainTab.Type type = MainTab.Type.valueOf(hidden);
        viewTabs.get(type).setChecked(false);
        hiddenTabs.add(type);
      } catch (IllegalArgumentException e) {
        // Ignore invalid tab names in the settings file.
      }
    }

    shell.setText(Messages.WINDOW_TITLE);
    shell.setImage(widgets().theme.logo());

    super.configureShell(shell);

    models().capture.addListener(new Capture.Listener() {
      @Override
      public void onCaptureLoadingStart() {
        gotoAtom.setEnabled(false);
        gotoMemory.setEnabled(false);
      }
    });
    models().atoms.addListener(new AtomStream.Listener() {
      @Override
      public void onAtomsLoaded() {
        gotoAtom.setEnabled(models().atoms.isLoaded());
        gotoMemory.setEnabled(models().atoms.getSelectedAtoms() != null);
      }

      @Override
      public void onAtomsSelected(CommandRange path) {
        gotoMemory.setEnabled(path != null);
      }
    });
    widgets().copypaste.addListener(new CopyPaste.Listener() {
      @Override
      public void onCopyEnabled(boolean enabled) {
        editCopy.setEnabled(enabled);
      }
    });

    shell.addListener(SWT.Move, e -> models().settings.windowLocation = shell.getLocation());
    shell.addListener(SWT.Resize, e -> models().settings.windowSize = shell.getSize());

    if (OS.isMac) {
      MacApplication.init(shell.getDisplay(), () -> showAbout(shell, widgets().theme));
    }
  }

  @Override
  protected Point getInitialSize() {
    Point size = models().settings.windowSize;
    return (size != null) ? size : super.getInitialSize();
  }

  @Override
  protected Point getInitialLocation(Point initialSize) {
    Point location = models().settings.windowLocation;
    return (location != null) ? location : super.getInitialLocation(initialSize);
  }

  @Override
  protected MenuManager createMenuManager() {
    MenuManager manager = new MenuManager();
    manager.add(createFileMenu());
    manager.add(createEditMenu());
    manager.add(createGotoMenu());
    manager.add(createViewMenu());
    manager.add(createHelpMenu());
    return manager;
  }

  @Override
  protected Control createContents(Composite parent) {
    Composite shell = Widgets.createComposite(parent, new GridLayout(1, false));
    new ContextSelector(shell, models()).setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));

    SashForm splitter = new SashForm(shell, SWT.VERTICAL);
    splitter.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    scrubber = new ThumbnailScrubber(splitter, models(), widgets());
    scrubber.setVisible(!models().settings.hideScrubber);
    tabs = new TabArea(splitter, new Persistance() {
      @Override
      public void store(FolderInfo[] folders) {
        MainTab.store(models(), folders);
      }

      @Override
      public FolderInfo[] restore() {
        return MainTab.getFolders(client, models(), widgets(), hiddenTabs);
      }
    });
    tabs.setLeftVisible(!models().settings.hideLeft);
    tabs.setRightVisible(!models().settings.hideRight);

    splitter.setWeights(models().settings.splitterWeights);
    splitter.addListener(SWT.Dispose, e -> {
      models().settings.splitterWeights = splitter.getWeights();
    });

    models().capture.addListener(new Capture.Listener() {
      @Override
      public void onCaptureLoadingStart() {
        getShell().setText(Messages.WINDOW_TITLE + " - " + models().capture.getName());
      }
    });
    models().follower.addListener(new Follower.Listener() {
      @Override
      public void onMemoryFollowed(Path.Memory path) {
        tabs.showTab(MainTab.Type.Memory);
      }

      @Override
      public void onStateFollowed(Path.Any path) {
        tabs.showTab(MainTab.Type.ApiState);
      }
    });
    return shell;
  }

  @Override
  public boolean close() {
    if (super.close()) {
      maw.dispose();
      return true;
    }
    return false;
  }

  private MenuManager createFileMenu() {
    MenuManager manager = new MenuManager("&File");
    manager.add(MenuItems.FileOpen.create(() -> showOpenTraceDialog(getShell(), models())));
    manager.add(MenuItems.FileTrace.create(
        () -> showTracingDialog(getShell(), models(), widgets())));
    manager.add(MenuItems.FileExit.create(() -> close()));

    return manager;
  }

  private MenuManager createEditMenu() {
    MenuManager manager = new MenuManager("&Edit");
    editCopy = MenuItems.EditCopy.create(() -> widgets().copypaste.doCopy());

    manager.add(editCopy);

    editCopy.setEnabled(false);

    return manager;
  }

  private MenuManager createGotoMenu() {
    MenuManager manager = new MenuManager("&Goto");
    gotoAtom = MenuItems.GotoAtom.create(() -> showGotoAtomDialog(getShell(), models().atoms));
    gotoMemory = MenuItems.GotoMemory.create(() -> showGotoMemoryDialog(getShell(), models()));

    manager.add(gotoAtom);
    manager.add(gotoMemory);

    gotoAtom.setEnabled(false);
    gotoMemory.setEnabled(false);

    return manager;
  }

  private MenuManager createViewMenu() {
    MenuManager manager = new MenuManager("&View");
    viewScrubber = MenuItems.ViewThumbnails.createCheckbox(show -> {
      if (scrubber != null) {
        scrubber.setVisible(show);
        scrubber.requestLayout();
        models().settings.hideScrubber = !show;
      }
    });
    viewLeft = MenuItems.ViewLeft.createCheckbox(show -> {
      if (tabs != null) {
        tabs.setLeftVisible(show);
        models().settings.hideLeft = !show;
      }
    });
    viewRight = MenuItems.ViewRight.createCheckbox(show -> {
      if (tabs != null) {
        tabs.setRightVisible(show);
        models().settings.hideRight = !show;
      }
    });

    manager.add(viewScrubber);
    manager.add(viewLeft);
    manager.add(viewRight);
    manager.add(createViewTabsMenu());
    return manager;
  }

  private MenuManager createViewTabsMenu() {
    MenuManager manager = new MenuManager("&Tabs");
    for (MainTab.Type type : MainTab.Type.values()) {
      Action action = type.createAction(shown -> {
        if (shown) {
          tabs.addNewTabToCenter(new MainTab(type, parent -> {
            Tab tab = type.factory.create(parent, client, models(), widgets());
            tab.reinitialize();
            return tab.getControl();
          }));
          tabs.showTab(type);
          hiddenTabs.remove(type);
        } else {
          tabs.removeTab(type);
          hiddenTabs.add(type);
        }
        models().settings.hiddenTabs =
            hiddenTabs.stream().map(MainTab.Type::name).toArray(n -> new String[n]);
      });
      manager.add(action);
      viewTabs.put(type, action);
    }
    return manager;
  }

  private MenuManager createHelpMenu() {
    MenuManager manager = new MenuManager("&Help");
    manager.add(MenuItems.HelpOnlineHelp.create(() -> showHelp()));
    manager.add(MenuItems.HelpAbout.create(() -> showAbout(getShell(), widgets().theme)));
    manager.add(MenuItems.HelpLicenses.create(() -> showLicensesDialog(getShell())));
    manager.add(MenuItems.HelpWelcome.create(
        () -> showWelcomeDialog(getShell(), models(), widgets())));
    return manager;
  }

  protected Models models() {
    return maw.models();
  }

  protected Widgets widgets() {
    return maw.widgets();
  }

  /**
   * Manages the lifetime of the {@link Models} and {@link Widgets}.
   */
  public static interface ModelsAndWidgets {
    /**
     * Initializes the models and widgets for the given window shell.
     */
    public void init(Shell shell);

    /**
     * @return the {@link Models}.
     */
    public Models models();

    /**
     * @return the {@link Widgets}.
     */
    public Widgets widgets();

    /**
     * Disposes the models and widgets.
     */
    public void dispose();
  }

  /**
   * Information about the tabs to be shown in the main window.
   */
  private static class MainTab extends TabInfo {
    public MainTab(Type type, Function<Composite, Control> contentFactory) {
      super(type, type.label, contentFactory);
    }

    public static FolderInfo[] getFolders(
        Client client, Models models, Widgets widgets, Set<Type> hidden) {
      Set<Type> allTabs = Sets.newLinkedHashSet(Arrays.asList(Type.values()));
      allTabs.removeAll(hidden);
      List<TabInfo> left = getTabs(models.settings.leftTabs, allTabs, client, models, widgets);
      List<TabInfo> center = getTabs(models.settings.centerTabs, allTabs, client, models, widgets);
      List<TabInfo> right = getTabs(models.settings.rightTabs, allTabs, client, models, widgets);

      for (Type missing : allTabs) {
        switch (missing.defaultLocation) {
          case Left:
            left.add(new MainTab(missing,
                parent -> missing.factory.create(parent, client, models, widgets).getControl()));
            break;
          case Center:
            center.add(new MainTab(missing,
                parent -> missing.factory.create(parent, client, models, widgets).getControl()));
            break;
          case Right:
            right.add(new MainTab(missing,
                parent -> missing.factory.create(parent, client, models, widgets).getControl()));
            break;
          default:
            throw new AssertionError();
        }
      }

      int[] weights = models.settings.tabWeights;
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

    private static int[] getWeights(FolderInfo[] folders) {
      return new int[] { folders[0].weight, folders[1].weight, folders[2].weight };
    }

    private static List<TabInfo> getTabs(
        String[] names, Set<Type> left, Client client, Models models, Widgets widgets) {

      List<TabInfo> result = Lists.newArrayList();
      for (String name : names) {
        try {
          Type type = Type.valueOf(name);
          if (left.remove(type)) {
            result.add(new MainTab(type,
                parent -> type.factory.create(parent, client, models, widgets).getControl()));
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
      ApiCalls(Location.Left, "Commands", (p, c, m, w) -> new AtomTree(p, m, w)),

      Framebuffer(Location.Center, "Framebuffer", (p, c, m, w) -> new FramebufferView(p, c, m, w)),
      Textures(Location.Center, "Textures", (p, c, m, w) -> new TextureView(p, c, m, w)),
      Geometry(Location.Center, "Geometry", (p, c, m, w) -> new GeometryView(p, c, m, w)),
      Shaders(Location.Center, "Shaders", (p, c, m, w) -> new ShaderView(p, c, m, w)),
      Report(Location.Center, "Report", (p, c, m, w) -> new ReportView(p, m, w)),
      Log(Location.Center, "Log", (p, c, m, w) -> new LogView(p)),

      ApiState(Location.Right, "State", (p, c, m, w) -> new StateView(p, m, w)),
      Memory(Location.Right, "Memory", (p, c, m, w) -> new MemoryView(p, c, m, w));

      public final Location defaultLocation;
      public final String label;
      public final TabFactory factory;

      private Type(Location defaultLocation, String label, TabFactory factory) {
        this.defaultLocation = defaultLocation;
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
      public Tab create(Composite parent, Client client, Models models, Widgets widgets);
    }
  }

  /**
   * The menu items shown in the main application window menus.
   */
  private static enum MenuItems {
    FileOpen("&Open", 'O'),
    FileTrace("Capture &Trace", 'T'),
    FileExit("&Exit", 'Q'),

    EditCopy("&Copy", 'C'),

    GotoAtom("Api &Call", 'G'),
    GotoMemory("&Memory Location", 'M'),

    ViewThumbnails("Show Filmstrip"),
    ViewLeft("Show Left Tabs"),
    ViewRight("Show Right Tabs"),

    HelpOnlineHelp("&Online Help\tF1", SWT.F1),
    HelpAbout("&About"),
    HelpLicenses("&Licenses"),
    HelpWelcome("Show &Welcome Screen");


    private final String label;
    private final int accelerator;

    private MenuItems(String label) {
      this(label, 0);
    }

    private MenuItems(String label, char ctrlAcc) {
      this(label + "\tCtrl+" + ctrlAcc, SWT.MOD1 + ctrlAcc);
    }

    private MenuItems(String label, int accelerator) {
      this.label = label;
      this.accelerator = accelerator;
    }

    public Action create(Runnable listener) {
      return configure(new Action() {
        @Override
        public void run() {
          listener.run();
        }
      });
    }

    public Action createCheckbox(Consumer<Boolean> listener) {
      return configure(new Action(null, IAction.AS_CHECK_BOX) {
        @Override
        public void run() {
          listener.accept(isChecked());
        }
      });
    }

    private Action configure(Action action) {
      action.setText(label);
      action.setAccelerator(accelerator);
      return action;
    }
  }
}

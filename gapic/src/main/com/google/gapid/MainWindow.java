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

import static com.google.gapid.views.AboutDialog.showAbout;
import static com.google.gapid.views.AboutDialog.showHelp;
import static com.google.gapid.views.AboutDialog.showLogDir;
import static com.google.gapid.views.GotoCommand.showGotoCommandDialog;
import static com.google.gapid.views.GotoMemory.showGotoMemoryDialog;
import static com.google.gapid.views.Licenses.showLicensesDialog;
import static com.google.gapid.views.SettingsDialog.showSettingsDialog;
import static com.google.gapid.views.TracerDialog.showOpenTraceDialog;
import static com.google.gapid.views.TracerDialog.showSaveTraceDialog;
import static com.google.gapid.views.TracerDialog.showTracingDialog;
import static com.google.gapid.views.WelcomeDialog.showWelcomeDialog;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.Sets;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.MacApplication;
import com.google.gapid.util.Messages;
import com.google.gapid.util.OS;
import com.google.gapid.util.UpdateWatcher;
import com.google.gapid.views.CommandTree;
import com.google.gapid.views.ContextSelector;
import com.google.gapid.views.FramebufferView;
import com.google.gapid.views.GeometryView;
import com.google.gapid.views.LogView;
import com.google.gapid.views.MemoryView;
import com.google.gapid.views.ReportView;
import com.google.gapid.views.ShaderView;
import com.google.gapid.views.StateView;
import com.google.gapid.views.StatusBar;
import com.google.gapid.views.Tab;
import com.google.gapid.views.TextureView;
import com.google.gapid.views.ThumbnailScrubber;
import com.google.gapid.widgets.CopyPaste;
import com.google.gapid.widgets.FixedTopSplitter;
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
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.program.Program;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Shell;

import java.io.File;
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
  protected Action gotoCommand, gotoMemory;
  protected Action viewScrubber, viewLeft, viewRight;
  protected final Map<MainTab.Type, Action> viewTabs = Maps.newHashMap();
  protected final Set<MainTab.Type> hiddenTabs = Sets.newHashSet();
  protected Action editCopy;
  private FixedTopSplitter splitter;
  private StatusBar statusBar;
  protected TabArea tabs;

  public MainWindow(Client client, ModelsAndWidgets maw) {
    super(null);
    this.client = client;
    this.maw = maw;

    addMenuBar();
    setBlockOnOpen(true);
  }

  /*
  @Override
  public int open() {
    setBlockOnOpen(false);
    super.open();
    Shell shell = getShell();
    Display display = shell.getDisplay();
    while (!shell.isDisposed()) {
      long start = System.nanoTime();
      boolean sleep = !display.readAndDispatch();
      long end = System.nanoTime();
      System.err.println(TimeUnit.NANOSECONDS.toMillis(end - start) + " " + sleep);
      if (sleep) {
        display.sleep();
      }
    }
    if (!display.isDisposed()) {
      display.update();
    }
    return 0;
  }
  */

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
    shell.setImages(widgets().theme.windowLogo());

    super.configureShell(shell);

    models().capture.addListener(new Capture.Listener() {
      @Override
      public void onCaptureLoadingStart(boolean maintainState) {
        gotoCommand.setEnabled(false);
        gotoMemory.setEnabled(false);
      }
    });
    models().commands.addListener(new CommandStream.Listener() {
      @Override
      public void onCommandsLoaded() {
        gotoCommand.setEnabled(models().commands.isLoaded());
        gotoMemory.setEnabled(models().commands.getSelectedCommands() != null);
      }

      @Override
      public void onCommandsSelected(CommandIndex selection) {
        gotoMemory.setEnabled(selection != null);
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
      MacApplication.init(shell.getDisplay(),
          () -> showAbout(shell, models().analytics, widgets().theme),
          () -> showSettingsDialog(shell, models(), widgets().theme),
          file -> models().capture.loadCapture(new File(file)));
    }
  }

  @Override
  protected Point getInitialSize() {
    Point size = models().settings.windowSize;
    return (size != null) ? size : getDefaultInitialSize();
  }

  private Point getDefaultInitialSize() {
    Rectangle bounds = getShell().getDisplay().getPrimaryMonitor().getClientArea();
    return new Point((int)(0.6 * bounds.width), (int)(0.8 * bounds.height));
  }

  @Override
  protected Point getInitialLocation(Point initialSize) {
    Point location = models().settings.windowLocation;
    return (location != null) ? location : getDefaultInitialLocation(initialSize);
  }

  private Point getDefaultInitialLocation(Point size) {
    Rectangle bounds = getShell().getDisplay().getPrimaryMonitor().getClientArea();
    // Center horizontally, split vertical space 1/3 - 2/3.
    return new Point(Math.max(0, bounds.width - size.x ) / 2,
        Math.max(0, bounds.height - size.y) / 3);
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
    new ContextSelector(shell, models())
        .setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));

    splitter = new FixedTopSplitter(shell, models().settings.splitterTopHeight) {
      @Override
      protected Control createTopControl() {
        return new ThumbnailScrubber(this, models(), widgets());
      }

      @Override
      protected Control createBottomControl() {
        tabs = new TabArea(this, new Persistance() {
          @Override
          public void store(FolderInfo[] folders) {
            MainTab.store(models(), folders);
          }

          @Override
          public FolderInfo[] restore() {
            return MainTab.getFolders(client, models(), widgets(), hiddenTabs);
          }
        }, models().analytics);
        tabs.setLeftVisible(!models().settings.hideLeft);
        tabs.setRightVisible(!models().settings.hideRight);
        return tabs;
      }
    };
    splitter.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    splitter.setTopVisible(!models().settings.hideScrubber);

    statusBar = new StatusBar(shell);
    statusBar.setLayoutData(new GridData(SWT.FILL, SWT.BOTTOM, true, false));

    splitter.addListener(SWT.Dispose, e -> {
      models().settings.splitterTopHeight = splitter.getTopHeight();
    });

    models().capture.addListener(new Capture.Listener() {
      @Override
      public void onCaptureLoadingStart(boolean maintainState) {
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

    watchForUpdates();

    return shell;
  }

  private void watchForUpdates() {
    new UpdateWatcher(maw.models().settings, client, (release) -> {
      scheduleIfNotDisposed(statusBar, () -> {
        statusBar.setNotification("New update available", () -> {
          Program.launch(release.getBrowserUrl());
        });
      });
    });
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
    manager.add(MenuItems.FileSave.create(() -> showSaveTraceDialog(getShell(), models())));
    manager.add(createOpenRecentMenu());
    manager.add(MenuItems.FileTrace.create(
        () -> showTracingDialog(client, getShell(), models(), widgets())));
    manager.add(MenuItems.FileExit.create(() -> close()));

    return manager;
  }

  private MenuManager createOpenRecentMenu() {
    MenuManager manager = new MenuManager("Open &Recent");
    manager.setRemoveAllWhenShown(true);
    manager.addMenuListener(m -> {
      for (String file : models().settings.getRecent()) {
        m.add(new Action(file) {
          @Override
          public void run() {
            models().analytics.postInteraction(View.Main, ClientAction.OpenRecent);
            models().capture.loadCapture(new File(file));
          }
        });
      }
    });
    return manager;
  }

  private MenuManager createEditMenu() {
    MenuManager manager = new MenuManager("&Edit");
    editCopy = MenuItems.EditCopy.create(() -> {
      models().analytics.postInteraction(View.Main, ClientAction.Copy);
      widgets().copypaste.doCopy();
    });

    manager.add(editCopy);
    manager.add(MenuItems.EditSettings.create(
        () -> showSettingsDialog(getShell(), models(), widgets().theme)));

    editCopy.setEnabled(false);

    return manager;
  }

  private MenuManager createGotoMenu() {
    MenuManager manager = new MenuManager("&Goto");
    gotoCommand = MenuItems.GotoCommand.create(() -> showGotoCommandDialog(getShell(), models()));
    gotoMemory = MenuItems.GotoMemory.create(() -> showGotoMemoryDialog(getShell(), models()));

    manager.add(gotoCommand);
    manager.add(gotoMemory);

    gotoCommand.setEnabled(false);
    gotoMemory.setEnabled(false);

    return manager;
  }

  private MenuManager createViewMenu() {
    MenuManager manager = new MenuManager("&View");
    viewScrubber = MenuItems.ViewThumbnails.createCheckbox(show -> {
      if (splitter != null) {
        models().analytics.postInteraction(
            View.FilmStrip, show ? ClientAction.Enable : ClientAction.Disable);
        splitter.setTopVisible(show);
        models().settings.hideScrubber = !show;
      }
    });
    viewLeft = MenuItems.ViewLeft.createCheckbox(show -> {
      if (tabs != null) {
        models().analytics.postInteraction(
            View.LeftTabs, show ? ClientAction.Enable : ClientAction.Disable);
        tabs.setLeftVisible(show);
        models().settings.hideLeft = !show;
      }
    });
    viewRight = MenuItems.ViewRight.createCheckbox(show -> {
      if (tabs != null) {
        models().analytics.postInteraction(
            View.RightTabs, show ? ClientAction.Enable : ClientAction.Disable);
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
        models().analytics.postInteraction(
            type.view, shown ? ClientAction.Enable : ClientAction.Disable);
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
    manager.add(MenuItems.HelpOnlineHelp.create(() -> showHelp(models().analytics)));
    manager.add(MenuItems.HelpAbout.create(
        () -> showAbout(getShell(), models().analytics, widgets().theme)));
    manager.add(MenuItems.HelpShowLogs.create(() -> showLogDir(models().analytics)));
    manager.add(MenuItems.HelpLicenses.create(
        () -> showLicensesDialog(getShell(), models().analytics, widgets().theme)));
    manager.add(MenuItems.HelpWelcome.create(
        () -> showWelcomeDialog(client, getShell(), models(), widgets())));
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
      super(type, type.view, type.label, contentFactory);
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
      ApiCalls(Location.Left, View.Commands, "Commands", (p, c, m, w) -> new CommandTree(p, m, w)),

      Framebuffer(Location.Center, View.Framebuffer, "Framebuffer", (p, c, m, w) -> new FramebufferView(p, m, w)),
      Textures(Location.Center, View.Textures, "Textures", (p, c, m, w) -> new TextureView(p, m, w)),
      Geometry(Location.Center, View.Geometry, "Geometry", (p, c, m, w) -> new GeometryView(p, m, w)),
      Shaders(Location.Center, View.Shaders, "Shaders", (p, c, m, w) -> new ShaderView(p, m, w)),
      Report(Location.Center, View.Report, "Report", (p, c, m, w) -> new ReportView(p, m, w)),
      Log(Location.Center, View.Log, "Log", (p, c, m, w) -> new LogView(p, w)),

      ApiState(Location.Right, View.State, "State", (p, c, m, w) -> new StateView(p, m, w)),
      Memory(Location.Right, View.Memory, "Memory", (p, c, m, w) -> new MemoryView(p, m, w));

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
      public Tab create(Composite parent, Client client, Models models, Widgets widgets);
    }
  }

  /**
   * The menu items shown in the main application window menus.
   */
  private static enum MenuItems {
    FileOpen("&Open", 'O'),
    FileSave("&Save", 'S'),
    FileTrace("Capture &Trace", 'T'),
    FileExit("&Exit", 'Q'),

    EditCopy("&Copy", 'C'),
    EditSettings("&Preferences", ','),

    GotoCommand("&Command", 'G'),
    GotoMemory("&Memory Location", 'M'),

    ViewThumbnails("Show Filmstrip"),
    ViewLeft("Show Left Tabs"),
    ViewRight("Show Right Tabs"),

    HelpOnlineHelp("&Online Help\tF1", SWT.F1),
    HelpAbout("&About"),
    HelpShowLogs("Open &Log Directory"),
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

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
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;

import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.server.Client;
import com.google.gapid.util.Loadable.Message;
import com.google.gapid.util.MacApplication;
import com.google.gapid.util.Messages;
import com.google.gapid.util.OS;
import com.google.gapid.util.StatusWatcher;
import com.google.gapid.util.UpdateWatcher;
import com.google.gapid.views.StatusBar;
import com.google.gapid.widgets.CopyPaste;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.action.Action;
import org.eclipse.jface.action.IAction;
import org.eclipse.jface.action.MenuManager;
import org.eclipse.jface.window.ApplicationWindow;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.StackLayout;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.program.Program;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Shell;

import java.io.File;
import java.util.function.Consumer;

/**
 * The main {@link ApplicationWindow} containing all of the UI components.
 */
public class MainWindow extends ApplicationWindow {
  private final Settings settings;
  private final Theme theme;
  private Composite mainArea;
  private LoadingScreen loadingScreen;
  protected StatusBar statusBar;

  public MainWindow(Settings settings, Theme theme) {
    super(null);
    this.settings = settings;
    this.theme = theme;

    addMenuBar();
    setBlockOnOpen(true);
  }

  public StatusBar getStatusBar() {
    return statusBar;
  }

  public void showLoadingMessage(String status) {
    loadingScreen.setText(status);
  }

  public void initMainUi(Client client, Models models, Widgets widgets) {
    Shell shell = getShell();

    showLoadingMessage("Setting up UI...");
    initMenus(client, models, widgets);

    LoadablePanel<MainViewContainer> mainUi = new LoadablePanel<MainViewContainer>(
        mainArea, widgets, parent -> new MainViewContainer(parent, models, widgets));
    models.capture.addListener(new Capture.Listener() {
      @Override
      public void onCaptureLoadingStart(boolean maintainState) {
        shell.setText(Messages.WINDOW_TITLE + " - " + models.capture.getName());
        setTopControl(mainUi);
        mainUi.startLoading();
      }

      @Override
      public void onCaptureLoaded(Message error) {
        if (error != null) {
          mainUi.showMessage(error);
        } else {
          MainView view = mainUi.getContents().updateAndGet(
              models.capture.getData().capture.getType());
          view.updateViewMenu(findMenu(MenuItems.VIEW_ID));
          getMenuBarManager().updateAll(true);
          mainUi.stopLoading();
        }
      }
    });

    if (OS.isMac) {
      MacApplication.init(shell.getDisplay(),
          () -> showAbout(shell, models.analytics, widgets.theme),
          () -> showSettingsDialog(shell, models, widgets.theme),
          file -> models.capture.loadCapture(new File(file)));
    }

    if (settings.autoCheckForUpdates) {
      // Only show the status message if we're actually checking for updates. watchForUpdates only
      //schedules a periodic check to see if we should check for updates and if so, checks.
      showLoadingMessage("Watching for updates...");
    }
    watchForUpdates(client, models);

    showLoadingMessage("Tracking server status...");
    trackServerStatus(client);

    showLoadingMessage("Ready! Please open or capture a trace file.");
  }

  private void watchForUpdates(Client client, Models models) {
    new UpdateWatcher(models.settings, client, (release) -> {
      scheduleIfNotDisposed(statusBar, () -> {
        statusBar.setNotification("New update available", () -> {
          Program.launch(release.getBrowserUrl());
        });
      });
    });
  }

  private void trackServerStatus(Client client) {
    new StatusWatcher(client, new StatusWatcher.Listener() {
      @Override
      public void onStatus(String status) {
        scheduleIfNotDisposed(statusBar, () -> statusBar.setServerStatus(status));
      }

      @Override
      public void onHeap(long heap) {
        scheduleIfNotDisposed(statusBar, () -> statusBar.setServerHeapSize(heap));
      }
    });
  }

  @Override
  protected void configureShell(Shell shell) {
    shell.setText(Messages.WINDOW_TITLE);
    shell.setImages(theme.windowLogo());

    super.configureShell(shell);

    shell.addListener(SWT.Move, e -> settings.windowLocation = shell.getLocation());
    shell.addListener(SWT.Resize, e -> settings.windowSize = shell.getSize());
  }

  @Override
  protected Control createContents(Composite shell) {
    Composite parent = createComposite(shell, withMargin(new GridLayout(1, false), 0, 0));

    mainArea = withLayoutData(
        createComposite(parent, new StackLayout()), new GridData(SWT.FILL, SWT.FILL, true, true));
    loadingScreen = new LoadingScreen(mainArea, theme);
    setTopControl(loadingScreen);

    statusBar = new StatusBar(parent, theme);
    statusBar.setLayoutData(new GridData(SWT.FILL, SWT.BOTTOM, true, false));
    return parent;
  }

  protected void setTopControl(Control c) {
    ((StackLayout)mainArea.getLayout()).topControl = c;
    c.requestLayout();
  }

  @Override
  protected Point getInitialSize() {
    Point size = settings.windowSize;
    return (size != null) ? size : getDefaultInitialSize();
  }

  private Point getDefaultInitialSize() {
    Rectangle bounds = getShell().getDisplay().getPrimaryMonitor().getClientArea();
    return new Point((int)(0.6 * bounds.width), (int)(0.8 * bounds.height));
  }

  @Override
  protected Point getInitialLocation(Point initialSize) {
    Point location = settings.windowLocation;
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

    // Add a dummy file menu, so the UI doesn't move once the rest of the menus are created.
    MenuManager file = new MenuManager("&File", MenuItems.FILE_ID);
    file.add(MenuItems.FileExit.create(this::close));
    manager.add(file);

    return manager;
  }

  private void initMenus(Client client, Models models, Widgets widgets) {
    updateFileMenu(client, models, widgets);
    MenuManager manager = getMenuBarManager();
    manager.add(createEditMenu(models, widgets));
    manager.add(createGotoMenu(models));
    manager.add(createViewMenu());
    manager.add(createHelpMenu(client, models, widgets));
    manager.updateAll(true);
  }

  protected MenuManager findMenu(String id) {
    return (MenuManager)getMenuBarManager().find(id);
  }

  private MenuManager updateFileMenu(Client client, Models models, Widgets widgets) {
    MenuManager manager = findMenu(MenuItems.FILE_ID);
    manager.removeAll();

    manager.add(MenuItems.FileOpen.create(() -> showOpenTraceDialog(getShell(), models)));
    manager.add(MenuItems.FileSave.create(() -> showSaveTraceDialog(getShell(), models)));
    manager.add(createOpenRecentMenu(models));
    manager.add(MenuItems.FileTrace.create(
        () -> showTracingDialog(client, getShell(), models, widgets)));
    manager.add(MenuItems.FileExit.create(() -> close()));

    return manager;
  }

  private static MenuManager createOpenRecentMenu(Models models) {
    MenuManager manager = new MenuManager("Open &Recent");
    manager.setRemoveAllWhenShown(true);
    manager.addMenuListener(m -> {
      for (String file : models.settings.getRecent()) {
        m.add(new Action(file) {
          @Override
          public void run() {
            models.analytics.postInteraction(View.Main, ClientAction.OpenRecent);
            models.capture.loadCapture(new File(file));
          }
        });
      }
    });
    return manager;
  }

  private MenuManager createEditMenu(Models models, Widgets widgets) {
    MenuManager manager = new MenuManager("&Edit");
    Action editCopy = MenuItems.EditCopy.create(() -> {
      models.analytics.postInteraction(View.Main, ClientAction.Copy);
      widgets.copypaste.doCopy();
    });

    manager.add(editCopy);
    manager.add(MenuItems.EditSettings.create(
        () -> showSettingsDialog(getShell(), models, widgets.theme)));

    editCopy.setEnabled(false);
    widgets.copypaste.addListener(new CopyPaste.Listener() {
      @Override
      public void onCopyEnabled(boolean enabled) {
        editCopy.setEnabled(enabled);
      }
    });

    return manager;
  }

  private MenuManager createGotoMenu(Models models) {
    MenuManager manager = new MenuManager("&Goto");
    Action gotoCommand = MenuItems.GotoCommand.create(() -> showGotoCommandDialog(getShell(), models));
    Action gotoMemory = MenuItems.GotoMemory.create(() -> showGotoMemoryDialog(getShell(), models));

    manager.add(gotoCommand);
    manager.add(gotoMemory);

    gotoCommand.setEnabled(false);
    gotoMemory.setEnabled(false);
    models.capture.addListener(new Capture.Listener() {
      @Override
      public void onCaptureLoadingStart(boolean maintainState) {
        gotoCommand.setEnabled(false);
        gotoMemory.setEnabled(false);
      }
    });
    models.commands.addListener(new CommandStream.Listener() {
      @Override
      public void onCommandsLoaded() {
        gotoCommand.setEnabled(models.commands.isLoaded());
        gotoMemory.setEnabled(models.commands.getSelectedCommands() != null);
      }

      @Override
      public void onCommandsSelected(CommandIndex selection) {
        gotoMemory.setEnabled(selection != null);
      }
    });

    return manager;
  }

  private static MenuManager createViewMenu() {
    MenuManager manager = new MenuManager("&View", MenuItems.VIEW_ID);
    return manager;
  }

  private MenuManager createHelpMenu(Client client, Models models, Widgets widgets) {
    MenuManager manager = new MenuManager("&Help");
    manager.add(MenuItems.HelpOnlineHelp.create(() -> showHelp(models.analytics)));
    manager.add(MenuItems.HelpAbout.create(
        () -> showAbout(getShell(), models.analytics, widgets.theme)));
    manager.add(MenuItems.HelpShowLogs.create(() -> showLogDir(models.analytics)));
    manager.add(MenuItems.HelpLicenses.create(
        () -> showLicensesDialog(getShell(), models.analytics, widgets.theme)));
    manager.add(MenuItems.HelpWelcome.create(
        () -> showWelcomeDialog(client, getShell(), models, widgets)));
    return manager;
  }

  private static class MainViewContainer extends Composite {
    private final Models models;
    private final Widgets widgets;

    private Service.TraceType current;
    private MainView view;

    public MainViewContainer(Composite parent, Models models, Widgets widgets) {
      super(parent, SWT.NONE);
      this.models = models;
      this.widgets = widgets;

      setLayout(new FillLayout());
    }

    public MainView updateAndGet(Service.TraceType traceType) {
      if (traceType == current) {
        return view;
      }
      if (view != null) {
        ((Control)view).dispose();
      }

      current = traceType;
      switch (traceType) {
        case Graphics:
          view = new GraphicsTraceView(this, models, widgets);
          break;
        case Perfetto:
          view = new PerfettoTraceView(this, models, widgets);
          break;
        default:
          throw new AssertionError("Trace type not supported: " + traceType);
      }
      layout();
      return view;
    }
  }

  /**
   * The menu items shown in the main application window menus.
   */
  public static enum MenuItems {
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
    ViewDarkMode("Dark Mode", 'D'),

    HelpOnlineHelp("&Online Help\tF1", SWT.F1),
    HelpAbout("&About"),
    HelpShowLogs("Open &Log Directory"),
    HelpLicenses("&Licenses"),
    HelpWelcome("Show &Welcome Screen");

    public static final String FILE_ID = "file";
    public static final String VIEW_ID = "view";

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

  /**
   * Main view shown once a trace is loaded.
   */
  public static interface MainView {
    public void updateViewMenu(MenuManager manager);
  }
}

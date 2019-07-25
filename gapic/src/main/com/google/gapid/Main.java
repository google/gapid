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

import static com.google.gapid.util.GapidVersion.GAPID_VERSION;
import static com.google.gapid.views.ErrorDialog.showErrorDialog;
import static com.google.gapid.views.WelcomeDialog.showFirstTimeDialog;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;

import com.google.common.base.Throwables;
import com.google.gapid.Server.GapisInitException;
import com.google.gapid.models.Analytics;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.perfetto.PerfettoConfig;
import com.google.gapid.server.GapiPaths;
import com.google.gapid.server.GapisProcess;
import com.google.gapid.util.Crash2ExceptionHandler;
import com.google.gapid.util.ExceptionHandler;
import com.google.gapid.util.Flags;
import com.google.gapid.util.Flags.Flag;
import com.google.gapid.util.Logging;
import com.google.gapid.util.Messages;
import com.google.gapid.util.Scheduler;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.window.Window;
import org.eclipse.swt.widgets.Display;
import org.eclipse.swt.widgets.Shell;

import java.io.File;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Main entry point of the application.
 */
public class Main {
  protected static final Logger LOG = Logger.getLogger(Main.class.getName());

  public static void main(String[] args) throws Exception {
    args = Flags.initFlags(ALL_FLAGS, args);
    Logging.init();

    Display.setAppName(Messages.WINDOW_TITLE);
    Display.setAppVersion(GAPID_VERSION.toString());

    Settings settings = Settings.load();
    Theme theme = Theme.load(Display.getDefault());
    ExceptionHandler handler = Crash2ExceptionHandler.register(settings);

    try {
      new UI(settings, theme, handler, args).show();
    } finally {
      Scheduler.EXECUTOR.shutdownNow();
    }
  }

  /**
   * Manages the main UI.
   */
  private static class UI implements GapisProcess.Listener {
    private final Settings settings;
    private final Theme theme;
    private final ExceptionHandler handler;
    private final String[] args;
    protected final MainWindow window;
    private final Server server;

    private Models models;
    private Widgets widgets;

    public UI(Settings settings, Theme theme, ExceptionHandler handler, String[] args) {
      this.settings = settings;
      this.theme = theme;
      this.handler = handler;
      this.args = args;
      this.window = new MainWindow(settings, theme) {
        @Override
        public void create() {
          super.create();
          scheduleIfNotDisposed(getShell(), () -> Scheduler.EXECUTOR.execute(UI.this::startup));
        }
      };
      server = new Server(settings);

      registerWindowExceptionHandler();
    }

    private void registerWindowExceptionHandler() {
      Window.setExceptionHandler(thrown -> {
        if (thrown instanceof ThreadDeath) {
          throw (ThreadDeath) thrown;
        }

        LOG.log(Level.WARNING, "Unhandled exception in the UI thread.", thrown);
        handler.reportException(thrown);
        showErrorDialog(null, getAnalytics(), "Unhandled exception in the UI thread.", thrown);
      });
    }

    public void show() {
      try {
        window.open();
      } finally {
        server.disconnect();

        if (widgets != null) {
          widgets.dispose();
          models.dispose();
        }

        models = null;
        widgets = null;
      }
    }

    protected void startup() {
      try {
        server.connect(this);
        scheduleIfStillOpen(this::uiStartup);
      } catch (GapisInitException e) {
        onServerExit(-42, Throwables.getStackTraceAsString(e));
      }
    }

    private void uiStartup(Shell shell) {
      models = Models.create(shell, settings, handler, server.getClient(), window.getStatusBar());
      widgets = Widgets.create(shell.getDisplay(), theme, server.getClient(), models);

      Runnable onStart = () -> {
        if (args.length == 1) {
          models.capture.loadCapture(new File(args[0]));
        }
      };

      window.initMainUi(server.getClient(), models, widgets);
      if (models.settings.skipFirstRunDialog) {
        shell.getDisplay().asyncExec(onStart);
      } else {
        shell.getDisplay().asyncExec(() -> showFirstTimeDialog(shell, models, widgets, onStart));
      }

      // Add the links on Loading Screen after the server set up.
      window.updateLoadingScreen(server.getClient(), models, widgets);
    }

    @Override
    public void onStatus(String message) {
      scheduleIfStillOpen(shell -> window.showLoadingMessage(message));
    }

    @Override
    public void onServerExit(int code, String panic) {
      scheduleIfStillOpen(shell ->
        // TODO: try to restart the server?
        showErrorDialog(shell, getAnalytics(),
            "The gapis server has exited with an error code of: " + code, panic)
      );
    }

    private void scheduleIfStillOpen(ShellRunnable run) {
      Shell shell = window.getShell();
      if (shell == null) {
        return;
      }
      scheduleIfNotDisposed(shell, () -> run.run(shell));
    }

    private Analytics getAnalytics() {
      return (models == null) ? null : models.analytics;
    }

    private static interface ShellRunnable {
      public void run(Shell shell);
    }
  }

  private static final Flag<?>[] ALL_FLAGS = {
    Flags.help,
    Flags.version,
    GapiPaths.gapidPath,
    GapiPaths.adbPath,
    GapisProcess.disableGapisTimeout,
    Server.gapis,
    Server.gapisAuthToken,
    GapisProcess.gapirArgs,
    GapisProcess.gapisArgs,
    Logging.logLevel,
    Logging.gapisLogLevel,
    Logging.gapirLogLevel,
    Logging.logDir,
    Follower.logFollowRequests,
    Server.useCache,
    PerfettoConfig.perfettoConfig,
  };
}

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
import static com.google.gapid.views.ErrorDialog.showErrorDialogWithTwoButtons;
import static com.google.gapid.views.WelcomeDialog.showFirstTimeDialog;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;

import com.google.common.base.Throwables;
import com.google.gapid.Server.GapisInitException;
import com.google.gapid.models.Analytics;
import com.google.gapid.models.Devices;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.perfetto.canvas.PanelCanvas;
import com.google.gapid.server.Client;
import com.google.gapid.server.GapiPaths;
import com.google.gapid.server.GapisProcess;
import com.google.gapid.util.Crash2ExceptionHandler;
import com.google.gapid.util.ExceptionHandler;
import com.google.gapid.util.Experimental;
import com.google.gapid.util.Flags;
import com.google.gapid.util.Flags.Flag;
import com.google.gapid.util.Logging;
import com.google.gapid.util.MacApplication;
import com.google.gapid.util.Messages;
import com.google.gapid.util.OS;
import com.google.gapid.util.Scheduler;
import com.google.gapid.views.TracerDialog;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.dialogs.IDialogConstants;
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

    if (OS.isMac) {
      MacApplication.listenForOpenDocument(Display.getDefault());
    }

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

    // Client is an immutable wrapper keeping track of inner server-dependent GapidClient instance.
    private final Client client;
    private Server server;
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
      this.client = new Client();
      server = new Server(settings, client);

      registerWindowExceptionHandler();
    }

    private void registerWindowExceptionHandler() {
      Window.setExceptionHandler(thrown -> {
        if (thrown instanceof ThreadDeath) {
          throw (ThreadDeath) thrown;
        }

        LOG.log(Level.WARNING, "Unhandled exception in the UI thread.", thrown);
        handler.reportException(thrown);
        if (shouldShowUIErrorDialog(thrown)) {
          showErrorDialog(null, getAnalytics(), "Unhandled exception in the UI thread.", thrown);
        }
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
      models = Models.create(shell, settings, handler, client, window.getStatusBar());
      widgets = Widgets.create(shell.getDisplay(), theme, client, models);

      Runnable onStart = () -> {
        if (args.length == 1) {
          models.capture.loadCapture(new File(args[0]));
        }
      };

      window.initMainUi(client, models, widgets);
      if (models.settings.preferences().getSkipFirstRunDialog()) {
        shell.getDisplay().asyncExec(onStart);
      } else {
        shell.getDisplay().asyncExec(() -> showFirstTimeDialog(shell, models, widgets, onStart));
      }

      // Add the links on Loading Screen after the server set up.
      window.updateLoadingScreen(client, models, widgets);
    }

    @Override
    public void onStatus(String message) {
      scheduleIfStillOpen(shell -> window.showLoadingMessage(message));
    }

    @Override
    public void onServerExit(int code, String panic) {
      scheduleIfStillOpen(shell -> showErrorDialogWithTwoButtons(
          shell, getAnalytics(), String.format(Messages.SERVER_ERROR_MESSAGE, code), panic,
          IDialogConstants.RETRY_ID, "Restart Server", this::restartServer,
          IDialogConstants.CLOSE_ID, "Exit", window::close)
      );
    }

    private void restartServer() {
      try {
        server = new Server(settings, client);
        server.connect(this);
        scheduleIfStillOpen(this::restartUi);
      } catch (GapisInitException e) {
        onServerExit(-42, Throwables.getStackTraceAsString(e));
      }
    }

    private void restartUi(Shell shell) {
      models.reset();
      window.showWelcomeScreen();
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

    private boolean shouldShowUIErrorDialog(Throwable throwable) {
      // TODO b/178397207: Disable error dialog for a known UI issue, while waiting solution from the SWT side.
      if (OS.isMac && throwable != null && throwable.getStackTrace().length > 0
          && "org.eclipse.swt.widgets.Widget".equals(throwable.getStackTrace()[0].getClassName())
          && "drawRect".equals(throwable.getStackTrace()[0].getMethodName())) {
        return false;
      }
      return true;
    }

    private static interface ShellRunnable {
      public void run(Shell shell);
    }
  }

  private static final Flag<?>[] ALL_FLAGS = {
    Flags.help,
    Flags.fullHelp,
    Flags.version,
    Devices.skipDeviceValidation,
    Experimental.enableAll,
    Experimental.enableVulkanTracing,
    Experimental.enableAngleTracing,
    Experimental.enablePerfTab,
    Experimental.enableProfileExperiments,
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
    PanelCanvas.showRedraws,
    TracerDialog.maxFrames,
    TracerDialog.maxPerfetto,
    TracerDialog.enableLoadValidationLayer,
  };
}

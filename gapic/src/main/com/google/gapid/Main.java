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
import static com.google.gapid.views.WelcomeDialog.showWelcomeDialog;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;

import com.google.gapid.models.Analytics;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.server.Client;
import com.google.gapid.server.GapiPaths;
import com.google.gapid.server.GapisProcess;
import com.google.gapid.util.Crash2ExceptionHandler;
import com.google.gapid.util.ExceptionHandler;
import com.google.gapid.util.Flags;
import com.google.gapid.util.Flags.Flag;
import com.google.gapid.util.Logging;
import com.google.gapid.util.Messages;
import com.google.gapid.util.Scheduler;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.window.ApplicationWindow;
import org.eclipse.jface.window.Window;
import org.eclipse.swt.widgets.Display;
import org.eclipse.swt.widgets.Shell;

import java.io.File;
import java.util.concurrent.atomic.AtomicReference;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Main entry point of the application.
 */
public class Main {
  public static void main(String[] args) throws Exception {
    args = Flags.initFlags(ALL_FLAGS, args);
    Logging.init();

    Display.setAppName(Messages.WINDOW_TITLE);
    Display.setAppVersion(GAPID_VERSION.toString());
    Settings settings = Settings.load();
    ExceptionHandler handler = Crash2ExceptionHandler.register(settings);

    Server server = new Server(settings);
    AtomicReference<UI> uiRef = new AtomicReference<UI>(null);
    try {
      server.connect((code, panic) -> {
        UI ui = uiRef.get();
        if (ui != null) {
          ui.showServerDiedMessage(code, panic);
        }
      });
      uiRef.set(new UI(settings, handler, server.getClient(), args));
      uiRef.get().show();
    } finally {
      uiRef.set(null);
      server.disconnect();
      Scheduler.EXECUTOR.shutdownNow();
    }
  }

  /**
   * Manages the main UI.
   */
  private static class UI implements MainWindow.ModelsAndWidgets {
    protected static final Logger LOG = Logger.getLogger(UI.class.getName());

    private final Settings settings;
    private final ExceptionHandler handler;
    private final Client client;
    private final String[] args;
    private final ApplicationWindow window;
    private Models models;
    private Widgets widgets;

    public UI(Settings settings, ExceptionHandler handler, Client client, String[] args) {
      this.settings = settings;
      this.handler = handler;
      this.client = client;
      this.args = args;
      this.window = new MainWindow(client, this);

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
      window.open();
    }

    public void showServerDiedMessage(int code, String panic) {
      Shell shell = window.getShell();
      if (shell == null) {
        return;
      }

      scheduleIfNotDisposed(shell, () -> {
        // TODO: try to restart the server?
        showErrorDialog(shell, getAnalytics(),
            "The gapis server has exited with an error code of: " + code, panic);
      });
    }

    @Override
    public void init(Shell shell) {
      models = Models.create(shell, settings, handler, client);
      widgets = Widgets.create(shell.getDisplay(), client, models);

      if (models.settings.skipFirstRunDialog) {
        shell.getDisplay().asyncExec(this::startUp);
      } else {
        shell.getDisplay().asyncExec(
            () -> showFirstTimeDialog(shell, models, widgets, this::startUp));
      }
    }

    private void startUp() {
      if (args.length == 1) {
        models.capture.loadCapture(new File(args[0]));
      } else if (!models.settings.skipWelcomeScreen) {
        showWelcomeDialog(client, window.getShell(), models, widgets);
      }
    }

    @Override
    public Models models() {
      return models;
    }

    @Override
    public Widgets widgets() {
      return widgets;
    }

    @Override
    public void dispose() {
      if (widgets != null) {
        widgets.dispose();
        models.dispose();
      }

      models = null;
      widgets = null;
    }

    private Analytics getAnalytics() {
      return (models == null) ? null : models.analytics;
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
  };
}

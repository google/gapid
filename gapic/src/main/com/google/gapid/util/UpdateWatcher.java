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
package com.google.gapid.util;

import static com.google.gapid.util.MoreFutures.logFailureIgnoringCancel;
import static com.google.gapid.util.Scheduler.EXECUTOR;
import static com.google.gapid.views.ErrorDialog.showErrorDialog;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static java.util.concurrent.TimeUnit.MILLISECONDS;
import static org.eclipse.jface.dialogs.MessageDialog.openInformation;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.SettingsProto;
import com.google.gapid.proto.service.Service;
import com.google.gapid.server.Client;
import com.google.gapid.views.StatusBar;

import org.eclipse.jface.dialogs.MessageDialog;
import org.eclipse.swt.program.Program;
import org.eclipse.swt.widgets.Shell;

import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.logging.Logger;

/**
 * Utility class for checking for new releases of GAPID.
 */
public class UpdateWatcher {
  private static final Logger LOG = Logger.getLogger(UpdateWatcher.class.getName());

  private static final long CHECK_INTERVAL_MS = TimeUnit.HOURS.toMillis(4);

  private final Settings settings;
  private final Client client;
  private final StatusBar statusBar;
  private final AtomicBoolean scheduled = new AtomicBoolean(false);
  private ListenableFuture<?> scheduledCheck;

  public UpdateWatcher(Settings settings, Client client, StatusBar statusBar) {
    this.settings = settings;
    this.client = client;
    this.statusBar = statusBar;
  }

  public void watchForUpdates() {
    if (!scheduled.getAndSet(true)) {
      scheduleCheck(Listener.NULL_LISTENER, settings.preferences().getUpdateAvailable());
    }
  }

  public void checkNow(Listener listener) {
    if (scheduled.getAndSet(true)) {
      scheduledCheck.cancel(false);
    }
    scheduleCheck(listener, true);
  }

  public static void manualUpdateCheck(Shell shell, Models models) {
    models.updateWatcher.checkNow(new Listener() {
      @Override
      public boolean forceCheck() {
        return true;
      }

      @Override
      public void onCompleted(Service.Releases releases) {
        scheduleIfNotDisposed(shell, () -> {
          if (GapidVersion.GAPID_VERSION.isOlderThan(releases.getAGI())) {
            MessageDialog dialog = new MessageDialog(shell, Messages.UPDATE_CHECK_TITLE,
                null, Messages.UPDATE_CHECK_UPDATE, MessageDialog.INFORMATION,
                new String[] { "Download", "Ignore" }, 0);
            if (dialog.open() == 0) {
              // Download was clicked.
              Program.launch(releases.getAGI().getBrowserUrl());
            }
          } else {
            openInformation(shell, Messages.UPDATE_CHECK_TITLE, Messages.UPDATE_CHECK_NO_UPDATE);
          }
        });
      }

      @Override
      public void onFailure(Throwable t) {
        scheduleIfNotDisposed(shell, () ->
        showErrorDialog(shell, models.analytics, "Failed to check for updates", t));
      }
    });
  }

  private void scheduleCheck(Listener listener, boolean immediate) {
    long delay = 0;
    if (!immediate) {
      long now = System.currentTimeMillis();
      long timeSinceLastUpdateMS = now - settings.preferences().getLastCheckForUpdates();
      delay = Math.max(CHECK_INTERVAL_MS - timeSinceLastUpdateMS, 0);
    }
    scheduledCheck = logFailureIgnoringCancel(
        LOG, EXECUTOR.schedule(() -> doCheck(listener), delay, MILLISECONDS));
  }

  private void doCheck(Listener listener) {
    SettingsProto.Preferences.Builder prefs = settings.writePreferences();
    if (listener.forceCheck() || prefs.getCheckForUpdates()) {
      ListenableFuture<Service.Releases> future =
          client.checkForUpdates(prefs.getIncludeDevReleases());
      prefs.setUpdateAvailable(false);
      try {
        Service.Releases releases = future.get();
        if (GapidVersion.GAPID_VERSION.isOlderThan(releases.getAGI())) {
          prefs.setUpdateAvailable(true);
          onNewReleaseAvailable(releases.getAGI());
        }
        prefs.setLatestAngleRelease(releases.getANGLE());
        listener.onCompleted(releases);
      } catch (InterruptedException e) {
        /* never mind */
      } catch (ExecutionException e) {
        listener.onFailure(e.getCause());
      }
    }
    prefs.setLastCheckForUpdates(System.currentTimeMillis());
    settings.save();
    scheduleCheck(Listener.NULL_LISTENER, false);
  }

  private void onNewReleaseAvailable(Service.Releases.AGIRelease release) {
    scheduleIfNotDisposed(statusBar, () -> {
      statusBar.setNotification("New update available", () -> {
        Program.launch(release.getBrowserUrl());
      });
    });
  }

  /**
   * Callback interface for manual update checks.
   */
  @SuppressWarnings("unused")
  public static interface Listener {
    public static final Listener NULL_LISTENER = new Listener() { /* empty */ };

    public default boolean forceCheck() { return false; }
    public default void onCompleted(Service.Releases releases) { /* do nothing */ }
    public default void onFailure(Throwable t) { /* do nothing */ }
  }
}

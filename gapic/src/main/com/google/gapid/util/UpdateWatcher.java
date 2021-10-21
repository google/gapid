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
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static java.util.concurrent.TimeUnit.MILLISECONDS;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.SettingsProto;
import com.google.gapid.proto.service.Service;
import com.google.gapid.server.Client;
import com.google.gapid.views.StatusBar;

import org.eclipse.swt.program.Program;

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
      scheduleCheck(settings.preferences().getUpdateAvailable());
    }
  }

  public void checkNow() {
    if (scheduled.getAndSet(true)) {
      scheduledCheck.cancel(false);
    }
    scheduleCheck(true);
  }

  private void scheduleCheck(boolean immediate) {
    long delay = 0;
    if (!immediate) {
      long now = System.currentTimeMillis();
      long timeSinceLastUpdateMS = now - settings.preferences().getLastCheckForUpdates();
      delay = Math.max(CHECK_INTERVAL_MS - timeSinceLastUpdateMS, 0);
    }
    scheduledCheck =
        logFailureIgnoringCancel(LOG, EXECUTOR.schedule(this::doCheck, delay, MILLISECONDS));
  }

  private void doCheck() {
    SettingsProto.Preferences.Builder prefs = settings.writePreferences();
    if (prefs.getCheckForUpdates()) {
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
      } catch (InterruptedException | ExecutionException e) {
        /* never mind */
      }
    }
    prefs.setLastCheckForUpdates(System.currentTimeMillis());
    settings.save();
    scheduleCheck(false);
  }

  private void onNewReleaseAvailable(Service.Releases.AGIRelease release) {
    scheduleIfNotDisposed(statusBar, () -> {
      statusBar.setNotification("New update available", () -> {
        Program.launch(release.getBrowserUrl());
      });
    });
  }
}

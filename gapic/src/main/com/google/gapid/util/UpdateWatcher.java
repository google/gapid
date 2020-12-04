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

import static com.google.gapid.util.MoreFutures.logFailure;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.SettingsProto;
import com.google.gapid.proto.service.Service;
import com.google.gapid.server.Client;

import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeUnit;
import java.util.logging.Logger;

/**
 * Utility class for checking for new releases of GAPID.
 */
public class UpdateWatcher {
  private static final Logger LOG = Logger.getLogger(UpdateWatcher.class.getName());

  private static final long CHECK_INTERVAL_MS = TimeUnit.HOURS.toMillis(4);

  private final Settings settings;
  private final Client client;
  private final Listener listener;

  /** Callback interface */
  public interface Listener {
    /** Called whenever a new release is found. */
    void onNewReleaseAvailable(Service.Releases.AGIRelease release);
  }

  public UpdateWatcher(Settings settings, Client client, Listener listener) {
    this.settings = settings;
    this.client = client;
    this.listener = listener;
    if (settings.preferences().getUpdateAvailable()) {
      logFailure(LOG, Scheduler.EXECUTOR.schedule(this::doCheck, 0, TimeUnit.MILLISECONDS));
    } else {
      scheduleCheck();
    }
  }

  private void scheduleCheck() {
    long now = System.currentTimeMillis();
    long timeSinceLastUpdateMS = now - settings.preferences().getLastCheckForUpdates();
    long delay = Math.max(CHECK_INTERVAL_MS - timeSinceLastUpdateMS, 0);
    logFailure(LOG, Scheduler.EXECUTOR.schedule(this::doCheck, delay, TimeUnit.MILLISECONDS));
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
          listener.onNewReleaseAvailable(releases.getAGI());
        }
      } catch (InterruptedException | ExecutionException e) {
        /* never mind */
      }
    }
    prefs.setLastCheckForUpdates(System.currentTimeMillis());
    settings.save();
    scheduleCheck();
  }
}

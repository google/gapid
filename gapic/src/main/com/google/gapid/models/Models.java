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
package com.google.gapid.models;

import com.google.gapid.server.Client;

import org.eclipse.swt.widgets.Shell;

public class Models {
  public final Settings settings;
  public final Follower follower;
  public final Capture capture;
  public final Devices devices;
  public final AtomStream atoms;
  public final ApiContext contexts;
  public final AtomHierarchies hierarchies;
  public final Resources resources;
  public final ApiState state;
  public final Reports reports;
  public final Thumbnails thumbs;

  public Models(Settings settings, Follower follower, Capture capture, Devices devices,
      AtomStream atoms, ApiContext contexts, AtomHierarchies hierarchies, Resources resources,
      ApiState state, Reports reports, Thumbnails thumbs) {
    this.settings = settings;
    this.follower = follower;
    this.capture = capture;
    this.devices = devices;
    this.atoms = atoms;
    this.contexts = contexts;
    this.hierarchies = hierarchies;
    this.resources = resources;
    this.state = state;
    this.reports = reports;
    this.thumbs = thumbs;
  }

  public static Models create(Shell shell, Client client) {
    Settings settings = Settings.load();
    Follower follower = new Follower(shell, client);
    Capture capture = new Capture(shell, client, settings);
    Devices devices = new Devices(shell, client, capture);
    ApiContext contexts = new ApiContext(shell, client, capture);
    AtomStream atoms = new AtomStream(shell, client, capture, contexts);
    AtomHierarchies hierarchies = new AtomHierarchies(shell, client, capture);
    Resources resources = new Resources(shell, client, capture);
    ApiState state = new ApiState(shell, client, follower, atoms);
    Reports reports = new Reports(shell, client, devices, capture);
    Thumbnails thumbs = new Thumbnails(client, devices, atoms);
    return new Models(settings, follower, capture, devices, atoms, contexts, hierarchies, resources,
        state, reports, thumbs);
  }

  public void dispose() {
    settings.save();
  }
}

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
import com.google.gapid.util.ExceptionHandler;

import org.eclipse.swt.widgets.Shell;

public class Models {
  public final Settings settings;
  public final Analytics analytics;
  public final Follower follower;
  public final Capture capture;
  public final Devices devices;
  public final CommandStream commands;
  public final ApiContext contexts;
  public final Timeline timeline;
  public final Resources resources;
  public final ApiState state;
  public final Reports reports;
  public final Thumbnails thumbs;
  public final ConstantSets constants;
  public final Geometries geos;

  public Models(Settings settings, Analytics analytics, Follower follower, Capture capture,
      Devices devices, CommandStream commands, ApiContext contexts, Timeline timeline,
      Resources resources, ApiState state, Reports reports, Thumbnails thumbs,
      ConstantSets constants, Geometries geos) {
    this.settings = settings;
    this.analytics = analytics;
    this.follower = follower;
    this.capture = capture;
    this.devices = devices;
    this.commands = commands;
    this.contexts = contexts;
    this.timeline = timeline;
    this.resources = resources;
    this.state = state;
    this.reports = reports;
    this.thumbs = thumbs;
    this.constants = constants;
    this.geos = geos;
  }

  public static Models create(
      Shell shell, Settings settings, ExceptionHandler handler, Client client) {
    Analytics analytics = new Analytics(client, settings, handler);
    ConstantSets constants = new ConstantSets(client);
    Follower follower = new Follower(shell, client);
    Capture capture = new Capture(shell, analytics, client, settings);
    Devices devices = new Devices(shell, analytics, client, capture);
    ApiContext contexts = new ApiContext(shell, analytics, client, capture);
    Timeline timeline = new Timeline(shell, analytics, client, capture, contexts);
    CommandStream commands = new CommandStream(shell, analytics, client, capture, contexts, constants);
    Resources resources = new Resources(shell, analytics, client, capture, commands);
    ApiState state = new ApiState(shell, analytics, client, follower, commands, contexts, constants);
    Reports reports = new Reports(shell, analytics, client, capture, devices, contexts);
    Thumbnails thumbs = new Thumbnails(client, devices, capture, settings);
    Geometries geometries = new Geometries(shell, analytics, client, commands);
    return new Models(settings, analytics, follower, capture, devices, commands, contexts, timeline,
        resources, state, reports, thumbs, constants, geometries);
  }

  public void dispose() {
    settings.save();
  }
}

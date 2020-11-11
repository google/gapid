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
import com.google.gapid.views.StatusBar;

import org.eclipse.swt.widgets.Shell;

public class Models {
  public final Settings settings;
  public final Analytics analytics;
  public final Follower follower;
  public final Capture capture;
  public final Devices devices;
  public final CommandStream commands;
  public final Resources resources;
  public final ApiState state;
  public final Reports reports;
  public final ImagesModel images;
  public final ConstantSets constants;
  public final Geometries geos;
  public final Memory memory;
  public final MemoryTypes types;
  public final Perfetto perfetto;
  public final Profile profile;
  public final StatusBar status; // The "model" part of this "widget".

  public Models(Settings settings, Analytics analytics, Follower follower, Capture capture,
      Devices devices, CommandStream commands, Resources resources,
      ApiState state, Reports reports, ImagesModel images, ConstantSets constants, Geometries geos,
      Memory memory, MemoryTypes types, Perfetto perfetto, Profile profile, StatusBar status) {
    this.settings = settings;
    this.analytics = analytics;
    this.follower = follower;
    this.capture = capture;
    this.devices = devices;
    this.commands = commands;
    this.resources = resources;
    this.state = state;
    this.reports = reports;
    this.images = images;
    this.constants = constants;
    this.geos = geos;
    this.memory = memory;
    this.types = types;
    this.perfetto = perfetto;
    this.profile = profile;
    this.status = status;
  }

  public static Models create(
      Shell shell, Settings settings, ExceptionHandler handler, Client client, StatusBar status) {
    Analytics analytics = new Analytics(client, settings, handler);
    Capture capture = new Capture(shell, analytics, client, settings);
    Devices devices = new Devices(shell, analytics, client, capture, settings);
    ConstantSets constants = new ConstantSets(client, devices);
    CommandStream commands = new CommandStream(
        shell, analytics, client, capture, devices, constants);
    Follower follower = new Follower(shell, client);
    Resources resources = new Resources(
        shell, analytics, client, capture, devices, commands, follower);
    ApiState state = new ApiState(
        shell, analytics, client, devices, follower, commands, constants);
    Reports reports = new Reports(shell, analytics, client, capture, devices);
    ImagesModel images = new ImagesModel(client, devices, capture, settings);
    Geometries geometries = new Geometries(shell, analytics, client, devices, commands);
    Memory memory = new Memory(shell, analytics, client, devices, commands);
    MemoryTypes types = new MemoryTypes(client, devices, constants);
    Perfetto perfetto = new Perfetto(shell, analytics, client, capture, status);
    Profile profile = new Profile(shell, analytics, client, capture, devices);
    return new Models(settings, analytics, follower, capture, devices, commands, resources, state,
        reports, images, constants, geometries, memory, types, perfetto, profile, status);
  }

  public void reset() {
    capture.reset();
    devices.resetReplayDevice();
    commands.reset();
    resources.reset();
    state.reset();
    reports.reset();
    geos.reset();
    memory.reset();
    perfetto.reset();
    profile.reset();
  }

  public void dispose() {
    settings.save();
  }
}

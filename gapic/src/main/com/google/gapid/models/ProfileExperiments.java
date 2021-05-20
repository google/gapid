/*
 * Copyright (C) 2021 Google Inc.
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

import com.google.common.collect.ImmutableList;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

public final class ProfileExperiments {
  public final boolean disableAnisotropicFiltering;
  public final ImmutableList<Path.Command> disabledCommands;

  public ProfileExperiments() {
    this(false, Collections.emptyList());
  }

  public ProfileExperiments(boolean disableAnisotropicFiltering, List<Path.Command> disabledCommands) {
    this.disableAnisotropicFiltering = disableAnisotropicFiltering;
    this.disabledCommands = ImmutableList.copyOf(disabledCommands);
  }

  public final Service.ProfileExperiments toProto() {
    return Service.ProfileExperiments.newBuilder()
      .setDisableAnisotropicFiltering(disableAnisotropicFiltering)
      .addAllDisabledCommands(disabledCommands)
      .build();
  }
}

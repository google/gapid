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

import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.api.API;

/**
 * Utility functions to deal with {@code Service.Value} protos.
 */
public class Values {
  private Values() {
  }

  public static Service.Value value(API.Command command) {
    return Service.Value.newBuilder()
        .setCommand(command)
        .build();
  }

  public static byte[] getBytes(Service.Value value) {
    switch (value.getValCase()) {
      case BOX: return Boxes.getBytes(value.getBox());
      default:
        throw new RuntimeException("Don't know how to get bytes out of " + value.getValCase());
    }
  }
}

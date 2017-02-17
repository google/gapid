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
package com.google.gapid.rpclib.binary;

import com.google.gapid.rpclib.schema.Entity;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 *
 */
public class Namespace {
  private static Map<String, BinaryClass> registry = new ConcurrentHashMap<String, BinaryClass>(300);

  public static void register(BinaryClass creator) {
    registry.put(creator.entity().signature(), creator);
  }

  public static BinaryClass lookup(Entity entity) {
    return registry.get(entity.signature());
  }
}

/*
 * Copyright (C) 2018 Google Inc.
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
"use strict";

// The devices module provides access to the devices stash.
define(["queriable"],
function(queriable) {
  class Device extends queriable.Base {
    constructor(json) {
      super(json);
    }

    get info_() {
      return this.json_.information || {};
    }

    get name() {
      return this.info_.Name || "";
    }
  }

  return queriable.new("/devices/", Device);
});

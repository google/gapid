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

// The packages module provides access to the packages stash.
define(["queriable", "xhr"],
function(queriable, xhr) {
  class Package extends queriable.Base {
    constructor(json) {
      super(json);
    }

    get info_() {
      return this.json_.information || {};
    }

    get sha() {
      return this.info_.cl || "";
    }

    get description() {
      return this.info_.description || "";
    }
  }

  var pkgs = queriable.new("/packages/", Package)
  return Object.assign(pkgs, {
    // Returns the package chain rooted at the given package (i.e. all the ancestors).
    getChain: (pkgId) => xhr.getJson("/packageChain/?head=" + encodeURIComponent(pkgId)).then(pkgs.toObjs_),
  });
});

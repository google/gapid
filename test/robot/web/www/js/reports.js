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

// The reports module provides access to the reports stash.
define(["actions", "queriable"],
function(actions, queriable) {
  class Report extends actions.Action {
    constructor(json) {
      super(json);
    }
  }

  var q = queriable.new("/reports/", Report);
  return Object.assign(q, {
    getByPackageAndTrace: (pkg, trace) =>
        q.query("Input.Package == \"" + pkg + "\" and Input.Trace == \"" + trace + "\"").then(q.onlyOne_),
  });
});

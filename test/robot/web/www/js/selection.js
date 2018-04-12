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

// The selection module provides a singleton to handle the current user
// selection in the UI and updates and monitors the URL's hash.
define(function() {
  class Selection {
    constructor() {
      this.track_ = "";
      this.trackDefault = "";
      this.pkg_ = "";
      this.pkgDefault = "";

      this.listeners_ = [];
    }

    get track() {
      return this.track_ || this.trackDefault;
    }

    set track(track) {
      this.track_ = track;
    }

    get pkg() {
      return this.pkg_ || this.pkgDefault;
    }

    set pkg(pkg) {
      this.pkg_ = pkg;
    }

    // Causes the hash of the URL to change and notifies listeners of changes.
    commit() {
      var hash = [], sel = this;
      function value(name) {
        var val = sel[name + "_"], dflt = sel[name + "Default"];
        if (val && val != dflt) {
          hash.push(name + "=" + encodeURIComponent(val));
        }
      }
      value("track");
      value("pkg");

      location.hash = hash.join("@");
      this.notify_();
    }

    // Calls the provided callback where changes to the selection should be made
    // and then commits these changes.
    update(f) {
      f(this);
      this.commit();
    }

    // Registers the provided callback as a listener.
    listen(f) {
      this.listeners_.push(f);
    }

    notify_() {
      this.listeners_.forEach(l => l(this));
    }
  }

  var sel = new Selection();
  function parse() {
    var params = {};
    location.hash.substr(1).split("@").forEach(kv => {
      kv = kv.split("=", 2);
      if (kv.length != 2) {
        return;
      }
      params[kv[0]] = decodeURIComponent(kv[1]);
    });

    var changed = false;
    function update(name) {
      var value = params[name] || "";
      changed = changed || sel[name] != value;
      sel[name] = value;
    }

    update("track");
    update("pkg");

    return changed;
  };
  parse();

  window.addEventListener("hashchange", function() {
    if (parse()) {
      sel.notify_();
    }
  });

  return sel;
});

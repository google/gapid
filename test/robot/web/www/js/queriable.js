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

// The queriable module provides a base class and constructor that can be used
// by modules providing access to the various queriable stashes.
define(["xhr"],
function(xhr) {
  class Base {
    constructor(json) {
      this.json_ = json;
    }

    get id() {
      return this.json_.id || "";
    }
  }

  function build(url, cls) {
    function toObjs(vs) {
      // The server returns "null" if nothing is found.
      return vs && vs.map(json => new cls(json)) || [];
    }

    function onlyOne(vs) {
      return vs && vs.length == 1 ? vs[0] : null;
    }

    function query(q) {
      return xhr.getJson(url + "?q=" + encodeURIComponent(q)).then(toObjs);
    }

    return {
      // Returns all items in this shelf.
      getAll: () => xhr.getJson(url).then(toObjs),
      // Returns the item with the given id in this shelf.
      get: (id) => query("Id == \"" + id + "\"").then(onlyOne),
      // Returns the items that match the given query in this shelf.
      query: query,

      url_: url,
      toObjs_: toObjs,
      onlyOne_: onlyOne,
    }
  }

  return {
    Base: Base,
    // Creates a new module using the given URL and class. Use this as the
    // return from your module.
    new: build,
  }
});

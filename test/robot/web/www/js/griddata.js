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

// The griddata module handles loading the grid action status data.
define(["xhr"],
function(xhr) {
  class Cell {
    constructor(json) {
      this.json_ = json;
    }

    get scheduled() {
      return this.json_.Scheduled || 0;
    }

    get running() {
      return this.json_.Running || 0;
    }

    get succeeded() {
      return this.json_.Succeeded || 0;
    }

    get failed() {
      return this.json_.Failed || 0;
    }

    get total() {
      return this.scheduled + this.running + this.succeeded + this.failed;
    }

    add_(c) {
      this.json_.Scheduled = this.scheduled + c.scheduled;
      this.json_.Running = this.running + c.running;
      this.json_.Succeeded = this.succeeded + c.succeeded;
      this.json_.Failed = this.failed + c.failed;
      return this;
    }
  }

  class Row {
    constructor(json) {
      this.json_ = json;
    }

    get id() {
      return this.json_.ID || "";
    }

    cell(i) {
      var cs = this.json_.Cells || [];
      return new Cell(cs[i] || {});
    }

    get summary() {
      var cs = this.json_.Cells || [];
      return cs.reduce((a, b) => a.add_(new Cell(b)), new Cell({}));
    }
  }

  class Grid {
    constructor(json) {
      this.json_ = json;
      this.columns_ = {};
      this.json_.Columns && this.json_.Columns.forEach((name, idx) => this.columns_[name] = idx);
    }

    columnIdx(name) {
      var r = this.columns_[name];
      return r == undefined ? -1 : r;
    }

    forEachRow(f) {
      this.json_.Rows && this.json_.Rows.forEach((r, idx) => f(new Row(r), idx));
    }
  }

  function get(pkg) {
    return xhr.getJson("/gridData/?pkg=" + encodeURIComponent(pkg)).then(json => new Grid(json || {}));
  }

  return {
    get: get,
    emptyCell: () => new Cell({}),
  }
});

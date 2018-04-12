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

// This is the main entry point for the Robot grid UI.

requirejs.config({
    baseUrl: "js",
});

require([
  "draw",
  "griddata",
  "packages",
  "replays",
  "reports",
  "selection",
  "subjects",
  "traces",
  "tracks",
  "viewer",
],
function(
  draw,
  griddata,
  packages,
  replays,
  reports,
  selection,
  subjects,
  traces,
  tracks,
  viewer,
) {
  function main() {
    registerEventHandlers();
    tracks.getAll()
      .then(ts => updateTracks(ts))
      .then(loadPackages)
      .then(loadGrid);
    subjects.getAll().then(updateSubjects);
  }

  // Registers the DOM and selection event handlers.
  function registerEventHandlers() {
    $("#tracks").change(function() {
      selection.update(s => s.track = this.value);
    });
    $("#packages").change(function() {
      selection.update(s => s.pkg = this.value);
    });

    function updateValue(el, value, dflt) {
      var old = el.data("loaded");
      var now = el.val(value).val();
      if (now == null) {
        now = el.val(dflt).val();
      }
      return old != now;
    }

    selection.listen(() => {
      if (updateValue($("#tracks"), selection.track, selection.trackDefault)) {
        loadPackages();
      }
      if (updateValue($("#packages"), selection.pkg, selection.pkgDefault)) {
        loadGrid();
      }
    });
  }

  // Updates the track selection UI with the loaded tracks
  function updateTracks(ts) {
    var sel = $("#tracks");
    sel.empty();
    selection.trackDefault = "";
    ts.forEach(track => {
      var val = track.id;
      if (track.name == "master") {
        selection.trackDefault = val;
      }
      sel.append($("<option>").attr("value", val).text(track.name));
    });
    if (sel.val(selection.track).val() != selection.track) {
      selection.update((s) => s.track = "");
    }
  }

  // Loads the packages of the currently selected track.
  async function loadPackages() {
    var trackId = selection.track;
    $("#tracks").data("loaded", trackId);
    var track = await tracks.get(trackId);
    return packages.getChain(track.head).then(ps => updatePackages(ps, track.head));
  }

  // Updats the package selection UI with the loaded packages.
  function updatePackages(ps, head) {
    var sel = $("#packages");
    sel.empty();
    selection.pkgDefault = head;
    ps.forEach(pkg => {
      sel.append($("<option>").attr("value", pkg.id).text(pkg.sha));
    });
    if (sel.val(selection.pkg).val() != selection.pkg) {
      selection.update((s) => s.pkg = "");
    }
  }

  // Updates the subjects display in the grid with the loaded subject data.
  function updateSubjects(subjs) {
    subjs.forEach(subj => {
      var row = getSubjectRow(subj.id);
      row.children().first().text(subj.name);
    });
  }

  // Loads the grid based on the current selection.
  function loadGrid() {
    var pkg = selection.pkg;
    $("#packages").data("loaded", pkg);
    griddata.get(pkg).then(grid => updateGrid(grid));
  }

  // Updtes the grid UI with the loaded data.
  function updateGrid(grid) {
    var traceIdx = grid.columnIdx("trace"), reportIdx = grid.columnIdx("report"), replayIdx = grid.columnIdx("replay");
    var processed = {};
    grid.forEachRow(r => {
      processed[r.id] = true;
      var row = getSubjectRow(r.id);
      updateCell(row.children().eq(1), r.summary);
      updateCell(row.children().eq(2), r.cell(replayIdx));
      updateCell(row.children().eq(3), r.cell(reportIdx));
      updateCell(row.children().eq(4), r.cell(traceIdx));
    });

    $("#grid").find("tr").each((idx, rowEl) => {
      var row = $(rowEl), id = row.data("subject");
      if (id && !processed[id]) {
        updateCell(row.children().eq(1), griddata.emptyCell());
        updateCell(row.children().eq(2), griddata.emptyCell());
        updateCell(row.children().eq(3), griddata.emptyCell());
        updateCell(row.children().eq(4), griddata.emptyCell());
      }
    });
  }

  function updateCell(cell, data) {
    draw.cell(cell, data);
  }

  // Gets or creates a row in the grid for the given subject.
  function getSubjectRow(subjId) {
    var row = $("#grid").find("[data-subject='" + subjId + "']");
    if (!row.length) {
      row = $("<tr>").attr("data-subject", subjId).appendTo(grid);
      row.append($("<td>")
          .addClass("subject"));
      for (var i = 0; i < 4; i++) {
        row.append($("<td>")
            .addClass("cell")
            .addClass((i == 0) ? "summary" : "clickable")
            .append($("<canvas>")
                .attr("width", 48)
                .attr("height", 48)
                .detectPixelRatio()));
      }
      row.children().eq(2).click(() => showReplays(row));
      row.children().eq(3).click(() => showReports(row));
      row.children().eq(4).click(() => showTraces(row));
    }
    return row;
  }

  async function showReplays(row) {
    var ts = await traces.getByPackageAndSubject(selection.pkg, row.data("subject"));
    viewer.show(ts, t => replays.getByPackageAndTrace(selection.pkg, t.file).then(r => r.json_));
  }

  async function showReports(row) {
    var ts = await traces.getByPackageAndSubject(selection.pkg, row.data("subject"));
    viewer.show(ts, t => reports.getByPackageAndTrace(selection.pkg, t.file).then(r => r.json_));
  }

  async function showTraces(row) {
    var ts = await traces.getByPackageAndSubject(selection.pkg, row.data("subject"));
    viewer.show(ts, t => Promise.resolve(t.json_));
  }

  main();
});

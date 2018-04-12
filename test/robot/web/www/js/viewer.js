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

// The viewer module provides functions to display a robot Object.
define(["devices", "draw"],
function(devices, draw) {
  function linkify(td, val) {
    if (/^[0-9a-fA-F]{40}/.test(val)) {
      td.append($("<a>")
          .attr("href", "/entities/" + val)
          .attr("target", "_blank")
          .text(val));
    } else {
      td.text(val);
    }
    return td;
  }

  function genDom(root, obj, expand) {
    var table = $("<table>")
    for (var k in obj) {
      var row = $("<tr>").append($("<td>").text(k + ":"));
      var v = obj[k];
      switch (typeof(v)) {
        case "boolean":
        case "number":
          row.append($("<td>").text(v));
          break;
        case "string":
          row.append(linkify($("<td>"), v));
          break;
        default:
          (function() {
            var td = $("<td>");
            td.append(draw.icon(expand > 0 ? "expand_less" : "expand_more")
                .click(function() {
                  var el = td.children().eq(1), vis = el.css("display") == "none";
                  el.css("display", vis ? "" : "none");
                  $(this).text(vis ? "expand_less" : "expand_more");
                }));
            row.append(genDom(td, v, expand - 1));
            td.children().eq(1).css("display", expand > 0 ? "" : "none");
          })();
      }
      table.append(row);
    }
    return root.append(table);
  }

  function showDropDown(traces, loader) {
    var v = $("#viewer").empty();
    if (!traces || !traces.length) {
      v.text("no traces");
      return
    }

    var dd = $("<select>");
    var traceById = {};
    traces.forEach(trace => {
      var opt = $("<option>")
          .attr("value", trace.id)
          .text("loading...");
      devices.get(trace.target).then(d => opt.text(d.name));
      dd.append(opt);
      traceById[trace.id] = trace;
    })
    v.append(dd);

    var content = $("<div>");
    v.append(content);

    function load(traceId) {
      loader(traceById[traceId]).then(val => {
        if (dd.val() == traceId) {
          content.empty();
          genDom(content, val, 1);
        }
      });
    }
    load(dd.val());
    dd.change(() => load(dd.val()));
  }

  return {
    show: showDropDown,
  }
});

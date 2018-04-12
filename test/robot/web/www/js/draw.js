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

// The draw module provides functions to draw the grid cells.
define([],
function() {
  function rgba(r, g, b, a) {
    return "rgba(" + (r * 255).toFixed(0) + "," + (g * 255).toFixed(0) + "," + (b * 255).toFixed(0) + "," + a + ")";
  }

  var BackgroundColor = "transparent";
  var CurrentSucceededBackgroundColor = rgba(0.91, 0.96, 0.91, 1.0);
  var CurrentSucceededForegroundColor = rgba(0.30, 0.69, 0.31, 0.9);
  var StaleSucceededBackgroundColor =   rgba(0.91, 0.96, 0.91, 0.3);
  var StaleSucceededForegroundColor =   rgba(0.30, 0.69, 0.31, 0.3);
  var CurrentFailedBackgroundColor =    rgba(1.00, 0.80, 0.82, 1.0);
  var CurrentFailedForegroundColor =    rgba(0.95, 0.26, 0.21, 0.9);
  var StaleFailedBackgroundColor =      rgba(1.00, 0.80, 0.82, 0.3);
  var StaleFailedForegroundColor =      rgba(0.95, 0.26, 0.21, 0.3);
  var InProgressForegroundColor =       rgba(0.00, 0.50, 1.00, 0.9);
  var RegressedForegroundColor =        rgba(1.00, 0.40, 0.41, 0.9);
  var FixedForegroundColor =            rgba(0.18, 0.85, 0.20, 0.9);
  var UnknownBackgroundColor =          rgba(1.00, 1.00, 1.00, 1.0);
  var UnknownForegroundColor =          rgba(0.60, 0.60, 0.60, 0.9);
  var StaleUnknownForegroundColor =     rgba(0.60, 0.60, 0.60, 0.3);
  var IconUnknown =   "help_outline";
  var IconSucceeded = "done";
  var IconFailed =    "close";

  function style(data) {
    var i = IconUnknown, b = BackgroundColor, f = UnknownForegroundColor;
    if (data.failed > 0) {
      i = IconFailed; b = CurrentFailedBackgroundColor; f = CurrentFailedForegroundColor;
    } else if (data.scheduled > 0 || data.running > 0) {
      i = IconSucceeded; b = StaleSucceededBackgroundColor; f = StaleSucceededForegroundColor;
    } else if (data.succeeded > 0) {
      i = IconSucceeded; b = CurrentSucceededBackgroundColor; f = CurrentSucceededForegroundColor;
    }

    return {
      icon: i,
      background: b,
      foreground: f,
    };
  }

  function drawCell(cell, data) {
    var cvs = cell.find("canvas");

    var halfWidth = cvs.width() / 2, radius = .8 * halfWidth;
    var dashLen = (2 * Math.PI * radius) / 10, countToAngle = 360 / data.total;
    var s = style(data);

    cvs.drawRect({
      x: halfWidth, y: halfWidth,
      width: 2 * halfWidth, height: 2 * halfWidth,
      fillStyle: s.background, strokeStyle: s.background,
    });

    var angle = 0;
    function drawSegment(color, dash, count) {
      if (!count) {
        return;
      }
      var step = countToAngle * count;
      cvs.drawArc({
        x: halfWidth, y: halfWidth,
        radius: radius,
        start: angle, end: angle + step,
        strokeStyle: color, strokeWidth: 5,
      });
      if (dash) {
        cvs.drawArc({
          x: halfWidth, y: halfWidth,
          radius: radius,
          start: angle, end: angle + step,
          strokeStyle: InProgressForegroundColor, strokeWidth: 3,
          strokeDash: [dashLen * .4, dashLen * .6],
        });
      }
      angle += step;
    }

    drawSegment(StaleUnknownForegroundColor, false, data.scheduled);
    drawSegment(CurrentSucceededForegroundColor, false, data.succeeded);
    drawSegment(StaleSucceededForegroundColor, true, data.scheduled);
    drawSegment(CurrentFailedForegroundColor, false, data.failed);

    cvs.drawText({
      x: halfWidth, y: halfWidth,
      fillStyle: s.foreground,
      fontFamily: "Material Icons",
      fontSize: "25px",
      text: s.icon,
    });
  }

  function icon(name) {
    return $("<i>")
        .addClass("material-icons")
        .text(name);
  }

  return {
    cell: drawCell,
    icon: icon,
  }
});

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
/*jslint white: true*/
'use strict';

var taskStats;
taskStats = function (tasks) {
  if (taskStats.initialized === undefined) {
    taskStats.CurrentSucceeded = function (t) { return t.result === "Succeeded" && t.status === "Current"; };
    taskStats.StaleSucceeded = function (t) { return t.result === "Succeeded" && t.status === "Stale"; };
    taskStats.InProgressWasSucceeded = function (t) { return t.result === "Succeeded" && t.status === "InProgress"; };
    taskStats.SucceededWasFailed = function (t) { return t.result === "Succeeded" && t.status === "Changed"; };
    taskStats.CurrentFailed = function (t) { return t.result === "Failed" && t.status === "Current"; };
    taskStats.StaleFailed = function (t) { return t.result === "Failed" && t.status === "Stale"; };
    taskStats.InProgressWasFailed = function (t) { return t.result === "Failed" && t.status === "InProgress"; };
    taskStats.FailedWasSucceeded = function (t) { return t.result === "Failed" && t.status === "Changed"; };
    taskStats.InProgressWasUnknown = function (t) { return t.result === "Unknown" && t.status === "InProgress"; };
    taskStats.StaleUnknown = function (t) { return t.result === "Unknown" && t.status === "Stale"; };
    taskStats.countif = function (tasks, pred) {
      var count = 0;
      tasks.forEach(function (task) {
        if (pred(task)) {
          count += 1;
        }
      });
      return count;
    };
    taskStats.initialized = 0;
  }
  return {
    numCurrentSucceeded: taskStats.countif(tasks, taskStats.CurrentSucceeded),
    numStaleSucceeded: taskStats.countif(tasks, taskStats.StaleSucceeded),
    numInProgressWasSucceeded: taskStats.countif(tasks, taskStats.InProgressWasSucceeded),
    numSucceededWasFailed: taskStats.countif(tasks, taskStats.SucceededWasFailed),
    numInProgressWasUnknown: taskStats.countif(tasks, taskStats.InProgressWasUnknown),
    numInProgressWasFailed: taskStats.countif(tasks, taskStats.InProgressWasFailed),
    numStaleFailed: taskStats.countif(tasks, taskStats.StaleFailed),
    numCurrentFailed: taskStats.countif(tasks, taskStats.CurrentFailed),
    numFailedWasSucceeded: taskStats.countif(tasks, taskStats.FailedWasSucceeded),
    numStaleUnknown: taskStats.countif(tasks, taskStats.StaleUnknown),
    numTasks: tasks.length
  };
};

// The grid view presents a div containing canvases that draw the state of a model in a 2D representation.
var newGridView = function () {
  var view;
  view = {
    div: document.createElement('div'),
    style: {
      gridPadding: 4,
      cellSize: 48,
      cellShadowColor: "rgba(0, 0, 0, 0.3)",
      headerFont: "16px Verdana",
      headerFontColor: "black",
      gridLineColor: "rgba(127, 127, 127)",
      gridLineWidth: 0.4,
      backgroundColor: "white",
      currentSucceededBackgroundColor: "rgba(232, 245, 232, 1)",
      staleSucceededBackgroundColor: "rgba(232, 245, 232,  0.3)",
      currentSucceededForegroundColor: "rgba(77,  176,  79, 0.9)",
      staleSucceededForegroundColor: "rgba(77,  176,  79,  0.3)",
      currentFailedBackgroundColor: "rgba(255, 204, 209, 1)",
      staleFailedBackgroundColor: "rgba(255, 204, 209,  0.3)",
      currentFailedForegroundColor: "rgba(242,  66,  54, 0.9)",
      staleFailedForegroundColor: "rgba(242,  66,  54,  0.3)",
      inProgressForegroundColor: "rgba(  0, 127, 255, 0.9)",
      regressedForegroundColor: "rgba(255, 102, 105, 0.9)",
      fixedForegroundColor: "rgba( 46, 217,  51, 0.9)",
      unknownBackgroundColor: "rgba(255, 255, 255, 1)",
      unknownForegroundColor: "rgba(153, 153, 153, 0.9)",
      staleUnknownForegroundColor: "rgba(153, 153, 153, 0.3)",
      selectedBackgroundColor: "rgb( 227, 242, 247)",
      iconsFont: "25px Material Icons",
      icons: {
        Succeeded: String.fromCharCode(parseInt("E876", 16)),
        Failed: String.fromCharCode(parseInt("E5CD", 16)),
        Unknown: String.fromCharCode(parseInt("E8FD", 16))
      }
    },
    clusterBackgroundColor: function (clusterStats) {
      if (clusterStats.numFailedWasSucceeded + clusterStats.numCurrentFailed > 0) {
        return view.style.currentFailedBackgroundColor;
      } else if (clusterStats.numSucceededWasFailed > 0) {
        return view.style.currentSucceededBackgroundColor;
      } else if (clusterStats.numInProgressWasFailed + clusterStats.numStaleFailed > 0) {
        return view.style.staleFailedBackgroundColor;
      } else if (clusterStats.numInProgressWasSucceeded + clusterStats.numStaleSucceeded > 0) {
        return view.style.staleSucceededBackgroundColor;
      } else if (clusterStats.numCurrentSucceeded > 0) {
        return view.style.currentSucceededBackgroundColor;
      } else if (clusterStats.numInprogressWasUnknown + clusterStats.numStaleUnknown > 0) {
        return view.style.unknownBackgroundColor;
      } else {
        return view.style.backgroundColor;
      }
    },
    clusterForegroundColor: function (clusterStats) {
      if (clusterStats.numFailedWasSucceeded > 0) {
        return view.style.regressedForegroundColor;
      } else if (clusterStats.numCurrentFailed > 0) {
        return view.style.currentFailedForegroundColor;
      } else if (clusterStats.numSucceededWasFailed > 0) {
        return view.style.fixedForegroundColor;
      } else if (clusterStats.numInProgressWasFailed + clusterStats.numStaleFailed > 0) {
        return view.style.staleFailedForegroundColor;
      } else if (clusterStats.numInProgressWasSucceeded + clusterStats.numStaleSucceeded > 0) {
        return view.style.staleSucceededForegroundColor;
      } else if (clusterStats.numCurrentSucceeded > 0) {
        return view.style.currentSucceededForegroundColor;
      } else {
        return view.style.unknownForegroundColor;
      }
    },
    clusterIcon: function (clusterStats) {
      if (clusterStats.numFailedWasSucceeded + clusterStats.numCurrentFailed > 0) {
        return view.style.icons.Failed;
      } else if (clusterStats.numSucceededWasFailed > 0) {
        return view.style.icons.Succeeded;
      } else if (clusterStats.numInProgressWasFailed + clusterStats.numStaleFailed > 0) {
        return view.style.icons.Failed;
      } else if (clusterStats.numInProgressWasSucceeded + clusterStats.numStaleSucceeded + clusterStats.numCurrentSucceeded > 0) {
        return view.style.icons.Succeeded;
      } else {
        return view.style.icons.Unknown;
      }
    },
    measureHeader: function (header) {
      var measure;
      var ctx = header.canvas.getContext('2d');
      ctx.save();
      ctx.font = view.style.headerFont;
      measure = ctx.measureText(header.text);
      ctx.restore();
      return Math.ceil(measure.width);
    },

    // input data from model
    dataset: {},
    resetDiv: function (newDataset, rowsWidth, columnsHeight) {
      // fastest way to clear out all canvas elements
      var cDiv = view.div.cloneNode(false);
      view.div.parentNode.replaceChild(cDiv, view.div);
      view.div = cDiv;
      view.div.style.position = "relative";

      newDataset.canvases.forEach((canvas) => view.div.appendChild(canvas));
      view.div.addEventListener("mouseleave", view.onMouseLeave);
      view.div.addEventListener("mousemove", view.onMouseMove);
      view.div.addEventListener("click", view.onClick);
      view.width = view.style.gridPadding * 2 + rowsWidth + view.style.cellSize * newDataset.columns.length;
      view.height = view.style.gridPadding * 2 + columnsHeight + view.style.cellSize * newDataset.rows.length;
      view.div.style.width = view.width + "px";
      view.div.style.height = view.height + "px";
    },
    fillEmptyCells: function (newDataset, rowsWidth, columnsHeight) {
      var i, j, index, cell;
      var x = view.style.gridPadding + rowsWidth;
      var y = view.style.gridPadding + columnsHeight;
      for (i = 0; i < newDataset.columns.length; i += 1) {
        for (j = 0; j < newDataset.rows.length; j += 1) {
          index = newDataset.cellIndex(i, j);
          if (newDataset.cells[index] == null) {
            cell = {
              canvas: document.createElement('canvas'),
              clusterStats: taskStats([]),
              rect: {
                left: x + view.style.cellSize * i,
                top: y + view.style.cellSize * j,
                width: view.style.cellSize,
                height: view.style.cellSize
              },
              dirty: true
            };
            newDataset.cells[index] = cell;
            cell.canvas.style.left = cell.rect.left;
            cell.canvas.style.top = cell.rect.top;
            cell.canvas.style.position = "absolute";
            newDataset.canvases.push(cell.canvas);
          }
        }
      }
    },
    setData: function (tasks, rowDim, columnDim, filterFn) {
      var newDataset;
      var rowTasks = {};
      var columnTasks = {};
      var cellTasks = {};
      var keyToRows = {}, keyToColumns = {};
      var maxRowWidth = 0, maxColumnHeight = 0;
      var rowsWidth, columnsHeight;
      var x, y;

      newDataset = {
        cells: [],
        rows: [],
        columns: [],
        canvases: [],
        cellIndex: function (cIndex, rIndex) {
          return (cIndex * newDataset.rows.length) + rIndex;
        }
      };

      tasks.forEach(function (task) {
        var rowKey, columnKey;
        if (filterFn(task) === false) {
          return;
        }
        rowKey = rowDim.keyOf(task);
        if (rowTasks[rowKey] == null) {
          rowTasks[rowKey] = [task];
        } else {
          rowTasks[rowKey].push(task);
        }
        columnKey = columnDim.keyOf(task);
        if (columnTasks[columnKey] == null) {
          columnTasks[columnKey] = [task];
        } else {
          columnTasks[columnKey].push(task);
        }
        if (cellTasks[rowKey] == null) {
          cellTasks[rowKey] = {};
          cellTasks[rowKey][columnKey] = [task];
        } else if (cellTasks[rowKey][columnKey] == null) {
          cellTasks[rowKey][columnKey] = [task];
        } else {
          cellTasks[rowKey][columnKey].push(task);
        }
      });

      // Build rows and columns, then sort them.
      rowDim.items.forEach(function (item) {
        var key = item.key;
        if (rowTasks[key] == null) {
          // no tasks for this key
          rowTasks[key] = [];
        }
        var header = {
          key: key,
          text: rowDim.displayName(key),
          canvas: document.createElement('canvas'),
          tasks: rowTasks[key],
          clusterStats: taskStats(rowTasks[key]),
          dirty: true
        };
        header.textMeasure = view.measureHeader(header);
        newDataset.rows.push(header);
        keyToRows[key] = header;
        maxRowWidth = Math.max(maxRowWidth, header.textMeasure);
      });
      maxRowWidth += 20;
      rowsWidth = maxRowWidth + view.style.cellSize;

      columnDim.items.forEach(function (item) {
        var key = item.key;
        if (columnTasks[key] == null) {
          // no tasks for this key
          columnTasks[key] = [];
        }
        var header = {
          key: key,
          text: columnDim.displayName(key),
          canvas: document.createElement('canvas'),
          tasks: columnTasks[key],
          clusterStats: taskStats(columnTasks[key]),
          textRotate: 90,
          dirty: true
        };
        header.textMeasure = view.measureHeader(header);
        newDataset.columns.push(header);
        keyToColumns[key] = header;
        maxColumnHeight = Math.max(maxColumnHeight, header.textMeasure);
      });
      maxColumnHeight += 20;
      columnsHeight = maxColumnHeight + view.style.cellSize;

      if (rowDim.sort != null) {
        newDataset.rows.sort(rowDim.sort);
      } else {
        newDataset.rows.sort();
      }
      if (columnDim.sort != null) {
        newDataset.columns.sort(columnDim.sort);
      } else {
        newDataset.columns.sort();
      }

      // Finalize the rows and columns.
      x = view.style.gridPadding;
      y = view.style.gridPadding + columnsHeight;
      newDataset.rows.forEach(function (header, index) {
        header.index = index;
        header.rect = {
          left: x,
          top: y,
          width: rowsWidth,
          height: view.style.cellSize
        };
        header.textOffset = {
          x: (maxRowWidth - header.textMeasure) / 2,
          y: view.style.cellSize / 2
        };
        header.clusterRect = {
          left: rowsWidth - view.style.cellSize - 5,
          top: 0,
          width: view.style.cellSize,
          height: view.style.cellSize
        };
        header.canvas.style.left = header.rect.left;
        header.canvas.style.top = header.rect.top;
        header.canvas.style.position = "absolute";
        newDataset.canvases.push(header.canvas);
        y += view.style.cellSize;
      });

      x = view.style.gridPadding + rowsWidth;
      y = view.style.gridPadding;
      newDataset.columns.forEach(function (header, index) {
        header.index = index;
        header.rect = {
          left: x,
          top: y,
          width: view.style.cellSize,
          height: columnsHeight
        };
        header.textOffset = {
          x: view.style.cellSize / 2,
          y: (maxColumnHeight - header.textMeasure) / 2
        };
        header.clusterRect = {
          left: 0,
          top: columnsHeight - view.style.cellSize - 5,
          width: view.style.cellSize,
          height: view.style.cellSize
        };
        header.canvas.style.left = header.rect.left;
        header.canvas.style.top = header.rect.top;
        header.canvas.style.position = "absolute";
        newDataset.canvases.push(header.canvas);
        x += view.style.cellSize;
      });

      // Sort all of the cells wrt rows and columns.
      x = view.style.gridPadding + rowsWidth;
      y = view.style.gridPadding + columnsHeight;
      Object.keys(cellTasks).forEach(function (rowKey) {
        var row = keyToRows[rowKey];
        var cellRowTasks = cellTasks[rowKey];
        Object.keys(cellRowTasks).forEach(function (columnKey) {
          var column = keyToColumns[columnKey];
          var index = newDataset.cellIndex(column.index, row.index);

          var cell = {
            rowKey: rowKey,
            columnKey: columnKey,
            tasks: cellRowTasks[columnKey],
            canvas: document.createElement('canvas'),
            clusterStats: taskStats(cellRowTasks[columnKey]),
            rect: {
              left: x + view.style.cellSize * column.index,
              top: y + view.style.cellSize * row.index,
              width: view.style.cellSize,
              height: view.style.cellSize
            },
            dirty: true
          };
          newDataset.cells[index] = cell;
          cell.canvas.style.left = cell.rect.left;
          cell.canvas.style.top = cell.rect.top;
          cell.canvas.style.position = "absolute";
          newDataset.canvases.push(cell.canvas);
        });
      });
      view.fillEmptyCells(newDataset, rowsWidth, columnsHeight);

      // transition here
      view.resetDiv(newDataset, rowsWidth, columnsHeight);
      view.dataset = newDataset;
      view.dirty = true;

      view.tick();
    },

    // events set by the controller
    rowAt: function (x, y) {
      var result;
      if (view.dataset.rows.some(function (row) { result = row; return contains(row.rect, x, y); })) {
        return result;
      }
      return null;
    },
    columnAt: function (x, y) {
      var result;
      if (view.dataset.columns.some(function (column) { result = column; return contains(column.rect, x, y); })) {
        return result;
      }
      return null;
    },
    cellAt: function (x, y) {
      var result;
      if (view.dataset.cells.some(function (cell) { result = cell; return contains(cell.rect, x, y); })) {
        return result;
      }
      return null;
    },
    /*onMouseDown: function (event) {
      var x = event.pageX - view.offsetLeft;
      var y = event.pageY - view.offsetTop;
      switch (event.which) {
        case 0: { // no button
        } break;
        case 1: { // left
          // add ripples?
        } break;
        case 2: { // middle
        } break;
        case 3: { // right
          // show dataview
        } break;
      }
    },*/
    highlighted: null,
    onMouseMove: function (event) {
      var x = event.pageX - view.div.offsetLeft;
      var y = event.pageY - view.div.offsetTop;
      var hit;
      // set highlighted element
      if (view.highlighted != null) {
        view.highlighted.dirty = true;
      }
      hit = view.rowAt(x, y);
      if (hit !== null) {
        view.highlighted = hit;
        view.highlighted.dirty = false;
        view.tick();
        return;
      }
      hit = view.columnAt(x, y);
      if (hit !== null) {
        view.highlighted = hit;
        view.highlighted.dirty = false;
        view.tick();
        return;
      }
      hit = view.cellAt(x, y);
      if (hit !== null) {
        view.highlighted = hit;
        view.highlighted.dirty = false;
        view.tick();
        return;
      }

      view.highlighted = null;
      view.tick();
    },
    onMouseLeave: function () {
      if (view.highlighted !== null) {
        view.highlighted.dirty = true;
      }
      view.tick();
    },
    onRowClicked: [],
    onColumnClicked: [],
    onCellClicked: [],
    onClick: function (event) {
      var x = event.pageX - view.div.offsetLeft;
      var y = event.pageY - view.div.offsetTop;
      // row? column? cell?
      var hit = view.rowAt(x, y);
      if (hit !== null) {
        view.onRowClicked.forEach((callback) => callback(hit));
        return;
      }
      hit = view.columnAt(x, y);
      if (hit !== null) {
        view.onColumnClicked.forEach((callback) => callback(hit));
        return;
      }

      hit = view.cellAt(x, y);
      if (hit !== null) {
        view.onCellClicked.forEach((callback) => callback(hit));
        return;
      }
    },

    // drawing, animation, etc.
    startTime: Date.now(),
    animating: false,
    dirty: false,
    queueTick: function () {
      if (view.tickPending) {
        return;
      }
      window.requestAnimationFrame(function () {
        view.tickPending = false;
        view.tick();
      });
      view.tickPending = true;
    },
    tick: function () {
      var time;

      // check if tick is already pending
      if (view.tickPending) {
        return;
      }

      time = (Date.now() - view.startTime) / 1000;
      view.time = time;

      // update transitions

      // true when drawing transitions
      view.animating = false;

      view.draw();

      if (view.animating) {
        view.queueTick();
      }
    },
    draw: function () {
      // find what we are drawing
      var toDraw = [];
      var vRect = visibleRect(view.div);
      var dpr = window.devicePixelRatio;

      var doesAOverlapB = function (rectA, rectB) {
        return rectA.left < rectB.left + rectB.width
          && rectA.left + rectA.width > rectB.left
          && rectA.top < rectB.top + rectB.height
          && rectA.top + rectA.height > rectB.top;
      };

      toDraw = [];
      view.dataset.rows.forEach(function (row) {
        if (!doesAOverlapB(vRect, row.rect) || view.highlighted === row) {
          return;
        }
        if (row.dirty || view.dirty) {
          toDraw.push(row);
        }
      });
      view.dataset.columns.forEach(function (column) {
        if (!doesAOverlapB(vRect, column.rect) || view.highlighted === column) {
          return;
        }
        if (column.dirty || view.dirty) {
          toDraw.push(column);
        }
      });
      view.dataset.cells.forEach(function (cell) {
        if (!doesAOverlapB(vRect, cell.rect) || view.highlighted === cell) {
          return;
        }
        if (cell.dirty || view.dirty) {
          toDraw.push(cell);
        }
      });

      if (view.highlighted !== null) {
        toDraw.push(view.highlighted);
      }

      toDraw.forEach(function (drawn) {
        var ctx = drawn.canvas.getContext('2d');
        var rect, clusterRect;
        ctx.save();
        ctx.translate(0.5, 0.5);

        rect = drawn.rect;
        drawn.canvas.width = rect.width * dpr;
        drawn.canvas.height = rect.height * dpr;
        drawn.canvas.style.width = rect.width;
        drawn.canvas.style.height = rect.height;
        ctx.scale(dpr, dpr);

        // draw background
        ctx.fillStyle = view.clusterBackgroundColor(drawn.clusterStats);
        ctx.fillRect(0, 0, rect.width, rect.height);

        if (drawn === view.highlighted) {
          // draw shadow
          ctx.shadowBlur = 30;
          ctx.shadowOffsetX = 3;
          ctx.shadowOffsetY = 3;
          ctx.shadowColor = view.style.cellShadowColor;
          ctx.fillStyle = "white";
          ctx.fillRect(0, 0, rect.width, rect.height);
          ctx.shadowBlur = 0;
          ctx.shadowOffsetX = 0;
          ctx.shadowOffsetY = 0;
        }

        // draw ripples
        // view.drawClickRipples();

        // draw grid
        ctx.strokeStyle = view.style.gridLineColor;
        ctx.lineWidth = view.style.gridLineWidth;
        ctx.strokeRect(0, 0, rect.width, rect.height);

        // draw cluster
        if (drawn.clusterRect == null) {
          clusterRect = { left: 0, top: 0, width: rect.width, height: rect.height };
        } else {
          clusterRect = drawn.clusterRect;
        }
        drawn.dirty = view.drawCluster(ctx, drawn.clusterStats, clusterRect);

        // draw text
        if (drawn.text != null) {
          ctx.font = view.style.headerFont;
          ctx.fillStyle = view.style.headerFontColor;
          ctx.textBaseline = "middle";
          ctx.translate(drawn.textOffset.x, drawn.textOffset.y);
          ctx.rotate(drawn.textRotate * Math.PI / 180);
          ctx.fillText(drawn.text, 0, 0);
        }
        ctx.restore();
      });
      view.dirty = false;
    },
    drawCluster: function (ctx, clusterStats, rect) {
      var halfWidth = rect.width / 2;
      var centerX = rect.left + halfWidth;
      var centerY = rect.top + halfWidth;
      var radius = halfWidth * 0.8;
      var dashLen = (2 * Math.PI * radius) / 10;
      var angle = -Math.PI / 2;
      var countToAngle = 2 * Math.PI / clusterStats.numTasks;
      var animating = false;
      var icon = view.clusterIcon(clusterStats);
      var iconColor = view.clusterForegroundColor(clusterStats);
      function drawClusterSegment(color, dash, count) {
        var step = countToAngle * count;
        var nextAngle = angle - step;

        if (count === 0) {
          return;
        }
        ctx.save();
        ctx.beginPath();
        ctx.arc(centerX, centerY, radius, angle, nextAngle, true);
        ctx.strokeStyle = color;
        ctx.stroke();

        if (dash) {
          ctx.lineDashOffset = view.time * 5;
          ctx.lineWidth = 3;
          ctx.strokeStyle = view.style.inProgressForegroundColor;
          ctx.setLineDash([dashLen * 0.4, dashLen * 0.6]);
          ctx.beginPath();
          ctx.arc(centerX, centerY, radius, angle, nextAngle, true);
          ctx.stroke();
          animating = true;
          view.animating = true;
        }

        angle = nextAngle;
        ctx.restore();
      }

      ctx.save();
      ctx.lineWidth = 5;

      drawClusterSegment(view.style.staleUnknownForegroundColor, false, clusterStats.numStaleUnknown);
      drawClusterSegment(view.style.currentSucceededForegroundColor, false, clusterStats.numCurrentSucceeded);
      drawClusterSegment(view.style.staleSucceededForegroundColor, true, clusterStats.numInProgressWasSucceeded + clusterStats.numInProgressWasUnknown);
      drawClusterSegment(view.style.staleSucceededForegroundColor, false, clusterStats.numStaleSucceeded);
      drawClusterSegment(view.style.staleFailedForegroundColor, false, clusterStats.numStaleFailed);
      drawClusterSegment(view.style.staleFailedForegroundColor, true, clusterStats.numInProgressWasFailed);
      drawClusterSegment(view.style.currentFailedForegroundColor, false, clusterStats.numCurrentFailed);
      drawClusterSegment(view.style.fixedForegroundColor, false, clusterStats.numSucceededWasFailed);
      drawClusterSegment(view.style.regressedForegroundColor, false, clusterStats.numFailedWasSucceeded);

      if (icon !== 0) {
        // draw icon
        ctx.save();
        ctx.translate(centerX, centerY);
        ctx.font = view.style.iconsFont;
        ctx.fillStyle = iconColor;
        ctx.textAlign = "center";
        ctx.textBaseline = "middle";
        ctx.fillText(icon, 0, 0);
        ctx.restore();
      }

      ctx.restore();
      return animating;
    }
    /* drawClickRipples: function (ctx, dt, ripples, clip) {
      // ripples.update(dt)
      // if len(*ripples) == 0 {
      //   return
      // }
      // g.animating = true
      // ctx.Save()
      // if clip != nil {
      //   ctx.BeginPath()
      //   clip()
      //   ctx.Clip()
      // }
      // for _, r := range *ripples {
      //   ctx.Save()
      //   ctx.GlobalAlpha = r.alpha()
      //   ctx.BeginPath()
      //   ctx.Arc(r.center, r.radius(), 0, 360, false)
      //   ctx.FillStyle = g.Style.SelectedBackgroundColor
      //   ctx.Fill()
      //   ctx.Restore()
      // }
      // ctx.Restore()
    }*/
  };

  window.addEventListener("scroll", function () {
    view.queueTick();
  });
  return view;
};

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

var visibleRect = function (element) {
  var boundingRect = element.getBoundingClientRect();
  return {
    left: Math.max(-boundingRect.left, 0),
    top: Math.max(-boundingRect.top, 0),
    width: Math.min(boundingRect.right, window.innerWidth),
    height: Math.min(boundingRect.bottom, window.innerHeight)
  };
};
var contains = function (rect, x, y) {
  return rect.left <= x && rect.left + rect.width >= x
    && rect.top <= y && rect.top + rect.height >= y;
};
var taskStats = function (tasks) {
  if (typeof taskStats.initialized == 'undefined') {
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
      for (var i = 0; i < tasks.length; ++i) {
        if (pred(tasks[i])) {
          count++;
        }
      }
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
var newGridView = function () {
  var grid = {
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
        return grid.style.currentFailedBackgroundColor;
      } else if (clusterStats.numSucceededWasFailed > 0) {
        return grid.style.currentSucceededBackgroundColor;
      } else if (clusterStats.numInProgressWasFailed + clusterStats.numStaleFailed > 0) {
        return grid.style.staleFailedBackgroundColor;
      } else if (clusterStats.numInProgressWasSucceeded + clusterStats.numStaleSucceeded > 0) {
        return grid.style.staleSucceededBackgroundColor;
      } else if (clusterStats.numCurrentSucceeded > 0) {
        return grid.style.currentSucceededBackgroundColor;
      } else if (clusterStats.numInprogressWasUnknown + clusterStats.numStaleUnknown > 0) {
        return grid.style.unknownBackgroundColor;
      } else {
        return grid.style.backgroundColor;
      }
    },
    clusterForegroundColor: function (clusterStats) {
      if (clusterStats.numFailedWasSucceeded > 0) {
        return grid.style.regressedForegroundColor;
      } else if (clusterStats.numCurrentFailed > 0) {
        return grid.style.currentFailedForegroundColor;
      } else if (clusterStats.numSucceededWasFailed > 0) {
        return grid.style.fixedForegroundColor;
      } else if (clusterStats.numInProgressWasFailed + clusterStats.numStaleFailed > 0) {
        return grid.style.staleFailedForegroundColor;
      } else if (clusterStats.numInProgressWasSucceeded + clusterStats.numStaleSucceeded > 0) {
        return grid.style.staleSucceededForegroundColor;
      } else if (clusterStats.numCurrentSucceeded > 0) {
        return grid.style.currentSucceededForegroundColor;
      } else {
        return grid.style.unknownForegroundColor;
      }
    },
    clusterIcon: function (clusterStats) {
      if (clusterStats.numFailedWasSucceeded + clusterStats.numCurrentFailed > 0) {
        return grid.style.icons.Failed;
      } else if (clusterStats.numSucceededWasFailed > 0) {
        return grid.style.icons.Succeeded;
      } else if (clusterStats.numInProgressWasFailed + clusterStats.numStaleFailed > 0) {
        return grid.style.icons.Failed;
      } else if (clusterStats.numInProgressWasSucceeded + clusterStats.numStaleSucceeded + clusterStats.numCurrentSucceeded > 0) {
        return grid.style.icons.Succeeded;
      } else {
        return grid.style.icons.Unknown;
      }
    },
    measureHeader: function (header) {
      var ctx = header.canvas.getContext('2d');
      ctx.save();
      ctx.font = grid.style.headerFont;
      measure = ctx.measureText(header.text);
      ctx.restore();
      return Math.ceil(measure.width);
    },

    // input data from model
    dataset: {},
    setData: function (tasks, rowDim, columnDim, filterFn) {
      var newDataset = {
        cells: [],
        rows: [],
        columns: [],
        canvases: []
      };
      var cellIndex = function (cIndex, rIndex) {
        return (cIndex * newDataset.rows.length) + rIndex;
      }

      var rowTasks = {};
      var columnTasks = {};
      var cellTasks = {};
      for (var i = 0; i < tasks.length; ++i) {
        var task = tasks[i];
        if (filterFn(task) == false) {
          continue;
        }
        var rowKey = rowDim.keyOf(task);
        if (rowTasks[rowKey] == null) {
          rowTasks[rowKey] = [task];
        } else {
          rowTasks[rowKey].push(task);
        }
        var columnKey = columnDim.keyOf(task);
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
      }

      // Build rows and columns, then sort them.
      var keyToRows = {};
      var maxRowWidth = 0;
      for (var i = 0; i < rowDim.items.length; ++i) {
        var key = rowDim.items[i].key;
        if (rowTasks[key] == null) {
          // no tasks for this key
          rowTasks[key] = []
        }
        var header = {
          key: key,
          text: rowDim.displayName(key),
          canvas: document.createElement('canvas'),
          tasks: rowTasks[key],
          clusterStats: taskStats(rowTasks[key]),
          dirty: true
        };
        header.textMeasure = grid.measureHeader(header);
        newDataset.rows.push(header);
        keyToRows[key] = header;
        maxRowWidth = Math.max(maxRowWidth, header.textMeasure)
      }
      maxRowWidth += 20;
      var rowsWidth = maxRowWidth + grid.style.cellSize;

      var keyToColumns = {};
      var maxColumnHeight = 0;
      for (var i = 0; i < columnDim.items.length; ++i) {
        var key = columnDim.items[i].key;
        if (columnTasks[key] == null) {
          // no tasks for this key
          columnTasks[key] = []
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
        header.textMeasure = grid.measureHeader(header);
        newDataset.columns.push(header);
        keyToColumns[key] = header;
        maxColumnHeight = Math.max(maxColumnHeight, header.textMeasure);
      }
      maxColumnHeight += 20;
      var columnsHeight = maxColumnHeight + grid.style.cellSize;

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
      var x = grid.style.gridPadding;
      var y = grid.style.gridPadding + columnsHeight;
      for (var i = 0; i < newDataset.rows.length; ++i) {
        header = newDataset.rows[i];
        header.index = i;
        header.rect = {
          left: x,
          top: y,
          width: rowsWidth,
          height: grid.style.cellSize
        };
        header.textOffset = {
          x: (maxRowWidth - header.textMeasure) / 2,
          y: grid.style.cellSize / 2
        };
        header.clusterRect = {
          left: rowsWidth - grid.style.cellSize - 5,
          top: 0,
          width: grid.style.cellSize,
          height: grid.style.cellSize
        };
        header.canvas.style.left = header.rect.left;
        header.canvas.style.top = header.rect.top;
        header.canvas.style.position = "absolute";
        newDataset.canvases.push(header.canvas);
        y += grid.style.cellSize;
      }

      x = grid.style.gridPadding + rowsWidth;
      y = grid.style.gridPadding;
      for (var i = 0; i < newDataset.columns.length; ++i) {
        header = newDataset.columns[i];
        header.index = i;
        header.rect = {
          left: x,
          top: y,
          width: grid.style.cellSize,
          height: columnsHeight
        };
        header.textOffset = {
          x: grid.style.cellSize / 2,
          y: (maxColumnHeight - header.textMeasure) / 2,
        };
        header.clusterRect = {
          left: 0,
          top: columnsHeight - grid.style.cellSize - 5,
          width: grid.style.cellSize,
          height: grid.style.cellSize
        };
        header.canvas.style.left = header.rect.left;
        header.canvas.style.top = header.rect.top;
        header.canvas.style.position = "absolute";
        newDataset.canvases.push(header.canvas);
        x += grid.style.cellSize;
      }

      // Sort all of the cells wrt rows and columns.
      x = grid.style.gridPadding + rowsWidth;
      y = grid.style.gridPadding + columnsHeight;
      for (var rowKey in cellTasks) {
        if (!cellTasks.hasOwnProperty(rowKey)) {
          continue;
        }
        var row = keyToRows[rowKey];
        var cellRowTasks = cellTasks[rowKey];
        for (var columnKey in cellRowTasks) {
          if (!cellRowTasks.hasOwnProperty(columnKey)) {
            continue;
          }
          var column = keyToColumns[columnKey];
          var index = cellIndex(column.index, row.index);

          var cell = {
            rowKey: rowKey,
            columnKey: columnKey,
            tasks: cellRowTasks[columnKey],
            canvas: document.createElement('canvas'),
            clusterStats: taskStats(cellRowTasks[columnKey]),
            rect: {
              left: x + grid.style.cellSize * column.index,
              top: y + grid.style.cellSize * row.index,
              width: grid.style.cellSize,
              height: grid.style.cellSize
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


      // Create an empty cell for any missing cells.
      for (var i = 0; i < newDataset.columns.length; ++i) {
        for (var j = 0; j < newDataset.rows.length; ++j) {
          var index = cellIndex(i, j);
          if (newDataset.cells[index] == null) {
            var cell = {
              canvas: document.createElement('canvas'),
              clusterStats: taskStats([]),
              rect: {
                left: x + grid.style.cellSize * i,
                top: y + grid.style.cellSize * j,
                width: grid.style.cellSize,
                height: grid.style.cellSize
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


      // TODO: transitions

      // fastest way to clear out all canvas elements
      var cDiv = grid.div.cloneNode(false);
      grid.div.parentNode.replaceChild(cDiv, grid.div);
      grid.div = cDiv;
      grid.div.style.position = "relative";

      for (var i = 0; i < newDataset.canvases.length; ++i) {
        grid.div.appendChild(newDataset.canvases[i]);
      }
      grid.div.addEventListener("mouseleave", grid.onMouseLeave);
      grid.div.addEventListener("mousemove", grid.onMouseMove);
      grid.div.addEventListener("click", grid.onClick);
      grid.width = grid.style.gridPadding * 2 + rowsWidth + grid.style.cellSize * newDataset.columns.length;
      grid.height = grid.style.gridPadding * 2 + columnsHeight + grid.style.cellSize * newDataset.rows.length;
      grid.div.style.width = grid.width + "px";
      grid.div.style.height = grid.height + "px";


      grid.dataset = newDataset;
      grid.dirty = true;

      grid.tick();
    },

    // events set by the controller
    rowAt: function (x, y) {
      for (var i = 0; i < grid.dataset.rows.length; ++i) {
        var row = grid.dataset.rows[i];
        if (contains(row.rect, x, y)) {
          return row;
        }
      }
    },
    columnAt: function (x, y) {
      for (var i = 0; i < grid.dataset.columns.length; ++i) {
        var column = grid.dataset.columns[i];
        if (contains(column.rect, x, y)) {
          return column;
        }
      }
    },
    cellAt: function (x, y) {
      for (var i = 0; i < grid.dataset.cells.length; ++i) {
        var cell = grid.dataset.cells[i];
        if (contains(cell.rect, x, y)) {
          return cell;
        }
      }
    },
    onMouseDown: function (event) {
      var x = event.pageX - grid.offsetLeft;
      var y = event.pageY - grid.offsetTop;
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
    },
    onMouseUp: function (event) { },
    highlighted: null,
    onMouseMove: function (event) {
      var x = event.pageX - grid.div.offsetLeft;
      var y = event.pageY - grid.div.offsetTop;;
      // set highlighted element
      if (grid.highlighted != null) {
        grid.highlighted.dirty = true;
      }
      var hit = grid.rowAt(x, y);
      if (hit != null) {
        grid.highlighted = hit;
        grid.highlighted.dirty = false;
      } else if ((hit = grid.columnAt(x, y)) != null) {
        grid.highlighted = hit;
        grid.highlighted.dirty = false;
      } else if ((hit = grid.cellAt(x, y)) != null) {
        grid.highlighted = hit;
        grid.highlighted.dirty = false;
      } else {
        grid.highlighted = null;
      }
      grid.tick();
    },
    onMouseLeave: function (event) {
      if (grid.highlighted != null) {
        grid.highlighted.dirty = true;
      }
      grid.tick();
    },
    onRowClicked: [],
    onColumnClicked: [],
    onCellClicked: [],
    onClick: function (event) {
      var x = event.pageX - grid.div.offsetLeft;
      var y = event.pageY - grid.div.offsetTop;
      // row? column? cell?
      var hit = grid.rowAt(x, y);
      if (hit != null) {
        for (var j = 0; j < grid.onRowClicked.length; ++j) {
          grid.onRowClicked[j](hit);
        }
        return;
      }
      hit = grid.columnAt(x, y);
      if (hit != null) {
        for (var j = 0; j < grid.onColumnClicked.length; ++j) {
          grid.onColumnClicked[j](hit);
        }
        return;
      }

      hit = grid.cellAt(x, y);
      if (hit != null) {
        for (var j = 0; j < grid.onCellClicked.length; ++j) {
          grid.onCellClicked[j](hit);
        }
        return
      }
    },

    // drawing, animation, etc.
    startTime: Date.now(),
    time: this.startTime,
    animating: false,
    dirty: false,
    queueTick: function () {
      if (grid.tickPending) {
        return;
      }
      window.requestAnimationFrame(function () {
        grid.tickPending = false
        grid.tick()
      });
      grid.tickPending = true;
    },
    tick: function () {
      // check if tick is already pending
      if (grid.tickPending) {
        return;
      }

      // convert to seconds
      var time = (grid.startTime - Date.now()) / 1000;
      var dt = time - grid.time;
      grid.time = time;

      // update transitions

      // true when drawing transitions
      grid.animating = false;

      grid.draw(dt);

      if (grid.animating) {
        grid.queueTick();
      }
    },
    draw: function (dt) {
      // find what we are drawing
      var vRect = visibleRect(grid.div);

      var doesAOverlapB = function (rectA, rectB) {
        return rectA.left < rectB.left + rectB.width
          && rectA.left + rectA.width > rectB.left
          && rectA.top < rectB.top + rectB.height
          && rectA.top + rectA.height > rectB.top;
      };

      var toDraw = [];
      for (var i = 0; i < grid.dataset.rows.length; ++i) {
        var row = grid.dataset.rows[i];
        if (!doesAOverlapB(vRect, row.rect) || grid.highlighted == row) {
          continue;
        } else if (row.dirty || grid.dirty) {
          toDraw.push(row);
        }
      }
      for (var i = 0; i < grid.dataset.columns.length; ++i) {
        var column = grid.dataset.columns[i];
        if (!doesAOverlapB(vRect, column.rect) || grid.highlighted == column) {
          continue;
        } else if (column.dirty || grid.dirty) {
          toDraw.push(column);
        }
      }
      for (var i = 0; i < grid.dataset.cells.length; ++i) {
        var cell = grid.dataset.cells[i];
        if (!doesAOverlapB(vRect, cell.rect) || grid.highlighted == cell) {
          continue;
        } else if (cell.dirty || grid.dirty) {
          toDraw.push(cell);
        }
      }

      if (grid.highlighted != null) {
        toDraw.push(grid.highlighted);
      }

      var dpr = window.devicePixelRatio
      for (var i = 0; i < toDraw.length; ++i) {
        var drawn = toDraw[i]

        var ctx = drawn.canvas.getContext('2d');
        ctx.save();
        ctx.translate(0.5, 0.5);

        var rect = drawn.rect;
        drawn.canvas.width = rect.width * dpr;
        drawn.canvas.height = rect.height * dpr;
        ctx.scale(dpr, dpr);

        // draw background
        ctx.fillStyle = grid.clusterBackgroundColor(drawn.clusterStats);
        ctx.fillRect(0, 0, rect.width, rect.height);

        if (drawn == grid.highlighted) {
          // draw shadow
          ctx.shadowBlur = 30;
          ctx.shadowOffsetX = 3;
          ctx.shadowOffsetY = 3;
          ctx.shadowColor = grid.style.cellShadowColor;
          ctx.fillStyle = "white";
          ctx.fillRect(0, 0, rect.width, rect.height);
          ctx.shadowBlur = 0;
          ctx.shadowOffsetX = 0;
          ctx.shadowOffsetY = 0;
        }

        // draw ripples
        grid.drawClickRipples();

        // draw grid
        ctx.strokeStyle = grid.style.gridLineColor;
        ctx.lineWidth = grid.style.gridLineWidth;
        ctx.strokeRect(0, 0, rect.width, rect.height);

        // draw cluster

        var clusterRect
        if (drawn.clusterRect == null) {
          clusterRect = { left: 0, top: 0, width: rect.width, height: rect.height };
        } else {
          clusterRect = drawn.clusterRect;
        }
        drawn.dirty = grid.drawCluster(ctx, drawn.clusterStats, clusterRect);

        // draw text
        if (drawn.text != null) {
          ctx.font = grid.style.headerFont;
          ctx.fillStyle = grid.style.headerFontColor;
          ctx.textBaseline = "middle";
          ctx.translate(drawn.textOffset.x, drawn.textOffset.y);
          ctx.rotate(drawn.textRotate * Math.PI / 180);
          ctx.fillText(drawn.text, 0, 0);
        }
        ctx.restore();
      }
      grid.dirty = false;
    },
    drawCluster: function (ctx, clusterStats, rect) {
      ctx.save();

      var halfWidth = rect.width / 2;
      var centerX = rect.left + halfWidth;
      var centerY = rect.top + halfWidth;
      ctx.lineWidth = 5;
      var radius = halfWidth * 0.8;
      var dashLen = (2 * Math.PI * radius) / 10;
      var angle = -Math.PI / 2;
      var countToAngle = 2 * Math.PI / clusterStats.numTasks;
      var animating = false;

      var drawClusterSegment = function (color, dash, count) {
        if (count === 0) {
          return;
        }
        ctx.save();
        var step = countToAngle * count;
        var nextAngle = angle - step;
        ctx.beginPath();
        ctx.arc(centerX, centerY, radius, angle, nextAngle, true);
        ctx.strokeStyle = color;
        ctx.stroke();

        if (dash) {
          ctx.lineDashOffset = grid.time * 5;
          ctx.lineWidth = 3;
          ctx.strokeStyle = grid.style.inProgressForegroundColor;
          ctx.setLineDash([dashLen * 0.4, dashLen * 0.6]);
          ctx.beginPath();
          ctx.arc(centerX, centerY, radius, angle, nextAngle, true);
          ctx.stroke();
          animating = true;
          grid.animating = true;
        }

        angle = nextAngle;
        ctx.restore();
      }

      drawClusterSegment(grid.style.staleUnknownForegroundColor, false, clusterStats.numStaleUnknown);
      drawClusterSegment(grid.style.currentSucceededForegroundColor, false, clusterStats.numCurrentSucceeded);
      drawClusterSegment(grid.style.staleSucceededForegroundColor, true, clusterStats.numInProgressWasSucceeded + clusterStats.numInProgressWasUnknown);
      drawClusterSegment(grid.style.staleSucceededForegroundColor, false, clusterStats.numStaleSucceeded);
      drawClusterSegment(grid.style.staleFailedForegroundColor, false, clusterStats.numStaleFailed);
      drawClusterSegment(grid.style.staleFailedForegroundColor, true, clusterStats.numInProgressWasFailed);
      drawClusterSegment(grid.style.currentFailedForegroundColor, false, clusterStats.numCurrentFailed);
      drawClusterSegment(grid.style.fixedForegroundColor, false, clusterStats.numSucceededWasFailed);
      drawClusterSegment(grid.style.regressedForegroundColor, false, clusterStats.numFailedWasSucceeded);

      var icon = grid.clusterIcon(clusterStats);
      var iconColor = grid.clusterForegroundColor(clusterStats);
      if (icon != 0) {
        // draw icon
        ctx.save();
        ctx.translate(centerX, centerY);
        ctx.font = grid.style.iconsFont;
        ctx.fillStyle = iconColor;
        ctx.textAlign = "center";
        ctx.textBaseline = "middle";
        ctx.fillText(icon, 0, 0);
        ctx.restore();
      }

      ctx.restore();
      return animating;
    },
    drawClickRipples: function (/*ctx, dt, ripples, clip*/) {
      //todo implement grid?
      // ripples.update(dt)
      // if len(*ripples) == 0 {
      //   return
      // }
      // g.animating = true
      // ctx.Save()
      // if clip != nil {
      // 	 ctx.BeginPath()
      // 	 clip()
      // 	 ctx.Clip()
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
    }
  }

  window.addEventListener("scroll", function (event) {
    grid.queueTick();
  });
  return grid
}

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

// The grid controller handles changing the view and setting it's data when
// it's axes or it's filters change. It also deals with changing those filters
// when the grid is clicked.
var newGridController = function (model) {
  var controller;
  controller = {
    model: model,
    filters: {},
    filterFn: function (task) {
      var pass = true;
      var filter;

      Object.keys(controller.filters).forEach(function (filterKey) {
        filter = controller.filters[filterKey];
        pass = pass && filter(task);
      });
      return pass;
    },
    view: newGridView(),
    axes: {
      row: {},
      column: {}
    },
    clear: function () {
      controller.filters = {};
      controller.axes.row = {};
      controller.axes.column = {};
    },
    initialize: function () {
      controller.axes.column = controller.nextUnfiltered();
      controller.axes.row = controller.nextUnfiltered();
      controller.commitViewState();
    },
    onFilterChanged: [],
    addKeyEqualityFilter: function (filterDim, key) {
      // Only add a filter if there is either enough dimensions to still have a grid
      // or the filter would override a dimension that is already filtered.
      if (controller.filters[filterDim.name] != null
        || controller.model.dimensions.length > Object.keys(controller.filters).length + 2) {
        controller.filters[filterDim.name] = function (task) { return filterDim.keyOf(task) === key; };
        controller.onFilterChanged.forEach((callback) => callback(filterDim.name, "==", key));
        return true;
      }

      return false;
    },
    nextUnfiltered: function () {
      var result;

      if (controller.model.dimensions.some(function (dim) {
        // The first dimension without a filter that is also not an axis will be the end result.
        result = dim;
        return (controller.filters[dim.name] == null && controller.axes.row !== dim && controller.axes.column !== dim);
      })) {
        return result;
      }
      return null;
    },
    onAxisChanged: [],
    setAxis: function (axisName, newAxisDim) {
      var oldAxisDim;

      if (controller.axes[axisName].name === newAxisDim.name) {
        return;
      }
      oldAxisDim = controller.axes[axisName];
      controller.axes[axisName] = newAxisDim;
      controller.onAxisChanged.forEach((callback) => callback(axisName, oldAxisDim, newAxisDim));
    },
    commitViewState: function () {
      controller.view.setData(controller.model.tasks, controller.axes.row, controller.axes.column, controller.filterFn);
    }
  };

  controller.view.onRowClicked.push(function (row) {
    var rowDim = controller.axes.row;
    if (controller.addKeyEqualityFilter(rowDim, row.key)) {
      controller.setAxis("row", controller.nextUnfiltered());
      controller.commitViewState();
    }
  });
  controller.view.onColumnClicked.push(function (column) {
    var columnDim = controller.axes.column;
    if (controller.addKeyEqualityFilter(columnDim, column.key)) {
      controller.setAxis("column", controller.nextUnfiltered());
      controller.commitViewState();
    }
  });
  controller.view.onCellClicked.push(function (cell) {
    var rowDim = controller.axes.row;
    var columnDim = controller.axes.column;
    if (cell.tasks.length !== 1) {
      if (controller.addKeyEqualityFilter(rowDim, cell.rowKey)) {
        controller.setAxis("row", controller.nextUnfiltered());
        if (controller.addKeyEqualityFilter(columnDim, cell.columnKey)) {
          controller.setAxis("column", controller.nextUnfiltered());
        }
        controller.commitViewState();
      }
    }
  });
  return controller;
};

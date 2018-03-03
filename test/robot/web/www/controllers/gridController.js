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


var newGridController = function (model) {
  var controller = {
    model: model,
    filters: {},
    filterFn: function (task) {
      var pass = true;
      for (var filterKey in controller.filters) {
        if (!controller.filters.hasOwnProperty(filterKey)) {
          continue;
        }
        var filter = controller.filters[filterKey];
        pass = pass && filter(task)
      }
      return pass;
    },
    view: newGridView(),
    axes: {
      row: {},
      column: {}
    },
    clear: function () {
      controller.filters = {}
      controller.axes.row = {}
      controller.axes.column = {}
    },
    initialize: function () {
      controller.axes.column = controller.nextUnfiltered();
      controller.axes.row = controller.nextUnfiltered();
      controller.commitViewState();
    },
    onFilterChanged: [],
    addKeyEqualityFilter: function (filterDim, key) {
      if (controller.filters[filterDim.name] != null
        || controller.model.dimensions.length > Object.keys(controller.filters).length + 2) {
        controller.filters[filterDim.name] = function (task) { filterDim.keyOf(task) == key };
        for (var i = 0; i < controller.onFilterChanged.length; ++i) {
          controller.onFilterChanged[i](filterDim.name, "==", key);
        }
        return true;
      }

      return false;
    },
    nextUnfiltered: function () {
      for (var i = 0; i < controller.model.dimensions.length; ++i) {
        var dim = controller.model.dimensions[i];
        if (controller.filters[dim.name] == null && controller.axes.row != dim && controller.axes.column != dim) {
          return dim;
        }
      }
      return null;
    },
    onAxisChanged: [],
    setAxis: function (axisName, newAxisDim) {
      if (controller.axes[axisName].name === newAxisDim.name) {
        return;
      }
      var oldAxisDim = controller.axes[axisName]
      controller.axes[axisName] = newAxisDim;
      for (var i = 0; i < controller.onAxisChanged.length; ++i) {
        controller.onAxisChanged[i](axisName, oldAxisDim, newAxisDim);
      }
    },
    commitViewState: function () {
      controller.view.setData(controller.model.tasks, controller.axes.row, controller.axes.column, controller.filterFn);
    },
  }

  controller.view.onRowClicked.push(function (row) {
    var rowDim = controller.axes.row;
    console.log(rowDim.displayName(row.key))
    if (controller.addKeyEqualityFilter(rowDim, row.key)) {
      controller.setAxis("row", controller.nextUnfiltered());
      controller.commitViewState();
    }
  });
  controller.view.onColumnClicked.push(function (column) {
    var columnDim = controller.axes.column;
    console.log(columnDim.displayName(column.key))
    if (controller.addKeyEqualityFilter(columnDim, column.key)) {
      controller.setAxis("column", controller.nextUnfiltered());
      controller.commitViewState();
    }
  });
  controller.view.onCellClicked.push(function (cell) {
    var rowDim = controller.axes.row;
    var columnDim = controller.axes.column;
    console.log(rowDim.displayName(cell.rowKey) + ", " + columnDim.displayName(cell.columnKey))
    if (cell.tasks.length != 1) {
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
}

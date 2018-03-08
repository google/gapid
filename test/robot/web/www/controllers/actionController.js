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

var newActionController = function (commitState) {
  var controller;
  controller = {
    view: newBreadcrumbView(),
    changing: false,
    targetHash: "",
    actionList: [],
    pushAction: function (actionHash, popFunction, forcedHash) {
      var action = {
        hash: actionHash,
        popStack: [popFunction]
      };
      var fullHash;

      action.index = controller.actionList.push(action);
      if (forcedHash != null) {
        fullHash = forcedHash;
      } else if (window.location.hash === "") {
        fullHash = "#" + actionHash;
      } else {
        fullHash = window.location.hash + "&" + actionHash;
      }
      action.breadcrumb = controller.view.addBreadcrumb(actionHash, fullHash);
      action.breadcrumb.a.onclick = function () {
        controller.popActions(controller.actionList.length - action.index, true);
      };
      if (forcedHash == null) {
        controller.changing = true;
        controller.targetHash = fullHash;
        window.location.hash = fullHash;
      }

    },
    popActions: function (count, moveLocation) {
      if (count === 0) {
        return;
      }
      var newLocation = window.location.hash;
      var oldCount = controller.actionList.length;
      var i, j, action;

      for (i = 0; i < Math.min(count, oldCount); i += 1) {
        action = controller.actionList.pop();
        // undo the action
        action.popStack.forEach((callback) => callback(action.hash));
        controller.view.removeBreadcrumb(action.breadcrumb);
        if (controller.actionList.length !== 0) {
          action.hash = "&" + action.hash;
        }
        newLocation = newLocation.replace(action.hash, "");
      }
      if (moveLocation) {
        commitState();
        controller.changing = true;
        controller.targetHash = newLocation;
        window.location.hash = newLocation;
      }
    }
  };
  return controller;
};

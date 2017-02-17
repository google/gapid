/*
 * Copyright (C) 2017 Google Inc.
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

services.factory('$gapidSelection', function ($rootScope, $location) {
  var listeners = [];
  var selected_view;
  var select = function (view, options) {
    if (selected_view === view) {
      return;
    }
    selected_view = view;
    $location.path('/' + view);
    $location.search(options !== undefined ? options : {});
    angular.forEach(listeners, function (listener) {
      listener(view);
    })
  };

  $rootScope.$on('$locationChangeSuccess', function (event) {
    var view = $location.path().substr(1);
    select(view);
  });

  return {
    onselect: function (callback) {
      listeners.push(callback);
    },
    select: select,
  }
});

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

var newBreadcrumbView = function () {
  var view = {
    element: document.createElement('span'),
    addBreadcrumb: function (name, href) {
      var breadcrumb = {
        separator: document.createTextNode(" >> "),
        a: document.createElement('a')
      };
      view.element.appendChild(breadcrumb.separator);
      breadcrumb.a.text = name;
      breadcrumb.a.href = href;
      view.element.appendChild(breadcrumb.a);
      return breadcrumb;
    },
    removeBreadcrumb: function (breadcrumb) {
      view.element.removeChild(breadcrumb.a);
      view.element.removeChild(breadcrumb.separator);
    }
  };

  view.rootCrumb = view.element.appendChild(document.createElement('a'));
  view.rootCrumb.text = "grid"
  view.rootCrumb.href = "#";
  return view;
}

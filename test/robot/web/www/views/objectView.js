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

var newObjectView = function () {
  var view;

  view = {
    element: document.createElement('div'),
    newFormatterGroup: function () {
      var group;
      group = {
        formatters: [],
        add: function (pattern, fun) {
          var fmt;
          fmt = {
            re: pattern,
            format: fun
          };
          group.formatters.push(fmt);
          return group;
        },
        format: function (path, obj) {
          var result;
          if (group.formatters.some(function (fmt) {
            if (fmt.re.exec(path)) {
              result = fmt.format(path, obj);
              return true;
            }
          })) {
            return { found: true, result: result };
          }
          return { found: false }
        }
      };
      return group;
    },
    objectStack: [],
    selection: -1,
    newContainer: function () {
      return document.createElement('div');
    },
    newLink: function (text, href) {
      var a = document.createElement('a');
      a.text = text;
      a.href = href;
      return a;
    },
    newPusher: function (label, path, obj, fmtGroup) {
      var link = view.newLink(label, "#");
      link.onclick = function (event) {
        view.objectStack = view.objectStack.slice(0, view.selection + 1);
        view.objectStack.push({ name: path, obj: obj, fmtGroup: fmtGroup });
        view.render(view.objectStack.length - 1);
        event.preventDefault();
      };
      return link;
    },
    newTextPreview: function (preview) {
      var text = document.createElement('div');
      text.style.maxWidth = 600;
      text.style.maxHeight = 420;
      text.style.overflow = "auto";
      text.style.whiteSpace = "pre";
      if (preview) {
        text.append(preview);
      }
      return text;
    },
    newVideo: function (path) {
      var v = document.createElement('video');
      var s = document.createElement('source');
      var dpr = window.devicePixelRatio;
      s.src = path;
      s.typ = "video/mp4";
      v.autoplay = true;
      v.controls = true;
      v.append(s);
      v.width = 600 * dpr;
      v.height = 420 * dpr;
      v.style.width = 600;
      v.style.height = 420;
      return v;
    },
    expandable: function (path, obj) {
      return view.newPusher("(view...)", path, obj, null);
    },
    render: function (index) {
      function clearView() {
        var cDiv = view.element.cloneNode(false);
        view.element.parentNode.replaceChild(cDiv, view.element);
        view.element = cDiv;
        view.element.style.position = "sticky";
        view.element.style.top = 0;
        view.element.style.float = "right";
      }
      function addCrumbs() {
        var crumbs = newBreadcrumbView("(hide)", "#");
        //crumbs.element.prepend("Object Viewer: ");
        crumbs.separatorText = " / ";
        crumbs.rootCrumb.onclick = function (event) {
          clearView();
          view.selection = -1;
          addCrumbs();
          event.preventDefault();
        };

        view.objectStack.forEach(function (obj, index) {
          var text;
          if (index == view.selection) {
            text = "[" + obj.name + "]";
          } else {
            text = obj.name;
          }
          crumbs.addBreadcrumb(text, "#").a.onclick = function (event) {
            view.render(index);
            event.preventDefault();
          };
        });

        view.element.append(crumbs.element);
      }
      var selected = view.objectStack[index];
      view.selection = index;

      clearView();
      addCrumbs();
      view.element.append(view.build("", selected.obj, selected.fmtGroup));
    },
    set: function (name, obj, fmtGroup) {
      view.objectStack = [{ name: name, obj: obj, fmtGroup: fmtGroup }];
      view.render(0);
    },
    build: function (path, obj, fmtGroup) {
      var res;
      var table;
      var formatted;
      if (fmtGroup != null) {
        formatted = fmtGroup.format(path, obj);
        if (formatted.found) {
          return formatted.result;
        }
      }

      if (typeof obj === "undefined") {
        return "undefined";
      } else if (obj instanceof Function) {
        return view.build(path, obj(), fmtGroup);
      } else if (obj instanceof Array) {
        table = document.createElement('table');
        table.border = 1;
        obj.forEach(function (val, index) {
          var r = document.createElement('tr');
          var td = document.createElement('td');
          td.append(view.build(path + '/' + index, val, fmtGroup));
          r.append(td);
          table.append(r);
        });
        return table;
      } else if (obj instanceof Object) {
        if ("representation" in obj) {
          obj = obj.representation();
        }
        table = document.createElement('table');
        table.border = 1;
        Object.keys(obj).forEach(function (key) {
          var val = obj[key];
          var r = document.createElement('tr');
          var td = document.createElement('td');
          td.append(key.toString());
          r.append(td);
          td = document.createElement('td');
          td.append(view.build(path + '/' + key, val, fmtGroup));
          r.append(td);
          table.append(r);
        });
        return table
      } else {
        return obj.toString();
      }
    }
  };

  return view;
};

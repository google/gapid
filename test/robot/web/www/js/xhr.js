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

// The xhr module provides Promise-ified XHR functions.
define(function() {
  function xhrDo(method, url, opt_responseType, opt_contentType, opt_data) {
    return new Promise(function(resolve, reject) {
      var xhr = new XMLHttpRequest();
      xhr.open(method, url, true);
      if (opt_responseType) {
        xhr.responseType = opt_responseType;
      }
      if (opt_contentType) {
        xhr.setRequestHeader("Content-Type", opt_contentType);
      }
      var fail = function(e) {
        var msg = "xhr " + method + " to " + url + " failed: " + e;
        console.log(msg)
        reject(new Error(msg));
      };
      xhr.onload = function() {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve(xhr.response);
        } else {
          fail(xhr.statusText);
        }
      };
      xhr.onerror = () => fail("network error");
      xhr.send(opt_data);
    });
  }

  return {
    "get": (url, opt_responseType) => xhrDo("GET", url, opt_responseType),
    "getJson": (url) => xhrDo("GET", url).then(v => JSON.parse(v)),
  }
});

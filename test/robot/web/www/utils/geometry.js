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

function visibleRect(element) {
  var boundingRect = element.getBoundingClientRect();
  return {
    left: Math.max(-boundingRect.left, 0),
    top: Math.max(-boundingRect.top, 0),
    width: Math.min(boundingRect.right, window.innerWidth),
    height: Math.min(boundingRect.bottom, window.innerHeight)
  };
}
function contains(rect, x, y) {
  return rect.left <= x && rect.left + rect.width >= x
    && rect.top <= y && rect.top + rect.height >= y;
}

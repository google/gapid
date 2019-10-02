// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

/** @module swarming-ui/templates
 * @description
 *
 * <p>
 *  A general set of useful templates.
 * </p>
 */

import { html } from 'lit-html'
import 'elements-sk/icon/expand-less-icon-sk'
import 'elements-sk/icon/expand-more-icon-sk'

/** moreOrLess shows an expand-more or expand-less icon
 *  depdending on the passedin param.
 */
export function moreOrLess(conditionForMore) {
  if (conditionForMore) {
    return html`<expand-less-icon-sk></expand-less-icon-sk>`;
  } else {
    return html`<expand-more-icon-sk></expand-more-icon-sk>`;
  }
}
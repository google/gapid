// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import { $, $$ } from 'common-sk/modules/dom'


/**
 * @module swarming-ui/modules/dialog-pop-over
 * @description <h2><code>dialog-pop-over<code></h2>
 *
 * <p>
 *   A dialog box that prevents interaction with the rest of the page
 *   while it is being presented.
 * </p>
 *
 */


const backdrop_template = document.createElement('template');
backdrop_template.innerHTML =`<div class=backdrop></div>`;

window.customElements.define('dialog-pop-over', class extends HTMLElement {

  constructor() {
    super();
    this._backdrop = null;
    this._content = null;
  }

  connectedCallback() {
    const backdrop = backdrop_template.content.cloneNode(true);
    this.appendChild(backdrop);
    // variable backdrop is a #document-fragment, so we need to
    // search for the expanded node after it has been added.
    this._backdrop = $$('.backdrop', this);

    this._content = $$('.content', this);
    if (!this._content) {
      throw 'You must have an element with class content to show.';
    }
  }

  /** hide makes the content and backdrop not visable */
  hide() {
    this._backdrop.classList.remove('opened');
    this._content.classList.remove('opened');
  }

  /** show makes the content centered and visible. It also makes the semi-transparent
   *  backdrop show up and prevent interaction with the elemnts behind it.
   */
  show() {
    // Do some math to center it. This cannot be done in pure CSS because
    // calc doesn't support min/max.
    const availWidth = window.innerWidth;
    const availHeight = window.innerHeight;

    const width = Math.min(this._content.offsetWidth, availWidth - 50);
    const height = Math.min(this._content.offsetHeight, availHeight - 50);
    this._content.style.width = width;
    this._content.style.left = (availWidth - width) / 2;
    this._content.style.top = (availHeight - height) / 2;

    this._backdrop.classList.add('opened');
    this._content.classList.add('opened');
  }
});
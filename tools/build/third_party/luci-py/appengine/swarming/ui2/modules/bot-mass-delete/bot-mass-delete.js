// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import { errorMessage } from 'elements-sk/errorMessage'
import { html, render } from 'lit-html'
import { jsonOrThrow } from 'common-sk/modules/jsonOrThrow'
import { until } from 'lit-html/directives/until';

import { initPropertyFromAttrOrProperty } from '../util'

// query.fromObject is more readable than just 'fromObject'
import * as query from 'common-sk/modules/query'

import 'elements-sk/styles/buttons'

/**
 * @module swarming-ui/modules/bot-mass-delete
 * @description <h2><code>bot-mass-delete<code></h2>
 *
 * <p>
 *   bot-mass-delete offers an interface for the user to delete multiple bots
 *   (and hopefully avoid doing so on accident). Care is taken such that only dead
 *   bots are deleted.
 * </p>
 *
 * @fires bots-deleting-started
 * @fires bots-deleting-finished
 */

const listItem = (dim) => html`<li>${dim}</li>`;

const template = (ele) => html`
  <div>
    You are about to delete all DEAD bots with the following dimensions:
    <ul>
      ${ele.dimensions.map(listItem)}
    </ul>

    This is about ${ele._count} bots.
    Once you start the process, the only way to partially stop it is to close this
    browser window.

    If that sounds good, click the button below.
  </div>

  <button class=delete ?disabled=${!ele._readyToDelete || ele._started}
                       @click=${ele._deleteAll}
                       tabindex=0>
    Delete the bots
  </button>

  <div>
    <div ?hidden=${!ele._started}>
      Progress: ${ele._progress} deleted${ele._finished ? ' - DONE.': '.'}
    </div>
    <div>
      Note: the bot deletion is being done in browser -
      closing the window will stop the mass deletion.
    </div>
  </div>
`;

function fetchError(e, loadingWhat) {
  const message = `Unexpected error loading ${loadingWhat}: ${e.message}`;
  console.error(message);
  errorMessage(message, 5000);
}

window.customElements.define('bot-mass-delete', class extends HTMLElement {

  constructor() {
    super();
    this._count = '...';
    this._readyToDelete = false;
    this._started = false;
    this._finished = false;
    this._progress = 0;
  }

  connectedCallback() {
    initPropertyFromAttrOrProperty(this, 'auth_header');
    initPropertyFromAttrOrProperty(this, 'dimensions');
    // Used for when default was loaded via attribute.
    if (typeof this.dimensions === 'string') {
      this.dimensions = this.dimensions.split(',');
    }
    // sort for determinism
    this.dimensions.sort();
    this.render();
  }

  _deleteAll() {
    this._started = true;
    this.dispatchEvent(new CustomEvent('bots-deleting-started', {bubbles: true}));

    let queryParams = query.fromObject({
      dimensions: this.dimensions,
      limit: 200, // see https://crbug.com/908423
      fields: 'items/bot_id',
    });
    queryParams += '&is_dead=TRUE';

    const extra = {
      headers: {'authorization': this.auth_header},
    };

    let bots = [];
    fetch(`/_ah/api/swarming/v1/bots/list?${queryParams}`, extra)
      .then(jsonOrThrow)
      .then((json) => {
        const maybeLoadMore = (json) => {
          bots = bots.concat(json.items);
          this.render();
          if (json.cursor) {
            queryParams = query.fromObject({
              cursor: json.cursor,
              dimensions: this.dimensions,
              limit: 200, // see https://crbug.com/908423
              fields: 'items/bot_id'
            });
            fetch(`/_ah/api/swarming/v1/bots/list?${queryParams}`, extra)
              .then(jsonOrThrow)
              .then(maybeLoadMore)
              .catch((e) => fetchError(e, 'bot-mass-delete/list (paging)'));
          } else {
            // Now that we have the complete list of bots (e.g. no paging left)
            // delete the bots one at a time, updating this._progress to be the
            // number completed.
            const post = {
              headers: {'authorization': this.auth_header},
              method: 'POST',
            };
            const deleteNext = (bots) => {
              if (!bots.length) {
                this._finished = true;
                this.render();
                this.dispatchEvent(new CustomEvent('bots-deleting-finished', {bubbles: true}));
                return;
              }
              const toDelete = bots.pop();
              fetch(`/_ah/api/swarming/v1/bot/${toDelete.bot_id}/delete`, post)
                .then(() => {
                  this._progress++;
                  this.render();
                  deleteNext(bots);
                }).catch((e) => fetchError(e, 'bot-mass-delete/delete'));
            }
            deleteNext(bots);
          }
        }
        maybeLoadMore(json);
      }).catch((e) => fetchError(e, 'bot-mass-delete/list'));

    this.render();
  }

  _fetchCount() {
    if (!this.auth_header) {
      // This should never happen
      console.warn('no auth_header recieved, try refreshing the page?');
      return;
    }
    const extra = {
      headers: {'authorization': this.auth_header},
    };
    const queryParams = query.fromObject({dimensions: this.dimensions});

    const countPromise = fetch(`/_ah/api/swarming/v1/bots/count?${queryParams}`, extra)
      .then(jsonOrThrow)
      .then((json) => {
        this._readyToDelete = true;
        this.render();
        return parseInt(json.dead);
      }).catch((e) => fetchError(e, 'bot-mass-delete/count'));
    this._count = html`${until(countPromise, '...')}`;
  }

  render() {
    render(template(this), this, {eventContext: this});
  }

  /** show prepares the UI to be shown to the user */
  show() {
    this._readyToDelete = false;
    this._started = false;
    this._finished = false;
    this._progress = 0;
    this._fetchCount();
    this.render();
  }

});
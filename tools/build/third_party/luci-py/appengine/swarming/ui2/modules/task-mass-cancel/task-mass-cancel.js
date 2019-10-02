// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import { $$ } from 'common-sk/modules/dom'
import { errorMessage } from 'elements-sk/errorMessage'
import { html, render } from 'lit-html'
import { jsonOrThrow } from 'common-sk/modules/jsonOrThrow'
import { until } from 'lit-html/directives/until';

import { initPropertyFromAttrOrProperty } from '../util'

// query.fromObject is more readable than just 'fromObject'
import * as query from 'common-sk/modules/query'

import 'elements-sk/checkbox-sk'
import 'elements-sk/styles/buttons'

/**
 * @module swarming-ui/modules/task-mass-cancel
 * @description <h2><code>task-mass-cancel<code></h2>
 *
 * <p>
 * task-mass-cancel offers an interface for the user to cancel multiple tasks
 * (and hopefully avoid doing so on accident).
 * </p>
 *
 * @fires tasks-canceling-started
 * @fires tasks-canceling-finished
 */

const listItem = (tag) => html`<li>${tag}</li>`;

const template = (ele) => html`
  <div>
    You are about to cancel all PENDING bots with the following tags:
    <ul>
      ${ele.tags.map(listItem)}
    </ul>
    <div>
      <checkbox-sk ?checked=${ele._both}
                   ?disabled=${ele._started}
                   @click=${ele._toggleBoth}
                   tabindex=0>
      </checkbox-sk> Also include RUNNING tasks.
    </div>

    This is about ${ele._count()} tasks.
    Once you start the process, the only way to partially stop it is to close this
    browser window.

    If that sounds good, click the button below.
  </div>

  <button class=cancel ?disabled=${!ele._readyToCancel || ele._started}
                       @click=${ele._cancelAll}>
    Cancel the tasks
  </button>

  <div>
    <div ?hidden=${!ele._started}>
      Progress: ${ele._progress} canceled${ele._finished ? ' - DONE.': '.'}
    </div>
    <div>
      Note: tasks queued for cancellation will be canceled as soon as possible, but there may
      be some delay between when this dialog box is closed and all tasks actually being canceled.
    </div>
  </div>
`;

function fetchError(e, loadingWhat) {
  const message = `Unexpected error loading ${loadingWhat}: ${e.message}`;
  console.error(message);
  errorMessage(message, 5000);
}

function nowInSeconds() {
  // convert milliseconds to seconds
  return Math.round(Date.now() / 1000);
}

const CANCEL_BATCH_SIZE = 100;

window.customElements.define('task-mass-cancel', class extends HTMLElement {

  constructor() {
    super();
    this._readyToCancel = false;
    this._started = false;
    this._finished = false;
    this._both = false;
    this._progress = 0;
  }

  connectedCallback() {
    initPropertyFromAttrOrProperty(this, 'auth_header');
    initPropertyFromAttrOrProperty(this, 'tags');
    // Used for when default was loaded via attribute.
    if (typeof this.tags === 'string') {
      this.tags = this.tags.split(',');
    }
    // sort for determinism
    this.tags.sort();
    this.render();
  }

  _cancelAll() {
    this._started = true;
    this.dispatchEvent(new CustomEvent('tasks-canceling-started', {bubbles: true}));
    this.render();

    let payload = {
      limit: CANCEL_BATCH_SIZE,
      tags: this.tags,
    };

    if (this._both) {
      payload.kill_running = true;
    }

    let options = {
      headers: {
        'authorization': this.auth_header,
        'content-type': 'application/json',
      },
      method: 'POST',
      body: JSON.stringify(payload),
    };

    const maybeCancelMore = (json) => {
      this._progress += parseInt(json.matched);
      this.render();
      if (json.cursor) {
        let payload = {
          limit: CANCEL_BATCH_SIZE,
          tags: this.tags,
          cursor: json.cursor,
        };
        if (this._both) {
          payload.kill_running = true;
        }
        let options = {
          headers: {
            'authorization': this.auth_header,
            'content-type': 'application/json',
          },
          method: 'POST',
          body: JSON.stringify(payload),
        };
        fetch('/_ah/api/swarming/v1/tasks/cancel', options)
          .then(jsonOrThrow)
          .then(maybeCancelMore)
          .catch((e) => fetchError(e, 'task-mass-cancel/cancel (paging)'));
      } else {
        this._finished = true;
        this.render();
        this.dispatchEvent(new CustomEvent('tasks-canceling-finished', {bubbles: true}));
      }
    };

    fetch('/_ah/api/swarming/v1/tasks/cancel', options)
        .then(jsonOrThrow)
        .then(maybeCancelMore)
        .catch((e) => fetchError(e, 'task-mass-cancel/cancel'));

  }

  _count() {
    if (this._pendingCount === undefined || this._runningCount === undefined) {
      return '...';
    }
    if (this._both) {
      return this._pendingCount + this._runningCount;
    }
    return this._pendingCount;
  }

  _fetchCount() {
    if (!this.auth_header) {
      // This should never happen
      console.warn('no auth_header recieved, try refreshing the page?');
      return;
    }
    let extra = {
      headers: {'authorization': this.auth_header},
    };

    let pendingParams = query.fromObject({
      state: 'PENDING',
      tags: this.tags,
      // Search in the last week to get the count.  PENDING tasks should expire
      // well before then, so this should be pretty accurate.
      start: nowInSeconds() - 7*24*60*60,
      end: nowInSeconds(),
    });

    let pendingPromise = fetch(`/_ah/api/swarming/v1/tasks/count?${pendingParams}`, extra)
      .then(jsonOrThrow)
      .then((json) => {
        this._pendingCount = parseInt(json.count);
      }).catch((e) => fetchError(e, 'task-mass-cancel/pending'));

    let runningParams = query.fromObject({
      state: 'RUNNING',
      tags: this.tags,
      // Search in the last week to get the count.  RUNNING tasks should finish
      // well before then, so this should be pretty accurate.
      start: nowInSeconds() - 7*24*60*60,
      end: nowInSeconds(),
    });

    let runningPromise = fetch(`/_ah/api/swarming/v1/tasks/count?${runningParams}`, extra)
      .then(jsonOrThrow)
      .then((json) => {
        this._runningCount = parseInt(json.count);
      }).catch((e) => fetchError(e, 'task-mass-cancel/running'));

    // re-render when both have returned
    Promise.all([pendingPromise, runningPromise]).then(() => {
      this._readyToCancel = true;
      this.render();
    });
  }

  render() {
    render(template(this), this, {eventContext: this});
  }

  /** show prepares the UI to be shown to the user */
  show() {
    this._readyToCancel = false;
    this._started = false;
    this._finished = false;
    this._progress = 0;
    this._fetchCount();
    this.render();
  }

  _toggleBoth(e) {
    // This prevents a double event from happening.
    e.preventDefault();
    if (this._started) {
      return;
    }
    this._both = !this._both;
    this.render();
  }
});

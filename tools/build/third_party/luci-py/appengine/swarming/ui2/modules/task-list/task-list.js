// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

/** @module swarming-ui/modules/task-list
 * @description <h2><code>task-list</code></h2>
 *
 * <p>
 *  Task List shows a filterable list of all swarming tasks.
 * </p>
 *
 * <p>This is a top-level element.</p>
 *
 * @attr client_id - The Client ID for authenticating via OAuth.
 * @attr testing_offline - If true, the real OAuth flow won't be used.
 *    Instead, dummy data will be used. Ideal for local testing.
 */

import { $, $$ } from 'common-sk/modules/dom'
import { errorMessage } from 'elements-sk/errorMessage'
import { html, render } from 'lit-html'
import { ifDefined } from 'lit-html/directives/if-defined';
import { until } from 'lit-html/directives/until';
import { jsonOrThrow } from 'common-sk/modules/jsonOrThrow'
import naturalSort from 'javascript-natural-sort/naturalSort'
import { stateReflector } from 'common-sk/modules/stateReflector'

import 'elements-sk/checkbox-sk'
import 'elements-sk/icon/add-circle-icon-sk'
import 'elements-sk/icon/cancel-icon-sk'
import 'elements-sk/icon/more-horiz-icon-sk'
import 'elements-sk/icon/more-vert-icon-sk'
import 'elements-sk/icon/search-icon-sk'
import 'elements-sk/select-sk'
import 'elements-sk/styles/buttons'
import '../dialog-pop-over'
import '../sort-toggle'
import '../swarming-app'
import '../task-mass-cancel'

import flatpickr from 'flatpickr'
import 'flatpickr/dist/flatpickr.css'

// query.fromObject is more readable than just 'fromObject'
import * as query from 'common-sk/modules/query'

import { applyAlias, handleLegacyFilters, maybeApplyAlias } from '../alias'
import { appendPossibleColumns, appendPrimaryMap, column, filterTasks,
         floorSecond, getColHeader, humanizePrimaryKey, legacyTags, listQueryParams,
         processTasks, sortColumns, sortPossibleColumns, specialSortMap,
         stripTag, stripTagFromFilter,  tagsOnly, taskClass,
         useNaturalSort } from './task-list-helpers'
import { botListLink, onSmallScreen } from '../util'
import { filterPossibleColumns, filterPossibleKeys,
         filterPossibleValues, makeFilter } from '../queryfilter'
import { moreOrLess } from '../templates'

import SwarmingAppBoilerplate from '../SwarmingAppBoilerplate'


const colHead = (col, ele) => html`
<th>${getColHeader(col)}
  <sort-toggle .key=${col} .currentKey=${ele._sort} .direction=${ele._dir}>
  </sort-toggle>
</th>`;

const taskCol = (col, task, ele) => html`
<td>${column(col, task, ele)}</td>`;

const taskRow = (task, ele) => html`
<tr class="task-row ${taskClass(task)}">
  ${ele._cols.map((col) => taskCol(col, task, ele))}
</tr>`;

const columnOption = (key, ele) => html`
<div class=item>
  <span class=key>${getColHeader(key)}</span>
  <span class=flex></span>
  <checkbox-sk ?checked=${ele._cols.indexOf(key) >= 0}
               @click=${(e) => ele._toggleCol(e, key)}
               @keypress=${(e) => ele._toggleCol(e, key)}
               data-key=${key}>
  </checkbox-sk>
</div>`;

const col_selector = (ele) => {
  if (!ele._showColSelector) {
    return '';
  }
  return html`
<!-- Stop clicks from traveling outside the popup.-->
<div class=col_selector @click=${e => e.stopPropagation()}>
  <input id=column_search class=search type=text
         placeholder='Search columns to show'
         @input=${ele._refilterPossibleColumns}
         <!-- Looking at the change event, but that had the behavior of firing
              any time the user clicked away, with seemingly no differentiation.
              Instead, we watch keyup and wait for the 'Enter' key. -->
         @keyup=${ele._columnSearch}>
  </input>
  ${ele._filteredPossibleColumns.map((key) => columnOption(key, ele))}
</div>`;
}

const col_options = (ele, firstCol) => html`
<!-- Put the click action here to make it bigger, especially for mobile.-->
<th class=col_options @click=${ele._toggleColSelector}>
  <span class=show_widget>
    <more-vert-icon-sk tabindex=0 @keypress=${ele._toggleColSelector}></more-vert-icon-sk>
  </span>
  <span>${getColHeader(firstCol)}</span>
  <sort-toggle @click=${e => (e.stopPropagation() && e.preventDefault())}
               key=name .currentKey=${ele._sort} .direction=${ele._dir}>
  </sort-toggle>
  ${col_selector(ele)}
</th>`;

const primaryOption = (key, ele) => html`
<div class=item ?selected=${ele._primaryKey === key}>
  <span class=key>${humanizePrimaryKey(key)}</span>
</div>`;

const secondaryOptions = (ele) => {
  if (!ele._primaryKey) {
    return '';
  }
  let values = ele._primaryMap[ele._primaryKey];
  if (!values) {
    return html`
<div class=information_only>
  Hmm... no preloaded values. Maybe try typing your filter like ${ele._primaryKey}:foo-bar in the
  above box and hitting enter.
</div>`;
  }
  values = filterPossibleValues(values, ele._primaryKey, ele._filterQuery);
  if (useNaturalSort(ele._primaryKey)) {
    values.sort(naturalSort);
  } else {
    values.sort();
  }
  return values.map((value) =>
    html`
<div class=item>
  <span class=value>${applyAlias(value, stripTag(ele._primaryKey))}</span>
  <span class=flex></span>
  <add-circle-icon-sk ?hidden=${ele._filters.indexOf(makeFilter(ele._primaryKey, value)) >= 0}
                      @click=${() => ele._addFilter(makeFilter(ele._primaryKey, value))}>
  </add-circle-icon-sk>
</div>`);
}

const filterChip = (filter, ele) => html`
<span class=chip>
  <span>${maybeApplyAlias(filter)}</span>
  <cancel-icon-sk @click=${() => ele._removeFilter(filter)}></cancel-icon-sk>
</span>`;

// can't use <select> and <option> because <option> strips out non-text
// (e.g. checkboxes)
const filters = (ele) => html`
<!-- primary key selector-->
<select-sk class="selector keys"
           @scroll=${ele._scrollCheck}
           @selection-changed=${ele._primaryKeyChanged}>
  ${ele._filteredPrimaryArr.map((key) => primaryOption(key, ele))}
</select-sk>
<!-- secondary value selector-->
<select-sk class="selector values" disabled>
  ${secondaryOptions(ele)}
</select-sk>`;

const options = (ele) => html`
<div class=options>
  <div class=verbose>
    <checkbox-sk ?checked=${ele._verbose}
                 @click=${ele._toggleVerbose}
                 label="Verbose Entries">
    </checkbox-sk>
  </div>
  <div class=picker>
    <div class=time>
      <span class=label title="Show only tasks created after this time">Start:</span>
      <input id=start_time></input>
    </div>
    <div class=time>
      <span class=label title="Show only tasks created before this time">End:</span>
      <input id=end_time ?disabled=${ele._now}></input>
      <checkbox-sk ?checked=${ele._now}
                   @click=${ele._toggleNow}>
      </checkbox-sk>
      <span>Now</span>
    </div>
  </div>
  <a href=${ele._matchingBotsLink()}>View Matching Bots</a>
  <button id=cancel_all
      ?disabled=${!ele.permissions.cancel_task}
      @click=${ele._promptMassCancel}>
    CANCEL ALL TASKS
  </button>
</div>`;

const summaryQueryRow = (ele, count) => html`
<tr>
  <td><a href=${ifDefined(ele._makeSummaryURL(count.filter))}>${count.label}</a>:</td>
  <td>${count.value}</td>
</tr>`;

const summary = (ele) => html`
<div class=summary>
  <div class=title @click=${ele._toggleAllStates}>
    Selected Tasks
    ${moreOrLess(ele._allStates)}
  </div>
  <table id=query_counts>
    ${summaryQueryRow(ele, {label: 'Displayed', value: ele._tasks.length})}
    ${ele._queryCounts.filter(ele._filterCounts.bind(ele))
                      .map((count) => summaryQueryRow(ele, count))}
  </table>
  <more-horiz-icon-sk @click=${ele._toggleAllStates}
                      ?hidden=${ele._allStates}>
  </more-horiz-icon-sk>
</div>`;

const header = (ele) => html`
<div class=header ?hidden=${!ele.loggedInAndAuthorized}>
  <div class=filter_box>
    <search-icon-sk></search-icon-sk>
    <input id=filter_search class=search type=text
           placeholder='Search filters or supply a filter
                        and press enter'
           @input=${ele._refilterPrimaryKeys}
           @keyup=${ele._filterSearch}>
    </input>
    <!-- The following div has display:block and divides the above and
         below inline-block groups-->
    <div></div>
    ${filters(ele)}

    ${options(ele)}
  </div>

    ${summary(ele)}
  </div>
</div>
<div class=chip_container>
  ${ele._filters.map((filter) => filterChip(filter, ele))}
</div>`;

const template = (ele) => html`
<swarming-app id=swapp
              client_id=${ele.client_id}
              ?testing_offline=${ele.testing_offline}>
  <header>
    <div class=title>Swarming Task List</div>
      <aside class=hideable>
        <a href=/>Home</a>
        <a href=/botlist>Bot List</a>
        <a href=/oldui/tasklist>Old Task List</a>
        <a href=/bot>Bot Page</a>
        <a href=/task>Task Page</a>
      </aside>
  </header>
  <!-- Allow clicking anywhere to dismiss the column selector-->
  <main @click=${e => ele._showColSelector && ele._toggleColSelector(e)}>
    <h2 class=message ?hidden=${ele.loggedInAndAuthorized}>${ele._message}</h2>

    ${header(ele)}

    <table class=task-table ?hidden=${!ele.loggedInAndAuthorized}>
      <thead>
        <tr>
          <tr>
          ${col_options(ele, ele._cols[0])}
          <!-- Slice off the first column so we can
               have a custom first box (including the widget to select columns).
            -->
          ${ele._cols.slice(1).map((col) => colHead(col, ele))}
        </tr>
        </tr>
      </thead>
      <tbody>${ele._sortTasks().map((task) => taskRow(task, ele))}</tbody>
    </table>

  </main>
  <footer></footer>
  <dialog-pop-over>
    <div class='cancel content'>
      <task-mass-cancel
          .auth_header=${ele.auth_header}
          .tags=${tagsOnly(ele._filters).map(stripTagFromFilter)}></task-mass-cancel>
      <button class=goback tabindex=0
              @click=${ele._closePopup}
              ?disabled=${ele._startedCanceling && !ele._finishedCanceling}>
        ${ele._startedCanceling ? 'DISMISS': "GO BACK - DON'T CANCEL ANYTHING"}
      </button>
    </div>
  </dialog-pop-over>
</swarming-app>`;

// How many items to load on the first load of tasks
// This is a relatively low number to make the initial page load
// seem snappier. After this, we can go up (see BATCH LOAD) to
// reduce the number of queries, since the user can expect to wait
// a bit more when their interaction (e.g. adding a filter) causes
// more data to be fetched.
const INITIAL_LOAD = 100;
// How many items to load on subsequent fetches.
// This number was picked from experience and experimentation.
const BATCH_LOAD = 200;

window.customElements.define('task-list', class extends SwarmingAppBoilerplate {

  constructor() {
    super(template);
    this._tasks = [];

    this._cols = [];
    this._dir = '';
    this._endTime = 0;
    this._filters = [];
    this._limit = 0; // _limit being 0 is a sentinel value for _fetch()
                     // We won't actually make a request if _limit is 0.
                     // So, we keep limit 0 until our params have been read in
                     // from the URL to avoid making a request until we are
                     // ready.
    this._now = true;
    this._primaryKey = '';
    this._sort = '';
    this._startTime = 0;
    this._verbose = false;
    // show it by default on small screens (e.g. mobile)
    this._allStates = onSmallScreen();

    this._stateChanged = stateReflector(
      /*getState*/() => {
        return {
          // provide empty values
          'c' : this._cols,
          'd' : this._dir,
          'et': this._endTime,
          'f' : this._filters,
          'k' : this._primaryKey,
          'n' : this._now,
          's' : this._sort,
          'st': this._startTime,
          'at': this._allStates,
          'v': this._verbose,
        }
    }, /*setState*/(newState) => {
      // default values if not specified.
      this._allStates = newState.at; // default to false
      this._cols = newState.c;
      if (!newState.c.length) {
        this._cols = ['name', 'state', 'bot', 'created_ts', 'pending_time',
                      'duration', 'pool-tag'];
      }
      this._dir = newState.d || 'desc';
      this._filters = handleLegacyFilters(newState.f); // default to []
      this._filters = legacyTags(this._filters);
      this._limit = INITIAL_LOAD;
      this._now = newState.n; // default to true
      this._primaryKey = newState.k; // default to ''
      // default to 24 hours ago, or if now is checked, update it to now
      if (this._now || !newState.et) {
        this._endTime = Date.now();
      } else {
        this._endTime = newState.et;
      }
      // default to 24 hours ago
      this._startTime = newState.st || floorSecond(Date.now() - 24*60*60*1000);
      this._sort = newState.s || 'created_ts';
      this._verbose = newState.v;         // default to false
      this._fetch();
      this.render();
      this._initDatePickers();
    });

    this._filteredPrimaryArr = [];

    this._possibleColumns = {};
    this._primaryMap = {};

    this._queryCounts = [
      {label: 'Total',        value: '...'},
      {label: 'Success',      value: '...', filter: 'COMPLETED_SUCCESS'},
      {label: 'Failure',      value: '...', filter: 'COMPLETED_FAILURE'},
      {label: 'Pending',      value: '...', filter: 'PENDING'},
      {label: 'Running',      value: '...', filter: 'RUNNING'},
      {label: 'Timed Out',    value: '...', filter: 'TIMED_OUT'},
      {label: 'Bot Died',     value: '...', filter: 'BOT_DIED'},
      {label: 'Deduplicated', value: '...', filter: 'DEDUPED'},
      {label: 'Expired',      value: '...', filter: 'EXPIRED'},
      {label: 'No Resource',  value: '...', filter: 'NO_RESOURCE'},
      {label: 'Canceled',     value: '...', filter: 'CANCELED'},
    ];

    this._message = 'You must sign in to see anything useful.';
    this._showColSelector = false;
    this._columnQuery = ''; // tracks what's typed into the input to search columns
    this._filterQuery = ''; // tracks what's typed into the input to search filters
    this._fetchController = null;
    this._knownDimensions = [];
    this._ignoreScrolls = 0;
  }

  connectedCallback() {
    super.connectedCallback();

    this._loginEvent = (e) => {
      this._fetch();
      this.render();
    };
    this.addEventListener('log-in', this._loginEvent);

    this._sortEvent = (e) => {
      this._sort = e.detail.key;
      this._dir = e.detail.direction;
      this._stateChanged();
      this.render();
    };
    this.addEventListener('sort-change', this._sortEvent);

    this._startedMassCancelingEvent = (e) => {
      this._startedCanceling = true;
      this._finishedCanceling = false;
      this.render();
    }
    this.addEventListener('tasks-canceling-started', this._startedMassCancelingEvent);
    this._finishedMassCancelingEvent = (e) => {
      this._startedCanceling = true;
      this._finishedCanceling = true;
      this.render();
    }
    this.addEventListener('tasks-canceling-finished', this._finishedMassCancelingEvent);
  }

  disconnectedCallback() {
    super.disconnectedCallback();

    this.removeEventListener('log-in', this._loginEvent);
    this.removeEventListener('sort-change', this._sortEvent);
    this.removeEventListener('tasks-canceling-started', this._startedMassCancelingEvent);
    this.removeEventListener('tasks-canceling-finished', this._finishedMassCancelingEvent);
  }

  _addFilter(filter) {
    if (this._filters.indexOf(filter) >= 0) {
      return;
    }
    this._filters.push(filter);
    this._stateChanged();
    // pre-filter what we have
    this._tasks = filterTasks(this._filters, this._tasks);
    // go fetch for all tasks that match the new filters.
    this._fetch();
    // render what we have now.  When _fetch() resolves it will
    // re-render.
    this.render();
  }

  _closePopup(e) {
    $$('dialog-pop-over', this).hide();
    this._startedCanceling = false;
    this._finishedCanceling = false;
    this.render();
  }

  _columnSearch(e) {
    if (e.key !== 'Enter') {
      return;
    }
    const input = $$('#column_search', this);
    const newCol = input.value.trim();
    if (!this._possibleColumns[newCol]) {
      errorMessage(`Column "${newCol}" is not valid.`, 5000);
      return;
    }
    input.value = '';
    this._columnQuery = '';
    if (this._cols.indexOf(newCol) !== -1) {
      this._refilterPossibleColumns();
      errorMessage(`Column "${newCol}" already displayed.`, 5000);
      return;
    }
    this._cols.push(newCol);
    this._stateChanged();
    this._refilterPossibleColumns();
  }

  _fetch() {
    if (!this.loggedInAndAuthorized || !this._limit) {
      return;
    }
    if (this._fetchController) {
      // Kill any outstanding requests that use the filters
      this._fetchController.abort();
    }
    // Make a fresh abort controller for each set of fetches. AFAIK, they
    // cannot be re-used once aborted.
    this._fetchController = new AbortController();
    const extra = {
      headers: {'authorization': this.auth_header},
      signal: this._fetchController.signal,
    };
    // Fetch the tasks
    this.app.addBusyTasks(1);
    let queryParams = listQueryParams(this._filters, {
      limit: this._limit,
      start: floorSecond(this._startTime),
      end: floorSecond(this._now ? Date.now() : this._endTime),
    });
    fetch(`/_ah/api/swarming/v1/tasks/list?${queryParams}`, extra)
      .then(jsonOrThrow)
      .then((json) => {
        this._tasks = [];
        const maybeLoadMore = (json) => {
          const tags = {};
          this._tasks = this._tasks.concat(processTasks(json.items, tags));
          appendPossibleColumns(this._possibleColumns, tags);
          appendPrimaryMap(this._primaryMap, tags);
          this._rebuildFilterables();

          this.render();
          // Special case: Don't load all the tasks when filters is empty to avoid
          // loading many many tasks unintentionally.
          if (this._filters.length && json.cursor) {
            this._limit = BATCH_LOAD;
            queryParams = listQueryParams(this._filters, {
              cursor: json.cursor,
              limit: this._limit,
              start: floorSecond(this._startTime),
              end: floorSecond(this._now ? Date.now() : this._endTime),
            });
            fetch(`/_ah/api/swarming/v1/tasks/list?${queryParams}`, extra)
              .then(jsonOrThrow)
              .then(maybeLoadMore)
              .catch((e) => this.fetchError(e, 'tasks/list (paging)'));
          } else {
            this.app.finishedTask();
          }
        }
        maybeLoadMore(json);
      })
      .catch((e) => this.fetchError(e, 'tasks/list'));

    this._fetchCounts(queryParams, extra);

    // fetch dimensions so we can fill out the filters.
    // We only need to do this once, because we don't expect it to
    // change (much) after the page has been loaded.
    if (!this._fetchedDimensions) {
      this._fetchedDimensions = true;
      this.app.addBusyTasks(1);
      const extra = {
        headers: {'authorization': this.auth_header},
        // No signal here because we shouldn't need to abort it.
        // This request does not depend on the filters.
      };
      fetch('/_ah/api/swarming/v1/bots/dimensions', extra)
      .then(jsonOrThrow)
      .then((json) => {
        appendPossibleColumns(this._possibleColumns, json.bots_dimensions);
        appendPrimaryMap(this._primaryMap, json.bots_dimensions);
        this._knownDimensions = json.bots_dimensions.map((d) => d.key);
        this._rebuildFilterables();

        this.render();
        this.app.finishedTask();
      })
      .catch((e) => this.fetchError(e, 'bots/dimensions'));
    }
  }

  _fetchCounts(queryParams, extra) {
    const states = ['COMPLETED_SUCCESS', 'COMPLETED_FAILURE', 'PENDING',
                    'RUNNING', 'TIMED_OUT', 'BOT_DIED', 'DEDUPED', 'EXPIRED',
                    'NO_RESOURCE', 'CANCELED'];

    this.app.addBusyTasks(1 + states.length);
    const totalPromise = fetch(`/_ah/api/swarming/v1/tasks/count?${queryParams}`, extra)
      .then(jsonOrThrow)
      .then((json) => {
        this.app.finishedTask();
        return json.count;
      })
      .catch((e) => this.fetchError(e, 'count/total'))
    this._queryCounts[0].value = html`${until(totalPromise, '...')}`;

    const stateRemoved = queryParams.replace(/state=.+?(&|$)/g, '');
    for (let i = 0; i < states.length; i++) {
      const promise = fetch(`/_ah/api/swarming/v1/tasks/count?${stateRemoved}&state=${states[i]}`, extra)
        .then(jsonOrThrow)
        .then((json) => {
          this.app.finishedTask();
          return json.count;
        })
        .catch((e) => this.fetchError(e, `count/${states[i]}`))
      this._queryCounts[1 + i].value = html`${until(promise, '...')}`;
    }
  }

  _filterSearch(e) {
    if (e.key !== 'Enter') {
      return;
    }
    const input = $$('#filter_search', this);
    let newFilter = input.value.trim();
    const idx = newFilter.indexOf(':');
    if (idx === -1) {
      errorMessage('Invalid filter.  Should be like "foo:bar"', 5000);
      return;
    }
    const key = newFilter.substring(0, idx);
    if (key !== 'state' && !key.endsWith('-tag')) {
      errorMessage(`Filters should only be on state or something-tag.
                   Auto-correcting ${key} to ${key}-tag.`, 6000);
      newFilter = key + '-tag' + newFilter.substring(idx);
    }
    input.value = '';
    this._filterQuery = '';
    this._primaryKey = '';
    if (this._filters.indexOf(newFilter) !== -1) {
      this._refilterPrimaryKeys();
      errorMessage(`Filter "${newFilter}" is already active`, 5000);
      return;
    }
    this._addFilter(newFilter);
    this._refilterPrimaryKeys();
  }

  _initDatePickers() {
    if (this._startEle || this._endEle) {
      return;
    }
    this._startEle = $$('#start_time', this);
    this._endEle = $$('#end_time', this);
    if (!this._startEle || !this._endEle) {
      // prevents errors in tests after element has been torn down.
      return;
    }
    flatpickr(this._startEle, {
      appendTo: $$('.picker', this), // otherwise, it leaks the calendar to <body>
      defaultDate: this._startTime,
      enableTime: true,
      onClose: (dates) => {
        this._startTime = dates[0].getTime(),
        this._stateChanged();
        this._fetch();
        this.render();
      },
      onOpen: () => {
        // prevent the end time picker from covering up the start time
        if (this._endEle._flatpickr.isOpen) {
          this._endEle._flatpickr.close();
        }
      }
    });
    flatpickr(this._endEle, {
      appendTo: $$('.picker', this), // otherwise, it leaks the calendar to <body>
      defaultDate: this._endTime,
      enableTime: true,
      onClose: (dates) => {
        this._endTime = dates[0].getTime(),
        this._stateChanged();
        this._fetch();
        this.render();
      }
    });
  }

  _filterCounts(count, idx) {
    if (this._allStates) {
      return true;
    }
    // The top 7 entries are ones that the users are most likely
    // going to care about by default. 7 uses the space most efficiently, with
    // the ellipsis to give an affordance for showing more.
    return idx < 7;
  }

  _makeSummaryURL(state) {
    if (!state) {
      return undefined;
    }
    // strip out any conflicting state filters.
    const withNewState = this._filters.filter((f) => {
      return !f.startsWith('state');
    });
    withNewState.push(`state:${state}`);
    const queryParams = query.fromObject({
          // provide empty values
          'c' : this._cols,
          'd' : this._dir,
          'et': this._endTime,
          'f' : withNewState,
          'k' : this._primaryKey,
          'n' : this._now,
          's' : this._sort,
          'st': this._startTime,
          'at': this._allStates,
          'v': this._verbose,
        });
    return `/tasklist?${queryParams}`;
  }

  _matchingBotsLink() {
    const cols = ['id', 'os', 'task', 'status'];

    const dimensions = this._filters.map((f) => {
      return f.replace('-tag', '');
    }).filter((f) => {
      const tag = f.split(':')[0];
      return tag !== 'state' && this._knownDimensions.indexOf(tag) !== -1;
    });

    for (const dim of dimensions) {
      const col = dim.split(':', 1)[0];
      if (cols.indexOf(col) === -1) {
        cols.push(col);
      }
    }
    return botListLink(dimensions, cols);
  }

  _primaryKeyChanged(e) {
    // e.detail.selection is the index of what was clicked
    this._primaryKey = this._filteredPrimaryArr[e.detail.selection];
    this._stateChanged();
    this.render();
  }

  _promptMassCancel(e) {
    $$('task-mass-cancel', this).show();
    $$('dialog-pop-over', this).show();
    $$('dialog-pop-over button.goback', this).focus();
  }

  _rebuildFilterables() {
    this._filteredPossibleColumns = filterPossibleColumns(Object.keys(this._possibleColumns),
                                                          this._columnQuery);

    this._primaryArr = Object.keys(this._primaryMap);
    this._primaryArr.sort();
    this._filteredPrimaryArr = this._primaryArr.slice();
  }

  _refilterPrimaryKeys(e) {
    this._filterQuery = $$('#filter_search', this).value;

    this._filteredPrimaryArr = filterPossibleKeys(this._primaryArr, this._primaryMap, this._filterQuery);
    // Update the selected to be the current one (if it is still with being
    // shown) or the first match.  This saves the user from having to click
    // the first result before seeing results.
    if (this._filterQuery && this._filteredPrimaryArr.length > 0 &&
        this._filteredPrimaryArr.indexOf(this._primaryKey) === -1) {
      this._primaryKey = this._filteredPrimaryArr[0];
      this._stateChanged();
    }

    this.render();
  }

  _refilterPossibleColumns(e) {
    const input = $$('#column_search', this);
    // If the column selector box is hidden, input will be null
    this._columnQuery = (input && input.value) || '';
    this._filteredPossibleColumns = filterPossibleColumns(Object.keys(this._possibleColumns),
                                                          this._columnQuery);
    sortPossibleColumns(this._filteredPossibleColumns, this._cols);
    this.render();
  }

  _removeFilter(filter) {
    const idx = this._filters.indexOf(filter);
    if (idx === -1) {
      return;
    }
    this._filters.splice(idx, 1);
    this._stateChanged();
    this._fetch();
    this.render();
  }

  render() {
    // Incorporate any data changes before rendering.
    sortColumns(this._cols);
    super.render();
    this._scrollToPrimaryKey();
  }

  _scrollCheck() {
    if (this._ignoreScrolls > 0) {
      this._ignoreScrolls--;
      return;
    }
    this._humanScrolledKeys = true;
  }

  _scrollToPrimaryKey() {
    // Especially on a page reload, the selected key won't be viewable.
    // This scrolls the little primary key box into view if it's not and,
    // since it runs every render, keeps it in view.
    // Do not use selectedKey.scrollIntoView since that will make the
    // whole page scroll and not just the selector box.
    //
    // We would like to avoid scrolling the primary key box if the user
    // has scrolled in that box. We cannot simply listen to scroll events
    // because calling element.scrollTo creates one scroll event that
    // happens asynchronously.
    // (of note, if 'smooth' scrolling behavior is specified, then an undetermined,
    // but finite amount of events are created, which is a bit of a mess)
    // So, anytime we trigger a scroll, we increment a counter and have
    // the scroll listener ignore that many events - if it hears any more
    // then the human must have scrolled.
    if (this._primaryKey && !this._humanScrolledKeys) {
      const keySelector = $$('.keys.selector', this);
      const selectedKey = $$('.item[selected]', keySelector);

      if (selectedKey) {
        this._ignoreScrolls++;
        keySelector.scrollTo({
          // 160 was found by experimentation with what looks good
          top: selectedKey.offsetTop - 160,
        });
      }
    }
  }

  /* sort the internal set of tasks based on the sort-toggle and direction
   * and returns it (for use in templating) */
  _sortTasks() {
    // All major supported browsers are now stable (or stable-ish)
    // https://stackoverflow.com/a/3027715
    this._tasks.sort((taskA, taskB) => {
      const sortOn = this._sort;
      if (!sortOn) {
        return 0;
      }
      let dir = 1;
      if (this._dir === 'desc') {
        dir = -1;
      }
      const sorter = specialSortMap[sortOn];
      if (sorter) {
        return sorter(dir, taskA, taskB);
      }
      // Default to a natural compare of the columns.
      let aCol = column(sortOn, taskA, this);
      if (aCol === 'none' || aCol === '--') {
        // put "none" at the bottom of the sort order
        aCol = 'zzz';
      }
      let bCol = column(sortOn, taskB, this);
      if (bCol === 'none' || bCol === '--') {
        // put "none" at the bottom of the sort order
        bCol = 'zzz';
      }
      return dir * naturalSort(aCol, bCol);
    });
    return this._tasks;
  }

  _toggleAllStates(e) {
    this._allStates = !this._allStates;
    this._stateChanged();
    this.render();
  }

  _toggleCol(e, col) {
    // This prevents a double event from happening (because of the
    // default 'click' event);
    e.preventDefault();
    // this prevents the click from bubbling up and being seen by the
    // <select-sk>
    e.stopPropagation();
    const idx = this._cols.indexOf(col);
    if (idx >= 0) {
      this._cols.splice(idx, 1);
    } else {
      this._cols.push(col);
    }
    this._refilterPossibleColumns();
    this._stateChanged();
    this.render();
  }

  _toggleColSelector(e) {
    e.preventDefault();
    // Prevent double click event from happening with the
    // click listener on <main>.
    e.stopPropagation();
    this._showColSelector = !this._showColSelector;
    this._refilterPossibleColumns(); // also renders
  }

  _toggleNow(e) {
    // This prevents a double event from happening.
    e.preventDefault();
    this._now = !this._now;
    this._stateChanged();
    this._fetch();
    this.render();
  }

  _toggleVerbose(e) {
    // This prevents a double event from happening.
    e.preventDefault();
    this._verbose = !this._verbose;
    this._stateChanged();
    this.render();
  }

});

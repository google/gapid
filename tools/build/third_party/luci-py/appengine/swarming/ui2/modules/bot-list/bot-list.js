// Copyright 2018 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

/** @module swarming-ui/modules/bot-list
 * @description <h2><code>bot-list</code></h2>
 *
 * <p>
 *  Bot List shows a filterable list of all bots in the fleet.
 * </p>
 *
 * <p>This is a top-level element.</p>
 *
 * @prop client_id - The Client ID for authenticating via OAuth.
 * @prop testing_offline - If true, the real OAuth flow won't be used.
 *    Instead, dummy data will be used. Ideal for local testing.
 */

import { $, $$ } from 'common-sk/modules/dom'
import { errorMessage } from 'elements-sk/errorMessage'
import { html, render } from 'lit-html'
import { ifDefined } from 'lit-html/directives/if-defined'
import { jsonOrThrow } from 'common-sk/modules/jsonOrThrow'
import naturalSort from 'javascript-natural-sort/naturalSort'
import * as query from 'common-sk/modules/query'
import { stateReflector } from 'common-sk/modules/stateReflector'

import 'elements-sk/checkbox-sk'
import 'elements-sk/icon/add-circle-icon-sk'
import 'elements-sk/icon/cancel-icon-sk'
import 'elements-sk/icon/more-vert-icon-sk'
import 'elements-sk/icon/search-icon-sk'
import 'elements-sk/select-sk'
import 'elements-sk/styles/buttons'
import '../bot-mass-delete'
import '../dialog-pop-over'
import '../sort-toggle'
import '../swarming-app'


import { applyAlias, handleLegacyFilters, maybeApplyAlias } from '../alias'
import { aggregateTemps, attribute, botLink, column, devices, dimensionsOnly,
         filterBots, forcedColumns, fromDimension, fromState, getColHeader,
         initCounts, listQueryParams, longestOrAll, processBots, processCounts,
         makePossibleColumns, processPrimaryMap, sortColumns,
         sortPossibleColumns, specialFilters, specialSortMap,
         useNaturalSort } from './bot-list-helpers'
import { filterPossibleColumns, filterPossibleKeys,
         filterPossibleValues, makeFilter } from '../queryfilter'
import { moreOrLess } from '../templates'
import { taskListLink } from '../util'

import SwarmingAppBoilerplate from '../SwarmingAppBoilerplate'

const colHead = (col, ele) => html`
<th>${getColHeader(col)}
  <sort-toggle .key=${col} .currentKey=${ele._sort} .direction=${ele._dir}>
  </sort-toggle>
</th>`;

const botCol = (col, bot, ele) => html`
<td>${column(col, bot, ele)}</td>`;

const botRow = (bot, ele) => html`
<tr class="bot-row ${ele._botClass(bot)}">
  ${ele._cols.map((col) => botCol(col, bot, ele))}
</tr>`;

const primaryOption = (key, ele) => html`
<div class=item ?selected=${ele._primaryKey === key}>
  <span class=key>${key}</span>
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
  <span class=value>${applyAlias(value, ele._primaryKey)}</span>
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
                 @click=${ele._toggleVerbose}>
    </checkbox-sk>
    <span>Verbose Entries</span>
  </div>
  <a href=${ele._matchingTasksLink()}>View Matching Tasks</a>
  <button id=delete_all
      ?disabled=${!ele.permissions.delete_bot}
      @click=${ele._promptMassDelete}>
    DELETE ALL DEAD BOTS
  </button>
</div>`;

const summaryFleetRow = (ele, count) => html`
<tr>
  <td><a href=${ifDefined(ele._makeSummaryURL(count, false))}>${count.label}</a>:</td>
  <td>${count.value}</td>
</tr>`;

const summaryQueryRow = (ele, count) => html`
<tr>
  <td><a href=${ifDefined(ele._makeSummaryURL(count, true))}>${count.label}</a>:</td>
  <td>${count.value}</td>
</tr>`;

const summary = (ele) => html`
<div class=summary ?hidden=${!ele._showFleetCounts}>
  <div class="fleet_header hider title" @click=${ele._toggleFleetsCount}>
    <span>Fleet</span>
    ${moreOrLess(ele._showFleetCounts)}
  </div>
  <table id=fleet_counts>
    ${ele._fleetCounts.map((count) => summaryFleetRow(ele, count))}
  </table>
</div>

<div class=summary>
  <div class="fleet_header shower title" ?hidden=${ele._showFleetCounts} @click=${ele._toggleFleetsCount}>
    <span>Fleet</span>
    ${moreOrLess(ele._showFleetCounts)}
  </div>

  <div class=title>Selected</div>
  <table id=query_counts>
    ${summaryQueryRow(ele, {label: 'Displayed', value: ele._bots.length})}
    ${ele._queryCounts.map((count) => summaryQueryRow(ele, count))}
  </table>
</div>`;

const header = (ele) => html`
<div class=header>
  <div class=filter_box ?hidden=${!ele.loggedInAndAuthorized}>
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

const columnOption = (key, ele) => html`
<div class=item>
  <span class=key>${key}</span>
  <span class=flex></span>
  <checkbox-sk ?checked=${ele._cols.indexOf(key) >= 0}
               ?disabled=${forcedColumns.indexOf(key) >= 0}
               @click=${(e) => ele._toggleCol(e, key)}
               @keypress=${(e) => ele._toggleCol(e, key)}>
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

const col_options = (ele) => html`
<!-- Put the click action here to make it bigger, especially for mobile.-->
<th class=col_options @click=${ele._toggleColSelector}>
  <span class=show_widget>
    <more-vert-icon-sk tabindex=0 @keypress=${ele._toggleColSelector}></more-vert-icon-sk>
  </span>
  <span>Bot Id</span>
  <sort-toggle @click=${e => (e.stopPropagation() && e.preventDefault())}
               key=id .currentKey=${ele._sort} .direction=${ele._dir}>
  </sort-toggle>
  ${col_selector(ele)}
</th>`;

const template = (ele) => html`
<swarming-app id=swapp
              client_id=${ele.client_id}
              ?testing_offline=${ele.testing_offline}>
  <header>
    <div class=title>Swarming Bot List</div>
      <aside class=hideable>
        <a href=/>Home</a>
        <a href=/oldui/botlist>Old Bot List</a>
        <a href=/tasklist>Task List</a>
        <a href=/bot>Bot Page</a>
        <a href=/task>Task Page</a>
      </aside>
  </header>
  <!-- Allow clicking anywhere to dismiss the column selector-->
  <main @click=${e => ele._showColSelector && ele._toggleColSelector(e)}>
    <h2 class=message ?hidden=${ele.loggedInAndAuthorized}>${ele._message}</h2>

    ${ele.loggedInAndAuthorized ? header(ele): ''}

    <table class=bot-table ?hidden=${!ele.loggedInAndAuthorized}>
      <thead>
        <tr>
          ${col_options(ele)}
          <!-- Slice off the first column (which is always 'id') so we can
               have a custom first box (including the widget to select columns).
            -->
          ${ele._cols.slice(1).map((col) => colHead(col, ele))}
        </tr>
      </thead>
      <tbody>${ele._sortBots().map((bot) => botRow(bot,ele))}</tbody>
    </table>
    <button ?hidden=${!ele.loggedInAndAuthorized || !!ele._filters.length || ele._showAll}
            @click=${ele._forceShowAll}>
      Show All
    </button>
  </main>
  <footer></footer>
  <dialog-pop-over>
    <div class='delete content'>
      <bot-mass-delete .auth_header=${ele.auth_header}
                       .dimensions=${dimensionsOnly(ele._filters)}>
      </bot-mass-delete>
      <button class=goback tabindex=0
              @click=${ele._closePopup}
              ?disabled=${ele._startedDeleting && !ele._finishedDeleting}>
        ${ele._startedDeleting ? 'DISMISS': "GO BACK - DON'T DELETE ANYTHING"}
      </button>
    </div>
  </dialog-pop-over>
</swarming-app>`;

// How many items to load on the first load of bots
// This is a relatively low number to make the initial page load
// seem snappier. After this, we can go up (see BATCH LOAD) to
// reduce the number of queries, since the user can expect to wait
// a bit more when their interaction (e.g. adding a filter) causes
// more data to be fetched.
const INITIAL_LOAD = 100;
// How many items to load on subsequent fetches.
// This number was picked from experience and experimentation.
const BATCH_LOAD = 200;

window.customElements.define('bot-list', class extends SwarmingAppBoilerplate {

  constructor() {
    super(template);
    this._bots = [];
    // Set empty values to allow empty rendering while we wait for
    // stateReflector (which triggers on DomReady). Additionally, these values
    // help stateReflector with types.
    this._cols = [];
    this._dir = '';
    this._filters = [];
    this._limit = 0; // _limit being 0 is a sentinel value for _fetch()
                     // We won't actually make a request if _limit is 0.
                     // So, we keep limit 0 until our params have been read in
                     // from the URL to avoid making a request until we are
                     // ready.
    this._primaryKey = '';
    this._showAll = false;
    this._showFleetCounts = false;
    this._sort = '';
    this._verbose = false;

    this._fleetCounts = initCounts();
    this._queryCounts = initCounts();

    this._stateChanged = stateReflector(
      /*getState*/() => {
        return {
          // provide empty values
          'c': this._cols,
          'd': this._dir,
          'e': this._showFleetCounts, // 'e' because 'f', 'l', are taken
          'f': this._filters,
          'k': this._primaryKey,
          's': this._sort,
          'show_all': this._showAll,
          'v': this._verbose,
        }
    }, /*setState*/(newState) => {
      // default values if not specified.
      this._cols = newState.c;
      if (!newState.c.length) {
        this._cols = ['id', 'task', 'os', 'status'];
      }
      this._dir = newState.d || 'asc';
      this._filters = handleLegacyFilters(newState.f); // default to []
      this._primaryKey = newState.k; // default to ''
      this._sort = newState.s || 'id';
      this._verbose = newState.v;         // default to false
      this._showFleetCounts = newState.e; // default to false
      this._limit = INITIAL_LOAD;
      this._showAll = newState.show_all; // default to false
      this._fetch();
      this.render();
    });

    /** _primaryArr: Array<String>, the display order of the primaryKeys, that is,
        anything that can be searched/filtered by. This should not be changed
        after it is set initially - it is the ground truth of primary keys.
     */
    this._primaryArr = [];
    /** _filteredPrimaryArr: Array<String>, the current, filtered display order
        of the primaryKeys. This can be mutated when filtering/sorting.
     */
    this._filteredPrimaryArr = [];
    /** _possibleColumns: Array<String>, Any valid columns that can be sorted by.
        This is a superset of _primaryArr, with some extra things that are useful,
        but can't be filtered by (the API only supports filtering by dimensions
        and a few special items). This should not be changed after it is set
        initially - it is the ground truth of columns.
     */
    this._possibleColumns = [];
    /** _filteredPossibleColumns: Array<String>, the current, filtered display
        order of columns that the user can select using the col_selector.
     */
    this._filteredPossibleColumns = [];
    /** _primaryMap: Object, a mapping of primary keys to secondary items.
        The primary keys are things that can be columns or sorted by.  The
        primary values (aka the secondary items) are things that can be filtered
        on. Primary consists of dimensions and state.  Secondary contains the
        values primary things can be.*/
    this._primaryMap = {};
    this._message = 'You must sign in to see anything useful.';
    this._showColSelector = false;
    this._columnQuery = ''; // tracks what's typed into the input to search columns
    this._filterQuery = ''; // tracks what's typed into the input to search filters
    // Allows us to abort fetches that are tied to filters when filters change.
    this._fetchController = null;
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

    this._startedMassDeletingEvent = (e) => {
      this._startedDeleting = true;
      this._finishedDeleting = false;
      this.render();
    }
    this.addEventListener('bots-deleting-started', this._startedMassDeletingEvent);
    this._finishedMassDeletingEvent = (e) => {
      this._startedDeleting = true;
      this._finishedDeleting = true;
      this.render();
    }
    this.addEventListener('bots-deleting-finished', this._finishedMassDeletingEvent);
  }

  disconnectedCallback() {
    super.disconnectedCallback();

    this.removeEventListener('log-in', this._loginEvent);
    this.removeEventListener('sort-change', this._sortEvent);
    this.removeEventListener('bots-deleting-started', this._startedMassDeletingEvent);
    this.removeEventListener('bots-deleting-finished', this._finishedMassDeletingEvent);
  }

  _addFilter(filter) {
    if (this._filters.indexOf(filter) >= 0) {
      return;
    }
    this._filters.push(filter);
    this._stateChanged();
    // pre-filter what we have
    this._bots = filterBots(this._filters, this._bots);
    // go fetch for all the bots that match the new filters.
    this._fetch();
    // render what we have now.  When _fetch() resolves it will
    // re-render.
    this.render();
  }

  _botClass(bot) {
    let classes = '';
    if (bot.is_dead) {
      classes += 'dead ';
    }
    if (bot.quarantined) {
      classes += 'quarantined ';
    }
    if (bot.maintenance_msg) {
      classes += 'maintenance ';
    }
    if (bot.version !== this.server_details.bot_version) {
      classes += 'old_version';
    }
    return classes;
  }

  _closePopup(e) {
    $$('dialog-pop-over', this).hide();
    this._startedDeleting = false;
    this._finishedDeleting = false;
  }

  _columnSearch(e) {
    if (e.key !== 'Enter') {
      return;
    }
    const input = $$('#column_search', this);
    const newCol = input.value.trim();
    if (this._possibleColumns.indexOf(newCol) === -1) {
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
    // limit of 0 is a sentinel value. See constructor for more details.
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
    // Fetch the bots
    this.app.addBusyTasks(1);
    let queryParams = listQueryParams(this._filters, this._limit);
    fetch(`/_ah/api/swarming/v1/bots/list?${queryParams}`, extra)
      .then(jsonOrThrow)
      .then((json) => {
        this._bots = [];
        const maybeLoadMore = (json) => {
          this._bots = this._bots.concat(processBots(json.items));
          this.render();
          // Special case: Don't load all the bots when filters is empty to avoid
          // loading many many bots unintentionally. A user can over-ride this
          // with the showAll button.
          if ((this._filters.length || this._showAll) && json.cursor) {
            this._limit = BATCH_LOAD;
            queryParams = listQueryParams(this._filters, this._limit, json.cursor);
            fetch(`/_ah/api/swarming/v1/bots/list?${queryParams}`, extra)
              .then(jsonOrThrow)
              .then(maybeLoadMore)
              .catch((e) => this.fetchError(e, 'bots/list (paging)'));
          } else {
            this.app.finishedTask();
          }
        }
        maybeLoadMore(json);
      })
      .catch((e) => this.fetchError(e, 'bots/list'));

    this.app.addBusyTasks(1);
    // We can re-use the query params from listQueryParams because
    // the backend will ignore those it doesn't understand (e.g limit
    // and is_dead, etc).
    fetch('/_ah/api/swarming/v1/bots/count?' + queryParams, extra)
      .then(jsonOrThrow)
      .then((json) => {
        this._queryCounts = processCounts(this._queryCounts, json);
        this.render();
        this.app.finishedTask();
      })
      .catch((e) => this.fetchError(e, 'bots/count (query)'));

    // We only need to do this once, because we don't expect it to
    // change (much) after the page has been loaded.
    if (!this._fleetCounts._queried) {
      this._fleetCounts._queried = true;
      this.app.addBusyTasks(1);
      fetch('/_ah/api/swarming/v1/bots/count', extra)
        .then(jsonOrThrow)
        .then((json) => {
          this._fleetCounts = processCounts(this._fleetCounts, json);
          this.render();
          this.app.finishedTask();
        })
        .catch((e) => this.fetchError(e, 'bots/count (fleet)'));
    }

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
        this._primaryMap = processPrimaryMap(json.bots_dimensions);
        this._possibleColumns = makePossibleColumns(json.bots_dimensions);
        this._filteredPossibleColumns = this._possibleColumns.slice();
        this._primaryArr = Object.keys(this._primaryMap);
        this._primaryArr.sort();
        this._filteredPrimaryArr = this._primaryArr.slice();
        this._refilterPossibleColumns(); // calls render
        this.app.finishedTask();
      })
      .catch((e) => this.fetchError(e, 'bots/dimensions'));
    }
  }

  _filterSearch(e) {
    if (e.key !== 'Enter') {
      return;
    }
    const input = $$('#filter_search', this);
    const newFilter = input.value.trim();
    if (newFilter.indexOf(':') === -1) {
      errorMessage('Invalid filter.  Should be like "foo:bar"', 5000);
      return;
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

  _forceShowAll() {
    this._showAll = true;
    this._stateChanged();
    this._fetch();
  }

  _makeSummaryURL(newFilter, preserveOthers) {
    if (!newFilter || newFilter.label === 'Displayed' || newFilter.label === 'All') {
      // no link
      return undefined;
    }
    const label = newFilter.label.toLowerCase();
    let filterStr = 'status:' + label;
    if (label === 'busy' || label === 'idle') {
      filterStr = 'task:' + label;
    }
    const currentURL = new URL(window.location.href);
    if (preserveOthers) {
      if (currentURL.searchParams.getAll('f').indexOf(filterStr) !== -1) {
        // The filter is already on the list.
        return undefined;
      }
      currentURL.searchParams.append('f', filterStr);
      return currentURL.href;
    }

    const params = {
      s: [this._sort],
      c: this._cols,
      v: [this._verbose],
      f: [filterStr],
      e: [true], // show fleet
    };

    return currentURL.pathname + '?' + query.fromParamSet(params);
  }

  _matchingTasksLink() {
    const cols = ['name', 'state', 'created_ts'];
    const dimensionFilters = this._filters.filter((f) => {
      // Strip out non-dimensions like "is_mp_bot"
      return !specialFilters[f.split(':')[0]];
    });
    // Add any dimensions as columns, so they are shown by default
    for (const f of dimensionFilters) {
      const col = f.split(':', 1)[0];
      if (cols.indexOf(col) === -1) {
        cols.push(col);
      }
    }
    return taskListLink(dimensionFilters, cols);
  }

  _primaryKeyChanged(e) {
    this._primaryKey = this._filteredPrimaryArr[e.detail.selection];
    this._stateChanged();
    this.render();
  }

  _promptMassDelete(e) {
    $$('bot-mass-delete', this).show();
    $$('dialog-pop-over', this).show();
    $$('dialog-pop-over button.goback', this).focus();
  }

  _refilterPossibleColumns(e) {
    const input = $$('#column_search', this);
    // If the column selector box is hidden, input will be null
    this._columnQuery = (input && input.value) || '';
    this._filteredPossibleColumns = filterPossibleColumns(this._possibleColumns, this._columnQuery);
    sortPossibleColumns(this._filteredPossibleColumns, this._cols);
    this.render();
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

  /* sort the internal set of bots based on the sort-toggle and direction
   * and returns it (for use in templating) */
  _sortBots() {
    // All major supported browsers are now stable (or stable-ish)
    // https://stackoverflow.com/a/3027715
    this._bots.sort((botA, botB) => {
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
        return sorter(dir, botA, botB);
      }
      // Default to a natural compare of the columns.
      let aCol = column(sortOn, botA, this);
      if (aCol === 'none') {
        // put "none" at the bottom of the sort order
        aCol = 'zzz';
      }
      let bCol = column(sortOn, botB, this);
      if (bCol === 'none') {
        // put "none" at the bottom of the sort order
        bCol = 'zzz';
      }
      return dir * naturalSort(aCol, bCol);
    });
    return this._bots;
  }

  _toggleCol(e, col) {
    if (forcedColumns.indexOf(col) >= 0) {
      return;
    }
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

  _toggleFleetsCount(e) {
    e.preventDefault();
    e.stopPropagation();
    this._showFleetCounts = !this._showFleetCounts;
    this._stateChanged();
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

// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import 'modules/task-list'

describe('task-list', function() {
  // Instead of using import, we use require. Otherwise,
  // the concatenation trick we do doesn't play well with webpack, which would
  // leak dependencies (e.g. bot-list's 'column' function to task-list) and
  // try to import things multiple times.
  const { deepCopy } = require('common-sk/modules/object');
  const { $, $$ } = require('common-sk/modules/dom');
  const { childrenAsArray, customMatchers, expectNoUnmatchedCalls, getChildItemWithText, mockAppGETs } = require('modules/test_util');
  const { fetchMock, MATCHED, UNMATCHED } = require('fetch-mock');

  const { column, filterTasks, getColHeader, listQueryParams, processTasks } = require('modules/task-list/task-list-helpers');
  const { tasks_20 } = require('modules/task-list/test_data');
  const { fleetDimensions } = require('modules/bot-list/test_data');

  beforeEach(function() {
    jasmine.addMatchers(customMatchers);
    // Clear out any query params we might have to not mess with our current state.
    history.pushState(null, '', window.location.origin + window.location.pathname + '?');
  });

  beforeEach(function() {
    // These are the default responses to the expected API calls (aka 'matched').
    // They can be overridden for specific tests, if needed.
    mockAppGETs(fetchMock, {
      cancel_task: false,
    });

    fetchMock.get('glob:/_ah/api/swarming/v1/tasks/list?*', tasks_20);
    fetchMock.get('/_ah/api/swarming/v1/bots/dimensions', fleetDimensions);
    fetchMock.get('glob:/_ah/api/swarming/v1/tasks/count?*', {'count': 12345});

    // Everything else
    fetchMock.catch(404);
  });

  afterEach(function() {
    // Completely remove the mocking which allows each test
    // to be able to mess with the mocked routes w/o impacting other tests.
    fetchMock.reset();
  });

  // A reusable HTML element in which we create our element under test.
  const container = document.createElement('div');
  document.body.appendChild(container);

  afterEach(function() {
    container.innerHTML = '';
  });

  beforeEach(function() {
    // Fix the time so all of our relative dates work.
    // Note, this turns off the default behavior of setTimeout and related.
    jasmine.clock().install();
    jasmine.clock().mockDate(new Date(Date.UTC(2018, 11, 19, 16, 46, 22, 1234)));
  });

  afterEach(function() {
    jasmine.clock().uninstall();
  });

  // calls the test callback with one element 'ele', a created <swarming-index>.
  // We can't put the describes inside the whenDefined callback because
  // that doesn't work on Firefox (and possibly other places).
  function createElement(test) {
    return window.customElements.whenDefined('task-list').then(() => {
      container.innerHTML = `<task-list client_id=for_test testing_offline=true></task-list>`;
      expect(container.firstElementChild).toBeTruthy();
      test(container.firstElementChild);
    });
  }

  function userLogsIn(ele, callback) {
    // The swarming-app emits the 'busy-end' event when all pending
    // fetches (and renders) have resolved.
    let ran = false;
    ele.addEventListener('busy-end', (e) => {
      if (!ran) {
        ran = true; // prevent multiple runs if the test makes the
                    // app go busy (e.g. if it calls fetch).
        callback();
      }
    });
    const login = $$('oauth-login', ele);
    login._logIn();
    fetchMock.flush();
  }

  // convenience function to save indentation and boilerplate.
  // expects a function test that should be called with the created
  // <task-list> after the user has logged in.
  function loggedInTasklist(test) {
    createElement((ele) => {
      userLogsIn(ele, () => {
        test(ele);
      });
    });
  }

//===============TESTS START====================================

  describe('html structure', function() {
    it('contains swarming-app as its only child', function(done) {
      createElement((ele) => {
        expect(ele.children.length).toBe(1);
        expect(ele.children[0].tagName).toBe('swarming-app'.toUpperCase());
        done();
      });
    });

    describe('when not logged in', function() {
      it('tells the user they should log in', function(done) {
        createElement((ele) => {
          const loginMessage = $$('swarming-app>main .message', ele);
          expect(loginMessage).toBeTruthy();
          expect(loginMessage).not.toHaveAttribute('hidden', 'Message should not be hidden');
          expect(loginMessage.textContent).toContain('must sign in');
          done();
        });
      });
      it('does not display filters or tasks', function(done) {
        createElement((ele) => {
          const taskTable = $$('.task-table', ele);
          expect(taskTable).toBeTruthy();
          expect(taskTable).toHaveAttribute('hidden', '.task-table should be hidden');
          expect($$('.header', ele)).toHaveAttribute('hidden', 'no filters seen');
          done();
        });
      });
    }); //end describe('when not logged in')

    describe('when logged in as unauthorized user', function() {

      function notAuthorized() {
        // overwrite the default fetchMock behaviors to have everything return 403.
        fetchMock.get('/_ah/api/swarming/v1/server/details', 403,
                      { overwriteRoutes: true });
        fetchMock.get('/_ah/api/swarming/v1/server/permissions', {},
                      { overwriteRoutes: true });
        fetchMock.get('glob:/_ah/api/swarming/v1/tasks/list?*', 403,
                      { overwriteRoutes: true });
        fetchMock.get('/_ah/api/swarming/v1/bots/dimensions', 403,
                      { overwriteRoutes: true });
        fetchMock.get('/_ah/api/swarming/v1/tasks/count', 403,
                      { overwriteRoutes: true });
      }

      beforeEach(notAuthorized);

      it('tells the user they should change accounts', function(done) {
        loggedInTasklist((ele) => {
          const loginMessage = $$('swarming-app>main .message', ele);
          expect(loginMessage).toBeTruthy();
          expect(loginMessage).not.toHaveAttribute('hidden', 'Message should not be hidden');
          expect(loginMessage.textContent).toContain('different account');
          done();
        });
      });
      it('does not display filters or tasks', function(done) {
        loggedInTasklist((ele) => {
          const taskTable = $$('.task-table', ele);
          expect(taskTable).toBeTruthy();
          expect(taskTable).toHaveAttribute('hidden', '.task-table should be hidden');
          expect($$('.header', ele)).toHaveAttribute('hidden', 'no filters seen');
          done();
        });
      });
    }); // end describe('when logged in as unauthorized user')

    describe('when logged in as user (not admin)', function() {

      describe('default landing page', function() {
        it('displays whatever tasks show up', function(done) {
          loggedInTasklist((ele) => {
            const rows = $('.task-table .task-row', ele);
            expect(rows).toBeTruthy();
            expect(rows.length).toBe(20, '(num taskRows)');
            done();
          });
        });

        it('shows the default set of columns', function(done) {
          loggedInTasklist((ele) => {
            // ensure sorting is deterministic.
            ele._sort = 'created_ts';
            ele._dir = 'desc';
            ele._verbose = false;
            ele.render();

            const colHeaders = $('.task-table thead th', ele);
            expect(colHeaders).toBeTruthy();
            expect(colHeaders.length).toBe(7, '(num colHeaders)');
            expect(colHeaders[0].innerHTML).toContain('<more-vert-icon-sk');
            expect(colHeaders[0]).toMatchTextContent('Task Name');
            expect(colHeaders[1]).toMatchTextContent('Created On');
            expect(colHeaders[2]).toMatchTextContent('Time Spent Pending');
            expect(colHeaders[3]).toMatchTextContent('Duration');
            expect(colHeaders[4]).toMatchTextContent('Bot Assigned');
            expect(colHeaders[5]).toMatchTextContent('pool (tag)');
            expect(colHeaders[6]).toMatchTextContent('state (of task)');

            const rows = $('.task-table .task-row', ele);
            expect(rows).toBeTruthy();
            expect(rows.length).toBe(20, '20 rows');

            const cells = $('.task-table .task-row td', ele);
            expect(cells).toBeTruthy();
            expect(cells.length).toBe(7 * 20, '7 columns * 20 rows');
            // little helper for readability
            const cell = (r, c) => cells[7*r+c];

            expect(rows[0]).toHaveClass('failed_task');
            expect(rows[0]).not.toHaveClass('exception');
            expect(rows[0]).not.toHaveClass('pending_task');
            expect(rows[0]).not.toHaveClass('bot_died');
            expect(cell(0, 0)).toMatchTextContent('Build-Win-Clang-x86_64-Debug-ANGLE');
            expect(cell(0, 0).innerHTML).toContain('<a ', 'has a link');
            expect(cell(0, 0).innerHTML).toContain('href="/task?id=41e031b2c8b46710"', 'link is correct');
            expect(cell(0, 2)).toMatchTextContent('2.36s'); // pending
            expect(cell(0, 4)).toMatchTextContent('skia-gce-610');
            expect(cell(0, 4).innerHTML).toContain('<a ', 'has a link');
            expect(cell(0, 4).innerHTML).toContain('href="/bot?id=skia-gce-610"', 'link is correct');
            expect(cell(0, 5)).toMatchTextContent('Skia');
            expect(cell(0, 6)).toMatchTextContent('COMPLETED (FAILURE)');

            expect(rows[2]).not.toHaveClass('failed_task');
            expect(rows[2]).not.toHaveClass('exception');
            expect(rows[2]).not.toHaveClass('pending_task');
            expect(rows[2]).not.toHaveClass('bot_died');
            expect(cell(2, 2)).toMatchTextContent('--'); // pending

            expect(rows[4]).not.toHaveClass('failed_task');
            expect(rows[4]).toHaveClass('exception');
            expect(rows[4]).not.toHaveClass('pending_task');
            expect(rows[4]).not.toHaveClass('bot_died');

            expect(rows[5]).not.toHaveClass('failed_task');
            expect(rows[5]).not.toHaveClass('exception');
            expect(rows[5]).toHaveClass('pending_task');
            expect(rows[5]).not.toHaveClass('bot_died');
            expect(cell(5, 3).textContent).toContain('12m 54s*'); // duration

            expect(rows[14]).not.toHaveClass('failed_task');
            expect(rows[14]).not.toHaveClass('exception');
            expect(rows[14]).not.toHaveClass('pending_task');
            expect(rows[14]).toHaveClass('bot_died');

            done();
          });
        });

        it('supplies past 24 hours for the time pickers', function(done) {
          loggedInTasklist((ele) => {
            const start = $$('#start_time', ele);
            expect(start).toBeTruthy();
            expect(start.disabled).toBeFalsy();
            expect(start.value).toBe('2018-12-18 11:46', '(start time is 24 hours ago)');

            const end = $$('#end_time', ele);
            expect(end).toBeTruthy();
            expect(end.disabled).toBeTruthy();
            expect(end.value).toBe('2018-12-19 11:46', '(end time is now)');

            const checkbox = $$('.picker checkbox-sk', ele);
            expect(checkbox).toBeTruthy();
            expect(checkbox.checked).toBeTruthy(); // defaults to using now
            done();
          });
        });

        it('shows the counts of the first 7 states', function(done) {
          loggedInTasklist((ele) => {
            ele.render();

            const countRows = $('#query_counts tr', ele);
            expect(countRows).toBeTruthy();
            expect(countRows.length).toBe(1+7, '(num counts, displayed + 7 states)');

            expect(countRows[0]).toMatchTextContent('Displayed: 20');

            expect(countRows[3].innerHTML).toContain('<a ', 'contains a link');
            const link = $$('a', countRows[3]);
            expect(link.href).toContain('/tasklist?at=false&c=name&c=created_ts&c=pending_time&' +
                  'c=duration&c=bot&c=pool-tag&c=state&d=desc&et=1545237983234&' +
                  'f=state%3ACOMPLETED_FAILURE&k=&n=true&s=created_ts&st=1545151583000&v=false');

            // The true on flush waits for res.json() to resolve too
            fetchMock.flush(true).then(() => {
              expect(countRows[5]).toMatchTextContent('Running: 12345');
              done();
            });

          });
        });

        it('shows aliases on filter chips', function(done) {
          loggedInTasklist((ele) => {
            ele._filters=['cpu-tag:x86-64-Haswell_GCE', 'gpu-tag:10de:1cb3-415.27',
                          'device_type-tag:flo']
            ele.render();

            const chips = $('.chip_container .chip', ele);
            expect(chips).toBeTruthy();
            expect(chips.length).toBe(3, '3 filters, 3 chips');

            // They are displayed in order, so check content
            expect(chips[0]).toMatchTextContent('cpu-tag:x86-64-Haswell_GCE');
            expect(chips[1]).toMatchTextContent('gpu-tag:NVIDIA Quadro P400 (10de:1cb3-415.27)');
            expect(chips[2]).toMatchTextContent('device_type-tag:Nexus 7 [2013] (flo)');
            done();
          });
        });
      }); // end describe('default landing page')
    });// end describe('when logged in as user')

  }); // end describe('html structure')

  describe('dynamic behavior', function() {
    it('updates the sort-toggles based on the current sort direction', function(done) {
      loggedInTasklist((ele) => {
        ele._sort = 'name';
        ele._dir = 'desc';
        ele.render();

        const sortToggles = $('.task-table thead sort-toggle', ele);
        expect(sortToggles).toBeTruthy();
        expect(sortToggles.length).toBe(7, '(num sort-toggles)');

        expect(sortToggles[0].key).toBe('name');
        expect(sortToggles[0].currentKey).toBe('name');
        expect(sortToggles[0].direction).toBe('desc');
        // spot check one of the other ones
        expect(sortToggles[5].key).toBe('pool-tag');
        expect(sortToggles[5].currentKey).toBe('name');
        expect(sortToggles[5].direction).toBe('desc');

        ele._sort = 'created_ts';
        ele._dir = 'asc';
        ele.render();

        expect(sortToggles[0].key).toBe('name');
        expect(sortToggles[0].currentKey).toBe('created_ts');
        expect(sortToggles[0].direction).toBe('asc');

        expect(sortToggles[1].key).toBe('created_ts');
        expect(sortToggles[1].currentKey).toBe('created_ts');
        expect(sortToggles[1].direction).toBe('asc');
        done();
      });
    });
    // This is done w/o interacting with the sort-toggles because that is more
    // complicated with adding the event listener and so on.
    it('can stable sort', function(done) {
      loggedInTasklist((ele) => {
        ele._verbose = false;
        // First sort in descending created_ts order
        ele._sort = 'created_ts';
        ele._dir = 'desc';
        ele.render();
        // next sort in ascending pool-tag
        ele._sort = 'pool-tag';
        ele._dir = 'asc';
        ele.render();

        const actualIDOrder = ele._tasks.map((t) => t.task_id);
        const actualPoolOrder = ele._tasks.map((t) => column('pool-tag', t, ele));

        expect(actualIDOrder).toEqual(['41e0284bc3ef4f10', '41e023035ecced10',
            '41e0222076a33010', '41e020504d0a5110', '41e0204f39d06210',
            '41e01fe02b981410',     '41dfffb4970ae410', '41e0284bf01aef10',
            '41e0222290be8110', '41e031b2c8b46710',     '41dfffb8b1414b10',
            '41dfa79d3bf29010', '41df677202f20310', '41e019d8b7aa2f10',
            '41e015d550464910', '41e0310fe0b7c410', '41e0182a00fcc110',
            '41e016dc85735b10',     '41dd3d950bb52710', '41dd3d9564402e10']);
        expect(actualPoolOrder).toEqual(['Chrome', 'Chrome', 'Chrome',
            'Chrome', 'Chrome', 'Chrome', 'Chrome', 'Chrome-CrOS-VM', 'Chrome-GPU',
            'Skia', 'Skia', 'Skia', 'Skia', 'fuchsia.tests', 'fuchsia.tests',
            'luci.chromium.ci', 'luci.chromium.ci', 'luci.chromium.ci',
            'luci.fuchsia.try', 'luci.fuchsia.try']);
        done();
      });
    });

    it('can sort durations correctly', function(done) {
      loggedInTasklist((ele) => {
        ele._verbose = false;
        ele._sort = 'duration';
        ele._dir = 'asc';
        ele.render();

        const actualDurationsOrder = ele._tasks.map((t) => t.human_duration.trim());

        expect(actualDurationsOrder).toEqual(['0.62s', '2.90s', '17.84s', '1m 38s',
            '2m  1s', '2m  1s', '12m 54s*', '12m 55s*', '1h  9m 47s', '2h 16m 15s',
            '--', '--', '--', '--', '--', '--', '--', '--', '--', '--']);

        ele._verbose = false;
        ele._sort = 'pending_time';
        ele._dir = 'asc';
        ele.render();

        const actualPendingOrder = ele._tasks.map((t) => t.human_pending_time.trim());

        expect(actualPendingOrder).toEqual(['0s', '0s', '0.63s', '0.66s', '0.72s', '2.35s',
          '2.36s', '2.58s', '5.74s', '8.21s', '24.58s', '1m 11s', '1m 17s', '5m  5s', '5m 36s',
          '11m 28s', '14m 54s*', '14m 55s*', '--', '--']);
        done();
      });
    });

    it('toggles columns by clicking on the boxes in the "column selector"', function(done) {
      loggedInTasklist((ele) => {
        ele._cols = ['name'];
        ele._showColSelector = true;
        ele.render();

        const keySelector = $$('.col_selector', ele);
        expect(keySelector).toBeTruthy();

        // click on first non checked checkbox.
        let keyToClick = null;
        let checkbox = null;
        for (let i = 0; i < keySelector.children.length; i++) {
          const child = keySelector.children[i];
          checkbox = $$('checkbox-sk', child);
          if (checkbox && !checkbox.checked) {
            keyToClick = checkbox.getAttribute('data-key');
            break;
          }
        }
        checkbox.click(); // click is synchronous, it returns after
                          // the clickHandler is run.
        // Check underlying data
        expect(ele._cols).toContain(keyToClick);
        // check the rendering changed
        let colHeaders = $('.task-table thead th');
        expect(colHeaders).toBeTruthy();
        expect(colHeaders.length).toBe(2, '(num colHeaders)');
        const expectedHeader = getColHeader(keyToClick);
        expect(colHeaders.map((c) => c.textContent.trim())).toContain(expectedHeader);

        // We have to find the checkbox again because the order
        // shuffles to keep selected ones on top.
        checkbox = null;
        for (let i = 0; i < keySelector.children.length; i++) {
          const child = keySelector.children[i];
          const key = $$('.key', child);
          if (key && key.textContent.trim() === getColHeader(keyToClick)) {
            checkbox = $$('checkbox-sk', child);
            break;
          }
        }
        expect(checkbox).toBeTruthy('We expected to find a checkbox with header '+
                                    getColHeader(keyToClick));

        // click it again
        checkbox.click();

        // Check underlying data
        expect(ele._cols).not.toContain(keyToClick);
        // check the rendering changed
        colHeaders = $('.task-table thead th');
        expect(colHeaders).toBeTruthy();
        expect(colHeaders.length).toBe(1, '(num colHeaders)');
        expect(colHeaders.map((c) => c.textContent.trim())).not.toContain(expectedHeader);
        done();
      });
    });

    it('shows values when a key row is selected', function(done) {
      loggedInTasklist((ele) => {
        ele._cols = ['name'];
        ele.render();
        let row = getChildItemWithText($$('.selector.keys'), 'device_type (tag)', ele);
        expect(row).toBeTruthy();
        row.click();
        expect(row.hasAttribute('selected')).toBeTruthy();
        expect(ele._primaryKey).toBe('device_type-tag');

        let valueSelector = $$('.selector.values');
        expect(valueSelector).toBeTruthy();
        let values = childrenAsArray(valueSelector).map((c) => c.textContent.trim());
        // spot check
        expect(values.length).toBe(15);
        expect(values).toContain('Nexus 9 (flounder)');
        expect(values).toContain('iPhone X');

        let oldRow = row;
        row = getChildItemWithText($$('.selector.keys'), 'state (of task)', ele);
        expect(row).toBeTruthy();
        row.click();
        expect(row.hasAttribute('selected')).toBeTruthy('new row only one selected');
        expect(oldRow.hasAttribute('selected')).toBeFalsy('old row unselected');
        expect(ele._primaryKey).toBe('state');

        valueSelector = $$('.selector.values');
        expect(valueSelector).toBeTruthy();
        values = childrenAsArray(valueSelector).map((c) => c.textContent.trim());
        // spot check
        expect(values.length).toBe(13);
        expect(values).toContain('RUNNING');
        expect(values).toContain('COMPLETED_FAILURE');

        done();
      });
    });

    it('orders columns in selector alphabetically with selected cols on top', function(done) {
      loggedInTasklist((ele) => {
        ele._cols = ['duration', 'created_ts', 'state', 'name'];
        ele._showColSelector = true;
        ele._refilterPossibleColumns(); // also calls render

        const keySelector = $$('.col_selector');
        expect(keySelector).toBeTruthy();
        const keys = childrenAsArray(keySelector).map((c) => c.textContent.trim());

        // Skip the first child, which is the input box
        expect(keys.slice(1, 7)).toEqual(['Task Name', 'Created On', 'Duration',
                                          'state (of task)', 'Abandoned On',
                                          'allow_milo (tag)']);

        done();
      });
    });

    it('adds a filter when the addIcon is clicked', function(done) {
      loggedInTasklist((ele) => {
        ele._cols = ['duration', 'created_ts', 'state', 'name'];
        ele._primaryKey = 'state';  // set 'os' selected
        ele._filters = [];  // no filters
        ele.render();

        const valueRow = getChildItemWithText($$('.selector.values'), 'BOT_DIED', ele);
        const addIcon = $$('add-circle-icon-sk', valueRow);
        expect(addIcon).toBeTruthy('there should be an icon for adding');
        addIcon.click();

        expect(ele._filters.length).toBe(1, 'a filter should be added');
        expect(ele._filters[0]).toEqual('state:BOT_DIED');

        const chipContainer = $$('.chip_container', ele);
        expect(chipContainer).toBeTruthy('there should be a filter chip container');
        expect(chipContainer.children.length).toBe(1);
        expect(addIcon.hasAttribute('hidden'))
              .toBeTruthy('addIcon should go away after being clicked');
        done();
      });
    });

    it('removes a filter when the circle clicked', function(done) {
      loggedInTasklist((ele) => {
        ele._cols = ['duration', 'created_ts', 'state', 'name'];
        ele._primaryKey = 'pool-tag';
        ele._filters = ['pool-tag:Skia', 'state:BOT_DIED'];
        ele.render();

        const filterChip = getChildItemWithText($$('.chip_container'), 'pool-tag:Skia', ele);
        const icon = $$('cancel-icon-sk', filterChip);
        expect(icon).toBeTruthy('there should be a icon to remove it');
        icon.click();

        expect(ele._filters.length).toBe(1, 'a filter should be removed');
        expect(ele._filters[0]).toEqual('state:BOT_DIED', 'pool-tag:Skia should be removed');

        const chipContainer = $$('.chip_container', ele);
        expect(chipContainer).toBeTruthy('there should be a filter chip container');
        expect(chipContainer.children.length).toBe(1);
        done();
      });
    });

    it('shows and hides the column selector', function(done) {
      loggedInTasklist((ele) => {
        ele._showColSelector = false;
        ele.render();

        let cs = $$('.col_selector', ele);
        expect(cs).toBeFalsy();

        let toClick = $$('.col_options', ele);
        expect(toClick).toBeTruthy('Thing to click to show col selector');
        toClick.click();

        cs = $$('.col_selector', ele);
        expect(cs).toBeTruthy();

        // click anywhere else to hide the column selector
        toClick = $$('.header', ele);
        expect(toClick).toBeTruthy('Thing to click to hide col selector');
        toClick.click();

        cs = $$('.col_selector', ele);
        expect(cs).toBeFalsy();

        done();
      });
    });

    it('filters the data it has when waiting for another request', function(done) {
      loggedInTasklist((ele) => {
        ele._cols = ['name'];
        ele._filters = [];
        ele.render();

        expect(ele._tasks.length).toBe(20, 'All 20 at the start');

        let wasCalled = false;
        fetchMock.get('glob:/_ah/api/swarming/v1/tasks/list?*', () => {
          expect(ele._tasks.length).toBe(2, '2 BOT_DIED there now.');
          wasCalled = true;
          return '[]'; // pretend no tasks match
        }, { overwriteRoutes: true });

        ele._addFilter('state:BOT_DIED');
        // The true on flush waits for res.json() to resolve too, which
        // is when we know the element has updated the _tasks.
        fetchMock.flush(true).then(() => {
          expect(wasCalled).toBeTruthy();
          expect(ele._tasks.length).toBe(0, 'none were actually returned');

          done();
        });
      });
    });

    it('allows filters to be added from the search box', function(done) {
      loggedInTasklist((ele) => {
        ele._filters = [];
        ele.render();

        const filterInput = $$('#filter_search', ele);
        filterInput.value = 'invalid filter';
        ele._filterSearch({key: 'Enter'});
        expect(ele._filters).toEqual([]);
        // Leave the input to let the user correct their mistake.
        expect(filterInput.value).toEqual('invalid filter');

        // Spy on the list call to make sure a request is made with the right filter.
        let calledTimes = 0;
        fetchMock.get('glob:/_ah/api/swarming/v1/tasks/list?*', (url, _) => {
          expect(url).toContain(encodeURIComponent('valid:filter:gpu:can:have:many:colons'));
          calledTimes++;
          return '[]'; // pretend no bots match
        }, {overwriteRoutes: true});

        filterInput.value = 'valid-tag:filter:gpu:can:have:many:colons';
        ele._filterSearch({key: 'Enter'});
        expect(ele._filters).toEqual(['valid-tag:filter:gpu:can:have:many:colons']);
        // Empty the input for next time.
        expect(filterInput.value).toEqual('');
        filterInput.value = 'valid-tag:filter:gpu:can:have:many:colons';
        ele._filterSearch({key: 'Enter'});
        // No duplicates
        expect(ele._filters).toEqual(['valid-tag:filter:gpu:can:have:many:colons']);

        fetchMock.flush(true).then(() => {
          expect(calledTimes).toEqual(1, 'Only request tasks once');

          done();
        });
      });
    });

    it('allows auto-corrects filters added from the search box', function(done) {
      loggedInTasklist((ele) => {
        ele._filters = [];
        ele._limit = 0; // turn off requests
        ele.render();

        const filterInput = $$('#filter_search', ele);
        filterInput.value = 'state:BOT_DIED';
        ele._filterSearch({key: 'Enter'});
        expect(ele._filters).toEqual(['state:BOT_DIED']);

        filterInput.value = 'gpu-tag:something';
        ele._filterSearch({key: 'Enter'});
        expect(ele._filters).toContain('gpu-tag:something');

        // there are no valid filters that aren't a tag or state, so
        // correct those that don't have a -tag.
        filterInput.value = 'gpu:something-else';
        ele._filterSearch({key: 'Enter'});
        expect(ele._filters).toContain('gpu-tag:something-else');

        done();
      });
    });

    it('searches filters by typing in the text box', function(done) {
      loggedInTasklist((ele) => {
        ele._filters = [];
        ele.render();

        const filterInput = $$('#filter_search', ele);
        filterInput.value = 'dev';
        ele._refilterPrimaryKeys();

        // Auto selects the first one
        expect(ele._primaryKey).toEqual('android_devices-tag');

        let row = getChildItemWithText($$('.selector.keys'), 'cpu (tag)', ele);
        expect(row).toBeFalsy('cpu (tag) should be hiding');
        row = getChildItemWithText($$('.selector.keys'), 'device_type (tag)', ele);
        expect(row).toBeTruthy('device_type (tag) should be there');
        row = getChildItemWithText($$('.selector.keys'), 'stepname (tag)', ele);
        expect(row).toBeTruthy('stepname (tag) should be there, because some values match');

        filterInput.value = 'pool:Chro';
        ele._refilterPrimaryKeys();
        // Auto selects the first one
        expect(ele._primaryKey).toEqual('pool-tag');

        row = getChildItemWithText($$('.selector.keys'), 'stepname (tag)', ele);
        expect(row).toBeFalsy('stepname (tag) should be hidden');
        row = getChildItemWithText($$('.selector.keys'), 'pool (tag)', ele);
        expect(row).toBeTruthy('pool (tag) should be visible');
        row = getChildItemWithText($$('.selector.keys'), 'sk_dim_pool (tag)', ele);
        expect(row).toBeFalsy('sk_dim_pool (tag) is not an exact match');

        row = getChildItemWithText($$('.selector.values'), 'Chrome-perf', ele);
        expect(row).toBeTruthy('Chrome-perf should be visible');
        row = getChildItemWithText($$('.selector.values'), 'AndroidBuilder', ele);
        expect(row).toBeFalsy('AndroidBuilder should be hidden, does not match Chro');

        done();
      });
    });

    it('filters keys/values by partial filters', function(done) {
      loggedInTasklist((ele) => {
        ele._filters = [];
        ele.render();

        const filterInput = $$('#filter_search', ele);
        filterInput.value = 'pool-tag:Ski';
        ele._refilterPrimaryKeys();

        // Auto selects the first one
        expect(ele._primaryKey).toEqual('pool-tag');

        const children = $$('.selector.keys', ele).children;
        expect(children.length).toEqual(1, 'only pool-tag should show up');
        expect(children[0].textContent).toContain('pool (tag)');

        let row = getChildItemWithText($$('.selector.values'), 'Chrome', ele);
        expect(row).toBeFalsy('Chrome does not match');
        row = getChildItemWithText($$('.selector.values'), 'SkiaTriggers', ele);
        expect(row).toBeTruthy('SkiaTriggers matches');

        done();
      });
    });

    it('searches columns by typing in the text box', function(done) {
      loggedInTasklist((ele) => {
        ele._cols = ['name'];
        ele._showColSelector = true;
        ele.render();

        const columnInput = $$('#column_search', ele);
        columnInput.value = 'build';
        ele._refilterPossibleColumns();

        const colSelector = $$('.col_selector', ele);
        expect(colSelector).toBeTruthy();
        expect(colSelector.children.length).toEqual(12); // 11 hits + the input box

        let row = getChildItemWithText(colSelector, 'state');
        expect(row).toBeFalsy('state should be hiding');
        row = getChildItemWithText(colSelector, 'build_is_experimental (tag)');
        expect(row).toBeTruthy('build_is_experimental (tag) should be there');

        done();
      });
    });

    it('shows and hide the extra state counts', function(done) {
          loggedInTasklist((ele) => {
            ele._allStates = false;
            ele.render();

            let countRows = $('#query_counts tr', ele);
            expect(countRows).toBeTruthy();
            expect(countRows.length).toBe(1+7, '(num counts, displayed + 7 states)');

            let showMore  = $$('.summary expand-more-icon-sk');
            let showMore2 = $$('.summary more-horiz-icon-sk');
            let showLess  = $$('.summary expand-less-icon-sk');
            expect(showMore).toBeTruthy();
            expect(showMore2).not.toHaveAttribute('hidden');
            expect(showLess).toBeFalsy();
            showMore.click();

            expect(ele._allStates).toBeTruthy();
            countRows = $('#query_counts tr', ele);
            expect(countRows).toBeTruthy();
            expect(countRows.length).toBe(1+11, '(num counts, displayed + 11 states)');

            showMore  = $$('.summary expand-more-icon-sk');
            showMore2 = $$('.summary more-horiz-icon-sk');
            showLess  = $$('.summary expand-less-icon-sk');
            expect(showMore).toBeFalsy();
            expect(showMore2).toHaveAttribute('hidden');
            expect(showLess).toBeTruthy();
            showLess.click();

            expect(ele._allStates).toBeFalsy();
            countRows = $('#query_counts tr', ele);
            expect(countRows).toBeTruthy();
            expect(countRows.length).toBe(1+7, '(num counts, displayed + 7 states)');

            showMore  = $$('.summary expand-more-icon-sk');
            showMore2 = $$('.summary more-horiz-icon-sk');
            showLess  = $$('.summary expand-less-icon-sk');
            expect(showMore).toBeTruthy();
            expect(showMore2).not.toHaveAttribute('hidden');
            expect(showLess).toBeFalsy();

            done();
          });
        });

    it('updates the links with filters and other settings', function(done) {
      loggedInTasklist((ele) => {
        ele._startTime = 1545151583000;
        ele._endTime = 1545237981000;
        ele._sort = 'completed_ts';
        ele._filters = ['pool-tag:Chrome', 'state:DEDUPED'];
        ele.render();

        const countRows = $('#query_counts tr', ele);
        expect(countRows).toBeTruthy();

        expect(countRows[3].innerHTML).toContain('<a ', 'contains a link');
        const link = $$('a', countRows[3]);
        expect(link.href).toContain('/tasklist?at=false&c=name&c=created_ts&' +
            'c=pending_time&c=duration&c=bot&c=pool-tag&c=state&d=desc&et=1545237981000&' +
            'f=pool-tag%3AChrome&f=state%3ACOMPLETED_FAILURE&k=&n=true&s=completed_ts&' +
            'st=1545151583000&v=false');
        done();
      });
    });

    it('updates the matching bots link', function(done) {
      loggedInTasklist((ele) => {
        ele._filters = ['device_type-tag:nemo', 'state:DEDUPED'];
        ele._knownDimensions = ['device_type', 'os'];
        ele.render();

        const link = $$('.options > a', ele);
        expect(link).toBeTruthy();

        expect(link.href).toContain('/botlist?c=id&c=os&c=task&c=status&c=device_type' +
                                    '&f=device_type%3Anemo');
        done();
      });
    });

    it('only tries to cancel all tasks based on tags', function(done) {
      loggedInTasklist((ele) => {
        ele.permissions.cancel_task = true;
        ele._filters = ['pool-tag:Skia', 'state:PENDING', 'os-tag:Windows'];
        ele.render();

        const cancelAll = $$('button#cancel_all', ele);
        expect(cancelAll).toBeTruthy();

        cancelAll.click();

        const prompt = $$('task-mass-cancel', ele);
        expect(prompt).toBeTruthy();

        expect(prompt.tags).toEqual(['pool:Skia', 'os:Windows']);

        done();
      });
    });
  }); // end describe('dynamic behavior')

  describe('api calls', function() {
    it('makes no API calls when not logged in', function(done) {
      createElement((ele) => {
        fetchMock.flush(true).then(() => {
          // MATCHED calls are calls that we expect and specified in the
          // beforeEach at the top of this file.
          let calls = fetchMock.calls(MATCHED, 'GET');
          expect(calls.length).toBe(0);
          calls = fetchMock.calls(MATCHED, 'POST');
          expect(calls.length).toBe(0);

          expectNoUnmatchedCalls(fetchMock);
          done();
        });
      });
    });

    function checkAuthorizationAndNoPosts(calls) {
      // check authorization headers are set
      calls.forEach((c) => {
        expect(c[1].headers).toBeDefined();
        expect(c[1].headers.authorization).toContain('Bearer ');
      });
      calls = fetchMock.calls(MATCHED, 'POST');
      expect(calls.length).toBe(0, 'no POSTs on task-list');

      expectNoUnmatchedCalls(fetchMock);
    }

    it('makes auth\'d API calls when a logged in user views landing page', function(done) {
      loggedInTasklist((ele) => {
        let calls = fetchMock.calls(MATCHED, 'GET');
        expect(calls.length).toBe(2+2+11, '2 GETs from swarming-app, 2 from task-list (11 counts)');
        // calls is an array of 2-length arrays with the first element
        // being the string of the url and the second element being
        // the options that were passed in
        const gets = calls.map((c) => c[0]);

        // limit=100 comes from the default limit value.
        expect(gets).toContainRegex(/\/_ah\/api\/swarming\/v1\/tasks\/list.+limit=100.*/);

        checkAuthorizationAndNoPosts(calls);
        done();
      });
    });

    it('counts correctly with filters', function(done) {
      loggedInTasklist((ele) => {
        ele._filters = ['os-tag:Android'];
        fetchMock.resetHistory();
        ele._addFilter('state:PENDING_RUNNING');
        fetchMock.flush(true).then(() => {
          const calls = fetchMock.calls(MATCHED, 'GET');
          expect(calls.length).toBe(1+11, '1 from task-list and 11 counts');

          const gets = calls.map((c) => c[0]);
          for (const get of gets) {
            // make sure there aren't two states when we do the count (which
            // appends a state)
            expect(get).not.toMatch(/state.+state/);
            // %3A is url encoded colon (:)
            expect(get).toMatch(/tags=os%3AAndroid/);
          }
          done();
        });
      });
    });

    it('counts correctly with just states', function(done) {
      loggedInTasklist((ele) => {
        ele._filters = [];
        fetchMock.resetHistory();
        ele._addFilter('state:PENDING_RUNNING');
        fetchMock.flush(true).then(() => {
          const calls = fetchMock.calls(MATCHED, 'GET');
          expect(calls.length).toBe(1+11, '1 from task-list and 11 counts');

          const gets = calls.map((c) => c[0]);
          for (const get of gets) {
            // make sure there aren't two states when we do the count (which
            // appends a state)
            expect(get).not.toMatch(/state.+state/);
            // Only one state requested.
            expect(get).toMatch(/state=/);
          }
          done();
        });
      });
    });

    it('counts correctly when preparing to cancel', function(done) {
      loggedInTasklist((ele) => {
        ele._filters = ['pool-tag:Chrome'];
        ele.permissions.cancel_task = true;
        ele.render();
        fetchMock.resetHistory();
        const showBtn = $$('#cancel_all');
        expect(showBtn).toBeTruthy('button should exist');
        showBtn.click();

        fetchMock.flush(true).then(() => {
          expectNoUnmatchedCalls(fetchMock);
          const calls = fetchMock.calls(MATCHED, 'GET');
          expect(calls.length).toBe(2, '2 counts, 1 running, 1 pending');

          const gets = calls.map((c) => c[0]);
          expect(gets).toContain('/_ah/api/swarming/v1/tasks/count?end=1545237983&'+
                                 'start=1544633183&state=PENDING&tags=pool%3AChrome');
          expect(gets).toContain('/_ah/api/swarming/v1/tasks/count?end=1545237983&'+
                                 'start=1544633183&state=RUNNING&tags=pool%3AChrome');
          done();
        });
      });
    });

    it('counts correctly when cancelling', function(done) {
      fetchMock.post('/_ah/api/swarming/v1/tasks/cancel', {'matched': 10});
      loggedInTasklist((ele) => {
        ele._filters = ['pool-tag:Chrome'];
        ele.permissions.cancel_task = true;
        ele.render();
        const showBtn = $$('#cancel_all');
        expect(showBtn).toBeTruthy('show button should exist');
        showBtn.click();

        fetchMock.flush(true).then(() => {
          fetchMock.resetHistory();
          const cancelBtn = $$('task-mass-cancel button.cancel');
          expect(cancelBtn).toBeTruthy('cancel button should exist');
          cancelBtn.click();

          fetchMock.flush(true).then(() => {
            expectNoUnmatchedCalls(fetchMock);
            let calls = fetchMock.calls(MATCHED, 'GET');
            expect(calls.length).toBe(0, 'Only posts');
            calls = fetchMock.calls(MATCHED, 'POST');
            expect(calls.length).toBe(1, '1 cancel request');
            // calls is an array of 2-length arrays with the first element
            // being the string of the url and the second element being
            // the options that were passed in
            const cancelPost = calls[0];
            expect(cancelPost[0]).toEqual('/_ah/api/swarming/v1/tasks/cancel');
            expect(cancelPost[1].body).toEqual('{"limit":100,"tags":["pool:Chrome"]}');

            done();
          });
        });
      });
    });
  }); // end describe('api calls')

  describe('data parsing', function() {
    const ANDROID_TASK = tasks_20.items[0];

    it('turns the dates into DateObjects', function() {
      // Make a copy of the object because processTasks will modify it in place.
      const tasks = processTasks([deepCopy(ANDROID_TASK)], {});
      const task = tasks[0];
      expect(task.created_ts).toBeTruthy();
      expect(task.created_ts instanceof Date).toBeTruthy('Should be a date object');
      expect(task.human_created_ts).toBeTruthy();
      expect(task.pending_time).toBeTruthy();
      expect(task.human_pending_time).toBeTruthy();
    });

    it('gracefully handles null data', function() {
      const tasks = processTasks(null);

      expect(tasks).toBeTruthy();
      expect(tasks.length).toBe(0);
    });

    it('produces a list of tags', function() {
      const tags = {};
      const tasks = processTasks(deepCopy(tasks_20.items), tags);
      const keys = Object.keys(tags);
      expect(keys).toBeTruthy();
      expect(keys.length).toBe(76);
      expect(keys).toContain('pool');
      expect(keys).toContain('purpose');
      expect(keys).toContain('source_revision');

      expect(tasks.length).toBe(20);
    });

    it('filters tasks based on special keys', function() {
      const tasks = processTasks(deepCopy(tasks_20.items), {});

      expect(tasks).toBeTruthy();
      expect(tasks.length).toBe(20);

      const filtered = filterTasks(['state:COMPLETED_FAILURE'], tasks);
      expect(filtered.length).toBe(2);
      const expectedIds = ['41e0310fe0b7c410', '41e031b2c8b46710'];
      const actualIds = filtered.map((task) => task.task_id);
      actualIds.sort();
      expect(actualIds).toEqual(expectedIds);
    });

    it('filters tasks based on tags', function() {
      const tasks = processTasks(deepCopy(tasks_20.items), {});

      expect(tasks).toBeTruthy();
      expect(tasks.length).toBe(20);

      let filtered = filterTasks(['pool-tag:Chrome'], tasks);
      expect(filtered.length).toBe(7);
      let actualIds = filtered.map((task) => task.task_id);
      expect(actualIds).toContain('41e0204f39d06210'); // spot check
      expect(actualIds).not.toContain('41e0182a00fcc110');

      // some tasks have multiple 'purpose' tags
      filtered = filterTasks(['purpose-tag:luci'], tasks);
      expect(filtered.length).toBe(8);
      actualIds = filtered.map((task) => task.task_id);
      expect(actualIds).toContain('41e020504d0a5110'); // spot check
      expect(actualIds).not.toContain('41e0310fe0b7c410');

      filtered = filterTasks(['pool-tag:Skia', 'gpu-tag:none'], tasks);
      expect(filtered.length).toBe(1);
      expect(filtered[0].task_id).toBe('41e031b2c8b46710');

      filtered = filterTasks(['pool-tag:Skia', 'gpu-tag:10de:1cb3-384.59'], tasks);
      expect(filtered.length).toBe(2);
      actualIds = filtered.map((task) => task.task_id);
      expect(actualIds).toContain('41dfa79d3bf29010');
      expect(actualIds).toContain('41df677202f20310');
    });

    it('correctly makes query params from filters', function() {
      // We know query.fromObject is used and it puts the query params in
      // a deterministic, sorted order. This means we can compare
      const expectations = [
        { // basic 'state'
          'extra': {
            'limit': 7,
            'start': 12345678,
            'end': 456789012,
          },
          'filters': ['state:BOT_DIED'],
          'output': 'end=456789&limit=7&start=12345&state=BOT_DIED',
        },
        { // two tags
          'extra': {
            'limit': 342,
            'start': 12345678,
            'end': 456789012,
          },
          'filters': ['os-tag:Window', 'gpu-tag:10de'],
          'output': 'end=456789&limit=342&start=12345&tags=os%3AWindow&tags=gpu%3A10de',
        },
        { // tags and state
          'extra': {
            'limit': 57,
            'start': 12345678,
            'end': 456789012,
          },
          'filters': ['os-tag:Window', 'state:RUNNING', 'gpu-tag:10de'],
          'output': 'end=456789&limit=57&start=12345&state=RUNNING&tags=os%3AWindow&tags=gpu%3A10de',
        },
      ];

      for (const testcase of expectations) {
        const qp = listQueryParams(testcase.filters, testcase.extra);
        expect(qp).toEqual(testcase.output);
      }

      const testcase = expectations[0];
      testcase.extra.cursor = 'mock_cursor12345';
      const qp = listQueryParams(testcase.filters, testcase.extra);
      expect(qp).toEqual('cursor=mock_cursor12345&'+testcase.output);
    });

  }); //end describe('data parsing')
});
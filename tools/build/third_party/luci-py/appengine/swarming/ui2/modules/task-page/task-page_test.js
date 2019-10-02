// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import 'modules/task-page'

// Tip from https://stackoverflow.com/a/37348710
// for catching "full page reload" errors.
beforeAll(() => {
  window.onbeforeunload = () => {
    expect(false).toBeTruthy();
    console.error('We should not have modified window.location.href directly.');
    throw 'We should not have modified window.location.href directly.';
  }
});

describe('task-page', function() {
  // Instead of using import, we use require. Otherwise,
  // the concatenation trick we do doesn't play well with webpack, which would
  // leak dependencies (e.g. bot-list's 'column' function to task-list) and
  // try to import things multiple times.
  const { $, $$ } = require('common-sk/modules/dom');
  const { customMatchers, expectNoUnmatchedCalls, mockAppGETs } = require('modules/test_util');
  const { fetchMock, MATCHED, UNMATCHED } = require('fetch-mock');
  const { taskOutput, taskResults, taskRequests } = require('modules/task-page/test_data');

  const TEST_TASK_ID = 'test0b3c0fac7810';

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

    // By default, don't have any handlers mocked out - this requires
    // tests to opt-in to wanting certain request data.

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
    jasmine.clock().mockDate(new Date(Date.UTC(2019, 1, 4, 16, 46, 22, 1234)));
  });

  afterEach(function() {
    jasmine.clock().uninstall();
  });

  // calls the test callback with one element 'ele', a created <task-page>.
  function createElement(test) {
    return window.customElements.whenDefined('task-page').then(() => {
      container.innerHTML = `<task-page client_id=for_test testing_offline=true></task-page>`;
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
  // <task-page> after the user has logged in.
  function loggedInTaskPage(test, emptyTaskId) {
    createElement((ele) => {
      if (!emptyTaskId) {
        ele._taskId = TEST_TASK_ID;
      }
      userLogsIn(ele, () => {
        test(ele);
      });
    });
  }

  function serveTask(idx, msg) {
    // msg is the name field in the task request, used to 1) give a human
    // readable description of the task data inline of the test and 2)
    // lessen the risk of copy-pasta mistakes.
    const request = taskRequests[idx];
    expect(request.name).toEqual(msg);
    const result = taskResults[idx];
    expect(result.name).toEqual(msg);

    fetchMock.get(`/_ah/api/swarming/v1/task/${TEST_TASK_ID}/request`, request);
    fetchMock.get(`/_ah/api/swarming/v1/task/${TEST_TASK_ID}/result?include_performance_stats=true`, result);
    if (idx === 0) {
      // The index 0 data has multiple tries that it requests data for (no perf stats),
      // so pass in some data for that.
      fetchMock.get('glob:/_ah/api/swarming/v1/task/*/result', taskResults[1]);
    }
    fetchMock.get(`/_ah/api/swarming/v1/task/${TEST_TASK_ID}/stdout`, taskOutput);
    fetchMock.get('glob:/_ah/api/swarming/v1/bots/count?*', {
      busy: 1024,
      count: 1337,
      dead: 13,
      quarantined: 1,
      maintenance: 0,
    });
    fetchMock.get('glob:/_ah/api/swarming/v1/tasks/count?*', (url, opts) => {
      if (url.indexOf('PENDING') !== -1) {
        return {count: 123};
      } else {
        return {count: 56};
      }
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
          const topDivs = $('main > div', ele);
          expect(topDivs).toBeTruthy();
          expect(topDivs.length).toBe(2);
          expect(topDivs[0]).toHaveAttribute('hidden', 'left side hidden');
          expect(topDivs[1]).toHaveAttribute('hidden', 'right side hidden');
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
        fetchMock.get('glob:/_ah/api/swarming/v1/task/*', 403,
                      { overwriteRoutes: true });
      }

      beforeEach(notAuthorized);

      it('tells the user they should change accounts', function(done) {
        loggedInTaskPage((ele) => {
          const loginMessage = $$('swarming-app>main .message', ele);
          expect(loginMessage).toBeTruthy();
          expect(loginMessage).not.toHaveAttribute('hidden', 'Message should not be hidden');
          expect(loginMessage.textContent).toContain('different account');
          done();
        });
      });

      it('does not display logs or task details', function(done) {
        loggedInTaskPage((ele) => {
          const topDivs = $('main > div', ele);
          expect(topDivs).toBeTruthy();
          expect(topDivs.length).toBe(2);
          expect(topDivs[0]).toHaveAttribute('hidden', 'left side hidden');
          expect(topDivs[1]).toHaveAttribute('hidden', 'right side hidden');
          done();
        });
      });
    }); // end describe('when logged in as unauthorized user')

    describe('authorized user, but no taskid', function() {

      it('tells the user they should enter a task id', function(done) {
        loggedInTaskPage((ele) => {
          const loginMessage = $$('.id_buttons .message', ele);
          expect(loginMessage).toBeTruthy();
          expect(loginMessage.textContent).toContain('Enter a Task ID');
          done();
        }, true);
      });

      it('does not display filters or tasks', function(done) {
        loggedInTaskPage((ele) => {
          const topDivs = $('main > div', ele);
          expect(topDivs).toBeTruthy();
          expect(topDivs.length).toBe(2);
          expect(topDivs[0].children.length).toEqual(2); // only .id_buttons and task not found
          expect(topDivs[1].children.length).toEqual(0); // everything else removed
          done();
        }, true);
      });
    }); // end describe('authorized user, but no taskid')

    describe('Completed task with 2 slices', function() {
      beforeEach(() => serveTask(1, 'Completed task with 2 slices'));

      it('shows relevant task request data', function(done) {
        loggedInTaskPage((ele) => {
          const taskInfo = $$('table.request-info', ele);
          expect(taskInfo).toBeTruthy();
          const rows = $('tr', taskInfo);
          expect(rows.length).toBeTruthy('Has some rows');

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];
          // Spot check some of the content
          expect(cell(0, 0)).toMatchTextContent('Name');
          expect(cell(0, 1)).toMatchTextContent('Completed task with 2 slices');
          expect(cell(1, 0)).toMatchTextContent('State');
          expect(cell(1, 1)).toMatchTextContent('COMPLETED (SUCCESS)');
          expect(cell(2, 1)).toMatchTextContent('1337 bots could possibly run this task ' +
                                        '(1024 busy, 13 dead, 1 quarantined, 0 maintenance)');
          expect(cell(3, 1)).toMatchTextContent('123  similar pending tasks, '+
                                                '56  similar running tasks');
          expect(rows[5]).toHaveAttribute('hidden', 'deduped message hidden');
          expect(cell(7, 0)).toMatchTextContent('Wait for Capacity');
          expect(cell(7, 1)).toMatchTextContent('false');
          // 5 dimensions shown on slice 2 + 1 for header
          expect(cell(14, 0).rowSpan).toEqual(6);
          expect(cell(14, 0).textContent).toContain('Dimensions');

          const subsections = $('tbody', taskInfo);
          expect(subsections.length).toEqual(2);
          expect(subsections[0]).not.toHaveAttribute('hidden');
          expect(subsections[1]).toHaveAttribute('hidden');

          const taskDisamb = $$('table.task-disambiguation', ele);
          expect(taskDisamb).toBeFalsy();

          done();
        });
      });

      it('shows relevant task timing data', function(done) {
        loggedInTaskPage((ele) => {
          const taskTiming = $$('table.task-timing', ele);
          expect(taskTiming).toBeTruthy();
          const rows = $('tr', taskTiming);
          expect(rows.length).toEqual(9);

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];
          // Spot check some of the content
          expect(rows[1]).not.toHaveAttribute('hidden', 'show started');
          expect(cell(6, 0)).toMatchTextContent('Pending Time');
          expect(cell(6, 1)).toMatchTextContent('3m 22s');
          expect(cell(7, 0)).toMatchTextContent('Total Overhead');
          expect(cell(7, 1)).toMatchTextContent('12.63s');
          expect(cell(8, 0)).toMatchTextContent('Running Time');
          expect(cell(8, 1)).toMatchTextContent('14m 41s');

          done();
        });
      });

      it('shows relevant task execution data', function(done) {
        loggedInTaskPage((ele) => {
          const taskExecution = $$('table.task-execution', ele);
          expect(taskExecution).toBeTruthy();
          const rows = $('tr', taskExecution);
          expect(rows.length).toBeTruthy();

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];
          // Spot check some of the content
          expect(cell(0, 0)).toMatchTextContent('Bot assigned to task');
          expect(cell(0, 1).innerHTML).toContain('<a ', 'has a link');
          expect(cell(0, 1).innerHTML).toContain('href="/bot?id=swarm1931-c4"', 'link is correct');
          expect(cell(1, 0).rowSpan).toEqual(15); // 14 dimensions shown + 1 for header
          expect(cell(5, 0)).toMatchTextContent('gpu:Intel (8086)' +
                    'Intel Sandy Bridge HD Graphics 2000 (8086:0102)');

          done();
        });
      });

      it('shows relevant performance stats', function(done) {
        loggedInTaskPage((ele) => {
          const taskPerformance = $$('table.performance-stats', ele);
          expect(taskPerformance).toBeTruthy();
          const rows = $('tr', taskPerformance);
          expect(rows.length).toBeTruthy();

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];
          // Spot check some of the content
          expect(cell(0, 0)).toMatchTextContent('Total Overhead');
          expect(cell(0, 1)).toMatchTextContent('12.63s');
          expect(cell(6, 0)).toMatchTextContent('Uploaded Cold Items');
          expect(cell(6, 1)).toMatchTextContent('2 items; 12 KB');
          expect(cell(7, 0)).toMatchTextContent('Uploaded Hot Items');
          expect(cell(7, 1)).toMatchTextContent('0 items; 0 B');
          done();
        });
      });

      it('shows a tab slice picker', function(done) {
        loggedInTaskPage((ele) => {
          const picker = $$('.slice-picker', ele);
          expect(picker).toBeTruthy();
          const tabs = $('.tab', picker);
          expect(tabs.length).toEqual(2);

          // The 2nd tab ran, so it should be shown by default.
          expect(tabs[0]).not.toHaveAttribute('selected');
          expect(tabs[1]).toHaveAttribute('selected');
          done();
        });
      });

      it('tells the user when slices did not run', function(done) {
        loggedInTaskPage((ele) => {
          ele._setSlice(0); // calls render

          const stateRow = $$('.request-info.inactive tr:nth-child(2)', ele);
          expect(stateRow).toBeTruthy();
          const header = stateRow.children[0];
          expect(header).toMatchTextContent('State');
          const message = stateRow.children[1];
          expect(message).toMatchTextContent('THIS SLICE DID NOT RUN. '+
                                             'Select another slice above.');
          done();
        });
      });

      it('shows rich logs in an iframe', function(done) {
        loggedInTaskPage((ele) => {
          ele._showRawOutput = false;
          ele.render();

          const frame = $$('#richLogsFrame', ele);
          expect(frame).toBeTruthy();
          expect(frame.src).toEqual('https://example.com/#id='+TEST_TASK_ID);

          // Stdout logs aren't rendered then
          const logs = $$('.stdout.code', ele);
          expect(logs).toBeFalsy();
          done();
        });
      });

      it('shows stdout logs in a box', function(done) {
        loggedInTaskPage((ele) => {
          ele._showRawOutput = true;
          ele._wideLogs = false;
          ele.render();

          // Rich logs aren't rendered then
          const frame = $$('#richLogsFrame', ele);
          expect(frame).toBeFalsy();

          const logs = $$('.stdout.code', ele);
          expect(logs).toBeTruthy();
          expect(logs.textContent).toContain('Lorem ipsum dolor');
          expect(logs).not.toHaveClass('wide');
          done();
        });
      });

      it('can show wide logs', function(done) {
        loggedInTaskPage((ele) => {
          ele._showRawOutput = true;
          ele._wideLogs = true;
          ele.render();

          const logs = $$('.stdout.code.wide', ele);
          expect(logs).toBeTruthy();
          expect(logs.textContent).toContain('Lorem ipsum dolor');
          done();
        });
      });

       it('shows neither a cancel button nor a kill button', function(done) {
        loggedInTaskPage((ele) => {
          const cancelBtn = $$('.id_buttons button.cancel', ele);
          expect(cancelBtn).toBeTruthy();
          expect(cancelBtn).toHaveAttribute('hidden', 'cancel should be hidden');

          const killBtn = $$('.id_buttons button.kill', ele);
          expect(killBtn).toBeTruthy();
          expect(killBtn).toHaveAttribute('hidden', 'Kill should be hidden');

          done();
        });
      });

      it('hides the retry button (because idempotent:false)', function(done) {
        loggedInTaskPage((ele) => {
          const retryBtn = $$('.id_buttons button.retry', ele);
          expect(retryBtn).toBeTruthy();
          expect(retryBtn).toHaveAttribute('hidden', 'retry should be hidden');

          const debugBtn = $$('.id_buttons button.debug', ele);
          expect(debugBtn).toBeTruthy();
          expect(debugBtn).not.toHaveAttribute('hidden', 'debug should be visible');

          done();
        });
      });
    }); // end describe('Completed task with 2 slices')


    describe('Pending task - 1 slice - no rich logs', function() {
      beforeEach(() => serveTask(2, 'Pending task - 1 slice - no rich logs'));

      it('has some pending specific request data', function(done) {
        loggedInTaskPage((ele) => {
          const taskInfo = $$('table.request-info', ele);
          expect(taskInfo).toBeTruthy();
          const rows = $('tr', taskInfo);
          expect(rows.length).toBeTruthy('Has some rows');

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];
          // Spot check some of the content
          expect(cell(1, 0)).toMatchTextContent('State');
          expect(cell(1, 1)).toMatchTextContent('PENDING');
          expect(cell(1, 1)).toHaveClass('pending_task');
          expect(cell(2, 0)).toMatchTextContent('Why Pending?');
          expect(rows[5]).toHaveAttribute('hidden', 'deduped message hidden');
          expect(cell(14, 0).rowSpan).toEqual(5); // 4 dimensions + 1 for header

          done();
        });
      });

      it('shows rich output as disabled', function(done) {
        loggedInTaskPage((ele) => {
          const picker = $$('.output-picker', ele);
          expect(picker).toBeTruthy();
          const tabs = $('.tab', picker);
          expect(tabs.length).toEqual(2);
          expect(tabs[0]).toHaveAttribute('disabled');
          expect(tabs[0]).not.toHaveAttribute('selected');
          expect(tabs[1]).not.toHaveAttribute('disabled');
          expect(tabs[1]).toHaveAttribute('selected');

          done();
        });
      });

      it('shows no task execution data', function(done) {
        loggedInTaskPage((ele) => {
          const output = $$('div.task-execution', ele);
          expect(output).toBeTruthy();
          expect(output.textContent).toContain('left blank');

          const outTable = $$('table.task-execution', ele);
          expect(outTable).toBeFalsy();

          done();
        });
      });

      it('shows a cancel button', function(done) {
        loggedInTaskPage((ele) => {
          ele.permissions.cancel_task = true;
          ele.render();
          const cancelBtn = $$('.id_buttons button.cancel', ele);
          expect(cancelBtn).toBeTruthy();
          expect(cancelBtn).not.toHaveAttribute('hidden', 'cancel should be showing');
          expect(cancelBtn).not.toHaveAttribute('disabled', 'cancel should be enabled');

          const killBtn = $$('.id_buttons button.kill', ele);
          expect(killBtn).toBeTruthy();
          // Kill is only for running tasks.
          expect(killBtn).toHaveAttribute('hidden', 'Kill should be hidden');

          ele.permissions.cancel_task = false;
          ele.render();
          expect(cancelBtn).not.toHaveAttribute('hidden', 'cancel should be showing');
          expect(cancelBtn).toHaveAttribute('disabled', 'cancel should be disabled');
          done();
        });
      });

      it('shows the retry button (because idempotent:true)', function(done) {
        loggedInTaskPage((ele) => {
          const retryBtn = $$('.id_buttons button.retry', ele);
          expect(retryBtn).toBeTruthy();
          expect(retryBtn).not.toHaveAttribute('hidden', 'retry should be visible');

          const debugBtn = $$('.id_buttons button.debug', ele);
          expect(debugBtn).toBeTruthy();
          expect(debugBtn).not.toHaveAttribute('hidden', 'debug should be visible');

          done();
        });
      });

    }); // end describe('Pending task - 1 slice - no rich logs')

    describe('running task on try number 3', function() {
      beforeEach(() => serveTask(0, 'running task on try number 3'));

      it('has some running specific request data', function(done) {
        loggedInTaskPage((ele) => {
          const taskInfo = $$('table.request-info', ele);
          expect(taskInfo).toBeTruthy();
          const rows = $('tr', taskInfo);
          expect(rows.length).toBeTruthy('Has some rows');

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];
          // Spot check some of the content
          expect(cell(1, 0)).toMatchTextContent('State');
          expect(cell(1, 1)).toMatchTextContent('RUNNING');
          expect(cell(1, 1)).toHaveClass('pending_task');
          expect(cell(2, 0)).toMatchTextContent('Fleet Capacity');
          expect(rows[5]).toHaveAttribute('hidden', 'deduped message hidden');
          expect(cell(14, 0).rowSpan).toEqual(5); // 4 dimensions + 1 for header

          done();
        });
      });

      it('shows task execution data', function(done) {
        loggedInTaskPage((ele) => {
          const output = $$('div.task-execution', ele);
          expect(output).toBeFalsy();

          const outTable = $$('table.task-execution', ele);
          expect(outTable).toBeTruthy();
          const rows = $('tr', outTable);
          expect(rows.length).toBeTruthy('Has some rows');

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];

          expect(cell(18, 0)).toMatchTextContent('Try Number');
          expect(cell(18, 1)).toMatchTextContent('3');
          done();
        });
      });

      it('shows relevant task timing data', function(done) {
        loggedInTaskPage((ele) => {
          const taskTiming = $$('table.task-timing', ele);
          expect(taskTiming).toBeTruthy();
          const rows = $('tr', taskTiming);
          expect(rows.length).toEqual(9);

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];
          // Spot check some of the content
          expect(rows[1]).not.toHaveAttribute('hidden', 'show started');
          expect(cell(6, 0)).toMatchTextContent('Pending Time');
          expect(cell(6, 1)).toMatchTextContent('31s');
          expect(cell(7, 0)).toMatchTextContent('Total Overhead');
          expect(cell(7, 1)).toMatchTextContent('--');
          expect(cell(8, 0)).toMatchTextContent('Running Time');
          expect(cell(8, 1)).toMatchTextContent('40m 33s*');

          done();
        });
      });

      it('tells the user about the other running tries', function(done) {
        loggedInTaskPage((ele) => {
          const taskDisamb = $$('table.task-disambiguation', ele);
          expect(taskDisamb).toBeTruthy();
          const rows = $('tr', taskDisamb);
          expect(rows.length).toEqual(4);

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];

          expect(cell(1, 0)).toMatchTextContent('testid001');
          expect(cell(1, 0).innerHTML).toContain('<a', 'has a link');
          expect(cell(1, 0).innerHTML).toContain('href="/task?id=testid001"', 'link is correct');
          expect(cell(2, 0)).toMatchTextContent('testid002');
          expect(cell(2, 0).innerHTML).toContain('<a', 'has a link');
          expect(cell(2, 0).innerHTML).toContain('href="/task?id=testid002"', 'link is correct');
          expect(cell(3, 0)).toMatchTextContent('testid003');
          expect(cell(3, 0).innerHTML).toContain('<a', 'has a link');
          expect(cell(3, 0).innerHTML).toContain('href="/task?id=testid003"', 'link is correct');
          done();
        });
      });

      it('shows a kill button', function(done) {
        loggedInTaskPage((ele) => {
          ele.permissions.cancel_task = true;
          ele.render();
          const cancelBtn = $$('.id_buttons button.cancel', ele);
          expect(cancelBtn).toBeTruthy();
          // Cacnel is only for pending tasks.
          expect(cancelBtn).toHaveAttribute('hidden', 'cancel should be hidden');

          const killBtn = $$('.id_buttons button.kill', ele);
          expect(killBtn).toBeTruthy();
          expect(killBtn).not.toHaveAttribute('hidden', 'Kill should be shown');
          expect(killBtn).not.toHaveAttribute('disabled', 'Kill should be enabled');

          ele.permissions.cancel_task = false;
          ele.render();
          expect(killBtn).not.toHaveAttribute('hidden', 'Kill should be shown');
          expect(killBtn).toHaveAttribute('disabled', 'Kill should be disabled');
          done();
        });
      });
    }); // end describe('running task on try number 3')

    describe('deduplicated task with gpu dim', function() {
      beforeEach(() => serveTask(3, 'deduplicated task with gpu dim'));

      it('has some running specific request data', function(done) {
        loggedInTaskPage((ele) => {
          const taskInfo = $$('table.request-info', ele);
          expect(taskInfo).toBeTruthy();
          const rows = $('tr', taskInfo);
          expect(rows.length).toBeTruthy('Has some rows');

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];
          // Spot check some of the content
          expect(cell(1, 0)).toMatchTextContent('State');
          expect(cell(1, 1)).toMatchTextContent('COMPLETED (DEDUPED)');
          expect(cell(2, 0)).toMatchTextContent('Fleet Capacity');
          expect(rows[4]).not.toHaveAttribute('hidden', 'deduped message shown');
          expect(rows[5]).not.toHaveAttribute('hidden', 'deduped message shown');
          expect(cell(4, 0)).toMatchTextContent('Deduped From');
          expect(cell(4, 1)).toMatchTextContent('42e0ec5f54b04411');
          expect(cell(5, 0)).toMatchTextContent('Deduped On');

          expect(cell(14, 0).rowSpan).toEqual(4); // 3 dimensions + 1 for header
          expect(cell(16, 0)).toMatchTextContent(
                  'gpu:Intel Sandy Bridge HD Graphics 2000 (8086:0102)');

          done();
        });
      });

      it('shows a deduplication message instead of execution', function(done) {
        loggedInTaskPage((ele) => {
          const output = $$('div.task-execution', ele);
          expect(output).toBeFalsy();
          const outTable = $$('table.task-execution', ele);
          expect(outTable).toBeFalsy();

          const dedupedText = $$('p.deduplicated', ele);
          expect(dedupedText).toBeTruthy();
          expect(dedupedText.innerHTML).toContain('<a href');

          done();
        });
      });

      it('does not show the multiple tries summary', function(done) {
        loggedInTaskPage((ele) => {
          const dTable = $$('table.task-disambiguation', ele);
          expect(dTable).toBeFalsy();
          done();
        });
      });
    }); // end describe('deduplicated task with gpu dim')

    describe('Expired Task', function() {
      beforeEach(() => serveTask(4, 'Expired Task'));

      it('shows relevant task timing data', function(done) {
        loggedInTaskPage((ele) => {
          const taskTiming = $$('table.task-timing', ele);
          expect(taskTiming).toBeTruthy();
          const rows = $('tr', taskTiming);
          expect(rows.length).toEqual(9);

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];
          // Spot check some of the content
          expect(rows[1]).toHaveAttribute('hidden', 'Started hidden');
          expect(rows[4]).not.toHaveAttribute('hidden', 'Abandoned shown');
          expect(cell(6, 0)).toMatchTextContent('Pending Time');
          expect(cell(6, 1)).toMatchTextContent('10m  6s');
          expect(cell(7, 0)).toMatchTextContent('Total Overhead');
          expect(cell(7, 1)).toMatchTextContent('--');
          expect(cell(8, 0)).toMatchTextContent('Running Time');
          expect(cell(8, 1)).toMatchTextContent('--');

          done();
        });
      });

      it('does not show the multiple tries summary', function(done) {
        loggedInTaskPage((ele) => {
          const dTable = $$('table.task-disambiguation', ele);
          expect(dTable).toBeFalsy();
          done();
        });
      });
    }); // end describe('Expired Task')

  }); // end describe('html structure')

  describe('dynamic behavior', function() {
    describe('Completed task with 2 slices', function() {
      beforeEach(() => serveTask(1, 'Completed task with 2 slices'));

      it('shows and hides the extra details', function(done) {
        loggedInTaskPage((ele) => {
          ele._showDetails = false;
          ele.render();

          const lowerHalf = $('.task-info > tbody', ele)[1];
          expect(lowerHalf).toHaveAttribute('hidden');

          const btn = $$('.details button', ele);
          btn.click();
          expect(lowerHalf).not.toHaveAttribute('hidden');
          btn.click();
          expect(lowerHalf).toHaveAttribute('hidden');

          done();
        });
      });

      it('switches between slices with a tab', function(done) {
        loggedInTaskPage((ele) => {
          ele._setSlice(1); // also calls render

          const taskinfo = $$('table.task-info', ele);
          expect(taskinfo).not.toHaveClass('inactive');

          const tabs = $('.slice-picker .tab', ele);
          expect(tabs.length).toEqual(2);

          tabs[0].click();
          expect(taskinfo).toHaveClass('inactive');

          tabs[1].click();
          expect(taskinfo).not.toHaveClass('inactive');

          done();
        });
      });

      it('switches between log representations with a tab', function(done) {
        loggedInTaskPage((ele) => {
          ele._showRawOutput = false;
          ele.render();
          const tabs = $('.output-picker .tab', ele);
          expect(tabs.length).toEqual(2);
          expect(tabs[0]).not.toHaveAttribute('disabled');
          expect(tabs[0]).toHaveAttribute('selected');
          expect(tabs[1]).not.toHaveAttribute('selected');

          tabs[1].click();

          expect(tabs[0]).not.toHaveAttribute('disabled');
          expect(tabs[0]).not.toHaveAttribute('selected');
          expect(tabs[1]).toHaveAttribute('selected');

          tabs[0].click();
          expect(tabs[0]).not.toHaveAttribute('disabled');
          expect(tabs[0]).toHaveAttribute('selected');
          expect(tabs[1]).not.toHaveAttribute('selected');

          done();
        });
      });

      it('switches between wide and narrow logs', function(done) {
        loggedInTaskPage((ele) => {
          ele._wideLogs = false;
          ele._showRawOutput = true;
          ele.render();
          const logs = $$('.stdout.code', ele);
          expect(logs).toBeTruthy();

          const checkbox = $$('#wide_logs', ele);
          expect(checkbox).not.toHaveAttribute('checked');
          expect(logs).not.toHaveClass('wide');

          checkbox.click();

          expect(checkbox).toHaveAttribute('checked');
          expect(logs).toHaveClass('wide');

          checkbox.click();

          expect(checkbox).not.toHaveAttribute('checked');
          expect(logs).not.toHaveClass('wide');

          done();
        });
      });
    });
  });  // end describe('dynamic behavior')

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
      expect(calls.length).toBe(0, 'no POSTs on task-page');

      expectNoUnmatchedCalls(fetchMock);
    }

    it('fetches some extra requests with try number > 1', function(done) {
      serveTask(0, 'running task on try number 3');
      loggedInTaskPage((ele) => {
        let calls = fetchMock.calls(MATCHED, 'GET');
        expect(calls.length).toBe(2+3+3+2, '2 GETs from swarming-app, 3 from task-page,' +
                                           '3 counts * 1 slice, 2 from extra tries');
        // calls is an array of 2-length arrays with the first element
        // being the string of the url and the second element being
        // the options that were passed in
        const gets = calls.map((c) => c[0]);

        expect(gets).toContain(`/_ah/api/swarming/v1/task/${TEST_TASK_ID}/request`);
        expect(gets).toContain(`/_ah/api/swarming/v1/task/${TEST_TASK_ID}/result`+
                               '?include_performance_stats=true');
        expect(gets).toContain(`/_ah/api/swarming/v1/task/${TEST_TASK_ID}/stdout`);
        expect(gets).toContain('/_ah/api/swarming/v1/task/test0b3c0fac78101/result');
        expect(gets).toContain('/_ah/api/swarming/v1/task/test0b3c0fac78102/result');

        checkAuthorizationAndNoPosts(calls);
        done();
      });
    });

    it('makes auth\'d API calls when a logged in user views landing page', function(done) {
      serveTask(1, 'Completed task with 2 slices');
      loggedInTaskPage((ele) => {
        let calls = fetchMock.calls(MATCHED, 'GET');
        expect(calls.length).toBe(2+3+6, '2 GETs from swarming-app, 3 from task-page, ' +
                                         '3 counts * 2 slices');
        // calls is an array of 2-length arrays with the first element
        // being the string of the url and the second element being
        // the options that were passed in
        const gets = calls.map((c) => c[0]);

        expect(gets).toContain(`/_ah/api/swarming/v1/task/${TEST_TASK_ID}/request`);
        expect(gets).toContain(`/_ah/api/swarming/v1/task/${TEST_TASK_ID}/result`+
                               '?include_performance_stats=true');
        expect(gets).toContain(`/_ah/api/swarming/v1/task/${TEST_TASK_ID}/stdout`);
        // spot check one of the counts
        expect(gets).toContain('/_ah/api/swarming/v1/bots/count?'+
          'dimensions=builder%3Alinux_chromium_cfi_rel_ng&'+
          'dimensions=cores%3A32&dimensions=os%3AUbuntu-14.04&dimensions=cpu%3Ax86-64&'+
          'dimensions=pool%3Aluci.chromium.try&'+
          'dimensions=caches%3Abuilder_86e11e72bf6f8c2c424eb2189ffc073b483485cf12a42'+
          'b403fb5526a59936253_v2');

        checkAuthorizationAndNoPosts(calls);
        done();
      });
    });

    it('makes counts correctly with 1 slice', function(done) {
      serveTask(2, 'Pending task - 1 slice - no rich logs');
      loggedInTaskPage((ele) => {
        let calls = fetchMock.calls(MATCHED, 'GET');
        expect(calls.length).toBe(2+3+3, '2 GETs from swarming-app, 3 from task-page, '+
                                         '3 counts * 1 slice');
        // calls is an array of 2-length arrays with the first element
        // being the string of the url and the second element being
        // the options that were passed in
        const gets = calls.map((c) => c[0]);

        expect(gets).toContain(`/_ah/api/swarming/v1/task/${TEST_TASK_ID}/request`);
        expect(gets).toContain(`/_ah/api/swarming/v1/task/${TEST_TASK_ID}/result`+
                               '?include_performance_stats=true');
        expect(gets).toContain(`/_ah/api/swarming/v1/task/${TEST_TASK_ID}/stdout`);
        expect(gets).toContain('/_ah/api/swarming/v1/tasks/count?start=1549212360&state=RUNNING&'+
          'tags=device_os%3AN&tags=os%3AAndroid&tags=pool%3AChrome-GPU&tags=device_type%3Afoster');

        checkAuthorizationAndNoPosts(calls);
        done();
      });
    });

    it('makes a POST to retry a job', function(done) {
      serveTask(0, 'running task on try number 3');
      loggedInTaskPage((ele) => {
        fetchMock.resetHistory();
        fetchMock.post('/_ah/api/swarming/v1/tasks/new', {task_id: TEST_TASK_ID});

        const retryBtn = $$('.id_buttons button.retry', ele);
        expect(retryBtn).toBeTruthy();

        retryBtn.click();

        const dialog = $$('.retry-dialog', ele);
        expect(dialog).toBeTruthy();
        expect(dialog).toHaveClass('opened');

        const okBtn = $$('button.ok', dialog);
        expect(okBtn).toBeTruthy();

        // stub out the fetch so the new task doesn't load.
        ele._fetch = () => {};
        okBtn.click();

        fetchMock.flush(true).then(() => {
          // MATCHED calls are calls that we expect and specified in the
          // beforeEach at the top of this file.
          let calls = fetchMock.calls(MATCHED, 'GET');
          expect(calls.length).toBe(0);
          calls = fetchMock.calls(MATCHED, 'POST');
          expect(calls.length).toBe(1);

          expectNoUnmatchedCalls(fetchMock);
          done();
        });
      });
    });

    it('makes a POST to debug a job', function(done) {
      serveTask(1, 'Completed task with 2 slices');
      loggedInTaskPage((ele) => {
        fetchMock.resetHistory();
        fetchMock.post('/_ah/api/swarming/v1/tasks/new', {task_id: TEST_TASK_ID});

        const debugBtn = $$('.id_buttons button.debug', ele);
        expect(debugBtn).toBeTruthy();

        debugBtn.click();

        const dialog = $$('.retry-dialog', ele);
        expect(dialog).toBeTruthy();
        expect(dialog).toHaveClass('opened');

        // https://crbug.com/935736
        const useSameBot = $$('checkbox-sk.same-bot');
        expect(useSameBot).toBeTruthy();

        useSameBot.click();

        const okBtn = $$('button.ok', dialog);
        expect(okBtn).toBeTruthy();

        // stub out the fetch so the new task doesn't load.
        ele._fetch = () => {};
        okBtn.click();

        fetchMock.flush(true).then(() => {
          // MATCHED calls are calls that we expect and specified in the
          // beforeEach at the top of this file.
          let calls = fetchMock.calls(MATCHED, 'GET');
          expect(calls.length).toBe(0);
          calls = fetchMock.calls(MATCHED, 'POST');
          expect(calls.length).toBe(1);

          const options = calls[0][1];
          expect(options.headers.authorization).toBeTruthy();

          const body = JSON.parse(options.body);
          expect(body.name).toContain('leased to');

          const dims = body.properties.dimensions;
          expect(dims).toBeTruthy();
          dims.sort((a,b) => {
            return a.key.localeCompare(b.key);
          });
          expect(dims).toEqual([{
            key: 'id',
            value: 'swarm1931-c4'
          }, {
            key: 'pool',
            value: 'luci.chromium.try',
          }]);

          expectNoUnmatchedCalls(fetchMock);
          done();
        });
      });
    });

    it('makes a post to cancel a pending job', function(done) {
      serveTask(2, 'Pending task - 1 slice - no rich logs');
      loggedInTaskPage((ele) => {
        ele.permissions.cancel_task = true;
        ele.render();
        fetchMock.resetHistory();
        fetchMock.post(`/_ah/api/swarming/v1/task/${TEST_TASK_ID}/cancel`, {success: true});

        const cancelBtn = $$('.id_buttons button.cancel', ele);
        expect(cancelBtn).toBeTruthy();

        cancelBtn.click();

        const dialog = $$('.cancel-dialog', ele);
        expect(dialog).toBeTruthy();
        expect(dialog).toHaveClass('opened');

        const okBtn = $$('button.ok', dialog);
        expect(okBtn).toBeTruthy();

        okBtn.click();

        fetchMock.flush(true).then(() => {
          // MATCHED calls are calls that we expect and specified in the
          // beforeEach at the top of this file.
          let calls = fetchMock.calls(MATCHED, 'GET');
          expect(calls.length).toBe(0);
          calls = fetchMock.calls(MATCHED, 'POST');
          expect(calls.length).toBe(1);
          const call = calls[0];
          const options = call[1];
          expect(options.body).toEqual('{}');

          expectNoUnmatchedCalls(fetchMock);
          done();
        });
      });
    });

    it('makes a post to kill a running job', function(done) {
      serveTask(0, 'running task on try number 3');
      loggedInTaskPage((ele) => {
        ele.permissions.cancel_task = true;
        ele.render();
        fetchMock.resetHistory();
        fetchMock.post(`/_ah/api/swarming/v1/task/${TEST_TASK_ID}/cancel`, {success: true});

        const killBtn = $$('.id_buttons button.kill', ele);
        expect(killBtn).toBeTruthy();

        killBtn.click();

        const dialog = $$('.cancel-dialog', ele);
        expect(dialog).toBeTruthy();
        expect(dialog).toHaveClass('opened');

        const okBtn = $$('button.ok', dialog);
        expect(okBtn).toBeTruthy();

        okBtn.click();

        fetchMock.flush(true).then(() => {
          // MATCHED calls are calls that we expect and specified in the
          // beforeEach at the top of this file.
          let calls = fetchMock.calls(MATCHED, 'GET');
          expect(calls.length).toBe(0);
          calls = fetchMock.calls(MATCHED, 'POST');
          expect(calls.length).toBe(1);
          const call = calls[0];
          const options = call[1];
          expect(options.body).toEqual('{"kill_running":true}');

          expectNoUnmatchedCalls(fetchMock);
          done();
        });
      });
    });

  }); // end describe('api calls')
});

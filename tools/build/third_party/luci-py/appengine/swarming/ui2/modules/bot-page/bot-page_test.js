// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import 'modules/bot-page'

describe('bot-page', function() {
  // Instead of using import, we use require. Otherwise,
  // the concatenation trick we do doesn't play well with webpack, which would
  // leak dependencies (e.g. bot-list's 'column' function to task-list) and
  // try to import things multiple times.
  const { $, $$ } = require('common-sk/modules/dom');
  const { customMatchers, expectNoUnmatchedCalls, mockAppGETs } = require('modules/test_util');
  const { fetchMock, MATCHED, UNMATCHED } = require('fetch-mock');
  const { botDataMap, eventsMap, tasksMap } = require('modules/bot-page/test_data');


  const TEST_BOT_ID = 'example-gce-001';

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
    jasmine.clock().mockDate(new Date(Date.UTC(2019, 1, 12, 18, 46, 22, 1234)));
  });

  afterEach(function() {
    jasmine.clock().uninstall();
  });

  // calls the test callback with one element 'ele', a created <bot-page>.
  function createElement(test) {
    return window.customElements.whenDefined('bot-page').then(() => {
      container.innerHTML = `<bot-page client_id=for_test testing_offline=true></bot-page>`;
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
  // <bot-page> after the user has logged in.
  function loggedInBotPage(test, emptyBotId) {
    createElement((ele) => {
      if (!emptyBotId) {
        ele._botId = TEST_BOT_ID;
      }
      userLogsIn(ele, () => {
        test(ele);
      });
    });
  }

  function serveBot(botName) {
    const data = botDataMap[botName];
    const tasks = {items: tasksMap['SkiaGPU']};
    const events = {items: eventsMap['SkiaGPU']};

    fetchMock.get(`/_ah/api/swarming/v1/bot/${TEST_BOT_ID}/get`, data);
    fetchMock.get(`glob:/_ah/api/swarming/v1/bot/${TEST_BOT_ID}/tasks*`, tasks);
    fetchMock.get(`glob:/_ah/api/swarming/v1/bot/${TEST_BOT_ID}/events*`, events);
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

      it('hides all other elements', function(done) {
        createElement((ele) => {
          // other stuff is hidden
          let content = $('main > *');
          expect(content.length).toEqual(4); // 3 top level sections and a message
          for (const div of content) {
            if (div.tagName !== 'H2') {
              expect(div).toHaveAttribute('hidden');
            }
          }
          ele._botId = TEST_BOT_ID;
          ele.render();
          // even if an id was given
          content = $('main > *');
          expect(content.length).toEqual(4); // 3 top level sections and a message
          for (const div of content) {
            if (div.tagName !== 'H2') {
              expect(div).toHaveAttribute('hidden');
            }
          }
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
        fetchMock.get('glob:/_ah/api/swarming/v1/bot/*', 403,
                      { overwriteRoutes: true });
      }

      beforeEach(notAuthorized);

      it('tells the user they should change accounts', function(done) {
        loggedInBotPage((ele) => {
          const loginMessage = $$('swarming-app>main .message', ele);
          expect(loginMessage).toBeTruthy();
          expect(loginMessage).not.toHaveAttribute('hidden', 'Message should not be hidden');
          expect(loginMessage.textContent).toContain('different account');
          done();
        });
      });

      it('does not display logs or task details', function(done) {
        loggedInBotPage((ele) => {
          const content = $$('main .content', ele);
          expect(content).toBeTruthy();
          expect(content).toHaveAttribute('hidden');
          done();
        });
      });
    }); // end describe('when logged in as unauthorized user')

    describe('authorized user, but no bot id', function() {

      it('tells the user they should enter a bot id', function(done) {
        loggedInBotPage((ele) => {
          const loginMessage = $$('.id_buttons .message', ele);
          expect(loginMessage).toBeTruthy();
          expect(loginMessage.textContent).toContain('Enter a Bot ID');
          done();
        }, true);
      });

      it('does not display filters or tasks', function(done) {
        loggedInBotPage((ele) => {
          const content = $$('main .content', ele);
          expect(content).toBeTruthy();
          expect(content).toHaveAttribute('hidden');
          done();
        }, true);
      });
    }); // end describe('authorized user, but no taskid')

    describe('gpu bot with a running task', function() {
      beforeEach(() => serveBot('running'));

      it('renders some of the bot data', function(done) {
        loggedInBotPage((ele) => {
          const dataTable = $$('table.data_table', ele);
          expect(dataTable).toBeTruthy();

          const rows = $('tr', dataTable);
          expect(rows).toBeTruthy();
          expect(rows.length).toBeTruthy();

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];

          const deleteBtn = $$('button.delete', cell(0, 2));
          expect(deleteBtn).toBeTruthy();
          expect(deleteBtn).toHaveClass('hidden');
          const shutDownBtn = $$('button.shut_down', cell(0, 2));
          expect(shutDownBtn).toBeTruthy();
          expect(shutDownBtn).not.toHaveClass('hidden');

          expect(rows[2]).toHaveClass('hidden', 'not quarantined');
          expect(rows[3]).toHaveClass('hidden', 'not dead');
          expect(rows[4]).toHaveClass('hidden', 'not in maintenance');
          expect(cell(5, 0)).toMatchTextContent('Current Task');
          expect(cell(5, 1)).toMatchTextContent('42fb00e06d95be11');
          expect(cell(5, 1).innerHTML).toContain('<a ', 'has a link');
          expect(cell(5, 1).innerHTML).toContain('href="/task?id=42fb00e06d95be10"');
          expect(cell(6, 0)).toMatchTextContent('Dimensions');
          expect(cell(11, 0)).toMatchTextContent('gpu');
          expect(cell(11, 1)).toMatchTextContent('NVIDIA (10de) | ' +
            'NVIDIA Quadro P400 (10de:1cb3) | NVIDIA Quadro P400 (10de:1cb3-25.21.14.1678)');
          expect(cell(23, 0)).toMatchTextContent('Bot Version');
          expect(rows[23]).toHaveClass('old_version');
          done();
        });
      });

      it('renders the tasks in a table', function(done) {
        loggedInBotPage((ele) => {
          ele._showEvents = false;
          ele.render();
          const tasksTable = $$('table.tasks_table', ele);
          expect(tasksTable).toBeTruthy();

          const rows = $('tr', tasksTable);
          expect(rows).toBeTruthy();
          expect(rows.length).toEqual(1 + 30, '1 for header, 30 tasks');

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];

          // row 0 is the header
          expect(cell(1, 0)).toMatchTextContent('Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Debug-All-ANGLE');
          expect(cell(1, 0).innerHTML).toContain('<a ', 'has a link');
          expect(cell(1, 0).innerHTML).toContain('href="/task?id=43004cb4fca98110"');
          expect(cell(1, 2)).toMatchTextContent('7m 20s');
          expect(cell(1, 3)).toMatchTextContent('RUNNING');
          expect(rows[1]).toHaveClass('pending_task');
          expect(cell(2, 2)).toMatchTextContent('3m 51s');
          expect(cell(2, 3)).toMatchTextContent('SUCCESS');
          expect(rows[2]).not.toHaveClass('pending_task');
          expect(rows[2]).not.toHaveClass('failed_task)');
          expect(rows[2]).not.toHaveClass('exception');
          expect(rows[2]).not.toHaveClass('bot_died');

          const eBtn = $$('main button.more_events', ele);
          expect(eBtn).toBeFalsy();

          const tBtn = $$('main button.more_tasks', ele);
          expect(tBtn).toBeTruthy();
          done();
        });
      });

      it('renders all events in a table', function(done) {
        loggedInBotPage((ele) => {
          ele._showEvents = true;
          ele._showAll = true;
          ele.render();
          const eventsTable = $$('table.events_table', ele);
          expect(eventsTable).toBeTruthy();

          const rows = $('tr', eventsTable);
          expect(rows).toBeTruthy();
          expect(rows.length).toEqual(1 + 50, '1 for header, 50 events');

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];

          // row 0 is the header
          expect(cell(1, 0)).toMatchTextContent('');
          expect(cell(1, 1)).toMatchTextContent('request_task');
          expect(cell(1, 3).innerHTML).toContain('<a ', 'has a link');
          expect(cell(1, 3).innerHTML).toContain('href="/task?id=4300ceb85b93e010"');
          expect(cell(1, 4)).toMatchTextContent('abcdoeraym');
          expect(cell(1, 4)).not.toHaveClass('old_version');

          expect(cell(15, 0)).toMatchTextContent('About to restart: ' +
                'Updating to abcdoeraymeyouandme');
          expect(cell(15, 1)).toMatchTextContent('bot_shutdown');
          expect(cell(15, 3)).toMatchTextContent('');
          expect(cell(15, 4)).toMatchTextContent('6fda8587d8');
          expect(cell(15, 4)).toHaveClass('old_version');
          done();
        });
      });

      it('renders some events in a table', function(done) {
        loggedInBotPage((ele) => {
          ele._showEvents = true;
          ele._showAll = false;
          ele.render();
          const eventsTable = $$('table.events_table', ele);
          expect(eventsTable).toBeTruthy();

          const rows = $('tr', eventsTable);
          expect(rows).toBeTruthy();
          expect(rows.length).toEqual(1 + 1, '1 for header, 1 shown event');

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];

          // row 0 is the header
          expect(cell(1, 0)).toMatchTextContent('About to restart: ' +
                'Updating to abcdoeraymeyouandme');
          expect(cell(1, 1)).toMatchTextContent('bot_shutdown');
          expect(cell(1, 3)).toMatchTextContent('');
          expect(cell(1, 4)).toMatchTextContent('6fda8587d8');
          expect(cell(1, 4)).toHaveClass('old_version');

          const eBtn = $$('main button.more_events', ele);
          expect(eBtn).toBeTruthy();

          const tBtn = $$('main button.more_tasks', ele);
          expect(tBtn).toBeFalsy();
          done();
        });
      });

      it('disables buttons for unprivileged users', function(done) {
        loggedInBotPage((ele) => {
          ele.permissions.cancel_task = false;
          ele.permissions.delete_bot = false;
          ele.permissions.terminate_bot = false;
          ele.render();
          const killBtn = $$('main button.kill', ele);
          expect(killBtn).toBeTruthy();
          expect(killBtn).toHaveAttribute('disabled');

          const deleteBtn = $$('main button.delete', ele);
          expect(deleteBtn).toBeTruthy();
          expect(deleteBtn).toHaveAttribute('disabled');

          const tBtn = $$('main button.shut_down', ele);
          expect(tBtn).toBeTruthy();
          expect(tBtn).toHaveAttribute('disabled');

          done();
        });
      });

      it('enables buttons for privileged users', function(done) {
        loggedInBotPage((ele) => {
          ele.permissions.cancel_task = true;
          ele.permissions.delete_bot = true;
          ele.permissions.terminate_bot = true;
          ele.render();
          const killBtn = $$('main button.kill', ele);
          expect(killBtn).toBeTruthy();
          expect(killBtn).not.toHaveAttribute('disabled');

          const deleteBtn = $$('main button.delete', ele);
          expect(deleteBtn).toBeTruthy();
          expect(deleteBtn).not.toHaveAttribute('disabled');

          const tBtn = $$('main button.shut_down', ele);
          expect(tBtn).toBeTruthy();
          expect(tBtn).not.toHaveAttribute('disabled');

          done();
        });
      });

      it('does not show android devices section', function(done) {
        loggedInBotPage((ele) => {
          const devTable = $$('table.devices', ele);
          expect(devTable).toBeFalsy();
          done();
        });
      });

      it('has a summary table of the tasks', function(done) {
        loggedInBotPage((ele) => {
          const sTable = $$('bot-page-summary table', ele);
          expect(sTable).toBeTruthy();

          const rows = $('tr', sTable);
          expect(rows).toBeTruthy();
          expect(rows.length).toEqual(1 + 15 + 1, 'header, 15 tasks, footer');

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];

          expect(cell(2, 0)).toMatchTextContent(
                'Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Rel...');

          expect(cell(5, 0)).toMatchTextContent(
                'Perf-Win10-Clang-Golo-GPU-QuadroP400-x86_64-Deb...');
          expect(cell(5, 1)).toMatchTextContent('2'); // Total
          expect(cell(5, 2)).toMatchTextContent('0'); // Success
          expect(cell(5, 3)).toMatchTextContent('1'); // Failed
          expect(cell(5, 4)).toMatchTextContent('1'); // Died
          expect(cell(5, 5)).toMatchTextContent('16m 57s'); // duration
          expect(cell(5, 6)).toMatchTextContent('11.85s'); // overhead
          expect(cell(5, 7)).toMatchTextContent('10.5%'); // percent

          done();
        });
      });
    }); // end describe('gpu bot with a running task')

    describe('quarantined android bot', function() {
      beforeEach(() => serveBot('quarantined'));

      it('displays a quarantined message', function(done) {
        loggedInBotPage((ele) => {
          const dataTable = $$('table.data_table', ele);
          expect(dataTable).toBeTruthy();

          const rows = $('tr', dataTable);
          expect(rows).toBeTruthy();
          expect(rows.length).toBeTruthy();

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];

          expect(rows[2]).not.toHaveClass('hidden', 'quarantined');
          expect(cell(2, 1)).toMatchTextContent('No available devices.');
          expect(rows[3]).toHaveClass('hidden', 'not dead');
          expect(rows[4]).toHaveClass('hidden', 'not in maintenance');
          expect(cell(5, 0)).toMatchTextContent('Current Task');
          expect(cell(5, 1)).toMatchTextContent('idle');

          done();
        });
      });

      it('shows android devices section', function(done) {
        loggedInBotPage((ele) => {
          const devTable = $$('table.devices', ele);
          expect(devTable).toBeTruthy();

          const rows = $('tr', devTable);
          expect(rows).toBeTruthy();
          expect(rows.length).toEqual(2); // 1 for header, 1 device

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];

          expect(cell(1, 0)).toMatchTextContent('3BE9F057');
          expect(cell(1, 1)).toMatchTextContent('100');
          expect(cell(1, 2)).toMatchTextContent('???');
          expect(cell(1, 3)).toMatchTextContent('still booting (sys.boot_completed)');

          done();
        });
      });
    }); // describe('quarantined android bot')

    describe('dead machine provider bot', function() {
      beforeEach(() => serveBot('dead'));

      it('displays dead and mp related info', function(done) {
        loggedInBotPage((ele) => {
          const dataTable = $$('table.data_table', ele);
          expect(dataTable).toBeTruthy();

          const rows = $('tr', dataTable);
          expect(rows).toBeTruthy();
          expect(rows.length).toBeTruthy();

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];

          const deleteBtn = $$('button.delete', cell(0, 2));
          expect(deleteBtn).toBeTruthy();
          expect(deleteBtn).not.toHaveClass('hidden');
          const shutDownBtn = $$('button.shut_down', cell(0, 2));
          expect(shutDownBtn).toBeTruthy();
          expect(shutDownBtn).toHaveClass('hidden');

          expect(rows[1]).toHaveClass('dead');
          expect(rows[2]).toHaveClass('hidden', 'not quarantined');
          expect(rows[3]).not.toHaveClass('hidden', 'dead');
          expect(rows[4]).toHaveClass('hidden', 'not in maintenance');
          expect(cell(5, 0)).toMatchTextContent('Died on Task');
          expect(rows[27]).not.toHaveAttribute('hidden');
          expect(cell(27, 0)).toMatchTextContent('Machine Provider Lease ID');
          expect(cell(27, 1)).toMatchTextContent('f69394d5f68b1f1e6c5f13e82ba4ccf72de7e6a0');
          expect(cell(27, 1).innerHTML).toContain('<a ', 'has a link');
          expect(cell(27, 1).innerHTML).toContain('href="https://example.com/leases/'+
                                                  'f69394d5f68b1f1e6c5f13e82ba4ccf72de7e6a0"');

          done();
        });
      });

      it('does not display kill task on dead bot', function(done) {
        loggedInBotPage((ele) => {
          ele._bot.task_id = 't1233';
          const dataTable = $$('table.data_table', ele);
          expect(dataTable).toBeTruthy();

          const rows = $('tr', dataTable);
          expect(rows).toBeTruthy();
          expect(rows.length).toBeTruthy();

          // little helper for readability
          const cell = (r, c) => rows[r].children[c];

          const killBtn = $$('button.kill', cell(5, 2));
          expect(killBtn).toBeTruthy();
          expect(killBtn).toHaveAttribute('hidden');

          done();
        });
      });
    }); // describe('dead machine provider bot')

  }); // end describe('html structure')

  describe('dynamic behavior', function() {
    it('hides and unhides extra details with a button', function(done) {
      serveBot('running');
      loggedInBotPage((ele) => {
        ele._showState = false;
        ele.render();

        const state = $$('.bot_state', ele);
        expect(state).toBeTruthy();
        expect(state).toHaveAttribute('hidden');

        const stateBtn = $$('button.state', ele);
        expect(stateBtn).toBeTruthy();

        stateBtn.click();

        expect(state).not.toHaveAttribute('hidden');

        stateBtn.click();
        expect(state).toHaveAttribute('hidden');

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
      expect(calls.length).toBe(0, 'no POSTs on bot-page');

      expectNoUnmatchedCalls(fetchMock);
    }

    it('makes auth\'d API calls when a logged in user views landing page', function(done) {
      serveBot('running');
      loggedInBotPage((ele) => {
        let calls = fetchMock.calls(MATCHED, 'GET');
        expect(calls.length).toBe(2+3, '2 GETs from swarming-app, 3 from bot-page');
        // calls is an array of 2-length arrays with the first element
        // being the string of the url and the second element being
        // the options that were passed in
        const gets = calls.map((c) => c[0]);

        expect(gets).toContain(`/_ah/api/swarming/v1/bot/${TEST_BOT_ID}/get`);
        checkAuthorizationAndNoPosts(calls);
        done();
      });
    });

    it('can kill a running task', function(done) {
      serveBot('running');
      loggedInBotPage((ele) => {
        ele.permissions.cancel_task = true;
        ele.render();
        fetchMock.resetHistory();
        // This is the task_id on the 'running' bot.
        fetchMock.post('/_ah/api/swarming/v1/task/42fb00e06d95be11/cancel', {success: true});

        const killBtn = $$('main button.kill', ele);
        expect(killBtn).toBeTruthy();

        killBtn.click();

        const dialog = $$('.prompt-dialog', ele);
        expect(dialog).toBeTruthy();
        expect(dialog).toHaveClass('opened');

        const okBtn = $$('button.ok', dialog);
        expect(okBtn).toBeTruthy();

        okBtn.click();

        fetchMock.flush(true).then(() => {
          // MATCHED calls are calls that we expect and specified in the
          // beforeEach at the top of this file.
          expectNoUnmatchedCalls(fetchMock);
          let calls = fetchMock.calls(MATCHED, 'GET');
          expect(calls.length).toBe(0);
          calls = fetchMock.calls(MATCHED, 'POST');
          expect(calls.length).toBe(1);
          const call = calls[0];
          const options = call[1];
          expect(options.body).toEqual('{"kill_running":true}');

          done();
        });
      });
    });

    it('can terminate a non-dead bot', function(done) {
      serveBot('running');
      loggedInBotPage((ele) => {
        ele.permissions.terminate_bot = true;
        ele.render();
        fetchMock.resetHistory();
        // This is the task_id on the 'running' bot.
        fetchMock.post(`/_ah/api/swarming/v1/bot/${TEST_BOT_ID}/terminate`, {success: true});

        const tBtn = $$('main button.shut_down', ele);
        expect(tBtn).toBeTruthy();

        tBtn.click();

        const dialog = $$('.prompt-dialog', ele);
        expect(dialog).toBeTruthy();
        expect(dialog).toHaveClass('opened');

        const okBtn = $$('button.ok', dialog);
        expect(okBtn).toBeTruthy();

        okBtn.click();

        fetchMock.flush(true).then(() => {
          // MATCHED calls are calls that we expect and specified in the
          // beforeEach at the top of this file.
          expectNoUnmatchedCalls(fetchMock);
          let calls = fetchMock.calls(MATCHED, 'GET');
          expect(calls.length).toBe(0);
          calls = fetchMock.calls(MATCHED, 'POST');
          expect(calls.length).toBe(1);

          done();
        });
      });
    });

    it('can delete a dead bot', function(done) {
      serveBot('dead');
      loggedInBotPage((ele) => {
        ele.permissions.delete_bot = true;
        ele.render();
        fetchMock.resetHistory();
        // This is the task_id on the 'running' bot.
        fetchMock.post(`/_ah/api/swarming/v1/bot/${TEST_BOT_ID}/delete`, {success: true});

        const deleteBtn = $$('main button.delete', ele);
        expect(deleteBtn).toBeTruthy();

        deleteBtn.click();

        const dialog = $$('.prompt-dialog', ele);
        expect(dialog).toBeTruthy();
        expect(dialog).toHaveClass('opened');

        const okBtn = $$('button.ok', dialog);
        expect(okBtn).toBeTruthy();

        okBtn.click();

        fetchMock.flush(true).then(() => {
          // MATCHED calls are calls that we expect and specified in the
          // beforeEach at the top of this file.
          expectNoUnmatchedCalls(fetchMock);
          let calls = fetchMock.calls(MATCHED, 'GET');
          expect(calls.length).toBe(0);
          calls = fetchMock.calls(MATCHED, 'POST');
          expect(calls.length).toBe(1);

          done();
        });
      });
    });

    it('can fetch more tasks', function(done) {
      serveBot('running');
      loggedInBotPage((ele) => {
        ele._taskCursor = 'myCursor';
        ele._showEvents = false;
        ele.render();
        fetchMock.reset(); // clears history and routes

        fetchMock.get(`glob:/_ah/api/swarming/v1/bot/${TEST_BOT_ID}/tasks*`, {
          items: tasksMap['SkiaGPU'],
          cursor: 'newCursor',
        });

        const tBtn = $$('main button.more_tasks', ele);
        expect(tBtn).toBeTruthy();

        tBtn.click();

        fetchMock.flush(true).then(() => {
          // MATCHED calls are calls that we expect and specified in the
          // beforeEach at the top of this file.
          expectNoUnmatchedCalls(fetchMock);
          let calls = fetchMock.calls(MATCHED, 'GET');
          expect(calls.length).toBe(1);

          const url = calls[0][0];
          // spot check a few fields
          expect(url).toContain('state');
          expect(url).toContain('name');
          expect(url).toContain('limit=30');
          // validate cursor
          expect(url).toContain('cursor=myCursor');
          expect(ele._taskCursor).toEqual('newCursor', 'cursor should update');
          expect(ele._tasks.length).toEqual(30+30, '30 initial tasks, 30 new tasks');

          done();
        });
      });
    });

    it('can fetch more events', function(done) {
      serveBot('running');
      loggedInBotPage((ele) => {
        ele._eventsCursor = 'myCursor';
        ele._showEvents = true;
        ele.render();
        fetchMock.reset(); // clears history and routes

        fetchMock.get(`glob:/_ah/api/swarming/v1/bot/${TEST_BOT_ID}/events*`, {
          items: eventsMap['SkiaGPU'],
          cursor: 'newCursor',
        });

        const eBtn = $$('main button.more_events', ele);
        expect(eBtn).toBeTruthy();

        eBtn.click();

        fetchMock.flush(true).then(() => {
          // MATCHED calls are calls that we expect and specified in the
          // beforeEach at the top of this file.
          expectNoUnmatchedCalls(fetchMock);
          let calls = fetchMock.calls(MATCHED, 'GET');
          expect(calls.length).toBe(1);

          const url = calls[0][0];
          // spot check a few fields
          expect(url).toContain('event_type');
          expect(url).toContain('task_id');
          expect(url).toContain('limit=50');
          // validate cursor
          expect(url).toContain('cursor=myCursor');
          expect(ele._eventsCursor).toEqual('newCursor', 'cursor should update');
          expect(ele._events.length).toEqual(50+50, '50 initial tasks, 50 new tasks');

          done();
        });
      });
    });

    it('reloads tasks and events on refresh', function(done) {
      serveBot('running');
      loggedInBotPage((ele) => {
        ele.render();
        fetchMock.reset(); // clears history and routes

        serveBot('running');

        const rBtn = $$('main button.refresh', ele);
        expect(rBtn).toBeTruthy();

        rBtn.click();

        fetchMock.flush(true).then(() => {
          // MATCHED calls are calls that we expect and specified in the
          // beforeEach at the top of this file.
          expectNoUnmatchedCalls(fetchMock);
          let calls = fetchMock.calls(MATCHED, 'GET');
          expect(calls.length).toBe(3);

          done();
        });
      });
    });

  }); // end describe('api calls')
});

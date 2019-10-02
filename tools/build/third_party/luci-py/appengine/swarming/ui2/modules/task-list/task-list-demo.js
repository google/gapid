// Copyright 2018 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import './index.js'

import { tasks_20 } from './test_data'
import { fleetDimensions } from '../bot-list/test_data'
import { requireLogin, mockAuthdAppGETs } from '../test_util'


(function(){
// Can't use import fetch-mock because the library isn't quite set up
// correctly for it, and we get strange errors about 'this' not being defined.
const fetchMock = require('fetch-mock');

mockAuthdAppGETs(fetchMock, {cancel_task: true});

fetchMock.get('glob:/_ah/api/swarming/v1/tasks/list?*',
              requireLogin(tasks_20, 2000));

fetchMock.get('/_ah/api/swarming/v1/bots/dimensions',
              requireLogin(fleetDimensions));

fetchMock.get('glob:/_ah/api/swarming/v1/tasks/count?*',
              requireLogin(() => {
                return {'count': "" + Math.round(Math.random() * 10000)};
              }, 800));


fetchMock.post('/_ah/api/swarming/v1/tasks/cancel', requireLogin({'matched': 17}, 1000));

// Everything else
fetchMock.catch(404);

// autologin for ease of testing locally - comment this out if using the real flow.
document.querySelector('oauth-login')._logIn();
})();

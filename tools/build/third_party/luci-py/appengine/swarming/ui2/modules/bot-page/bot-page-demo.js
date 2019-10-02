// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.
import './index.js'

import { botData, eventsMap, tasksMap } from './test_data'
import { requireLogin, mockAuthdAppGETs } from '../test_util'
import { $$ } from 'common-sk/modules/dom'

(function(){
// Can't use import fetch-mock because the library isn't quite set up
// correctly for it, and we get strange errors about 'this' not being defined.
const fetchMock = require('fetch-mock');

mockAuthdAppGETs(fetchMock, {
  delete_bot: true,
  terminate_bot: true,
  cancel_task: true,
});

fetchMock.get('glob:/_ah/api/swarming/v1/bot/*/get',
              requireLogin(botData, 100));

fetchMock.get('glob:/_ah/api/swarming/v1/bot/*/tasks?*',
              requireLogin({items: tasksMap['SkiaGPU']}, 100));

fetchMock.get('glob:/_ah/api/swarming/v1/bot/*/events?*',
              requireLogin({items: eventsMap['SkiaGPU']}, 100));

fetchMock.post('/_ah/api/swarming/v1/task/42fb00e06d95be11/cancel',
               requireLogin({success:true}, 200));

fetchMock.post('glob:/_ah/api/swarming/v1/bot/*/terminate',
               requireLogin({success:true}, 200));

fetchMock.post('glob:/_ah/api/swarming/v1/bot/dead/delete',
               requireLogin({success:true}, 200));

// Everything else
fetchMock.catch(404);

const ele = $$('bot-page');
if (!ele._botId) {
  ele._botId = 'running';
}
// autologin for ease of testing locally - comment this out if using the real flow.
$$('oauth-login')._logIn();
})();

// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.
import './index.js'

import { requireLogin, mockAuthdAppGETs } from '../test_util'
import { $$ } from 'common-sk/modules/dom'

(function(){
const fetchMock = require('fetch-mock');

mockAuthdAppGETs(fetchMock, {delete_bot: true});

fetchMock.get('glob:/_ah/api/swarming/v1/bots/count?*',
              requireLogin({dead: 3}, 700));

fetchMock.get('glob:/_ah/api/swarming/v1/bots/list?*',
              requireLogin({
                items: [{bot_id: 'bot-1'}, {bot_id: 'bot-2'}, {bot_id: 'bot-3'}]
              }, 700));

fetchMock.post('glob:/_ah/api/swarming/v1/bot/*/delete', requireLogin(200, 1500));

fetchMock.catch(404);

$$('bot-mass-delete').show();
})();
// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.
import './index.js'

import { requireLogin, mockAuthdAppGETs } from '../test_util'
import { $$ } from 'common-sk/modules/dom'

(function(){
const fetchMock = require('fetch-mock');

mockAuthdAppGETs(fetchMock, {cancel_task: true});

fetchMock.get('glob:/_ah/api/swarming/v1/tasks/count?*',
              requireLogin({count: 17}, 700));

fetchMock.post('/_ah/api/swarming/v1/tasks/cancel',
              requireLogin({matched: 22}, 700))

fetchMock.catch(404);

$$('task-mass-cancel').show();
})();
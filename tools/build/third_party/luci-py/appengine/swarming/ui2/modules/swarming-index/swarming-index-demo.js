// Copyright 2018 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import './index.js'
import { requireLogin, mockAuthdAppGETs } from '../test_util'

(function(){

// Can't use import fetch-mock because the library isn't quite set up
// correctly for it, and we get strange errors about 'this' not being defined.
const fetchMock = require('fetch-mock');


mockAuthdAppGETs(fetchMock, {
  get_bootstrap_token: true
});

const logged_in_token = {
  bootstrap_token: '8675309JennyDontChangeYourNumber8675309'
};

fetchMock.post('/_ah/api/swarming/v1/server/token',
               requireLogin(logged_in_token, 1500));

// Everything else
fetchMock.catch(404);

})();
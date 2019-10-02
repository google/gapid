# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

# Bootstraps a swarming bot to connect host_url. 'host_url = "foo"' and
# 'bootstrap_token = "<urlsafe string>"' are added on top of this file by the
# HTTP handler that serves this file. This file must be concise since it is
# executed directly at the CLI.
# pylint: disable=E0602
import os, urllib, sys
urllib.urlretrieve(
    '%s/bot_code?tok=%s' % (host_url, bootstrap_token), 'swarming_bot.zip')
os.execv(sys.executable, [sys.executable, 'swarming_bot.zip'])

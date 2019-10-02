# coding: utf-8
# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""This file is meant to be overriden by the server's specific copy.

You can upload a new version via /restricted/upload/bot_config.

There's 3 types of functions in this file:
  - get_*() to return properties to describe this bot.
  - on_*() as hooks based on events happening on the bot.
  - setup_*() to setup global state on the host.

This file shouldn't import from other scripts in this directory except
os_utilities which is guaranteed to be usable as an API. It's fine to import
from stdlib.

There can be two copies of this file. The base file and a bot specific version:
  - For get_*() functions, the bot specific version is called first,
    and if missing, the general version is called.
  - For on_*() functions, the hook in the general bot_config.py is called first,
    then the hook in the bot specific bot_config.py is called.
  - For setup_*(), the bot specific bot_config.py is never used.

This file contains unicode to confirm UTF-8 encoded file is well supported.
Here's a pile of poo: 💩
"""

import os
import sys

from api import os_utilities
from api import platforms

# pylint: disable=unused-argument


def get_dimensions(bot):
  # pylint: disable=line-too-long
  """Returns dict with the bot's dimensions.

  The dimensions are what are used to select the bot that can run each task.

  The bot id will be automatically selected based on the hostname with
  os_utilities.get_dimensions(). If you want something more special, specify it
  in your bot_config.py and override the item 'id'.

  The dimensions returned here will be joined with server defined dimensions
  (extracted from bots.cfg config file based on the bot id). Server defined
  dimensions override the ones provided by the bot. See bot.Bot.dimensions for
  more information.

  See
  https://chromium.googlesource.com/infra/luci/luci-py.git/+/master/appengine/swarming/doc/Magic-Values.md

  Arguments:
  - bot: bot.Bot instance or None. See ../api/bot.py.
  """
  return os_utilities.get_dimensions()


def get_settings(bot):
  """Returns settings for this bot.

  This function should be fast and mostly (preferably) constant.
  """
  # Here is the default values. Keep in sync with the default values in
  # ../bot_code/bot_main.py.
  return {
    # Free partition (disk) space to keep and to self-quarantine on.
    #
    # The exact minimum free space can be calculated with:
    #   max(disk_size * 'min_percent', min('size', disk_size * 'max_percent'))
    # where 'min_percent' and 'percent' are relative to the total partition
    # size. Setting any value to 0 disables this setting. In practice, with the
    # default values:
    # - For disks <27GB this will be "disk_size * max_percent"
    # - For disks 27GB-80GB this will be 4GB
    # - For disks >80GB this will be "disk_size * min_percent"
    #
    # When trimming the cache, 'wiggle' is added to the value selected above.
    #
    'free_partition': {
      # Settings specifically for the OS root partition: / on Linux
      # distributions and OSX, generally (but not necessarily) C:\ on Windows).
      # If the bot runs on the root partition, these values are ignored.
      'root': {
        # Minimum free space in bytes to use, if lower than 'max_percent'.
        'size': 1 * 1024*1024*1024,
        # Maximum free space in percent to ensure to keep free, if lower than
        # 'size'.
        'max_percent': 10.,
        # Minimum of of free space percentage, even if higher than 'size'.
        'min_percent': 6.,
      },
      # Settings specifically for the partition in which the bot runs on. These
      # values are expected to be higher than 'root' values.
      'bot': {
        'size': 4 * 1024*1024*1024,
        'max_percent': 15.,
        'min_percent': 7.,
        # Number of bytes to add to the minimum value selected above when
        # calculating the isolated cache trimming. This is to ensure that system
        # level processes writing logs and such do not cause to criss the
        # self-quarantine line while the bot is idle.
        'wiggle': 250 * 1024*1024,
      },
    },
    # Local caches settings.
    'caches': {
      # Local isolated cache settings, used for isolated tasks. The cache actual
      # size is bounded by the lesser of all 3:
      # - The cache total size in bytes
      # - The number of items in the cache
      # - The cache is further trimmed until 'free_partition' value is
      #   respected.
      'isolated': {
        # Maximum local isolated cache size in bytes.
        'size': 50 * 1024*1024*1024,
        # Maximum number of items in the local isolated cache.
        'items': 50*1024,
      },
    },
  }


def get_state(bot):
  # pylint: disable=line-too-long
  """Returns dict with a state of the bot reported to the server with each poll.

  It is only for dynamic state that changes while bot is running for information
  for the sysadmins.

  The server can not use this state for immediate scheduling purposes (use
  'dimensions' for that), but it can use it for maintenance and bookkeeping
  tasks.

  See
  https://chromium.googlesource.com/infra/luci/luci-py.git/+/master/appengine/swarming/doc/Magic-Values.md

  Arguments:
  - bot: bot.Bot instance or None. See ../api/bot.py.
  """
  return os_utilities.get_state()


def get_authentication_headers(bot):
  """Returns authentication headers and their expiration time.

  The returned headers will be passed with each HTTP request to the Swarming
  server (and only Swarming server). The bot will use the returned headers until
  they are close to expiration (usually 6 min, see AUTH_HEADERS_EXPIRATION_SEC
  in remote_client.py), and then it'll attempt to refresh them by calling
  get_authentication_headers again.

  Can be used to implement per-bot authentication. If no headers are returned,
  the server will use only IP whitelist for bot authentication.

  On GCE will use OAuth token of the default GCE service account. It should have
  "User info" API scope enabled (this can be set when starting an instance). The
  server should be configured (via bots.cfg) to trust this account (see
  'require_service_account' in bots.proto).

  May be called by different threads, but never concurrently.

  Arguments:
  - bot: bot.Bot instance. See ../api/bot.py.

  Returns:
    Tuple (dict with headers or None, unix timestamp of when they expire).
  """
  if platforms.is_gce():
    # By default, VMs do not have "User info" API enabled, as commented above.
    # When this is the case, the oauth token is unusable. So do not use the
    # oauth token in this case and fall back to IP based whitelisting.
    if ('https://www.googleapis.com/auth/userinfo.email' in
        platforms.gce.oauth2_available_scopes('default')):
      tok, exp = platforms.gce.oauth2_access_token_with_expiration('default')
      return {'Authorization': 'Bearer %s' % tok}, exp
  return (None, None)


### Hooks


def on_bot_shutdown(bot):
  """Hook function called when the bot shuts down, usually rebooting.

  It's a good time to do other kinds of cleanup.

  Arguments:
  - bot: bot.Bot instance. See ../api/bot.py.
  """
  pass


def on_bot_startup(bot):
  """Hook function called when the bot starts, before handshake with the server.

  Here the bot may initialize and examine its environment, pick initial state
  and dimensions to send to the server during the handshake.

  Arguments:
  - bot: bot.Bot instance. See ../api/bot.py.
  """
  pass


def on_handshake(bot):
  """Hook function called when the bot starts, after handshake with the server.

  Here the bot already knows server enforced dimensions (defined in server side
  bots.cfg file).

  This is called right before starting to poll for tasks. It's a good time to
  do some final initialization or cleanup that may depend on server provided
  configuration.

  Arguments:
  - bot: bot.Bot instance. See ../api/bot.py.
  """
  pass


def on_before_poll(bot):
  """Hook function called before polling the server for an action.

  This function is guaranteed to be called before fetching the dimensions and
  state of the bot.

  Arguments:
  - bot: bot.Bot instance. See ../api/bot.py.
  """
  pass


def on_after_poll(bot, cmd):
  """Hook function called immediately after polling the server for an action.

  Arguments:
  - bot: bot.Bot instance. See ../api/bot.py.
  - cmd: The action that the server asked the bot to perform (e.g. "sleep",
         "terminate", "run").
  """
  pass


def on_before_task(bot, bot_file, runner_cmd, runner_env):
  """Hook function called before running a task.

  It shouldn't do much, since it can't cancel the task so it shouldn't do
  anything too fancy.

  Arguments:
  - bot: bot.Bot instance. See ../api/bot.py.
  - bot_file: Path to file to write information about the state of the bot.
              This file can be used to pass certain info about the bot
              to tasks, such as which connected android devices to run on. See
              https://chromium.googlesource.com/infra/luci/luci-py.git/+/master/appengine/swarming/doc/Magic-Values.md#run_isolated
  - runner_cmd: Command to be executed to launch task runner. This variable can
                be mutated to override the task runner, modify its arguments
                and/or add a wrapper script around it. USE WITH CAUTION.
  - runner_env: Environment in which test runner is launched. Can be mutated.
  """
  pass


def on_after_task(bot, failure, internal_failure, task_dimensions, summary):
  """Hook function called after running a task.

  It is an excellent place to do post-task cleanup of temporary files.

  Arguments:
  - bot: bot.Bot instance. See ../api/bot.py.
  - failure: bool, True if the task failed.
  - internal_failure: bool, True if an internal failure happened.
  - task_dimensions: dict, Dimensions requested as part of the task.
  - summary: dict, Summary of the task execution.
  """
  # Example code:
  #if failure:
  #  bot.host_reboot('Task failure')
  #elif internal_failure:
  #  bot.host_reboot('Internal failure')


def on_bot_idle(bot, since_last_action):
  """Hook function called once when the bot has been idle; when it has no
  command to execute.

  This is an excellent place to put device in 'cool down' mode or any
  "pre-warming" kind of stuff that could take several seconds to do, that would
  not be appropriate to do in on_after_task(). It could be worth waiting for
  `since_last_action` to be several seconds before doing a more lengthy
  operation.

  This function is called repeatedly until an action is taken (a task, updating,
  etc).

  This is a good place to do "auto reboot" for hardware based bots that are
  rebooted periodically.

  Arguments:
  - bot: bot.Bot instance. See ../api/bot.py.
  - since_last_action: time in second since last action; e.g. amount of time the
                       bot has been idle.
  """
  # Don't try this if running inside docker.
  #if sys.platform != 'linux2' or not platforms.linux.get_inside_docker():
  #  uptime = os_utilities.get_uptime()
  #  if uptime > 12*60*60 * (1. + bot.get_pseudo_rand(0.2)):
  #    bot.host_reboot('Periodic reboot after %ds' % uptime)


### Setup


def setup_bot(bot):
  """Does one time initialization for this bot.

  Returns True if it's fine to start the bot right away. Otherwise, the calling
  script should exit.

  TODO(maruel): Have the user call bot.host_reboot() or bot.bot_restart()
  instead of returning a value.

  This is an excellent place to drop a README file in the bot directory, to give
  more information about the purpose of this bot.

  Example: making this script starts automatically on user login via
  os_utilities.set_auto_startup_win() or os_utilities.set_auto_startup_osx().
  """
  with open(os.path.join(bot.base_dir, 'README'), 'wb') as f:
    f.write(
"""This directory contains a Swarming bot.

Swarming source code is hosted at
https://chromium.googlesource.com/infra/luci/luci-py.git.

The bot was generated from the server %s. To get the bot's attributes, run:

  python swarming_bot.zip attributes
""" % bot.server)
    return True

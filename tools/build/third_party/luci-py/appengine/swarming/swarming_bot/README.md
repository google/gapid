# Swarming bot code

Contains the code to run on the bots. This code has to be resilient enough to be
able to self-update in case of partial destruction.

The server optionally has its own bot_config.py that will replace the file in
this directory. This script provides domain specific dimensions and should
self-register the bot on startup if desired.

The server replaces config.json with server-specific details.

## Behavior

When the bot starts, the first thing it looks up is if it is the primary copy.
If so, it immediately sets itself as swarming_bot.1.zip. Then it starts up
itself. When running as swarming_bot.1.zip and it needs to upgrade, it switches
to swarming_bot.2.zip. On the next upgrade, it goes back to .1.zip.


## Code layout

This directory contains the main entry point to the bot and symlinks to useful
directories (eg to the "client" directory, which contains the Isolated clients
used by the bot). The three key subdirectories are:

  * `api`: code usable by both the bot config hooks and by the Swarming code.
  * `bot_code`: the actual Swarming bot code. Implements the main bot process
    and the task runner process. This code cannot be accessed by the bot config
    hooks; it is responsible for *calling* those hooks.
  * `config`: a default implementation of the bot config, including a default
    implementation of every hook.

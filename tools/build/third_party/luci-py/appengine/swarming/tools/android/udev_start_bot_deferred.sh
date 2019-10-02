#!/bin/sh
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

# This script is run as the user as specified to ./setup_udev.py. It
# starts/signal the bot about plugged or unplugged devices.

# Unset sudo environment variables.
unset SUDO_COMMAND
unset SUDO_GID
unset SUDO_UID
unset SUDO_USER

base_path=`dirname $0`

# Convert udev environment variable to swarming bot expected variable.
export SWARMING_BOT_ANDROID="$ID_SERIAL_SHORT"

# It's recommended to have a log file. Set it to /dev/null if not desired.
LOG_FILE="$base_path/swarming_bot_udev.log"

# If you need to detail about the environment variable.
echo "" >> "$LOG_FILE"
echo "Running udev event" >> "$LOG_FILE"

# Warning: the user's PATH will not be available.
# TODO(maruel): Figure out a way to ensure adb is in the PATH.
PATH="$PATH:$HOME/src/android-sdk-linux/platform-tools/"

export >> "$LOG_FILE"
echo "" >> "$LOG_FILE"

# Defer execution.
# TODO(maruel): When ACTION=='remove', send the signal to the currently running
# swarming bot to stop itself.
# https://code.google.com/p/swarming/issues/detail?id=127
echo "python $base_path/swarming_bot.zip >> $LOG_FILE 2>&1" | at now

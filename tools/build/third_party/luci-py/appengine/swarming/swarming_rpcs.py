# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""This module defines ProtoRPC types for the Swarming Server handlers."""

from protorpc import message_types
from protorpc import messages


### Enums


class TaskStateQuery(messages.Enum):
  """Use one of the values in this enum to query for tasks in one of the
  specified state.

  Use 'ALL' to not use any filtering based on task state.

  As an example, this enum enables querying for all tasks with state COMPLETED
  but non-zero exit code via COMPLETED_FAILURE.

  Do not confuse TaskStateQuery and TaskState. TaskStateQuery is to query tasks
  via the API. TaskState is the current task state.
  """
  # Query for all tasks currently TaskState.PENDING.
  PENDING = 0
  # Query for all tasks currently TaskState.RUNNING. This includes tasks
  # currently in the overhead phase; mapping input files or archiving outputs
  # back to the server.
  RUNNING = 1
  # Query for all tasks currently TaskState.PENDING or TaskState.RUNNING. This
  # is the query for the 'active' tasks.
  PENDING_RUNNING = 2
  # Query for all tasks that completed normally as TaskState.COMPLETED,
  # independent of the process exit code.
  COMPLETED = 3
  # Query for all tasks that completed normally as TaskState.COMPLETED and that
  # had exit code 0.
  COMPLETED_SUCCESS = 4
  # Query for all tasks that completed normally as TaskState.COMPLETED and that
  # had exit code not 0.
  COMPLETED_FAILURE = 5
  # Query for all tasks that are TaskState.EXPIRED.
  EXPIRED = 6
  # Query for all tasks that are TaskState.TIMED_OUT.
  TIMED_OUT = 7
  # Query for all tasks that are TaskState.BOT_DIED.
  BOT_DIED = 8
  # Query for all tasks that are TaskState.CANCELED.
  CANCELED = 9
  # Query for all tasks, independent of the task state.
  #
  # In hindsight, this constant should have been the value 0. Sorry, the
  # original author was young and foolish.
  ALL = 10
  # Query for all tasks that are TaskState.COMPLETED but that actually didn't
  # run due to TaskProperties.idempotent being True *and* that a previous task
  # with the exact same TaskProperties had successfully run before, aka
  # COMPLETED_SUCCESS.
  DEDUPED = 11
  # Query for all tasks that are TaskState.KILLED.
  KILLED = 12
  # Query for all tasks that are TaskState.NO_RESOURCE.
  NO_RESOURCE = 13


class TaskState(messages.Enum):
  """Represents the current task state.

  Some states are still mutable: PENDING and RUNNING. The others are final and
  will not change afterward.

  A task is guaranteed to be in exactly one state at any point of time.

  Do not confuse TaskStateQuery and TaskState. TaskStateQuery is to query tasks
  via the API. TaskState is the current task state.

  As you read the following constants, astute readers may wonder why these
  constants look like a bitmask. This is because of historical reasons and this
  is effectively an enum, not a bitmask.
  """
  # Invalid state, do not use.
  INVALID = 0x00
  # The task is currently running. This is in fact 3 phases: the initial
  # overhead to fetch input files, the actual task running, and the tear down
  # overhead to archive output files to the server.
  RUNNING = 0x10
  # The task is currently pending. This means that no bot reaped the task. It
  # will stay in this state until either a task reaps it or the expiration
  # elapsed. The task pending expiration is specified as
  # TaskSlice.expiration_secs, one per task slice.
  PENDING = 0x20
  # The task is not pending anymore, and never ran due to lack of capacity. This
  # means that other higher priority tasks ran instead and that not enough bots
  # were available to run this task for TaskSlice.expiration_secs seconds.
  EXPIRED = 0x30
  # The task ran for longer than the allowed time in
  # TaskProperties.execution_timeout_secs or TaskProperties.io_timeout_secs.
  # This means the bot forcefully killed the task process as described in the
  # graceful termination dance in the documentation.
  TIMED_OUT = 0x40
  # The task ran but the bot had an internal failure, unrelated to the task
  # itself. It can be due to the server being unavailable to get task update,
  # the host on which the bot is running crashing or rebooting, etc.
  BOT_DIED = 0x50
  # The task never ran, and was manually cancelled via the 'cancel' API before
  # it was reaped.
  CANCELED = 0x60
  # The task ran and completed normally. The task process exit code may be 0 or
  # another value.
  COMPLETED = 0x70
  # The task ran but was manually killed via the 'cancel' API. This means the
  # bot forcefully killed the task process as described in the graceful
  # termination dance in the documentation.
  KILLED = 0x80
  # The task was never set to PENDING and was immediately refused, as the server
  # determined that there is no bot capacity to run this task. This happens
  # because no bot exposes a superset of the requested task dimensions.
  #
  # Set TaskSlice.wait_for_capacity to True to force the server to keep the task
  # slice pending even in this case. Generally speaking, the task will
  # eventually switch to EXPIRED, as there's no bot to run it. That said, there
  # are situations where it is known that in some not-too-distant future a wild
  # bot will appear that will be able to run this task.
  NO_RESOURCE = 0x100


class TaskSort(messages.Enum):
  """Flag to sort returned tasks. The natural sort is CREATED_TS."""
  CREATED_TS, MODIFIED_TS, COMPLETED_TS, ABANDONED_TS = range(4)


class PoolTaskTemplateField(messages.Enum):
  """Flag to control the application of the pool's TaskTemplate in a new
  TaskRequest.

  The non-endpoints counterpart is in task_request.
  """
  AUTO = 0
  CANARY_PREFER = 1
  CANARY_NEVER = 2
  SKIP = 3


### Pretend Associative Array


class StringPair(messages.Message):
  """Represents a mapping of string to string."""
  key = messages.StringField(1)
  value = messages.StringField(2)


class StringListPair(messages.Message):
  """Represents a mapping of string to list of strings."""
  key = messages.StringField(1)
  value = messages.StringField(2, repeated=True)


class ThreeStateBool(messages.Enum):
  FALSE = 1
  TRUE = 2
  NONE = 3


def to_bool(three_state):
  if three_state in (None, True, False):
    return three_state
  if three_state == ThreeStateBool.FALSE:
    return False
  if three_state == ThreeStateBool.TRUE:
    return True


### Server related.


class ServerDetails(messages.Message):
  """Reports details about the server."""
  server_version = messages.StringField(1)
  bot_version = messages.StringField(2)
  machine_provider_template = messages.StringField(3)
  display_server_url_template = messages.StringField(4)
  luci_config = messages.StringField(5)
  default_isolate_server = messages.StringField(6)
  default_isolate_namespace = messages.StringField(7)


class BootstrapToken(messages.Message):
  """Returns a token to bootstrap a new bot."""
  bootstrap_token = messages.StringField(1)


class ClientPermissions(messages.Message):
  """Reports the client's permissions."""
  delete_bot = messages.BooleanField(1)
  terminate_bot = messages.BooleanField(2)
  get_configs = messages.BooleanField(3)
  put_configs = messages.BooleanField(4)
  # Cancel one single task
  cancel_task = messages.BooleanField(5)
  get_bootstrap_token = messages.BooleanField(6)
  # Cancel multiple tasks at once, usually in emergencies.
  cancel_tasks = messages.BooleanField(7)


class FileContentRequest(messages.Message):
  """Content of a file."""
  content = messages.StringField(1)


class FileContent(messages.Message):
  """Content of a file."""
  content = messages.StringField(1)
  version = messages.StringField(2)
  who = messages.StringField(3)
  when = message_types.DateTimeField(4)


### Task-Related Requests


class FilesRef(messages.Message):
  """Defines a data tree reference for Swarming task inputs or outputs.

  It can either be:
    - a reference to an isolated file on an isolate server
    - a reference to an isolated file on a RBE CAS server

  In the RBE CAS case, the isolatedserver must be set to GCP name, and namespace
  must be set to "sha256-GCP". For the moment, RBE CAS requires SHA-256 and
  doesn't support precompressed data.
  """
  # The hash of an isolated archive.
  isolated = messages.StringField(1)
  # The hostname of the isolated server to use or the Google Cloud Project name.
  isolatedserver = messages.StringField(2)
  # Namespace on the isolate server or "sha256-GCP" for a GCP hosted RBE CAS.
  namespace = messages.StringField(3)


class CipdPackage(messages.Message):
  """A CIPD package to install in the run dir before task execution."""
  # A template of a full CIPD package name, e.g.
  # "infra/tools/authutil/${platform}"
  # See also cipd.ALL_PARAMS.
  package_name = messages.StringField(1)
  # Valid package version for all packages matched by package name.
  version = messages.StringField(2)
  # Path to dir, relative to the root dir, where to install the package.
  # If empty, the package will be installed a the root of the mapped directory.
  # If file names in the package and in the isolate clash, it will cause a
  # failure.
  path = messages.StringField(3)


class CipdInput(messages.Message):
  """Defines CIPD packages to install in task run directory."""
  # URL of the CIPD server. Must start with "https://" or "http://".
  # This field or its subfields are optional if default cipd client is defined
  # in the server config.
  server = messages.StringField(1)

  # CIPD package of CIPD client to use.
  # client_package.version is required.
  # This field is optional is default value is defined in the server config.
  # client_package.path must be empty.
  client_package = messages.MessageField(CipdPackage, 2)

  # List of CIPD packages to install.
  packages = messages.MessageField(CipdPackage, 3, repeated=True)


class CipdPins(messages.Message):
  """Defines pinned CIPD packages that were installed during the task."""

  # The pinned package + version of the CIPD client that was actually used.
  client_package = messages.MessageField(CipdPackage, 1)

  # List of CIPD packages that were installed in the task with fully resolved
  # package names and versions.
  packages = messages.MessageField(CipdPackage, 2, repeated=True)


class CacheEntry(messages.Message):
  """Describes a named cache that should be present on the bot.

  A CacheEntry in a task specified that the task prefers the cache to be present
  on the bot. A symlink to the cache directory is created at <run_dir>/|path|.
  If cache is not present on the machine, the directory is empty.
  If the tasks makes any changes to the contents of the cache directory, they
  are persisted on the machine. If another task runs on the same machine and
  requests the same named cache, even if mapped to a different path, it will see
  the changes.
  """

  # Unique name of the cache. Required. Length is limited to 4096.
  name = messages.StringField(1)
  # Relative path to the directory that will be linked to the named cache.
  # Required.
  # A path cannot be shared among multiple caches or CIPD installations.
  # A task will fail if a file/dir with the same name already exists.
  path = messages.StringField(2)


class TaskProperties(messages.Message):
  """Important metadata about a particular task."""
  # Specifies named caches to map into the working directory. These caches
  # outlives the task, which can then be reused by tasks later used on this bot
  # that request the same named cache.
  caches = messages.MessageField(CacheEntry, 11, repeated=True)
  # CIPD packages to install. These packages are meant to be software that is
  # needed (a dependency) to the task being run. Unlike isolated files, the CIPD
  # packages do not expire from the server.
  cipd_input = messages.MessageField(CipdInput, 10)
  # Command to run. This has priority over a command specified in the isolated
  # files. Only one of 'command' or 'extra_args' can be specified.
  command = messages.StringField(1, repeated=True)
  # Relative working directory to start the 'command' in, defaults to the root
  # mapped directory or what is provided in the isolated file, if any.
  relative_cwd = messages.StringField(15)
  # Dimensions are what is used to determine which bot can run the task. The
  # bot must have all the matching dimensions, even for repeated keys with
  # multiple different values. It is a logical AND, all values must match.
  #
  # It should have been a StringListPair but this would be a breaking change.
  dimensions = messages.MessageField(StringPair, 2, repeated=True)
  # Environment variables to set when running the task.
  env = messages.MessageField(StringPair, 3, repeated=True)
  # Swarming-root relative paths to prepend to a given environment variable.
  #
  # These allow you to put certain subdirectories of the task into PATH,
  # PYTHONPATH, or other PATH-like environment variables. The order of
  # operations is:
  #   * Turn slashes into native-platform slashes.
  #   * Make the path absolute
  #   * Prepend it to the current value of the envvar using the os-native list
  #     separator (i.e. `;` on windows, `:` on POSIX).
  #
  # Each envvar can have multiple paths to prepend. They will be prepended in
  # the order seen here.
  #
  # For example, if env_prefixes was:
  #   [("PATH", ["foo", "bar"]),
  #    ("CUSTOMPATH", ["custom"])]
  #
  # The task would see:
  #   PATH=/path/to/swarming/rundir/foo:/path/to/swarming/rundir/bar:$PATH
  #   CUSTOMPATH=/path/to/swarming/rundir/custom
  #
  # The path should always be specified here with forward-slashes, and it must
  # not attempt to escape the swarming root (i.e. must not contain `..`).
  #
  # These are applied AFTER evaluating `env` entries.
  env_prefixes = messages.MessageField(StringListPair, 14, repeated=True)
  # Maximum number of seconds the task can run before its process is forcibly
  # terminated and the task results in TIMED_OUT.
  execution_timeout_secs = messages.IntegerField(4)
  # Extraneous arguments to append to the command specified in the isolated
  # file. Can only be used when an isolated file specifies a command. Only one
  # of 'command' or 'extra_args' can be specified.
  extra_args = messages.StringField(5, repeated=True)
  # Number of second to give the child process after a SIGTERM before sending a
  # SIGKILL. See doc/Bot.md#timeout-handling
  grace_period_secs = messages.IntegerField(6)
  # True if the task does not access any service through the network and is
  # believed to be 100% reproducible with the same outcome. In the case of a
  # successful task, previous results will be reused if possible.
  idempotent = messages.BooleanField(7)
  # Isolated inputs to map in the working directory. The isolated file may
  # optionally specify a command to run. Otherwise, 'command' must be specified.
  inputs_ref = messages.MessageField(FilesRef, 8)
  # Maximum number of seconds the task may be silent (no output to stdout nor
  # stderr) before it is considered hung and it forcibly terminated early and
  # the task results in TIMED_OUT.
  io_timeout_secs = messages.IntegerField(9)
  # Paths in the working directory to archive back.
  outputs = messages.StringField(12, repeated=True)
  # Secret bytes to provide to the task. Cannot be retrieved back.
  secret_bytes = messages.BytesField(13)


class TaskSlice(messages.Message):
  """Defines a possible task execution for a task request to be run on the
  Swarming infrastructure.

  This is one of the possible fallback on a task request.
  """
  # The property of the task to try to run.
  #
  # If there is no bot that can serve this properties.dimensions when this task
  # slice is enqueued, it is immediately denied. This can trigger if:
  # - There is no bot with these dimensions currently known.
  # - Bots that could run this task are either all dead or quarantined.
  # Swarming considers a bot dead if it hasn't pinged in the last N minutes
  # (currently 10 minutes).
  properties = messages.MessageField(TaskProperties, 1)
  # If this task request slice is not scheduled after waiting this long, the
  # next one will be processed.
  expiration_secs = messages.IntegerField(2)
  # When a task is scheduled and there are currently no bots available to run
  # the task, the TaskSlice can either be PENDING, or be denied immediately.
  # When denied, the next TaskSlice is enqueued, and if there's no following
  # TaskSlice, the task state is set to NO_RESOURCE. This should normally be
  # set to False to avoid unnecessary waiting.
  wait_for_capacity = messages.BooleanField(3)


class NewTaskRequest(messages.Message):
  """Description of a new task request as described by the client.

  This message is used to create a new task.
  """
  # Maximum of seconds the task may stay PENDING. Must be specified with
  # properties. Cannot be used at the same time as task_slices.
  expiration_secs = messages.IntegerField(1)
  # Task name for display purpose.
  name = messages.StringField(2)
  # Parent Swarming task ID of the process requesting this task. This is to tell
  # the server about reentrancy: when a task creates children Swarming tasks, so
  # that the tree of tasks can be presented in the UI; the parent task will list
  # all the children tasks that were triggered.
  parent_task_id = messages.StringField(3)
  # Task priority, the lower the more important.
  priority = messages.IntegerField(4)
  # Task properties, which defines what to run.
  properties = messages.MessageField(TaskProperties, 5)
  # Slice of TaskSlice, along their scheduling parameters. Cannot be used at the
  # same time as properties and expiration_secs.
  #
  # This defines all the various possible task execution for a task request to
  # be run on the Swarming infrastructure. They are processed in order, and it
  # is guaranteed that at most one of these will be processed.
  task_slices = messages.MessageField(TaskSlice, 12, repeated=True)
  # Tags are 'key:value' strings that describes what the task is about. This can
  # later be leveraged to search for kinds of tasks per tag.
  tags = messages.StringField(6, repeated=True)
  # User on which behalf this task is run, if relevant. Not validated.
  user = messages.StringField(7)

  # Defines what OAuth2 credentials the task uses when calling other services.
  #
  # Possible values are:
  #   - 'none': do not use task service accounts at all, this is default.
  #   - 'bot': use bot's own account, works only if bots authenticate with
  #       OAuth2.
  #   - 'email': use this account (if token server's service_accounts.cfg rules
  #       allow it). Not implemented yet.
  #
  # Note that the service account name is specified outside of task properties,
  # and thus it is possible to have two tasks with different service accounts,
  # but identical properties hash (so one can be deduped). If this is unsuitable
  # use 'idempotent=False' or include a service account name in properties
  # separately.
  #
  # TODO(vadimsh): Link to a doc that describes Swarming Service Accounts, when
  # it exists.
  service_account = messages.StringField(8)

  # Full topic name to post task state updates to, e.g.
  # "projects/<id>/topics/<id>".
  pubsub_topic = messages.StringField(9)
  # Secret string to put into "auth_token" attribute of PubSub message.
  pubsub_auth_token = messages.StringField(10)
  # Will be but into "userdata" fields of PubSub message.
  pubsub_userdata = messages.StringField(11)

  # Only evaluate the task, as if we were going to schedule it, but don't
  # actually schedule it. This will return the TaskRequest, but without
  # a task_id.
  evaluate_only = messages.BooleanField(13)

  # Controls the application of the pool's TaskTemplate to the creation of this
  # task. By default this will automatically select the pool's preference for
  # template, but you can also instruct swarming to prefer/prevent the
  # application of canary templates, as well as skipping the template
  # altogether.
  pool_task_template = messages.EnumField(
      PoolTaskTemplateField, 14, default='AUTO')


class TaskRequest(messages.Message):
  """Description of a task request as registered by the server.

  This message is used when retrieving information about an existing task.

  See NewtaskRequest for more details.
  """
  expiration_secs = messages.IntegerField(1)
  name = messages.StringField(2)
  parent_task_id = messages.StringField(3)
  priority = messages.IntegerField(4)
  # For some amount of time, the properties will be copied into the
  # task_slices and vice-versa, to give time to the clients to update.
  # Eventually, only task_slices will be supported.
  properties = messages.MessageField(TaskProperties, 5)
  tags = messages.StringField(6, repeated=True)
  created_ts = message_types.DateTimeField(7)
  user = messages.StringField(8)
  # User name of whoever posted this task, extracted from the credentials.
  authenticated = messages.StringField(9)
  task_slices = messages.MessageField(TaskSlice, 13, repeated=True)
  # Indicates what OAuth2 credentials the task uses when calling other services.
  service_account = messages.StringField(10)

  pubsub_topic = messages.StringField(11)
  pubsub_userdata = messages.StringField(12)


class TaskCancelRequest(messages.Message):
  """Request to cancel one task."""
  kill_running = messages.BooleanField(1)


class TasksCancelRequest(messages.Message):
  """Request to cancel some subset of pending/running tasks."""
  tags = messages.StringField(1, repeated=True)
  cursor = messages.StringField(2)
  limit = messages.IntegerField(3, default=100)
  kill_running = messages.BooleanField(4)


### Task-Related Responses


class OperationStats(messages.Message):
  duration = messages.FloatField(1)
  initial_number_items = messages.IntegerField(2)
  initial_size = messages.IntegerField(3)
  # These buffers are compressed as deflate'd delta-encoded varints. They are
  # all the items for an isolated operation, which can scale in the 100k range.
  # So can be large! See //client/utils/large.py for the code to handle these.
  items_cold = messages.BytesField(4)
  items_hot = messages.BytesField(5)
  # Corresponding summaries; for each list above, sum of the number of files
  # and the sum bytes of the files.
  num_items_cold = messages.IntegerField(6)
  total_bytes_items_cold = messages.IntegerField(7)
  num_items_hot = messages.IntegerField(8)
  total_bytes_items_hot = messages.IntegerField(9)


class PerformanceStats(messages.Message):
  bot_overhead = messages.FloatField(1)
  isolated_download = messages.MessageField(OperationStats, 2)
  isolated_upload = messages.MessageField(OperationStats, 3)


class CancelResponse(messages.Message):
  """Result of a request to cancel a task."""
  ok = messages.BooleanField(1)
  was_running = messages.BooleanField(2)


class TasksCancelResponse(messages.Message):
  """Result of canceling some subset of pending tasks.
  """
  cursor = messages.StringField(1)
  now = message_types.DateTimeField(2)
  matched = messages.IntegerField(3)


class TaskOutput(messages.Message):
  """A task's output as a string."""
  output = messages.StringField(1)


class TaskResult(messages.Message):
  """Representation of the TaskResultSummary or TaskRunResult ndb model."""
  # Time when the task was abandoned instead of normal completion (e.g.
  # EXPIRED, BOT_DIED, KILLED).
  #
  # In the case of KILLED, this records the time the user requested the task to
  # stop.
  abandoned_ts = message_types.DateTimeField(1)
  # The same key cannot be repeated.
  bot_dimensions = messages.MessageField(StringListPair, 2, repeated=True)
  # Unique ID of the bot.
  bot_id = messages.StringField(3)
  # Hash of the bot code which ran the task.
  bot_version = messages.StringField(4)
  # List of task IDs that this task triggered, if any.
  children_task_ids = messages.StringField(5, repeated=True)
  # Time the task completed normally. Only one of abandoned_ts or completed_ts
  # can be set except for state == KILLED.
  #
  # In case of KILLED, completed_ts is the time the task completed.
  completed_ts = message_types.DateTimeField(6)
  # $ saved for task with state DEDUPED.
  cost_saved_usd = messages.FloatField(7)
  # Time the task was requested.
  created_ts = message_types.DateTimeField(8)
  # Task ID which results was reused for state DEDUPED.
  deduped_from = messages.StringField(9)
  # Duration of the task in seconds. This excludes overheads.
  duration = messages.FloatField(10)
  # Process exit code if relevant. May be forcibly set to -1 in exceptional
  # cases.
  exit_code = messages.IntegerField(11)
  # True if exit_code != 0.
  failure = messages.BooleanField(12)
  # True if state is BOT_DIED.
  internal_failure = messages.BooleanField(13)
  # Time the results was last updated in the DB.
  modified_ts = message_types.DateTimeField(14)
  # Isolated outputs, if any.
  outputs_ref = messages.MessageField(FilesRef, 15)
  # Server versions that touched this task.
  server_versions = messages.StringField(17, repeated=True)
  # Time the task started being run by a bot.
  started_ts = message_types.DateTimeField(18)
  # Current state of the task (e.g. PENDING, RUNNING, COMPLETED, EXPIRED, etc).
  state = messages.EnumField(TaskState, 19)
  # Summary task ID (ending with '0') when creating a new task.
  task_id = messages.StringField(20)
  # Can be 0, 1 or 2. It is 0 for a deduped task, since nothing ran. It is
  # normally 1. It is 2 if the first try had an internal failure.
  try_number = messages.IntegerField(21)

  # Can be multiple values only in TaskResultSummary.
  costs_usd = messages.FloatField(22, repeated=True)
  # Name of the task. Only set when requesting task ID summary, ending with '0'.
  name = messages.StringField(23)
  # Tags associated with the task when it was requested. Only set when
  # requesting task ID summary, ending with '0'.
  tags = messages.StringField(24, repeated=True)
  # User on behalf this task was requested. Only set when requesting task ID
  # summary, ending with '0'.
  user = messages.StringField(25)
  # Statistics about overhead for an isolated task. Only sent when requested.
  performance_stats = messages.MessageField(PerformanceStats, 26)

  # Listing of the ACTUAL pinned CipdPackages that the task used. These can vary
  # from the input packages if the inputs included non-identity versions (e.g. a
  # ref like "latest").
  cipd_pins = messages.MessageField(CipdPins, 27)
  # Actual executed task id that this task represents. For deduped tasks, it is
  # the same value as deduped_from. This value can be empty if there is no
  # execution, for example the task was cancelled.
  run_id = messages.StringField(28)

  # Index in the TaskRequest.task_slices (TaskSlice instance) that this result
  # represents. This is updated when a TaskSlice is enqueued to run.
  #
  # The TaskSlice contains a TaskProperties, which defines what is run.
  current_task_slice = messages.IntegerField(29)


class TaskStates(messages.Message):
  """Only holds states. Used in the 'get_states' RPC."""
  states = messages.EnumField(TaskState, 1, repeated=True)


class TaskList(messages.Message):
  """Wraps a list of TaskResult."""
  # TODO(maruel): Rename to TaskResults.
  cursor = messages.StringField(1)
  items = messages.MessageField(TaskResult, 2, repeated=True)
  now = message_types.DateTimeField(3)


class TaskRequests(messages.Message):
  """Wraps a list of TaskRequest."""
  cursor = messages.StringField(1)
  items = messages.MessageField(TaskRequest, 2, repeated=True)
  now = message_types.DateTimeField(3)


class TasksCount(messages.Message):
  """Returns the count, as requested."""
  count = messages.IntegerField(1)
  now = message_types.DateTimeField(2)


class TasksTags(messages.Message):
  """Returns all the tags and tag possibilities in the fleet."""
  tasks_tags = messages.MessageField(StringListPair, 1, repeated=True)
  # Time at which this summary was calculated.
  ts = message_types.DateTimeField(2)


class TaskRequestMetadata(messages.Message):
  """Provides the ID of the requested TaskRequest."""
  request = messages.MessageField(TaskRequest, 1)
  task_id = messages.StringField(2)
  # Set to finished task result in case task was deduplicated.
  task_result = messages.MessageField(TaskResult, 3)


### Task queues


class TaskQueue(messages.Message):
  # Must be a list of 'key:value' strings to filter the returned list of bots
  # on.
  dimensions = messages.StringField(1, repeated=True)
  valid_until_ts = message_types.DateTimeField(2)


class TaskQueueList(messages.Message):
  cursor = messages.StringField(1)
  # Note that it's possible that the RPC returns a tad more or less items than
  # requested limit.
  items = messages.MessageField(TaskQueue, 2, repeated=True)
  now = message_types.DateTimeField(3)


### Bots


### Bot-Related Responses


class BotInfo(messages.Message):
  """Representation of the BotInfo ndb model."""
  bot_id = messages.StringField(1)
  dimensions = messages.MessageField(StringListPair, 2, repeated=True)
  external_ip = messages.StringField(3)
  authenticated_as = messages.StringField(4)
  first_seen_ts = message_types.DateTimeField(5)
  is_dead = messages.BooleanField(6)
  last_seen_ts = message_types.DateTimeField(7)
  quarantined = messages.BooleanField(8)
  maintenance_msg = messages.StringField(18)
  task_id = messages.StringField(9)
  task_name = messages.StringField(10)
  version = messages.StringField(11)
  # Encoded as json since it's an arbitrary dict.
  state = messages.StringField(12)
  lease_id = messages.StringField(13)
  lease_expiration_ts = message_types.DateTimeField(14)
  deleted = messages.BooleanField(15)
  machine_type = messages.StringField(16)
  machine_lease = messages.StringField(17)
  leased_indefinitely = messages.BooleanField(19)


class BotList(messages.Message):
  """Wraps a list of BotInfo."""
  cursor = messages.StringField(1)
  items = messages.MessageField(BotInfo, 2, repeated=True)
  now = message_types.DateTimeField(3)
  death_timeout = messages.IntegerField(4)


class BotsCount(messages.Message):
  """Returns the count, as requested."""
  now = message_types.DateTimeField(1)
  count = messages.IntegerField(2)
  quarantined = messages.IntegerField(3)
  maintenance = messages.IntegerField(6)
  dead = messages.IntegerField(4)
  busy = messages.IntegerField(5)


class BotsDimensions(messages.Message):
  """Returns all the dimensions and dimension possibilities in the fleet."""
  bots_dimensions = messages.MessageField(StringListPair, 1, repeated=True)
  # Time at which this summary was calculated.
  ts = message_types.DateTimeField(2)


class BotEvent(messages.Message):
  # Timestamp of this event.
  ts = message_types.DateTimeField(1)
  # Type of event.
  event_type = messages.StringField(2)
  # Message included in the event.
  message = messages.StringField(3)
  # Bot dimensions at that moment.
  dimensions = messages.MessageField(StringListPair, 4, repeated=True)
  # Bot state at that moment, encoded as json.
  state = messages.StringField(5)
  # IP address as seen by the HTTP handler.
  external_ip = messages.StringField(6)
  # Bot identity as seen by the HTTP handler.
  authenticated_as = messages.StringField(7)
  # Version of swarming_bot.zip the bot is currently running.
  version = messages.StringField(8)
  # If True, the bot is not accepting task due to being quarantined.
  quarantined = messages.BooleanField(9)
  # If set, the bot is rejecting tasks due to maintenance.
  maintenance_msg = messages.StringField(11)
  # Affected by event_type == 'request_task', 'task_completed', 'task_error'.
  task_id = messages.StringField(10)


class BotEvents(messages.Message):
  cursor = messages.StringField(1)
  items = messages.MessageField(BotEvent, 2, repeated=True)
  now = message_types.DateTimeField(3)


class BotTasks(messages.Message):
  cursor = messages.StringField(1)
  items = messages.MessageField(TaskResult, 2, repeated=True)
  now = message_types.DateTimeField(3)


class DeletedResponse(messages.Message):
  """Indicates whether a bot was deleted."""
  deleted = messages.BooleanField(1)


class TerminateResponse(messages.Message):
  """Returns the pseudo taskid to wait for the bot to shut down."""
  task_id = messages.StringField(1)

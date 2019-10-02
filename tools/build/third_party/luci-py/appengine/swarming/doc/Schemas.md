# Schemas

This page documents schemas used for tasks and bots in the DB.


## Tasks

The task description core has to be read in order like a story in 4 parts; each
block depends on the previous ones:

  - task_request.py
  - task_to_run.py
  - task_result.py
  - task_scheduler.py

The scheduling optimisation is done via:

  - task_queues.py


### General workflow

  - A client wants to run something (a task) on the infrastructure and sends a
    HTTP POST to the Swarming server:
    - A TaskRequest describing this request is saved to note that a new request
      exists. The details of the task is saved in TaskProperties embedded in
      TaskRequest. If the task contains any SecretBytes, a child SecretBytes
      entity with id=1 will also be saved.
    - A TaskToRun is created to dispatch this request so it can be run on a
      bot. It is marked as ready to be triggered when created.
    - A TaskResultSummary is created to describe the request's overall status,
      taking in account retries.
    - A TaskDimensions is stored to describe the precise set of dimensions.
  - Bots poll for work. It looks at the queues described by BotTaskDimensions.
    Once a bot reaps a TaskToRun, the server creates the corresponding
    TaskRunResult for this run and updates it as required until completed. The
    TaskRunResult describes the result for this run on this specific bot.
  - When the bot is done, a PerformanceStats entity is saved as a child entity
    of the TaskRunResult.
  - If the TaskRequest is retried automatically due to the bot dying, an
    automatic on task failure or another infrastructure related failure, another
    TaskRunResult will be created when another bot reaps the task again.
    TaskResultSummary is the summary of the last relevant TaskRunResult.


### Task schema

This schema is an example of a task with two tries. This happens when the first
try resulted in `TaskRunResult.state == BOT_DIED`. This is a relatively rare
case.

**Note**: Entities marked with an asterisk `*` may not be stored in certain
situations, like for deduplicated tasks, tasks that didn't run due to internal
failure, or tasks with no secret bytes provided (for SecretBytes).

    +--------Root------------------------------------------------------+
    |TaskRequest                                                       |
    |  +--------------+      +----------------+     +----------------+ |
    |  |TaskProperties|      |TaskSlice       |     |TaskSlice       | |
    |  |  +--------+  |      |+--------------+| ... |+--------------+| |
    |  |  |FilesRef|  | *or* ||TaskProperties|| ... ||TaskProperties|| |
    |  |  +--------+  |      |+--------------+|     |+--------------+| |
    |  +--------------+      +----------------+     +----------------+ |
    |id=<based on epoch>                                               |
    +------------------------------------------------------------------+
        |                                                        task_request.py
        |
        +------+
        |      |
        |      v
        |  +-----------+
        |  |SecretBytes|*                                        task_request.py
        |  |id=1       |
        |  +-----------+
        |
        +------+
        |      |
        |      v
        |  +--------------+      +--------------+
        |  |TaskToRun     |* ... |TaskToRun     |*                task_to_run.py
        |  |id=<composite>|  ... |id=<composite>|
        |  +--------------+      +--------------+
        |
        v
    +-----------------+
    |TaskResultSummary|                                           task_result.py
    |  +--------+     |
    |  |FilesRef|     |
    |  +--------+     |
    |id=1             |
    +-----------------+
        |
        +----------------+
        |                |
        v                v
    +-------------+   +-------------+
    |TaskRunResult|*  |TaskRunResult|*                            task_result.py
    |  +--------+ |   |  +--------+ |
    |  |FilesRef| |   |  |FilesRef| |
    |  +--------+ |   |  +--------+ |
    |id=1 <try #> |   |id=2         |
    +-------------+   +-------------+
        |
        +----------------------+
        |                      |
        v                      v
    +-----------------+     +----------------+
    |TaskOutput       |*    |PerformanceStats|*                   task_result.py
    |id=1 (not stored)|     |id=1            |
    +-----------------+     +----------------+
        |
        +------------ ... ----+
        |                     |
        v                     v
    +---------------+      +---------------+
    |TaskOutputChunk|* ... |TaskOutputChunk|*                     task_result.py
    |id=1           |  ... |id=N           |
    +---------------+      +---------------+


### Task queues schema

This schema is to enable fast task scheduling.

    +-------Root------------+
    |TaskDimensionsRoot     |  (not stored)                       task_queues.py
    |id=<pool:foo or id:foo>|
    +-----------------------+
        |
        +---------------- ... -------+
        |                            |
        v                            v
    +----------------------+     +----------------------+
    |TaskDimensions        | ... |TaskDimensions        |         task_queues.py
    |  +-----------------+ | ... |  +-----------------+ |
    |  |TaskDimensionsSet| |     |  |TaskDimensionsSet| |
    |  +-----------------+ |     |  +-----------------+ |
    |id=<dimension_hash>   |     |id=<dimension_hash>   |
    +----------------------+     +----------------------+


## Bots

The bot activity generate entities to keep a trace of all the events happening
on the bot. A cache is kept in `BotInfo` to be able to provide APIs to query for
all active bots.


### Bot schema

This schema is about the audit of the events of bots.

    +-----------+
    |BotRoot    |                                              bot_management.py
    |id=<bot_id>|
    +-----------+
        |
        +------+--------------+
        |      |              |
        |      v              v
        |  +-----------+  +-------+
        |  |BotSettings|  |BotInfo|                            bot_management.py
        |  |id=settings|  |id=info|
        |  +-----------+  +-------+
        |
        +------+-----------+----- ... ----+
        |      |           |              |
        |      v           v              v
        |  +--------+  +--------+     +--------+
        |  |BotEvent|  |BotEvent| ... |BotEvent|               bot_management.py
        |  |id=fffff|  |if=ffffe| ... |id=00000|
        |  +--------+  +--------+     +--------+
        |
        +------+
        |      |
        |      v
        |  +-------------+
        |  |BotDimensions|                                        task_queues.py
        |  |id=1         |
        |  +-------------+
        |
        +---------------- ... ----+
        |                         |
        v                         v
    +-------------------+     +-------------------+
    |BotTaskDimensions  | ... |BotTaskDimensions  |               task_queues.py
    |id=<dimension_hash>| ... |id=<dimension_hash>|
    +-------------------+     +-------------------+


    +--------Root--------+
    |DimensionAggregation|                                     bot_management.py
    |id=current          |
    +--------------------+


# Keys

AppEngine's automatic key numbering is never used. The entities are directly
created with predefined keys so entity sharding can be tightly controlled to
reduce DB contention.

  - TaskRequest has almost monotonically decreasing key ids based on time NOT'ed
    to make it decreasing. See task_request.py for more detail.
  - TaskResultSummary has key ID = 1.
  - TaskRunResult has monotonically increasing key ID starting at 1.
  - TaskToRun has the key ID as `dimensions_hash` value is calculated as an
    int32 from the TaskRequest.properties.dimensions dictionary.
  - PerformanceStats has key ID = 1.
  - BotTaskDimensions and TaskDimensions have key ID `dimensions_hash`. This
    value is calculated as an int32 from the TaskRequest.properties.dimensions
    dictionary.


## Notes

  - Each root entity is tagged as Root.
  - Each line is annotated with the file that define the entities on this line.
  - Dotted line means a similar key relationship without actual entity
    hierarchy.

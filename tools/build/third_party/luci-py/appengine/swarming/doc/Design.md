# Design

Lightweight distribution server with a legion of dumb bots.

**One line description**: Running all tests concurrently, not be impacted by network or device flakiness.


## Introduction

Most task distribution systems involve a DB (MySQL, PostgreSQL, etc) and a task
distribution server where the bots connect to. This means managing these. It's
not too bad, until a drive fills up, RAM needs to be added, more CPU is needed,
connectivity to a second lab is needed, then a hard drive fails...

The authors of this project decided to try to create a completely **stateless
service**, with no single server, where everything is written to the DB. That
was not a given that it would work at all, making all states go through the DB
require a very scalable DB, very scalable frontends and well thought out schemas
so that the DB can sustain the load. We had a lot of funny looks. Most people
thought it'd fail. It worked out and it is used by the Chromium project at
scale.

We also wanted to go "outside the lab" so that everything would be **encrypted
over HTTPS and accessible from the internet** by default. Network management,
like NAT'ing switches, vlan, etc can take a significant amount of time to do and
to do proper bookkeeping. By having the service accessible from the internet with
one way connections always initiated from the bot, it saves a lot of network
topology configuration.

That's not enough, bot management is also a huge pain. So Swarming **bots are
continuously self-updating** from the server. Since the server is stateless,
multiple versions can run simultaneously on the same DB and switching the
default AppEngine version will instantly tell the idle bots to self update
immediately.

Choosing the right bot for the right task is tedious, so Swarming use an evolved
technique using **list based properties** to do task request -> bot matching
using a **priority queue of time based queues**. These can be either FIFO or
LIFO depending on configuration.

All 4 combined result in an incredible reduction of maintenance, where there's
no server to maintain, not much network to setup beside having HTTPS access to
the internet and bots manage their version by themselves.


## Goals

  - Transparently **handle flakiness** at all layers; unreliable network,
    unreliable hardware, flaky DB, etc.
  - **Elastic fleet**; seamlessly use bots are they come and disappear.
  - **Low maintenance**; no server at all, no DB server, everything is embedded
    in the AppEngine server. Bots self-configure themselves.
  - **Secure**; SSL certificate is verified to detect MITM attacks. All
    communications are encrypted. Strong ACLs with authentication via OAuth2.
  - **Multi-project aware**; seamlessly distribute tasks across an heterogeneous
    fleet with priority queue of FIFO or LIFO queues.
  - **Task deduplication**; do not run the same task twice, returns the results
    from previous requests if possible.


## Non Goals

  - Very low latency (<5s) tasks distribution.
  - File distribution. Use the [[Isolated Design|Isolate Server]] for that.
  - Leasing a bot. There's no connectivity between the clients and the bot.


## Use case

Here's an hypothetical build as people normally do:
```
+----------------------------+
|Sync and compile (5 minutes)|
+-----------+----------------+
            |
            v
    +-------------------+
    |Test Foo (1 minute)|
    +-------+-----------+
            |
            v
   +--------------------+
   |Test Bar (5 minutes)|
   +--------+-----------+
            |
            v
 +-------------------------+
 |Test Enlarge (30 minutes)|
 +-------------------------+
```

That's a 5+1+5+30 = **41 minutes build**. You may find this acceptable, we
don't.


Here's how it looks like once running via Swarming:
```
                            +----------------------------+
                            |Sync and compile (5 minutes)|
                            +------------+---------------+
                                         |
                                         v
                         +-----------------------------------+
                         |Archive to Isolate Server (30 secs)|
                         +---------------+-------------------+
                                         |
                                         |
       +-----------------------+---------+-------------+-------------------------+---------- ... --------+
       |                       |                       |                         |                       |
       v                       v                       v                         v                       v
+-------------------+ +--------------------+ +-------------------------+ +----------------+      +----------------+
|Test Foo (1 minute)| |Test Bar (5 minutes)| |Test Enlarge Shard 1 (5m)| |... Shard 2 (5m)|  ... |... Shard 6 (5m)|
+-------------------+ +--------------------+ +-------------------------+ +----------------+      +----------------+
```
That's a 5+0.5+5 = **10.5 minutes** build. What happens if you want to run a new
test named "Test Extra Slow"? It's still a 10 minutes builds. *Run tests in
O(1)*.

What's the secret sauce to make it work efficiently in practice and lower file
transfer overhead? The [[Isolated Design|Isolate Server]].


## No single point of failure

In general, a normal task distribution mechanism looks like:
```
          +-------+  +----+
          |Clients|  |Bots|
          +---+---+  +-+--+
              |        |
              v        v
         +----------------+
         |Load Balancer(s)|
         +-------+--------+
                 |
                 v
           +------------+
           |Front end(s)|
           +-----+------+
                 |
    +------------+-------------+
    |            |             |
    v            v             v
+--------+ +------------+ +----------------+
|Memcache| |DB server(s)| |Task distributor|
+--------+ +------------+ +----------------+
```

That's a lots of server to maintain, and that if any of them goes down, you are
SOL.


## Running on AppEngine

Here's how it looks like on AppEngine:
```
+-------+  +----+
|Clients|  |Bots|
+---+---+  +-+--+
    |        |
    v        v
  +-----------+
  | AppEngine |
  +-----------+
```


## Benefits from this design

Things you don't have to care about anymore:

   - Having to manage a single server. It's managed by Google SREs.
      - No server to restart.
      - No server's hard drive full.
      - No server CPU usage/RAM usage to care about.
   - Having to handle scaling. Need 500 frontends? Fine.
   - Having cache sizing. Need 15gb memcache? Fine.
   - Having to handle DB size. Want to save 20Tb worth of logs? Fine.
   - One of the frontend crashes? No problem, the client will retry.
   - DB is unavailable for 10s? Clients will retry.
   - NoSQL means no schema update.
   - No upfront cost, usage based cost.
   - Because there's no single server, all the states are always, and by
     definition, saved in the [Cloud DB](https://developers.google.com/datastore/).


## Performance expectations

The current design has the following runtime parameters:
   - Number of task queues (different kind of task request dimensions) in the
     1000s range.
   - Task creation rate below 20tps (task per second).
   - ~20000 bots live.


## Detailed Design

See [Detailed design](Detailed-Design.md).

# Monitoring

Monitoring is done in two parts: capacity and throughput.

Capacity is about the bots and what they are doing. Are bots efficient, mostly
dead, or wasting their time either continuously rebooting or running bot\_config
hooks?

Throughput is about the tasks that are enqueued and executed. Are tasks pending
for long, deduped at a good rate?


## Capacity

Swarming is designed to work with about 20k bots. To get data at a granularity
of 1 minute, this means aggregating data of 33 bots per second on a continuous
basis.


### Design

The way to achieve this is to precompute as much data in the current handlers,
so that the computation handler collects the data, and fill the holes.


## Throughput

Swarming is designed to work at a rate of 20 tasks created per second. In
practice task rate is much lower since the workload is either builds, which are
generally at least in the order of several minutes, or test suites which also
lasts several seconds, if not several minutes or hours.

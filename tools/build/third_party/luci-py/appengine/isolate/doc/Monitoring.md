# Monitoring

The isolate server monitors the lookups, uploads and downloads. It doesn't
monitor the effective cache hit rate on lookups yet.

The BigQuery definition is in [isolated.proto](../proto/isolated.proto) as a
series of StatsSnapshot.

See [../README.md](../README.md) to setup.

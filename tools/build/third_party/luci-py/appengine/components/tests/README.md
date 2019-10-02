Smoke tests for components
--------------------------

This directory contains smoke test that uses `components/*`. Such test can't be
located inside `components/` because it creates symbolic links cycle.

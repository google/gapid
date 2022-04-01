# Fuchsia Configuration

This directory contains two sub-directories: `enabled` and `disabled`. In
`fuchsia_config.bzl` we define a workspace rule that is used to define a
repository named `@local_fuchsia_config`. Inside it we symlink to the files in
either `enabled` or `disabled` based on flags passed to bazel as defined in the
.bazelrc file.

The `disabled` folder is intended to contain the same functions exposed to the
workspace as the `enabled` folder, but empty, thus disabling Fuchsia support and
loading of external dependencies, which could be quite large.

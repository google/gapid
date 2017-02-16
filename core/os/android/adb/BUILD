load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "adb.go",
        "bind.go",
        "commands.go",
        "device.go",
        "doc.go",
        "file.go",
        "forward.go",
        "forward_and_connect.go",
        "inputs.go",
        "installed_package.go",
        "logcat.go",
        "screen.go",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/app:go_default_library",
        "//core/event/task:go_default_library",
        "//core/fault:go_default_library",
        "//core/log:go_default_library",
        "//core/os/android:go_default_library",
        "//core/os/device:go_default_library",
        "//core/os/device/bind:go_default_library",
        "//core/os/file:go_default_library",
        "//core/os/shell:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = [
        "adb_data_test.go",
        "adb_test.go",
        "commands_test.go",
        "device_test.go",
        "file_test.go",
        "installed_package_test.go",
        "logcat_test.go",
        "screen_test.go",
    ],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
        "//core/event/task:go_default_library",
        "//core/log:go_default_library",
        "//core/os/android:go_default_library",
        "//core/os/device:go_default_library",
        "//core/os/file:go_default_library",
        "//core/os/shell:go_default_library",
        "//core/os/shell/stub:go_default_library",
    ],
)

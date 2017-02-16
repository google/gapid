load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")
load("//tools/build:rules.bzl", "api_library", "snippets", "apic", "go_proto_sources")

proto_library(
    name = "pb",
    srcs = ["resolvables.proto"],
    visibility = ["//visibility:public"],
    deps = [
        "//gapis/service/path:pb",
    ],
)

go_proto_sources(
    name = "pb_go",
    visibility = ["//visibility:public"],
    deps = [":pb"],
)

api_library(
    name = "api",
    api = "gles.api",
    apiname = "gles",
    includes = glob([
        "*.api",
        "api/*.api",
    ]),
    visibility = ["//visibility:public"],
    deps = ["//gapis/messages:api"],
)

snippets(
    name = "snippets",
    api = ":api",
    visibility = ["//visibility:public"],
)

apic(
    name = "generated",
    api = ":api",
    templates = [
        "//gapis/gfxapi/templates:api",
        "//gapis/gfxapi/templates:mutate",
        "//gapis/gfxapi/templates:convert",
        "//gapis/gfxapi/templates:proto",
    ],
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    srcs = [
        "backwards_compat.go",
        "compat.go",
        "context.go",
        "custom_replay.go",
        "dead_code_elimination.go",
        "dependency_graph.go",
        "doc.go",
        "draw_call.go",
        "draw_call_mesh.go",
        "externs.go",
        "extras.go",
        "find_issues.go",
        "gles.go",
        "glsl_compat.go",
        "guess_semantics.go",
        "helpers.go",
        "image.go",
        "issue_whitelist.go",
        "links.go",
        "markers.go",
        "metadata.go",
        "read_framebuffer.go",
        "replay.go",
        "resources.go",
        "state.go",
        "string.go",
        "stub_program.go",
        "texture_compat.go",
        "tweaker.go",
        "undefined_framebuffer.go",
        "version.go",
        "vertex_attribute_array.go",
        "wireframe.go",
        ":generated",  # keep
        ":pb_go",  # keep
        ":snippets",  # keep
        "//tools/build/codergen:gapis/gfxapi/gles/gles_binary.go",  # keep
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//core/app/benchmark:go_default_library",
        "//core/context/keys:go_default_library",
        "//core/data/endian:go_default_library",
        "//core/data/id:go_default_library",
        "//core/data/pod:go_default_library",
        "//core/event/task:go_default_library",
        "//core/image:go_default_library",
        "//core/log:go_default_library",
        "//core/math/interval:go_default_library",
        "//core/math/sint:go_default_library",
        "//core/math/u64:go_default_library",
        "//core/os/device:go_default_library",
        "//core/stream:go_default_library",
        "//core/stream/fmts:go_default_library",
        "//framework/binary:go_default_library",
        "//framework/binary/cyclic:go_default_library",
        "//framework/binary/registry:go_default_library",
        "//framework/binary/schema:go_default_library",
        "//framework/binary/vle:go_default_library",
        "//gapis/atom:go_default_library",
        "//gapis/atom/atom_pb:go_default_library",
        "//gapis/atom/transform:go_default_library",
        "//gapis/capture:go_default_library",
        "//gapis/config:go_default_library",
        "//gapis/database:go_default_library",
        "//gapis/gfxapi:go_default_library",
        "//gapis/gfxapi/gles/gles_pb:go_default_library",
        "//gapis/gfxapi/gles/glsl:go_default_library",
        "//gapis/gfxapi/gles/glsl/ast:go_default_library",
        "//gapis/gfxapi/gles/glsl/parser:go_default_library",
        "//gapis/memory:go_default_library",
        "//gapis/memory/memory_pb:go_default_library",
        "//gapis/messages:go_default_library",
        "//gapis/replay:go_default_library",
        "//gapis/replay/builder:go_default_library",
        "//gapis/replay/protocol:go_default_library",
        "//gapis/replay/value:go_default_library",
        "//gapis/resolve:go_default_library",
        "//gapis/service:go_default_library",
        "//gapis/service/path:go_default_library",
        "//gapis/service/pod:go_default_library",
        "//gapis/shadertools:go_default_library",
        "//gapis/stringtable:go_default_library",
        "//gapis/vertex:go_default_library",
        "@com_github_golang_protobuf//proto:go_default_library",
        "@com_github_pkg_errors//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = ["dead_code_elimination_test.go"],
    library = ":go_default_library",
    deps = [
        "//core/assert:go_default_library",
        "//core/log:go_default_library",
        "//gapis/atom:go_default_library",
        "//gapis/atom/test:go_default_library",
        "//gapis/capture:go_default_library",
        "//gapis/database:go_default_library",
        "//gapis/memory:go_default_library",
    ],
)

go_test(
    name = "go_default_xtest",
    size = "small",
    srcs = [
        "compat_test.go",
        "glsl_compat_test.go",
        "markers_test.go",
        "resources_test.go",
        "stub_program_test.go",
    ],
    deps = [
        ":go_default_library",
        "//core/assert:go_default_library",
        "//core/log:go_default_library",
        "//core/os/device:go_default_library",
        "//framework/binary/any:go_default_library",
        "//gapis/atom:go_default_library",
        "//gapis/atom/test:go_default_library",
        "//gapis/capture:go_default_library",
        "//gapis/config:go_default_library",
        "//gapis/database:go_default_library",
        "//gapis/gfxapi:go_default_library",
        "//gapis/gfxapi/gles/glsl/ast:go_default_library",
        "//gapis/memory:go_default_library",
    ],
)

filegroup(
    # This is a hack for as long as codergen lives on...
    name = "codergen_inputs",
    srcs = glob(["*.go"]) + [":generated"],
    visibility = ["//visibility:public"],
)

workspace(name = "gapid")

####################################################################
# Get repositories with rules we need for the rest of the file first

load("@//tools/build/rules:repository.bzl", "github_repository")

github_repository(
    name = "io_bazel_rules_go",
    organization = "bazelbuild",
    project = "rules_go",
    branch = "master",
    commit = "2e319588571f20fdaaf83058b690abd32f596e89", # Comment to use the master branch of this repository
    # path = "../rules_go", # Uncomment to use a local copy of this repository
)

github_repository(
    name = "io_bazel_rules_appengine",
    organization = "bazelbuild",
    project = "rules_appengine",
    commit = "ffe8c3fdc47d4ead45d02e908c56c21b1cc8967b",
)

github_repository(
    name = "com_google_protobuf",
    organization = "google",
    project = "protobuf",
    commit = "593e917c176b5bc5aafa57bf9f6030d749d91cd5",
)

github_repository(
    name = "com_github_grpc_grpc",
    organization = "grpc",
    project = "grpc",
    commit = "fa301e3674a1cc786eb4dd4253a0e677f2eb68e3",
)

##############################
# Load all our workspace rules

load("@io_bazel_rules_go//go:def.bzl", "go_repositories")
load("@io_bazel_rules_appengine//appengine:appengine.bzl", "appengine_repositories")
load("@bazel_tools//tools/cpp:cc_configure.bzl", "cc_configure")
load("@//tools/build:rules.bzl", "empty_repository", "github_go_repository", "windows_sdk")

#########################################################
# Run our workspace preparation rules

go_repositories()
appengine_repositories()
cc_configure()

windows_sdk(
    name="windows_sdk",
)

android_sdk_repository(
    name="androidsdk",
    api_level=21,
)

android_ndk_repository(
    name="androidndk",
    api_level=21,
)

####################################
# Now get all our other dependancies

github_repository(
    name = "io_bazel",
    organization = "bazelbuild",
    project = "bazel",
    commit = "b78d8c8b31a530e1a94d1feeac34aceac67a31df",
)

github_repository(
    name = "com_github_grpc_java",
    organization = "grpc",
    project = "grpc-java",
    commit = "009c51f2f793aabf516db90a14a52da2b613aa21",
    build_file = "//tools/build/third_party:grpc_java.BUILD",
)

github_repository(
    name = "cityhash",
    organization = "google",
    project = "cityhash",
    commit = "8af9b8c2b889d80c22d6bc26ba0df1afb79a30db",
    build_file = "//tools/build/third_party:cityhash.BUILD",
)

github_repository(
    name = "astc-encoder",
    organization = "ARM-software",
    project = "astc-encoder",
    commit = "a1b3a964ea999a7284a223ebd23bc1a65b991073",
    build_file = "//tools/build/third_party:astc-encoder.BUILD",
)

github_repository(
    name = "spirv-tools",
    organization = "KhronosGroup",
    project = "SPIRV-Tools",
    commit = "0b0454c42c6b6f6746434bd5c78c5c70f65d9c51",
    build_file = "//tools/build/third_party:spirv-tools.BUILD",
)

github_repository(
    name = "spirv-cross",
    organization = "KhronosGroup",
    project = "SPIRV-Cross",
    commit = "98a17431c24b47392cbe343da8dbd1f5ffbb23e8",
    build_file = "//tools/build/third_party:spirv-cross.BUILD",
)

github_repository(
    name = "spirv-headers",
    organization = "KhronosGroup",
    project = "SPIRV-Headers",
    commit = "2bf02308656f97898c5f7e433712f21737c61e4e",
    build_file = "//tools/build/third_party:spirv-headers.BUILD",
)

github_repository(
    name = "glslang",
    organization = "KhronosGroup",
    project = "glslang",
    commit = "778806a69246b8921e867e839c9e87ccddc924f2",
    build_file = "//tools/build/third_party:glslang.BUILD",
)

github_repository(
    name = "llvm",
    organization = "llvm-mirror",
    project = "llvm",
    commit = "4fba04fd9608115c1813dfba8909ab43e36ba92d",
    build_file = "//tools/build/third_party:llvm.BUILD",
)

github_go_repository(
    name = "org_golang_x_crypto",
    organization = "golang",
    project = "crypto",
    commit = "dc137beb6cce2043eb6b5f223ab8bf51c32459f4",
    importpath = "golang.org/x/crypto",
)

github_go_repository(
    name = "org_golang_x_net",
    organization = "golang",
    project = "net",
    commit = "f2499483f923065a842d38eb4c7f1927e6fc6e6d",
    importpath = "golang.org/x/net",
)

github_go_repository(
    name = "org_golang_x_sys",
    organization = "golang",
    project = "sys",
    commit = "d75a52659825e75fff6158388dddc6a5b04f9ba5",
    importpath = "golang.org/x/sys",
)

github_go_repository(
    name = "org_golang_x_tools",
    organization = "golang",
    project = "tools",
    commit = "3da34b1b520a543128e8441cd2ffffc383111d03",
    importpath = "golang.org/x/tools",
)

github_go_repository(
    name = "org_golang_google_grpc",
    organization = "grpc",
    project = "grpc-go",
    commit = "50955793b0183f9de69bd78e2ec251cf20aab121",
    importpath = "google.golang.org/grpc",
)

github_go_repository(
    name = "com_github_pkg_errors",
    organization = "pkg",
    project = "errors",
    commit = "248dadf4e9068a0b3e79f02ed0a610d935de5302",
    importpath = "github.com/pkg/errors",
)

github_go_repository(
    name = "com_github_spf13_pflag",
    organization = "spf13",
    project = "pflag",
    commit = "dc137beb6cce2043eb6b5f223ab8bf51c32459f4",
    importpath = "github.com/spf13/pflag",
)

github_go_repository(
    name = "com_github_spf13_cobra",
    organization = "spf13",
    project = "cobra",
    commit = "35136c09d8da66b901337c6e86fd8e88a1a255bd",
    importpath = "github.com/spf13/cobra",
)

github_go_repository(
    name = "com_github_golang_protobuf",
    organization = "golang",
    project = "protobuf",
    commit = "8ee79997227bf9b34611aee7946ae64735e6fd93",
    importpath = "github.com/golang/protobuf",
)

empty_repository(
    name = "ptypes",
    build_file = "//tools/build/third_party:ptypes.BUILD",
)

github_go_repository(
    name = "com_github_fsnotify_fsnotify",
    organization = "fsnotify",
    project = "fsnotify",
    commit = "a904159b9206978bb6d53fcc7a769e5cd726c737",
    importpath = "github.com/fsnotify/fsnotify",
)

github_go_repository(
    name = "com_github_gopherjs_gopherjs",
    organization = "gopherjs",
    project = "gopherjs",
    commit = "2967252ace8b112e63a5b5879e92de915fe731f4",
    importpath = "github.com/gopherjs/gopherjs",
)

github_go_repository(
    name = "com_github_kardianos_osext",
    organization = "kardianos",
    project = "osext",
    commit = "c2c54e542fb797ad986b31721e1baedf214ca413",
    importpath = "github.com/kardianos/osext",
)

github_go_repository(
    name = "com_github_neelance_sourcemap",
    organization = "neelance",
    project = "sourcemap",
    commit = "8c68805598ab8d5637b1a72b5f7d381ea0f39c31",
    importpath = "github.com/neelance/sourcemap",
)

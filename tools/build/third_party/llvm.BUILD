# Copyright (C) 2018 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

load("@gapid//tools/build/third_party:llvm/rules.bzl", "llvm_cc_copts", "llvm_sources", "tablegen")
load("@gapid//tools/build/third_party:llvm/libs.bzl", "llvm_auto_libs")
load("@gapid//tools/build/rules:cc.bzl", "cc_copts")
load("@io_bazel_rules_go//go:def.bzl", "go_library")

package(default_visibility = ["//visibility:public"])

filegroup(
    name = "table_includes",
    srcs = glob(["include/**/*.td"]),
)

tablegen(
    name = "Attributes",
    rules = [
        [
            "include/llvm/IR/Attributes.inc",
            "-gen-attrs",
        ],
    ],
    strip_include_prefix = "include",
    table = "include/llvm/IR/Attributes.td",
    deps = [":table_includes"],
)

tablegen(
    name = "InstCombineTables",
    rules = [
        [
            "lib/Transforms/InstCombine/InstCombineTables.inc",
            "-gen-searchable-tables",
        ],
    ],
    strip_include_prefix = "lib/Transforms/InstCombine",
    table = "lib/Transforms/InstCombine/InstCombineTables.td",
    deps = [":table_includes"] + glob(["lib/Transforms/InstCombine/*.td"]),
)

tablegen(
    name = "Intrinsics",
    rules = [
        [
            "include/llvm/IR/IntrinsicEnums.inc",
            "-gen-intrinsic-enums",
        ],
        [
            "include/llvm/IR/IntrinsicImpl.inc",
            "-gen-intrinsic-impl",
        ],
    ],
    strip_include_prefix = "include",
    table = "include/llvm/IR/Intrinsics.td",
    deps = [":table_includes"],
)

tablegen(
    name = "AttributesCompatFunc",
    rules = [
        [
            "lib/IR/AttributesCompatFunc.inc",
            "-gen-attrs",
        ],
    ],
    strip_include_prefix = "lib/IR",
    table = "lib/IR/AttributesCompatFunc.td",
    deps = [":table_includes"],
)

tablegen(
    name = "LibDriver/Options",
    rules = [
        [
            "lib/LibDriver/Options.inc",
            "-gen-opt-parser-defs",
        ],
    ],
    strip_include_prefix = "lib/LibDriver",
    table = "lib/LibDriver/Options.td",
    deps = [":table_includes"],
)

tablegen(
    name = "tables-AArch64",
    rules = [
        [
            "lib/Target/AArch64/AArch64GenRegisterInfo.inc",
            "-gen-register-info",
        ],
        [
            "lib/Target/AArch64/AArch64GenInstrInfo.inc",
            "-gen-instr-info",
        ],
        [
            "lib/Target/AArch64/AArch64GenMCCodeEmitter.inc",
            "-gen-emitter",
        ],
        [
            "lib/Target/AArch64/AArch64GenMCPseudoLowering.inc",
            "-gen-pseudo-lowering",
        ],
        [
            "lib/Target/AArch64/AArch64GenAsmWriter.inc",
            "-gen-asm-writer",
        ],
        [
            "lib/Target/AArch64/AArch64GenAsmWriter1.inc",
            "-gen-asm-writer",
            "-asmwriternum=1",
        ],
        [
            "lib/Target/AArch64/AArch64GenAsmMatcher.inc",
            "-gen-asm-matcher",
        ],
        [
            "lib/Target/AArch64/AArch64GenDAGISel.inc",
            "-gen-dag-isel",
        ],
        [
            "lib/Target/AArch64/AArch64GenFastISel.inc",
            "-gen-fast-isel",
        ],
        [
            "lib/Target/AArch64/AArch64GenCallingConv.inc",
            "-gen-callingconv",
        ],
        [
            "lib/Target/AArch64/AArch64GenSubtargetInfo.inc",
            "-gen-subtarget",
        ],
        [
            "lib/Target/AArch64/AArch64GenDisassemblerTables.inc",
            "-gen-disassembler",
        ],
        [
            "lib/Target/AArch64/AArch64GenSystemOperands.inc",
            "-gen-searchable-tables",
        ],
        [
            "lib/Target/AArch64/AArch64GenRegisterBank.inc",
            "-gen-register-bank",
        ],
        [
            "lib/Target/AArch64/AArch64GenGlobalISel.inc",
            "-gen-global-isel",
        ],
    ],
    strip_include_prefix = "lib/Target/AArch64",
    table = "lib/Target/AArch64/AArch64.td",
    deps = [":table_includes"] + glob(["lib/Target/AArch64/*.td"]),
)

cc_library(
    name = "headers-AArch64",
    hdrs = glob(["lib/Target/AArch64/**/*.h"]),
    strip_include_prefix = "lib/Target/AArch64",
    deps = [":tables-AArch64"],
)

tablegen(
    name = "tables-ARM",
    rules = [
        [
            "lib/Target/ARM/ARMGenRegisterInfo.inc",
            "-gen-register-info",
        ],
        [
            "lib/Target/ARM/ARMGenInstrInfo.inc",
            "-gen-instr-info",
        ],
        [
            "lib/Target/ARM/ARMGenMCCodeEmitter.inc",
            "-gen-emitter",
        ],
        [
            "lib/Target/ARM/ARMGenMCPseudoLowering.inc",
            "-gen-pseudo-lowering",
        ],
        [
            "lib/Target/ARM/ARMGenAsmWriter.inc",
            "-gen-asm-writer",
        ],
        [
            "lib/Target/ARM/ARMGenAsmMatcher.inc",
            "-gen-asm-matcher",
        ],
        [
            "lib/Target/ARM/ARMGenDAGISel.inc",
            "-gen-dag-isel",
        ],
        [
            "lib/Target/ARM/ARMGenFastISel.inc",
            "-gen-fast-isel",
        ],
        [
            "lib/Target/ARM/ARMGenCallingConv.inc",
            "-gen-callingconv",
        ],
        [
            "lib/Target/ARM/ARMGenSubtargetInfo.inc",
            "-gen-subtarget",
        ],
        [
            "lib/Target/ARM/ARMGenDisassemblerTables.inc",
            "-gen-disassembler",
        ],
        [
            "lib/Target/ARM/ARMGenSystemRegister.inc",
            "-gen-searchable-tables",
        ],
        [
            "lib/Target/ARM/ARMGenRegisterBank.inc",
            "-gen-register-bank",
        ],
        [
            "lib/Target/ARM/ARMGenGlobalISel.inc",
            "-gen-global-isel",
        ],
    ],
    strip_include_prefix = "lib/Target/ARM",
    table = "lib/Target/ARM/ARM.td",
    deps = [":table_includes"] + glob(["lib/Target/ARM/*.td"]),
)

cc_library(
    name = "headers-ARM",
    hdrs = glob(["lib/Target/ARM/**/*.h"]),
    strip_include_prefix = "lib/Target/ARM",
    deps = [":tables-ARM"],
)

cc_library(
    name = "headers-RuntimeDyld",
    hdrs = glob(["lib/ExecutionEngine/RuntimeDyld/*.h"]),
    strip_include_prefix = "lib/ExecutionEngine/RuntimeDyld",
)

cc_library(
    name = "headers-SupportWindows",
    hdrs = glob(["lib/Support/Windows/*.h"]),
    strip_include_prefix = "lib/Support",
)

# RuntimeDyldELFMips is referenced by RuntimeDyldELF.cpp whether we're building
# MIPS or not.
cc_library(
    name = "RuntimeDyldELFMips",
    srcs = glob([
        "lib/ExecutionEngine/RuntimeDyld/Targets/RuntimeDyldELFMips.cpp",
        "lib/ExecutionEngine/RuntimeDyld/**/*.h",
    ]),
    copts = llvm_cc_copts() + select({
        "@gapid//tools/build:windows": ["-D__STDC_FORMAT_MACROS"],
        "//conditions:default": [],
    }),
    strip_include_prefix = "lib/ExecutionEngine/RuntimeDyld/Targets",
    deps = [":headers"],
)

tablegen(
    name = "tables-X86",
    rules = [
        [
            "lib/Target/X86/X86GenRegisterInfo.inc",
            "-gen-register-info",
        ],
        [
            "lib/Target/X86/X86GenDisassemblerTables.inc",
            "-gen-disassembler",
        ],
        [
            "lib/Target/X86/X86GenInstrInfo.inc",
            "-gen-instr-info",
        ],
        [
            "lib/Target/X86/X86GenAsmWriter.inc",
            "-gen-asm-writer",
        ],
        [
            "lib/Target/X86/X86GenAsmWriter1.inc",
            "-gen-asm-writer",
            "-asmwriternum=1",
        ],
        [
            "lib/Target/X86/X86GenAsmMatcher.inc",
            "-gen-asm-matcher",
        ],
        [
            "lib/Target/X86/X86GenDAGISel.inc",
            "-gen-dag-isel",
        ],
        [
            "lib/Target/X86/X86GenFastISel.inc",
            "-gen-fast-isel",
        ],
        [
            "lib/Target/X86/X86GenCallingConv.inc",
            "-gen-callingconv",
        ],
        [
            "lib/Target/X86/X86GenSubtargetInfo.inc",
            "-gen-subtarget",
        ],
        [
            "lib/Target/X86/X86GenRegisterBank.inc",
            "-gen-register-bank",
        ],
        [
            "lib/Target/X86/X86GenEVEX2VEXTables.inc",
            "-gen-x86-EVEX2VEX-tables",
        ],
        [
            "lib/Target/X86/X86GenGlobalISel.inc",
            "-gen-global-isel",
        ],
    ],
    strip_include_prefix = "lib/Target/X86",
    table = "lib/Target/X86/X86.td",
    deps = [":table_includes"] + glob(["lib/Target/X86/*.td"]),
)

cc_library(
    name = "headers-X86",
    hdrs = glob(["lib/Target/X86/**/*.h"]),
    strip_include_prefix = "lib/Target/X86",
    deps = [":tables-X86"],
)

cc_library(
    name = "headers",
    hdrs = glob(["include/**/*"]),
    strip_include_prefix = "include",
    deps = [
        "@gapid//tools/build/third_party:llvm-config",
    ],
)

llvm_auto_libs(
    # The table below is the source files that should be excluded from the globs
    excludes = {
    },
    # The table below is the extra dependancies not declared in the LLVMBuild.txt files
    # They are added to the dependancies declared in the generated llvm/rules.bzl file
    extras = {
        "Core": [
            ":Intrinsics",
            ":Attributes",
            ":AttributesCompatFunc",
        ],
        "Demangle": [":headers"],
        "AArch64Utils": [":headers-AArch64"],
        "AArchInfo": [":headers-AArch64"],
        "ARMAsmPrinter": [":headers-ARM"],
        "ARMDesc": [":headers-ARM"],
        "ARMInfo": [":headers-ARM"],
        "ARMUtils": [":headers-ARM"],
        "LibDriver": [":LibDriver/Options"],
        "InstCombine": [":InstCombineTables"],
        "MC": [
            ":Intrinsics",
            ":Attributes",
        ],
        "RuntimeDyld": [
            ":headers-RuntimeDyld",
            ":RuntimeDyldELFMips",
        ],
        "Support": [":headers-SupportWindows"],
        "X86Info": [":headers-X86"],
        "X86Utils": [":headers-X86"],
    },
)

cc_binary(
    name = "llvm-tblgen",
    srcs = llvm_sources("utils/TableGen") + glob(["lib/Target/**/*.h"]),
    copts = llvm_cc_copts(),
    linkopts = select({
        "@gapid//tools/build:linux": [
            "-ldl",
            "-lpthread",
            "-lcurses",
        ],
        "@gapid//tools/build:darwin": [
            "-framework Cocoa",
            "-lcurses",
        ],
        "@gapid//tools/build:windows": [
            "-luuid",
            "-lole32",
        ],
    }),
    deps = [":TableGen"],
)

cc_library(
    name = "go_binding_lib",
    hdrs = glob(["bindings/go/llvm/*.h"]),
    linkopts = select({
        "@gapid//tools/build:linux": [
            "-ldl",
            "-lpthread",
            "-lcurses",
            "-lz",
            "-lm",
        ],
        "@gapid//tools/build:darwin": [
            "-framework Cocoa",
            "-lcurses",
            "-lz",
            "-lm",
        ],
        "@gapid//tools/build:windows": [
            "-luuid",
            "-lole32",
            "-lpsapi",
        ],
    }),
    strip_include_prefix = "bindings/go/llvm",
    visibility = ["//visibility:public"],
    deps = [
        ":AArch64AsmPrinter",
        ":AArch64CodeGen",
        ":AArch64Desc",
        ":AArch64Info",
        ":AArch64Utils",
        ":ARMAsmPrinter",
        ":ARMCodeGen",
        ":ARMDesc",
        ":ARMInfo",
        ":Analysis",
        ":AsmPrinter",
        ":Attributes",
        ":BitReader",
        ":BitWriter",
        ":CodeGen",
        ":Core",
        ":Coroutines",
        ":DebugInfoCodeView",
        ":DebugInfoMSF",
        ":Demangle",
        ":ExecutionEngine",
        ":GlobalISel",
        ":IPO",
        ":IRReader",
        ":InstCombine",
        ":Instrumentation",
        ":Interpreter",
        ":Intrinsics",
        ":Linker",
        ":MC",
        ":MCDisassembler",
        ":MCJIT",
        ":MCParser",
        ":Object",
        ":ProfileData",
        ":RuntimeDyld",
        ":Scalar",
        ":SelectionDAG",
        ":Support",
        ":Target",
        ":TransformUtils",
        ":Vectorize",
        ":X86AsmPrinter",
        ":X86CodeGen",
        ":X86Desc",
        ":X86Info",
        ":X86Utils",
        ":headers",
    ],
)

go_library(
    name = "go",
    srcs = glob(
        [
            "bindings/go/llvm/*.go",
            "bindings/go/llvm/*.cpp",
        ],
        exclude = ["bindings/go/llvm/llvm_dep.go"],
    ),
    cdeps = [":go_binding_lib"],
    cgo = True,
    importpath = "llvm/bindings/go/llvm",
    visibility = ["//visibility:public"],
)

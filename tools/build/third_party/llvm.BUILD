load("@//tools/build/third_party:llvm/rules.bzl", "llvm_sources", "tablegen")
load("@//tools/build/third_party:llvm/libs.bzl", "llvm_auto_libs")
load("@//tools/build/rules:cc.bzl", "cc_copts")

package(default_visibility = ["//visibility:public"])

filegroup(
    name="table_includes",
    srcs=glob(["include/**/*.td"]),
)

tablegen(
    name = "Attributes",
    table = "include/llvm/IR/Attributes.td",
    deps = [":table_includes"],
    rules = [
        ["include/llvm/IR/Attributes.gen", "-gen-attrs"],
    ],
    strip_include_prefix = "include",
)

tablegen(
    name = "Intrinsics",
    table = "include/llvm/IR/Intrinsics.td",
    deps = [":table_includes"],
    rules = [
        ["include/llvm/IR/Intrinsics.gen", "-gen-intrinsic"],
    ],
    strip_include_prefix = "include",
)

tablegen(
    name = "AttributesCompatFunc",
    table = "lib/IR/AttributesCompatFunc.td",
    deps = [":table_includes"],
    rules = [
        ["lib/IR/AttributesCompatFunc.inc", "-gen-attrs"],
    ],
    strip_include_prefix = "lib/IR",
)

tablegen(
    name = "LibDriver/Options",
    table = "lib/LibDriver/Options.td",
    deps = [":table_includes"],
    rules = [
        ["lib/LibDriver/Options.inc", "-gen-opt-parser-defs"],
    ],
    strip_include_prefix = "lib/LibDriver",
)

tablegen(
    name = "tables-AArch64",
    table = "lib/Target/AArch64/AArch64.td",
    deps = [":table_includes"] + glob(["lib/Target/AArch64/*.td"]),
    rules = [
        ["lib/Target/AArch64/AArch64GenRegisterInfo.inc", "-gen-register-info"],
        ["lib/Target/AArch64/AArch64GenInstrInfo.inc", "-gen-instr-info"],
        ["lib/Target/AArch64/AArch64GenMCCodeEmitter.inc", "-gen-emitter"],
        ["lib/Target/AArch64/AArch64GenMCPseudoLowering.inc", "-gen-pseudo-lowering"],
        ["lib/Target/AArch64/AArch64GenAsmWriter.inc", "-gen-asm-writer"],
        ["lib/Target/AArch64/AArch64GenAsmWriter1.inc", "-gen-asm-writer", "-asmwriternum=1"],
        ["lib/Target/AArch64/AArch64GenAsmMatcher.inc", "-gen-asm-matcher"],
        ["lib/Target/AArch64/AArch64GenDAGISel.inc", "-gen-dag-isel"],
        ["lib/Target/AArch64/AArch64GenFastISel.inc", "-gen-fast-isel"],
        ["lib/Target/AArch64/AArch64GenCallingConv.inc", "-gen-callingconv"],
        ["lib/Target/AArch64/AArch64GenSubtargetInfo.inc", "-gen-subtarget"],
        ["lib/Target/AArch64/AArch64GenDisassemblerTables.inc", "-gen-disassembler"],
        ["lib/Target/AArch64/AArch64GenSystemOperands.inc", "-gen-searchable-tables"],
    ],
    strip_include_prefix = "lib/Target/AArch64",
)

cc_library(
    name = "headers-AArch64",
    hdrs = glob(["lib/Target/AArch64/**/*.h"]),
    deps = [":tables-AArch64"],
    strip_include_prefix = "lib/Target/AArch64",
)


tablegen(
    name = "tables-ARM",
    table = "lib/Target/ARM/ARM.td",
    deps = [":table_includes"] + glob(["lib/Target/ARM/*.td"]),
    rules = [
        ["lib/Target/ARM/ARMGenRegisterInfo.inc", "-gen-register-info"],
        ["lib/Target/ARM/ARMGenInstrInfo.inc", "-gen-instr-info"],
        ["lib/Target/ARM/ARMGenMCCodeEmitter.inc", "-gen-emitter"],
        ["lib/Target/ARM/ARMGenMCPseudoLowering.inc", "-gen-pseudo-lowering"],
        ["lib/Target/ARM/ARMGenAsmWriter.inc", "-gen-asm-writer"],
        ["lib/Target/ARM/ARMGenAsmMatcher.inc", "-gen-asm-matcher"],
        ["lib/Target/ARM/ARMGenDAGISel.inc", "-gen-dag-isel"],
        ["lib/Target/ARM/ARMGenFastISel.inc", "-gen-fast-isel"],
        ["lib/Target/ARM/ARMGenCallingConv.inc", "-gen-callingconv"],
        ["lib/Target/ARM/ARMGenSubtargetInfo.inc", "-gen-subtarget"],
        ["lib/Target/ARM/ARMGenDisassemblerTables.inc", "-gen-disassembler"],
    ],
    strip_include_prefix = "lib/Target/ARM",
)

cc_library(
    name = "headers-ARM",
    hdrs = glob(["lib/Target/ARM/**/*.h"]),
    deps = [":tables-ARM"],
    strip_include_prefix = "lib/Target/ARM",
)

tablegen(
    name = "tables-Mips",
    table = "lib/Target/Mips/Mips.td",
    deps = [":table_includes"] + glob(["lib/Target/Mips/*.td"]),
    rules = [
        ["lib/Target/Mips/MipsGenRegisterInfo.inc", "-gen-register-info"],
        ["lib/Target/Mips/MipsGenInstrInfo.inc", "-gen-instr-info"],
        ["lib/Target/Mips/MipsGenDisassemblerTables.inc", "-gen-disassembler"],
        ["lib/Target/Mips/MipsGenMCCodeEmitter.inc", "-gen-emitter"],
        ["lib/Target/Mips/MipsGenAsmWriter.inc", "-gen-asm-writer"],
        ["lib/Target/Mips/MipsGenDAGISel.inc", "-gen-dag-isel"],
        ["lib/Target/Mips/MipsGenFastISel.inc", "-gen-fast-isel"],
        ["lib/Target/Mips/MipsGenCallingConv.inc", "-gen-callingconv"],
        ["lib/Target/Mips/MipsGenSubtargetInfo.inc", "-gen-subtarget"],
        ["lib/Target/Mips/MipsGenAsmMatcher.inc", "-gen-asm-matcher"],
        ["lib/Target/Mips/MipsGenMCPseudoLowering.inc", "-gen-pseudo-lowering"],
    ],
    strip_include_prefix = "lib/Target/Mips",
)

cc_library(
    name = "headers-Mips",
    hdrs = glob(["lib/Target/Mips/**/*.h"]),
    deps = [":tables-Mips"],
    strip_include_prefix = "lib/Target/Mips",
)

tablegen(
    name = "tables-X86",
    table = "lib/Target/X86/X86.td",
    deps = [":table_includes"] + glob(["lib/Target/X86/*.td"]),
    rules = [
        ["lib/Target/X86/X86GenRegisterInfo.inc", "-gen-register-info"],
        ["lib/Target/X86/X86GenDisassemblerTables.inc", "-gen-disassembler"],
        ["lib/Target/X86/X86GenInstrInfo.inc", "-gen-instr-info"],
        ["lib/Target/X86/X86GenAsmWriter.inc", "-gen-asm-writer"],
        ["lib/Target/X86/X86GenAsmWriter1.inc", "-gen-asm-writer", "-asmwriternum=1"],
        ["lib/Target/X86/X86GenAsmMatcher.inc", "-gen-asm-matcher"],
        ["lib/Target/X86/X86GenDAGISel.inc", "-gen-dag-isel"],
        ["lib/Target/X86/X86GenFastISel.inc", "-gen-fast-isel"],
        ["lib/Target/X86/X86GenCallingConv.inc", "-gen-callingconv"],
        ["lib/Target/X86/X86GenSubtargetInfo.inc", "-gen-subtarget"],
    ],
    strip_include_prefix = "lib/Target/X86",
)

cc_library(
    name = "headers-X86",
    hdrs = glob(["lib/Target/X86/**/*.h"]),
    deps = [":tables-X86"],
    strip_include_prefix = "lib/Target/X86",
)

cc_library(
    name = "headers",
    hdrs = glob(["include/**/*"]),
    strip_include_prefix = "include",
    deps = [
        "@//tools/build/third_party:llvm-config",
    ],
)

# The following are all excluded because they depend on LLVM_BUILD_GLOBAL_ISEL
ISEL_EXCLUDES=[
    "**/*CallLowering.cpp",
    "**/*InstructionSelector.cpp",
    "**/*LegalizerInfo.cpp",
    "**/*RegisterBankInfo.cpp"
]

llvm_auto_libs(
    # The table below is the source files that should be excluded from the globs
    excludes = {
        "AArch64CodeGen": ISEL_EXCLUDES,
        "ARMCodeGen": ISEL_EXCLUDES,
        "MipsCodeGen": ISEL_EXCLUDES,
        "X86CodeGen": ISEL_EXCLUDES,
    },
    # The table below is the extra dependancies not declared in the LLVMBuild.txt files
    # They are added to the depdendancies declared in the generated llvm/rules.bzl file
    extras = {
        "Demangle": [":headers"],
        "Core": [":Intrinsics", ":Attributes", ":AttributesCompatFunc"],
        "AArch64Utils": [":headers-AArch64"],
        "AArchInfo": [":headers-AArch64"],
        "ARMDesc":  [":headers-ARM"],
        "ARMAsmPrinter":  [":headers-ARM"],
        "ARMInfo":  [":headers-ARM"],
        "MipsDesc": [":headers-Mips"],
        "MipsAsmPrinter": [":headers-Mips"],
        "MipsInfo": [":headers-Mips", ":Intrinsics", ":Attributes"],
        "X86Utils": [":headers-X86"],
        "X86Info": [":headers-X86"],
        "LibDriver": [":LibDriver/Options"],
        "MC": [":Intrinsics", ":Attributes"],
    }
)

cc_binary(
    name = "llvm-tblgen",
    srcs = llvm_sources("utils/TableGen") + glob(["lib/Target/**/*.h"]),
    deps = [":TableGen"],
    linkopts = select({
        "@//tools/build:linux": ["-ldl", "-lpthread", "-lcurses"],
        "@//tools/build:darwin": ["-framework Cocoa"],
        "@//tools/build:windows": [],
    }),
    copts = cc_copts(),
)

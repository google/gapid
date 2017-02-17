// Copyright (c) 2015-2016 The Khronos Group Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and/or associated documentation files (the
// "Materials"), to deal in the Materials without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Materials, and to
// permit persons to whom the Materials are furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Materials.
//
// MODIFICATIONS TO THIS FILE MAY MEAN IT NO LONGER ACCURATELY REFLECTS
// KHRONOS STANDARDS. THE UNMODIFIED, NORMATIVE VERSIONS OF KHRONOS
// SPECIFICATIONS AND HEADER INFORMATION ARE LOCATED AT
//    https://www.khronos.org/registry/
//
// THE MATERIALS ARE PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
// CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
// TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
// MATERIALS OR THE USE OR OTHER DEALINGS IN THE MATERIALS.

// Validation tests for SSA

#include "UnitSPIRV.h"
#include "ValidateFixtures.h"
#include "gmock/gmock.h"

#include <sstream>
#include <string>
#include <utility>

using ::testing::HasSubstr;
using ::testing::MatchesRegex;

using std::string;
using std::pair;
using std::stringstream;

namespace {
using ValidateSSA = spvtest::ValidateBase<pair<string, bool>>;

TEST_F(ValidateSSA, Default) {
  char str[] = R"(
     OpCapability Shader
     OpMemoryModel Logical GLSL450
     OpEntryPoint GLCompute %3 ""
     OpExecutionMode %3 LocalSize 1 1 1
%1 = OpTypeVoid
%2 = OpTypeFunction %1
%3 = OpFunction %1 None %2
%4 = OpLabel
     OpReturn
     OpFunctionEnd
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_F(ValidateSSA, IdUndefinedBad) {
  char str[] = R"(
          OpCapability Shader
          OpMemoryModel Logical GLSL450
          OpName %missing "missing"
%voidt  = OpTypeVoid
%vfunct = OpTypeFunction %voidt
%func   = OpFunction %vfunct None %missing
%flabel = OpLabel
          OpReturn
          OpFunctionEnd
    )";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("missing"));
}

TEST_F(ValidateSSA, IdRedefinedBad) {
  char str[] = R"(
     OpCapability Shader
     OpMemoryModel Logical GLSL450
     OpName %2 "redefined"
%1 = OpTypeVoid
%2 = OpTypeFunction %1
%2 = OpFunction %1 None %2
%4 = OpLabel
     OpReturn
     OpFunctionEnd
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
}

TEST_F(ValidateSSA, DominateUsageBad) {
  char str[] = R"(
     OpCapability Shader
     OpMemoryModel Logical GLSL450
     OpName %1 "not_dominant"
%2 = OpTypeFunction %1              ; uses %1 before it's definition
%1 = OpTypeVoid
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("not_dominant"));
}

TEST_F(ValidateSSA, DominateUsageWithinBlockBad) {
  char str[] = R"(
     OpCapability Shader
     OpMemoryModel Logical GLSL450
     OpName %bad "bad"
%voidt = OpTypeVoid
%funct = OpTypeFunction %voidt
%uintt = OpTypeInt 32 0
%one   = OpConstant %uintt 1
%func  = OpFunction %voidt None %funct
%entry = OpLabel
%sum   = OpIAdd %uintt %one %bad
%bad   = OpCopyObject %uintt %sum
         OpReturn
         OpFunctionEnd
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(),
              MatchesRegex("ID .\\[bad\\] has not been defined"));
}

TEST_F(ValidateSSA, DominateUsageSameInstructionBad) {
  char str[] = R"(
     OpCapability Shader
     OpMemoryModel Logical GLSL450
     OpName %sum "sum"
%voidt = OpTypeVoid
%funct = OpTypeFunction %voidt
%uintt = OpTypeInt 32 0
%one   = OpConstant %uintt 1
%func  = OpFunction %voidt None %funct
%entry = OpLabel
%sum   = OpIAdd %uintt %one %sum
         OpReturn
         OpFunctionEnd
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(),
              MatchesRegex("ID .\\[sum\\] has not been defined"));
}

TEST_F(ValidateSSA, ForwardNameGood) {
  char str[] = R"(
     OpCapability Shader
     OpMemoryModel Logical GLSL450
     OpName %3 "main"
%1 = OpTypeVoid
%2 = OpTypeFunction %1
%3 = OpFunction %1 None %2
%4 = OpLabel
     OpReturn
     OpFunctionEnd
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_F(ValidateSSA, ForwardNameMissingTargetBad) {
  char str[] = R"(
      OpCapability Shader
      OpMemoryModel Logical GLSL450
      OpName %5 "main"              ; Target never defined
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("main"));
}

TEST_F(ValidateSSA, ForwardMemberNameGood) {
  char str[] = R"(
           OpCapability Shader
           OpMemoryModel Logical GLSL450
           OpMemberName %struct 0 "value"
           OpMemberName %struct 1 "size"
%intt   =  OpTypeInt 32 1
%uintt  =  OpTypeInt 32 0
%struct =  OpTypeStruct %intt %uintt
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_F(ValidateSSA, ForwardMemberNameMissingTargetBad) {
  char str[] = R"(
           OpCapability Shader
           OpMemoryModel Logical GLSL450
           OpMemberName %struct 0 "value"
           OpMemberName %bad 1 "size"     ; Target is not defined
%intt   =  OpTypeInt 32 1
%uintt  =  OpTypeInt 32 0
%struct =  OpTypeStruct %intt %uintt
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("size"));
}

TEST_F(ValidateSSA, ForwardDecorateGood) {
  char str[] = R"(
           OpCapability Shader
           OpMemoryModel Logical GLSL450
           OpDecorate %var Restrict
%intt   =  OpTypeInt 32 1
%ptrt   =  OpTypePointer UniformConstant %intt
%var    =  OpVariable %ptrt UniformConstant
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_F(ValidateSSA, ForwardDecorateInvalidIDBad) {
  char str[] = R"(
           OpCapability Shader
           OpMemoryModel Logical GLSL450
           OpName %missing "missing"
           OpDecorate %missing Restrict        ;Missing ID
%voidt  =  OpTypeVoid
%intt   =  OpTypeInt 32 1
%ptrt   =  OpTypePointer UniformConstant %intt
%var    =  OpVariable %ptrt UniformConstant
%2      =  OpTypeFunction %voidt
%3      =  OpFunction %voidt None %2
%4      =  OpLabel
           OpReturn
           OpFunctionEnd
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("missing"));
}

TEST_F(ValidateSSA, ForwardMemberDecorateGood) {
  char str[] = R"(
           OpCapability Shader
           OpMemoryModel Logical GLSL450
           OpMemberDecorate %struct 1 RowMajor
%intt   =  OpTypeInt 32 1
%vec3   =  OpTypeVector %intt 3
%mat33  =  OpTypeMatrix %vec3 3
%struct =  OpTypeStruct %intt %mat33
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_F(ValidateSSA, ForwardMemberDecorateInvalidIdBad) {
  char str[] = R"(
           OpCapability Shader
           OpMemoryModel Logical GLSL450
           OpName %missing "missing"
           OpMemberDecorate %missing 1 RowMajor ; Target not defined
%intt   =  OpTypeInt 32 1
%vec3   =  OpTypeVector %intt 3
%mat33  =  OpTypeMatrix %vec3 3
%struct =  OpTypeStruct %intt %mat33
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("missing"));
}

TEST_F(ValidateSSA, ForwardGroupDecorateGood) {
  char str[] = R"(
          OpCapability Shader
          OpMemoryModel Logical GLSL450
          OpDecorate %dgrp RowMajor
%dgrp   = OpDecorationGroup
          OpGroupDecorate %dgrp %mat33 %mat44
%intt   = OpTypeInt 32 1
%vec3   = OpTypeVector %intt 3
%vec4   = OpTypeVector %intt 4
%mat33  = OpTypeMatrix %vec3 3
%mat44  = OpTypeMatrix %vec4 4
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_F(ValidateSSA, ForwardGroupDecorateMissingGroupBad) {
  char str[] = R"(
           OpCapability Shader
           OpMemoryModel Logical GLSL450
           OpName %missing "missing"
           OpDecorate %dgrp RowMajor
%dgrp   =  OpDecorationGroup
           OpGroupDecorate %missing %mat33 %mat44 ; Target not defined
%intt   =  OpTypeInt 32 1
%vec3   =  OpTypeVector %intt 3
%vec4   =  OpTypeVector %intt 4
%mat33  =  OpTypeMatrix %vec3 3
%mat44  =  OpTypeMatrix %vec4 4
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("missing"));
}

TEST_F(ValidateSSA, ForwardGroupDecorateMissingTargetBad) {
  char str[] = R"(
           OpCapability Shader
           OpMemoryModel Logical GLSL450
           OpName %missing "missing"
           OpDecorate %dgrp RowMajor
%dgrp   =  OpDecorationGroup
           OpGroupDecorate %dgrp %missing %mat44 ; Target not defined
%intt   =  OpTypeInt 32 1
%vec3   =  OpTypeVector %intt 3
%vec4   =  OpTypeVector %intt 4
%mat33  =  OpTypeMatrix %vec3 3
%mat44  =  OpTypeMatrix %vec4 4
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("missing"));
}

TEST_F(ValidateSSA, ForwardGroupDecorateDecorationGroupDominateBad) {
  char str[] = R"(
           OpCapability Shader
           OpMemoryModel Logical GLSL450
           OpName %dgrp "group"
           OpDecorate %dgrp RowMajor
           OpGroupDecorate %dgrp %mat33 %mat44 ; Decoration group does not dominate usage
%dgrp   =  OpDecorationGroup
%intt   =  OpTypeInt 32 1
%vec3   =  OpTypeVector %intt 3
%vec4   =  OpTypeVector %intt 4
%mat33  =  OpTypeMatrix %vec3 3
%mat44  =  OpTypeMatrix %vec4 4
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("group"));
}

TEST_F(ValidateSSA, ForwardDecorateInvalidIdBad) {
  char str[] = R"(
           OpCapability Shader
           OpMemoryModel Logical GLSL450
           OpName %missing "missing"
           OpDecorate %missing Restrict        ; Missing target
%voidt  =  OpTypeVoid
%intt   =  OpTypeInt 32 1
%ptrt   =  OpTypePointer UniformConstant %intt
%var    =  OpVariable %ptrt UniformConstant
%2      =  OpTypeFunction %voidt
%3      =  OpFunction %voidt None %2
%4      =  OpLabel
           OpReturn
           OpFunctionEnd
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("missing"));
}

TEST_F(ValidateSSA, FunctionCallGood) {
  char str[] = R"(
         OpCapability Shader
         OpMemoryModel Logical GLSL450
%1    =  OpTypeVoid
%2    =  OpTypeInt 32 1
%3    =  OpTypeInt 32 0
%4    =  OpTypeFunction %1
%8    =  OpTypeFunction %1 %2 %3
%four =  OpConstant %2 4
%five =  OpConstant %3 5
%9    =  OpFunction %1 None %8
%10   =  OpFunctionParameter %2
%11   =  OpFunctionParameter %3
%12   =  OpLabel
         OpReturn
         OpFunctionEnd
%5    =  OpFunction %1 None %4
%6    =  OpLabel
%7    =  OpFunctionCall %1 %9 %four %five
         OpReturn
         OpFunctionEnd
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_F(ValidateSSA, ForwardFunctionCallGood) {
  char str[] = R"(
         OpCapability Shader
         OpMemoryModel Logical GLSL450
%1    =  OpTypeVoid
%2    =  OpTypeInt 32 1
%3    =  OpTypeInt 32 0
%four =  OpConstant %2 4
%five =  OpConstant %3 5
%8    =  OpTypeFunction %1 %2 %3
%4    =  OpTypeFunction %1
%5    =  OpFunction %1 None %4
%6    =  OpLabel
%7    =  OpFunctionCall %1 %9 %four %five
         OpReturn
         OpFunctionEnd
%9    =  OpFunction %1 None %8
%10   =  OpFunctionParameter %2
%11   =  OpFunctionParameter %3
%12   =  OpLabel
         OpReturn
         OpFunctionEnd
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_F(ValidateSSA, ForwardBranchConditionalGood) {
  char str[] = R"(
            OpCapability Shader
            OpMemoryModel Logical GLSL450
%voidt  =   OpTypeVoid
%boolt  =   OpTypeBool
%vfunct =   OpTypeFunction %voidt
%true   =   OpConstantTrue %boolt
%main   =   OpFunction %voidt None %vfunct
%mainl  =   OpLabel
            OpSelectionMerge %endl None
            OpBranchConditional %true %truel %falsel
%truel  =   OpLabel
            OpNop
            OpBranch %endl
%falsel =   OpLabel
            OpNop
            OpBranch %endl
%endl    =  OpLabel
            OpReturn
            OpFunctionEnd
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_F(ValidateSSA, ForwardBranchConditionalWithWeightsGood) {
  char str[] = R"(
           OpCapability Shader
           OpMemoryModel Logical GLSL450
%voidt  =  OpTypeVoid
%boolt  =  OpTypeBool
%vfunct =  OpTypeFunction %voidt
%true   =  OpConstantTrue %boolt
%main   =  OpFunction %voidt None %vfunct
%mainl  =  OpLabel
           OpSelectionMerge %endl None
           OpBranchConditional %true %truel %falsel 1 9
%truel  =  OpLabel
           OpNop
           OpBranch %endl
%falsel =  OpLabel
           OpNop
           OpBranch %endl
%endl   =  OpLabel
           OpReturn
           OpFunctionEnd
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_F(ValidateSSA, ForwardBranchConditionalNonDominantConditionBad) {
  char str[] = R"(
           OpCapability Shader
           OpMemoryModel Logical GLSL450
           OpName %tcpy "conditional"
%voidt  =  OpTypeVoid
%boolt  =  OpTypeBool
%vfunct =  OpTypeFunction %voidt
%true   =  OpConstantTrue %boolt
%main   =  OpFunction %voidt None %vfunct
%mainl  =  OpLabel
           OpSelectionMerge %endl None
           OpBranchConditional %tcpy %truel %falsel ;
%truel  =  OpLabel
           OpNop
           OpBranch %endl
%falsel =  OpLabel
           OpNop
           OpBranch %endl
%endl   =  OpLabel
%tcpy   =  OpCopyObject %boolt %true
           OpReturn
           OpFunctionEnd
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("conditional"));
}

TEST_F(ValidateSSA, ForwardBranchConditionalMissingTargetBad) {
  char str[] = R"(
           OpCapability Shader
           OpMemoryModel Logical GLSL450
           OpName %missing "missing"
%voidt  =  OpTypeVoid
%boolt  =  OpTypeBool
%vfunct =  OpTypeFunction %voidt
%true   =  OpConstantTrue %boolt
%main   =  OpFunction %voidt None %vfunct
%mainl  =  OpLabel
           OpSelectionMerge %endl None
           OpBranchConditional %true %missing %falsel
%truel  =  OpLabel
           OpNop
           OpBranch %endl
%falsel =  OpLabel
           OpNop
           OpBranch %endl
%endl   =  OpLabel
           OpReturn
           OpFunctionEnd
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("missing"));
}

const string kHeader = R"(
OpCapability Int8
OpCapability DeviceEnqueue
OpMemoryModel Logical OpenCL
)";

const string kBasicTypes = R"(
%voidt  =  OpTypeVoid
%boolt  =  OpTypeBool
%int8t  =  OpTypeInt 8 0
%intt   =  OpTypeInt 32 1
%uintt  =  OpTypeInt 32 0
%vfunct =  OpTypeFunction %voidt
%intptrt = OpTypePointer UniformConstant %intt
%zero      = OpConstant %intt 0
%one       = OpConstant %intt 1
%ten       = OpConstant %intt 10
%false     = OpConstantFalse %boolt
)";

const string kKernelTypesAndConstants = R"(
%queuet  = OpTypeQueue

%three   = OpConstant %uintt 3
%arr3t   = OpTypeArray %intt %three
%ndt     = OpTypeStruct %intt %arr3t %arr3t %arr3t

%eventt  = OpTypeEvent

%offset = OpConstant %intt 0
%local  = OpConstant %intt 1
%gl     = OpConstant %intt 1

%nevent = OpConstant %intt 0
%event  = OpConstantNull %eventt

%firstp = OpConstant %int8t 0
%psize  = OpConstant %intt 0
%palign = OpConstant %intt 32
%lsize  = OpConstant %intt 1
%flags  = OpConstant %intt 0 ; NoWait

%kfunct = OpTypeFunction %voidt %intptrt
)";

const string kKernelSetup = R"(
%dqueue = OpGetDefaultQueue %queuet
%ndval  = OpBuildNDRange %ndt %gl %local %offset
%revent = OpUndef %eventt

)";

const string kKernelDefinition = R"(
%kfunc  = OpFunction %voidt None %kfunct
%iparam = OpFunctionParameter %intptrt
%kfuncl = OpLabel
          OpNop
          OpReturn
          OpFunctionEnd
)";

TEST_F(ValidateSSA, EnqueueKernelGood) {
  string str = kHeader + kBasicTypes + kKernelTypesAndConstants +
               kKernelDefinition + R"(
                %main   = OpFunction %voidt None %vfunct
                %mainl  = OpLabel
                )" +
               kKernelSetup + R"(
                %err    = OpEnqueueKernel %uintt %dqueue %flags %ndval %nevent
                                        %event %revent %kfunc %firstp %psize
                                        %palign %lsize
                          OpReturn
                          OpFunctionEnd
                 )";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_F(ValidateSSA, ForwardEnqueueKernelGood) {
  string str = kHeader + kBasicTypes + kKernelTypesAndConstants + R"(
                %main   = OpFunction %voidt None %vfunct
                %mainl  = OpLabel
                )" +
               kKernelSetup + R"(
                %err    = OpEnqueueKernel %uintt %dqueue %flags %ndval %nevent
                                        %event %revent %kfunc %firstp %psize
                                        %palign %lsize
                         OpReturn
                         OpFunctionEnd
                 )" +
               kKernelDefinition;
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_F(ValidateSSA, EnqueueMissingFunctionBad) {
  string str = kHeader + "OpName %kfunc \"kfunc\"" + kBasicTypes +
               kKernelTypesAndConstants + R"(
                %main   = OpFunction %voidt None %vfunct
                %mainl  = OpLabel
                )" +
               kKernelSetup + R"(
                %err    = OpEnqueueKernel %uintt %dqueue %flags %ndval %nevent
                                        %event %revent %kfunc %firstp %psize
                                        %palign %lsize
                         OpReturn
                         OpFunctionEnd
                 )";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("kfunc"));
}

string forwardKernelNonDominantParameterBaseCode(string name = string()) {
  string op_name;
  if (name.empty()) {
    op_name = "";
  } else {
    op_name = "\nOpName %" + name + " \"" + name + "\"\n";
  }
  string out = kHeader + op_name + kBasicTypes + kKernelTypesAndConstants +
               kKernelDefinition +
               R"(
                %main   = OpFunction %voidt None %vfunct
                %mainl  = OpLabel
                )" +
               kKernelSetup;
  return out;
}

TEST_F(ValidateSSA, ForwardEnqueueKernelMissingParameter1Bad) {
  string str = forwardKernelNonDominantParameterBaseCode("missing") + R"(
                %err    = OpEnqueueKernel %missing %dqueue %flags %ndval
                                        %nevent %event %revent %kfunc %firstp
                                        %psize %palign %lsize
                          OpReturn
                          OpFunctionEnd
                )";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("missing"));
}

TEST_F(ValidateSSA, ForwardEnqueueKernelNonDominantParameter2Bad) {
  string str = forwardKernelNonDominantParameterBaseCode("dqueue2") + R"(
                %err     = OpEnqueueKernel %uintt %dqueue2 %flags %ndval
                                            %nevent %event %revent %kfunc
                                            %firstp %psize %palign %lsize
                %dqueue2 = OpGetDefaultQueue %queuet
                           OpReturn
                           OpFunctionEnd
                )";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("dqueue2"));
}

TEST_F(ValidateSSA, ForwardEnqueueKernelNonDominantParameter3Bad) {
  string str = forwardKernelNonDominantParameterBaseCode("ndval2") + R"(
                %err    = OpEnqueueKernel %uintt %dqueue %flags %ndval2
                                        %nevent %event %revent %kfunc %firstp
                                        %psize %palign %lsize
                %ndval2  = OpBuildNDRange %ndt %gl %local %offset
                          OpReturn
                          OpFunctionEnd
                )";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("ndval2"));
}

TEST_F(ValidateSSA, ForwardEnqueueKernelNonDominantParameter4Bad) {
  string str = forwardKernelNonDominantParameterBaseCode("nevent2") + R"(
              %err    = OpEnqueueKernel %uintt %dqueue %flags %ndval %nevent2
                                        %event %revent %kfunc %firstp %psize
                                        %palign %lsize
              %nevent2 = OpCopyObject %intt %nevent
                        OpReturn
                        OpFunctionEnd
              )";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("nevent2"));
}

TEST_F(ValidateSSA, ForwardEnqueueKernelNonDominantParameter5Bad) {
  string str = forwardKernelNonDominantParameterBaseCode("event2") + R"(
              %err     = OpEnqueueKernel %uintt %dqueue %flags %ndval %nevent
                                        %event2 %revent %kfunc %firstp %psize
                                        %palign %lsize
              %event2  = OpCopyObject %eventt %event
                         OpReturn
                         OpFunctionEnd
              )";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("event2"));
}

TEST_F(ValidateSSA, ForwardEnqueueKernelNonDominantParameter6Bad) {
  string str = forwardKernelNonDominantParameterBaseCode("revent2") + R"(
              %err     = OpEnqueueKernel %uintt %dqueue %flags %ndval %nevent
                                        %event %revent2 %kfunc %firstp %psize
                                        %palign %lsize
              %revent2 = OpCopyObject %eventt %revent
                         OpReturn
                         OpFunctionEnd
              )";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("revent2"));
}

TEST_F(ValidateSSA, ForwardEnqueueKernelNonDominantParameter8Bad) {
  string str = forwardKernelNonDominantParameterBaseCode("firstp2") + R"(
              %err     = OpEnqueueKernel %uintt %dqueue %flags %ndval %nevent
                                        %event %revent %kfunc %firstp2 %psize
                                        %palign %lsize
              %firstp2 = OpCopyObject %int8t %firstp
                         OpReturn
                         OpFunctionEnd
              )";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("firstp2"));
}

TEST_F(ValidateSSA, ForwardEnqueueKernelNonDominantParameter9Bad) {
  string str = forwardKernelNonDominantParameterBaseCode("psize2") + R"(
              %err    = OpEnqueueKernel %uintt %dqueue %flags %ndval %nevent
                                        %event %revent %kfunc %firstp %psize2
                                        %palign %lsize
              %psize2 = OpCopyObject %intt %psize
                        OpReturn
                        OpFunctionEnd
              )";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("psize2"));
}

TEST_F(ValidateSSA, ForwardEnqueueKernelNonDominantParameter10Bad) {
  string str = forwardKernelNonDominantParameterBaseCode("palign2") + R"(
              %err     = OpEnqueueKernel %uintt %dqueue %flags %ndval %nevent
                                        %event %revent %kfunc %firstp %psize
                                        %palign2 %lsize
              %palign2 = OpCopyObject %intt %palign
                        OpReturn
                        OpFunctionEnd
              )";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("palign2"));
}

TEST_F(ValidateSSA, ForwardEnqueueKernelNonDominantParameter11Bad) {
  string str = forwardKernelNonDominantParameterBaseCode("lsize2") + R"(
              %err     = OpEnqueueKernel %uintt %dqueue %flags %ndval %nevent
                                        %event %revent %kfunc %firstp %psize
                                        %palign %lsize2
              %lsize2  = OpCopyObject %intt %lsize
                         OpReturn
                         OpFunctionEnd
              )";

  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("lsize2"));
}

static const bool kWithNDrange = true;
static const bool kNoNDrange = false;
pair<string, bool> cases[] = {
    {"OpGetKernelNDrangeSubGroupCount", kWithNDrange},
    {"OpGetKernelNDrangeMaxSubGroupSize", kWithNDrange},
    {"OpGetKernelWorkGroupSize", kNoNDrange},
    {"OpGetKernelPreferredWorkGroupSizeMultiple", kNoNDrange}};

INSTANTIATE_TEST_CASE_P(KernelArgs, ValidateSSA, ::testing::ValuesIn(cases), );

static const string return_instructions = R"(
  OpReturn
  OpFunctionEnd
)";

TEST_P(ValidateSSA, GetKernelGood) {
  string instruction = GetParam().first;
  bool with_ndrange = GetParam().second;
  string ndrange_param = with_ndrange ? " %ndval " : " ";

  stringstream ss;
  // clang-format off
  ss << forwardKernelNonDominantParameterBaseCode() + " %numsg = "
     << instruction + " %uintt" + ndrange_param + "%kfunc %firstp %psize %palign"
     << return_instructions;
  // clang-format on

  CompileSuccessfully(ss.str());
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_P(ValidateSSA, ForwardGetKernelGood) {
  string instruction = GetParam().first;
  bool with_ndrange = GetParam().second;
  string ndrange_param = with_ndrange ? " %ndval " : " ";

  // clang-format off
  string str = kHeader + kBasicTypes + kKernelTypesAndConstants +
               R"(
            %main    = OpFunction %voidt None %vfunct
            %mainl   = OpLabel
                )"
            + kKernelSetup + " %numsg = "
            + instruction + " %uintt" + ndrange_param + "%kfunc %firstp %psize %palign"
            + return_instructions + kKernelDefinition;
  // clang-format on

  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_P(ValidateSSA, ForwardGetKernelMissingDefinitionBad) {
  string instruction = GetParam().first;
  bool with_ndrange = GetParam().second;
  string ndrange_param = with_ndrange ? " %ndval " : " ";

  stringstream ss;
  // clang-format off
  ss << forwardKernelNonDominantParameterBaseCode("missing") + " %numsg = "
     << instruction + " %uintt" + ndrange_param + "%missing %firstp %psize %palign"
     << return_instructions;
  // clang-format on

  CompileSuccessfully(ss.str());
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("missing"));
}

TEST_P(ValidateSSA, ForwardGetKernelNDrangeSubGroupCountMissingParameter1Bad) {
  string instruction = GetParam().first;
  bool with_ndrange = GetParam().second;
  string ndrange_param = with_ndrange ? " %ndval " : " ";

  stringstream ss;
  // clang-format off
  ss << forwardKernelNonDominantParameterBaseCode("missing") + " %numsg = "
     << instruction + " %missing" + ndrange_param + "%kfunc %firstp %psize %palign"
     << return_instructions;
  // clang-format on

  CompileSuccessfully(ss.str());
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("missing"));
}

TEST_P(ValidateSSA,
       ForwardGetKernelNDrangeSubGroupCountNonDominantParameter2Bad) {
  string instruction = GetParam().first;
  bool with_ndrange = GetParam().second;
  string ndrange_param = with_ndrange ? " %ndval2 " : " ";

  stringstream ss;
  // clang-format off
  ss << forwardKernelNonDominantParameterBaseCode("ndval2") + " %numsg = "
     << instruction + " %uintt" + ndrange_param + "%kfunc %firstp %psize %palign"
     << "\n %ndval2  = OpBuildNDRange %ndt %gl %local %offset"
     << return_instructions;
  // clang-format on

  if (GetParam().second) {
    CompileSuccessfully(ss.str());
    ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
    EXPECT_THAT(getDiagnosticString(), HasSubstr("ndval2"));
  }
}

TEST_P(ValidateSSA,
       ForwardGetKernelNDrangeSubGroupCountNonDominantParameter4Bad) {
  string instruction = GetParam().first;
  bool with_ndrange = GetParam().second;
  string ndrange_param = with_ndrange ? " %ndval " : " ";

  stringstream ss;
  // clang-format off
  ss << forwardKernelNonDominantParameterBaseCode("firstp2") + " %numsg = "
     << instruction + " %uintt" + ndrange_param + "%kfunc %firstp2 %psize %palign"
     << "\n %firstp2 = OpCopyObject %int8t %firstp"
     << return_instructions;
  // clang-format on

  CompileSuccessfully(ss.str());
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("firstp2"));
}

TEST_P(ValidateSSA,
       ForwardGetKernelNDrangeSubGroupCountNonDominantParameter5Bad) {
  string instruction = GetParam().first;
  bool with_ndrange = GetParam().second;
  string ndrange_param = with_ndrange ? " %ndval " : " ";

  stringstream ss;
  // clang-format off
  ss << forwardKernelNonDominantParameterBaseCode("psize2") + " %numsg = "
     << instruction + " %uintt" + ndrange_param + "%kfunc %firstp %psize2 %palign"
     << "\n %psize2  = OpCopyObject %intt %psize"
     << return_instructions;
  // clang-format on

  CompileSuccessfully(ss.str());
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("psize2"));
}

TEST_P(ValidateSSA,
       ForwardGetKernelNDrangeSubGroupCountNonDominantParameter6Bad) {
  string instruction = GetParam().first;
  bool with_ndrange = GetParam().second;
  string ndrange_param = with_ndrange ? " %ndval " : " ";

  stringstream ss;
  // clang-format off
  ss << forwardKernelNonDominantParameterBaseCode("palign2") + " %numsg = "
     << instruction + " %uintt" + ndrange_param + "%kfunc %firstp %psize %palign2"
     << "\n %palign2 = OpCopyObject %intt %palign"
     << return_instructions;
  // clang-format on

  if (GetParam().second) {
    CompileSuccessfully(ss.str());
    ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
    EXPECT_THAT(getDiagnosticString(), HasSubstr("palign2"));
  }
}

TEST_F(ValidateSSA, PhiGood) {
  string str = kHeader + kBasicTypes +
               R"(
%func      = OpFunction %voidt None %vfunct
%preheader = OpLabel
%init      = OpCopyObject %intt %zero
             OpBranch %loop
%loop      = OpLabel
%i         = OpPhi %intt %init %preheader %loopi %loop
%loopi     = OpIAdd %intt %i %one
             OpNop
%cond      = OpSLessThan %boolt %i %ten
             OpLoopMerge %endl %loop None
             OpBranchConditional %cond %loop %endl
%endl      = OpLabel
             OpReturn
             OpFunctionEnd
)";

  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_F(ValidateSSA, PhiMissingTypeBad) {
  string str = kHeader + "OpName %missing \"missing\"" + kBasicTypes +
               R"(
%func      = OpFunction %voidt None %vfunct
%preheader = OpLabel
%init      = OpCopyObject %intt %zero
             OpBranch %loop
%loop      = OpLabel
%i         = OpPhi %missing %init %preheader %loopi %loop
%loopi     = OpIAdd %intt %i %one
             OpNop
%cond      = OpSLessThan %boolt %i %ten
             OpLoopMerge %endl %loop None
             OpBranchConditional %cond %loop %endl
%endl      = OpLabel
             OpReturn
             OpFunctionEnd
)";

  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("missing"));
}

TEST_F(ValidateSSA, PhiMissingIdBad) {
  string str = kHeader + "OpName %missing \"missing\"" + kBasicTypes +
               R"(
%func      = OpFunction %voidt None %vfunct
%preheader = OpLabel
%init      = OpCopyObject %intt %zero
             OpBranch %loop
%loop      = OpLabel
%i         = OpPhi %intt %missing %preheader %loopi %loop
%loopi     = OpIAdd %intt %i %one
             OpNop
%cond      = OpSLessThan %boolt %i %ten
             OpLoopMerge %endl %loop None
             OpBranchConditional %cond %loop %endl
%endl      = OpLabel
             OpReturn
             OpFunctionEnd
)";

  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("missing"));
}

TEST_F(ValidateSSA, PhiMissingLabelBad) {
  string str = kHeader + "OpName %missing \"missing\"" + kBasicTypes +
               R"(
%func      = OpFunction %voidt None %vfunct
%preheader = OpLabel
%init      = OpCopyObject %intt %zero
             OpBranch %loop
%loop      = OpLabel
%i         = OpPhi %intt %init %missing %loopi %loop
%loopi     = OpIAdd %intt %i %one
             OpNop
%cond      = OpSLessThan %boolt %i %ten
             OpLoopMerge %endl %loop None
             OpBranchConditional %cond %loop %endl
%endl      = OpLabel
             OpReturn
             OpFunctionEnd
)";

  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(), HasSubstr("missing"));
}

TEST_F(ValidateSSA, IdDominatesItsUseGood) {
  string str = kHeader + kBasicTypes +
               R"(
%func      = OpFunction %voidt None %vfunct
%entry     = OpLabel
%cond      = OpSLessThan %intt %one %ten
%eleven    = OpIAdd %intt %one %ten
             OpSelectionMerge %merge None
             OpBranchConditional %cond %t %f
%t         = OpLabel
%twelve    = OpIAdd %intt %eleven %one
             OpBranch %merge
%f         = OpLabel
%twentytwo = OpIAdd %intt %eleven %ten
             OpBranch %merge
%merge     = OpLabel
             OpReturn
             OpFunctionEnd
)";

  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_F(ValidateSSA, IdDoesNotDominateItsUseBad) {
  string str = kHeader +
               "OpName %eleven \"eleven\"\n"
               "OpName %true_block \"true_block\"\n"
               "OpName %false_block \"false_block\"" +
               kBasicTypes +
               R"(
%func        = OpFunction %voidt None %vfunct
%entry       = OpLabel
%cond        = OpSLessThan %intt %one %ten
               OpSelectionMerge %merge None
               OpBranchConditional %cond %true_block %false_block
%true_block  = OpLabel
%eleven      = OpIAdd %intt %one %ten
%twelve      = OpIAdd %intt %eleven %one
               OpBranch %merge
%false_block = OpLabel
%twentytwo   = OpIAdd %intt %eleven %ten
               OpBranch %merge
%merge       = OpLabel
               OpReturn
               OpFunctionEnd
)";
  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(
      getDiagnosticString(),
      MatchesRegex("ID .\\[eleven\\] defined in block .\\[true_block\\] does "
                   "not dominate its use in block .\\[false_block\\]"));
}

TEST_F(ValidateSSA, PhiUseDoesntDominateDefinitionGood) {
  string str = kHeader + kBasicTypes +
               R"(
%func        = OpFunction %voidt None %vfunct
%entry       = OpLabel
%var_one     = OpVariable %intptrt Function %one
%one_val     = OpLoad %intt %var_one
               OpBranch %loop
%loop        = OpLabel
%i           = OpPhi %intt %one_val %entry %inew %cont
%cond        = OpSLessThan %intt %one %ten
               OpLoopMerge %merge %cont None
               OpBranchConditional %cond %body %merge
%body        = OpLabel
               OpBranch %cont
%cont        = OpLabel
%inew        = OpIAdd %intt %i %one
               OpBranch %loop
%merge       = OpLabel
               OpReturn
               OpFunctionEnd
)";

  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_F(ValidateSSA,
       PhiUseDoesntDominateUseOfPhiOperandUsedBeforeDefinitionBad) {
  string str = kHeader + "OpName %inew \"inew\"" + kBasicTypes +
               R"(
%func        = OpFunction %voidt None %vfunct
%entry       = OpLabel
%var_one     = OpVariable %intptrt Function %one
%one_val     = OpLoad %intt %var_one
               OpBranch %loop
%loop        = OpLabel
%i           = OpPhi %intt %one_val %entry %inew %cont
%bad         = OpIAdd %intt %inew %one
%cond        = OpSLessThan %intt %one %ten
               OpLoopMerge %merge %cont None
               OpBranchConditional %cond %body %merge
%body        = OpLabel
               OpBranch %cont
%cont        = OpLabel
%inew        = OpIAdd %intt %i %one
               OpBranch %loop
%merge       = OpLabel
               OpReturn
               OpFunctionEnd
)";

  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(getDiagnosticString(),
              MatchesRegex("ID .\\[inew\\] has not been defined"));
}

TEST_F(ValidateSSA, PhiUseMayComeFromNonDominatingBlockGood) {
  string str = kHeader + "OpName %if_true \"if_true\"\n" +
               "OpName %exit \"exit\"\n" + "OpName %copy \"copy\"\n" +
               kBasicTypes +
               R"(
%func        = OpFunction %voidt None %vfunct
%entry       = OpLabel
               OpBranchConditional %false %if_true %exit

%if_true     = OpLabel
%copy        = OpCopyObject %boolt %false
               OpBranch %exit

; The use of %copy here is ok, even though it was defined
; in a block that does not dominate %exit.  That's the point
; of an OpPhi.
%exit        = OpLabel
%value       = OpPhi %boolt %false %entry %copy %if_true
               OpReturn
               OpFunctionEnd
)";

  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions()) << getDiagnosticString();
}

TEST_F(ValidateSSA, PhiVariableDefNotDominatedByParentBlockBad) {
  string str = kHeader + "OpName %if_true \"if_true\"\n" +
               "OpName %if_false \"if_false\"\n" + "OpName %exit \"exit\"\n" +
               "OpName %value \"phi\"\n" + "OpName %true_copy \"true_copy\"\n" +
               "OpName %false_copy \"false_copy\"\n" + kBasicTypes +
               R"(
%func        = OpFunction %voidt None %vfunct
%entry       = OpLabel
               OpBranchConditional %false %if_true %if_false

%if_true     = OpLabel
%true_copy   = OpCopyObject %boolt %false
               OpBranch %exit

%if_false    = OpLabel
%false_copy  = OpCopyObject %boolt %false
               OpBranch %exit

; The (variable,Id) pairs are swapped.
%exit        = OpLabel
%value       = OpPhi %boolt %true_copy %if_false %false_copy %if_true
               OpReturn
               OpFunctionEnd
)";

  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(
      getDiagnosticString(),
      MatchesRegex("In OpPhi instruction .\\[phi\\], ID .\\[true_copy\\] "
                   "definition does not dominate its parent .\\[if_false\\]"));
}

TEST_F(ValidateSSA,
       PhiVariableDefDominatesButNotDefinedInParentBlock) {
  string str = kHeader + "OpName %if_true \"if_true\"\n" +
               kBasicTypes +
               R"(
%func        = OpFunction %voidt None %vfunct
%entry       = OpLabel
               OpBranchConditional %false %if_true %if_false

%if_true     = OpLabel
%true_copy   = OpCopyObject %boolt %false
               OpBranch %if_tnext
%if_tnext    = OpLabel
               OpBranch %exit

%if_false    = OpLabel
%false_copy  = OpCopyObject %boolt %false
               OpBranch %if_fnext
%if_fnext    = OpLabel
               OpBranch %exit

%exit        = OpLabel
%value       = OpPhi %boolt %true_copy %if_tnext %false_copy %if_fnext
               OpReturn
               OpFunctionEnd
)";

  CompileSuccessfully(str);
  ASSERT_EQ(SPV_SUCCESS, ValidateInstructions());
}

TEST_F(ValidateSSA,
       DominanceCheckIgnoresUsesInUnreachableBlocksDefInBlockGood) {
  string str = kHeader + kBasicTypes +
               R"(
%func        = OpFunction %voidt None %vfunct
%entry       = OpLabel
%def         = OpCopyObject %boolt %false
               OpReturn

%unreach     = OpLabel
%use         = OpCopyObject %boolt %def
               OpReturn
               OpFunctionEnd
)";

  CompileSuccessfully(str);
  EXPECT_EQ(SPV_SUCCESS, ValidateInstructions()) << getDiagnosticString();
}

TEST_F(ValidateSSA,
       DominanceCheckIgnoresUsesInUnreachableBlocksDefIsParamGood) {
  string str = kHeader + kBasicTypes +
               R"(
%void_fn_int = OpTypeFunction %voidt %intt
%func        = OpFunction %voidt None %void_fn_int
%int_param   = OpFunctionParameter %intt
%entry       = OpLabel
               OpReturn

%unreach     = OpLabel
%use         = OpCopyObject %intt %int_param
               OpReturn
               OpFunctionEnd
)";

  CompileSuccessfully(str);
  EXPECT_EQ(SPV_SUCCESS, ValidateInstructions()) << getDiagnosticString();
}

TEST_F(ValidateSSA, UseFunctionParameterFromOtherFunctionBad) {
  string str = kHeader +
               "OpName %first \"first\"\n"
               "OpName %func \"func\"\n" +
               "OpName %func2 \"func2\"\n" + kBasicTypes +
               R"(
%viifunct  = OpTypeFunction %voidt %intt %intt
%func      = OpFunction %voidt None %viifunct
%first     = OpFunctionParameter %intt
%second    = OpFunctionParameter %intt
             OpFunctionEnd
%func2     = OpFunction %voidt None %viifunct
%first2    = OpFunctionParameter %intt
%second2   = OpFunctionParameter %intt
%entry2    = OpLabel
%baduse    = OpIAdd %intt %first %first2
             OpReturn
             OpFunctionEnd
)";

  CompileSuccessfully(str);
  ASSERT_EQ(SPV_ERROR_INVALID_ID, ValidateInstructions());
  EXPECT_THAT(
      getDiagnosticString(),
      MatchesRegex("ID .\\[first\\] used in function .\\[func2\\] is used "
                   "outside of it's defining function .\\[func\\]"));
}
// TODO(umar): OpGroupMemberDecorate
}  // namespace

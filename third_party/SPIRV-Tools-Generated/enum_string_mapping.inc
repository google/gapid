std::string ExtensionToString(Extension extension) {
  switch (extension) {
    case Extension::kSPV_AMD_gcn_shader:
      return "SPV_AMD_gcn_shader";
    case Extension::kSPV_KHR_16bit_storage:
      return "SPV_KHR_16bit_storage";
    case Extension::kSPV_KHR_device_group:
      return "SPV_KHR_device_group";
    case Extension::kSPV_KHR_multiview:
      return "SPV_KHR_multiview";
    case Extension::kSPV_KHR_shader_ballot:
      return "SPV_KHR_shader_ballot";
    case Extension::kSPV_KHR_shader_draw_parameters:
      return "SPV_KHR_shader_draw_parameters";
    case Extension::kSPV_KHR_subgroup_vote:
      return "SPV_KHR_subgroup_vote";
    case Extension::kSPV_NVX_multiview_per_view_attributes:
      return "SPV_NVX_multiview_per_view_attributes";
    case Extension::kSPV_NV_geometry_shader_passthrough:
      return "SPV_NV_geometry_shader_passthrough";
    case Extension::kSPV_NV_sample_mask_override_coverage:
      return "SPV_NV_sample_mask_override_coverage";
    case Extension::kSPV_NV_stereo_view_rendering:
      return "SPV_NV_stereo_view_rendering";
    case Extension::kSPV_NV_viewport_array2:
      return "SPV_NV_viewport_array2";
  };

  return "";
}

bool GetExtensionFromString(const std::string& str, Extension* extension) {
 static const std::unordered_map<std::string, Extension> mapping =
{
  {"SPV_AMD_gcn_shader",
   Extension::kSPV_AMD_gcn_shader},
  {"SPV_KHR_16bit_storage",
   Extension::kSPV_KHR_16bit_storage},
  {"SPV_KHR_device_group",
   Extension::kSPV_KHR_device_group},
  {"SPV_KHR_multiview",
   Extension::kSPV_KHR_multiview},
  {"SPV_KHR_shader_ballot",
   Extension::kSPV_KHR_shader_ballot},
  {"SPV_KHR_shader_draw_parameters",
   Extension::kSPV_KHR_shader_draw_parameters},
  {"SPV_KHR_subgroup_vote",
   Extension::kSPV_KHR_subgroup_vote},
  {"SPV_NVX_multiview_per_view_attributes",
   Extension::kSPV_NVX_multiview_per_view_attributes},
  {"SPV_NV_geometry_shader_passthrough",
   Extension::kSPV_NV_geometry_shader_passthrough},
  {"SPV_NV_sample_mask_override_coverage",
   Extension::kSPV_NV_sample_mask_override_coverage},
  {"SPV_NV_stereo_view_rendering",
   Extension::kSPV_NV_stereo_view_rendering},
  {"SPV_NV_viewport_array2",
   Extension::kSPV_NV_viewport_array2}
};

  const auto it = mapping.find(str);

  if (it == mapping.end()) return false;

  *extension = it->second;
  return true;
}

std::string CapabilityToString(SpvCapability capability) {
  switch (capability) {
    case SpvCapabilityMatrix:
      return "Matrix";
    case SpvCapabilityShader:
      return "Shader";
    case SpvCapabilityGeometry:
      return "Geometry";
    case SpvCapabilityTessellation:
      return "Tessellation";
    case SpvCapabilityAddresses:
      return "Addresses";
    case SpvCapabilityLinkage:
      return "Linkage";
    case SpvCapabilityKernel:
      return "Kernel";
    case SpvCapabilityVector16:
      return "Vector16";
    case SpvCapabilityFloat16Buffer:
      return "Float16Buffer";
    case SpvCapabilityFloat16:
      return "Float16";
    case SpvCapabilityFloat64:
      return "Float64";
    case SpvCapabilityInt64:
      return "Int64";
    case SpvCapabilityInt64Atomics:
      return "Int64Atomics";
    case SpvCapabilityImageBasic:
      return "ImageBasic";
    case SpvCapabilityImageReadWrite:
      return "ImageReadWrite";
    case SpvCapabilityImageMipmap:
      return "ImageMipmap";
    case SpvCapabilityPipes:
      return "Pipes";
    case SpvCapabilityGroups:
      return "Groups";
    case SpvCapabilityDeviceEnqueue:
      return "DeviceEnqueue";
    case SpvCapabilityLiteralSampler:
      return "LiteralSampler";
    case SpvCapabilityAtomicStorage:
      return "AtomicStorage";
    case SpvCapabilityInt16:
      return "Int16";
    case SpvCapabilityTessellationPointSize:
      return "TessellationPointSize";
    case SpvCapabilityGeometryPointSize:
      return "GeometryPointSize";
    case SpvCapabilityImageGatherExtended:
      return "ImageGatherExtended";
    case SpvCapabilityStorageImageMultisample:
      return "StorageImageMultisample";
    case SpvCapabilityUniformBufferArrayDynamicIndexing:
      return "UniformBufferArrayDynamicIndexing";
    case SpvCapabilitySampledImageArrayDynamicIndexing:
      return "SampledImageArrayDynamicIndexing";
    case SpvCapabilityStorageBufferArrayDynamicIndexing:
      return "StorageBufferArrayDynamicIndexing";
    case SpvCapabilityStorageImageArrayDynamicIndexing:
      return "StorageImageArrayDynamicIndexing";
    case SpvCapabilityClipDistance:
      return "ClipDistance";
    case SpvCapabilityCullDistance:
      return "CullDistance";
    case SpvCapabilityImageCubeArray:
      return "ImageCubeArray";
    case SpvCapabilitySampleRateShading:
      return "SampleRateShading";
    case SpvCapabilityImageRect:
      return "ImageRect";
    case SpvCapabilitySampledRect:
      return "SampledRect";
    case SpvCapabilityGenericPointer:
      return "GenericPointer";
    case SpvCapabilityInt8:
      return "Int8";
    case SpvCapabilityInputAttachment:
      return "InputAttachment";
    case SpvCapabilitySparseResidency:
      return "SparseResidency";
    case SpvCapabilityMinLod:
      return "MinLod";
    case SpvCapabilitySampled1D:
      return "Sampled1D";
    case SpvCapabilityImage1D:
      return "Image1D";
    case SpvCapabilitySampledCubeArray:
      return "SampledCubeArray";
    case SpvCapabilitySampledBuffer:
      return "SampledBuffer";
    case SpvCapabilityImageBuffer:
      return "ImageBuffer";
    case SpvCapabilityImageMSArray:
      return "ImageMSArray";
    case SpvCapabilityStorageImageExtendedFormats:
      return "StorageImageExtendedFormats";
    case SpvCapabilityImageQuery:
      return "ImageQuery";
    case SpvCapabilityDerivativeControl:
      return "DerivativeControl";
    case SpvCapabilityInterpolationFunction:
      return "InterpolationFunction";
    case SpvCapabilityTransformFeedback:
      return "TransformFeedback";
    case SpvCapabilityGeometryStreams:
      return "GeometryStreams";
    case SpvCapabilityStorageImageReadWithoutFormat:
      return "StorageImageReadWithoutFormat";
    case SpvCapabilityStorageImageWriteWithoutFormat:
      return "StorageImageWriteWithoutFormat";
    case SpvCapabilityMultiViewport:
      return "MultiViewport";
    case SpvCapabilitySubgroupDispatch:
      return "SubgroupDispatch";
    case SpvCapabilityNamedBarrier:
      return "NamedBarrier";
    case SpvCapabilityPipeStorage:
      return "PipeStorage";
    case SpvCapabilitySubgroupBallotKHR:
      return "SubgroupBallotKHR";
    case SpvCapabilityDrawParameters:
      return "DrawParameters";
    case SpvCapabilitySubgroupVoteKHR:
      return "SubgroupVoteKHR";
    case SpvCapabilityStorageUniformBufferBlock16:
      return "StorageUniformBufferBlock16";
    case SpvCapabilityStorageUniform16:
      return "StorageUniform16";
    case SpvCapabilityStoragePushConstant16:
      return "StoragePushConstant16";
    case SpvCapabilityStorageInputOutput16:
      return "StorageInputOutput16";
    case SpvCapabilityDeviceGroup:
      return "DeviceGroup";
    case SpvCapabilityMultiView:
      return "MultiView";
    case SpvCapabilitySampleMaskOverrideCoverageNV:
      return "SampleMaskOverrideCoverageNV";
    case SpvCapabilityGeometryShaderPassthroughNV:
      return "GeometryShaderPassthroughNV";
    case SpvCapabilityShaderViewportIndexLayerNV:
      return "ShaderViewportIndexLayerNV";
    case SpvCapabilityShaderViewportMaskNV:
      return "ShaderViewportMaskNV";
    case SpvCapabilityShaderStereoViewNV:
      return "ShaderStereoViewNV";
    case SpvCapabilityPerViewAttributesNV:
      return "PerViewAttributesNV";
    case SpvCapabilityMax:
      assert(0 && "Attempting to convert SpvCapabilityMax to string");
      return "";
  };

  return "";
}

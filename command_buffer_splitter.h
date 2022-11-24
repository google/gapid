#pragma once

#include <vulkan/vulkan.h>

#include <memory>
#include <shared_mutex>
#include <unordered_map>
#include <unordered_set>

#include "command_buffer_recorder.h"
#include "creation_data_tracker.h"
#include "creation_tracker.h"
#include "struct_clone.h"
#include "transform.h"

namespace gapid2 {

enum class patch_type : uint32_t {
  none = 0,
  load = 1 << 0,
  store = 1 << 1,
  final_layout = 1 << 2
};

inline patch_type operator|(patch_type lhs, patch_type rhs) {
  using T = std::underlying_type_t<patch_type>;
  return static_cast<patch_type>(static_cast<T>(lhs) | static_cast<T>(rhs));
}

inline patch_type& operator|=(patch_type& lhs, patch_type rhs) {
  lhs = lhs | rhs;
  return lhs;
}

inline patch_type operator&(patch_type lhs, patch_type rhs) {
  using T = std::underlying_type_t<patch_type>;
  return static_cast<patch_type>(static_cast<T>(lhs) & static_cast<T>(rhs));
}

inline patch_type& operator&=(patch_type& lhs, patch_type rhs) {
  lhs = lhs & rhs;
  return lhs;
}

class command_buffer_splitter : public transform_base {
  using super = transform_base;

 public:
  void SplitCommandBuffer(VkCommandBuffer cb, transform_base* next, uint64_t* indices, uint32_t num_indices) {
    commands_to_split_ = std::vector<uint64_t>(indices, indices + num_indices);
    recorder->RerecordCommandBuffer(cb, next, [this, cb, next](uint64_t c) {
      if (std::find(commands_to_split_.begin(), commands_to_split_.end(), c) != commands_to_split_.end()) {
        if (on_command_buffer_split_) {
          if (current_render_pass_) {
            output_message(message_type::debug, std::format("Temporarily leaving renderpass {}",
                                                            reinterpret_cast<uintptr_t>(current_render_pass_)));
            super::vkCmdEndRenderPass(cb);
          }
          on_command_buffer_split_(cb);
          if (current_render_pass_) {
            VkRenderPassBeginInfo begin_info = original_begin_info;
            begin_info.renderPass =
                split_renderpasses[current_render_pass_].subpasses[current_subpass_].post_split_render_pass;

            output_message(message_type::debug, std::format("Re-entering renderpass {}",
                                                            reinterpret_cast<uintptr_t>(current_render_pass_)));
            super::vkCmdBeginRenderPass(cb, &begin_info, VK_SUBPASS_CONTENTS_INLINE);
          }
        }
      }
    });
    commands_to_split_.clear();
    current_render_pass_ = VK_NULL_HANDLE;
    current_subpass_ = 0;
  }

  VkPipeline rewrite_pipeline(VkPipeline pipeline, VkRenderPass pass) {
    if (rewritten_pipelines.contains(pipeline)) {
      return rewritten_pipelines[pipeline];
    }

    auto pl = state_block_->get(pipeline);
    if (!pl->get_graphics_create_info()) {
      return pipeline;
    }
    auto ci = pl->get_graphics_create_info();
    if (ci->subpass == 0) {
      return pipeline;
    }

    auto new_ci = *ci;
    new_ci.subpass = 0;
    new_ci.renderPass = pass;
    VkPipeline pipe;
    VkResult res = vkCreateGraphicsPipelines(pl->device, VK_NULL_HANDLE, 1, &new_ci, nullptr, &pipe);
    GAPID2_ASSERT(res == VK_SUCCESS, "Could not actually recreate this pipeline, thats wrong");

    rewritten_pipelines[pipeline] = pipe;
    return pipe;
  }

  struct subpasses {
    VkRenderPass pre_split_render_pass = VK_NULL_HANDLE;
    VkRenderPass post_split_render_pass = VK_NULL_HANDLE;
    VkRenderPass end_render_pass = VK_NULL_HANDLE;
  };

  struct renderpasses {
    std::vector<subpasses> subpasses;
  };

  std::unordered_map<VkRenderPass, renderpasses> split_renderpasses;

  renderpasses* split_renderpass(VkRenderPass render_pass) {
    output_message(message_type::debug, std::format("Splitting renderpass {}",
                                                    reinterpret_cast<uintptr_t>(render_pass)));
    if (split_renderpasses.contains(render_pass)) {
      return &split_renderpasses[render_pass];
    }
    std::vector<subpasses> new_subpasses;
    auto rp = state_block_->get(render_pass);

    auto ci = rp->get_create_info();
    std::vector<VkImageLayout> current_layouts;
    current_layouts.resize(ci->attachmentCount);
    for (size_t i = 0; i < ci->attachmentCount; ++i) {
      current_layouts[i] = ci->pAttachments[i].initialLayout;
    }

    auto patch_final_layout = [&current_layouts](
                                  std::vector<VkAttachmentDescription>& descriptions,
                                  std::vector<VkAttachmentReference>& references) {
      std::vector<VkAttachmentReference> new_references;
      for (uint32_t i = 0; i < references.size(); ++i) {
        auto ia = references[i];
        if (ia.attachment != VK_ATTACHMENT_UNUSED) {
          current_layouts[ia.attachment] = ia.layout;
          descriptions[ia.attachment].finalLayout = ia.layout;
        }
      }
    };

    auto patch_all_descriptions = [&current_layouts](std::vector<VkAttachmentDescription>& descriptions, patch_type patch) {
      GAPID2_ASSERT(descriptions.size() == current_layouts.size(), "We expect the attachment descriptions to match");
      for (size_t i = 0; i < descriptions.size(); ++i) {
        descriptions[i].initialLayout = current_layouts[i];
        if ((patch & patch_type::final_layout) == patch_type::final_layout) {
          descriptions[i].finalLayout = current_layouts[i];
        }
        if ((patch & patch_type::load) == patch_type::load) {
          descriptions[i].loadOp = VK_ATTACHMENT_LOAD_OP_LOAD;
          descriptions[i].stencilLoadOp = VK_ATTACHMENT_LOAD_OP_LOAD;
        }
        if ((patch & patch_type::store) == patch_type::store) {
          descriptions[i].storeOp = VK_ATTACHMENT_STORE_OP_STORE;
          descriptions[i].stencilStoreOp = VK_ATTACHMENT_STORE_OP_STORE;
        }
      }
    };

    for (size_t i = 0; i < ci->subpassCount; ++i) {
      VkRenderPass subpass_handles[3] = {VK_NULL_HANDLE, VK_NULL_HANDLE, VK_NULL_HANDLE};
      bool is_last_subpass = (i == (ci->subpassCount - 1));
      bool is_first_subpass = (i == 0);

      {
        auto nci = *ci;
        auto spd = ci->pSubpasses[i];
        patch_type patch = patch_type::none;
        if (!is_first_subpass) {
          patch |= patch_type::load;
        }
        std::vector<VkAttachmentDescription> ads(ci->pAttachments, ci->pAttachments + ci->attachmentCount);
        std::vector<VkAttachmentReference> ias(spd.pInputAttachments, spd.pInputAttachments + spd.inputAttachmentCount);
        std::vector<VkAttachmentReference> cas(spd.pColorAttachments, spd.pColorAttachments + spd.colorAttachmentCount);

        patch_all_descriptions(ads, patch);
        patch_final_layout(ads, ias);
        patch_final_layout(ads, cas);
        spd.pResolveAttachments = nullptr;
        if (spd.pDepthStencilAttachment) {
          auto ia = spd.pDepthStencilAttachment;
          if (ia->attachment != VK_ATTACHMENT_UNUSED) {
            current_layouts[ia->attachment] = ia->layout;
            ads[ia->attachment].finalLayout = ia->layout;
          }
        }
        spd.pPreserveAttachments = nullptr;
        spd.preserveAttachmentCount = 0;

        nci.subpassCount = 1;
        nci.pSubpasses = &spd;
        nci.dependencyCount = 0;
        nci.pDependencies = nullptr;
        nci.pAttachments = ads.data();
        spd.pInputAttachments = ias.data();
        spd.pColorAttachments = cas.data();

        VkResult res = vkCreateRenderPass(rp->device, &nci, nullptr, &subpass_handles[0]);
        GAPID2_ASSERT(res == VK_SUCCESS, "Expected success on the render pass create");
      }

      {
        auto nci = *ci;
        auto spd = ci->pSubpasses[i];
        patch_type patch = patch_type::load | patch_type::store | patch_type::final_layout;

        std::vector<VkAttachmentDescription> ads(ci->pAttachments, ci->pAttachments + ci->attachmentCount);
        std::vector<VkAttachmentReference> ias(spd.pInputAttachments, spd.pInputAttachments + spd.inputAttachmentCount);
        std::vector<VkAttachmentReference> cas(spd.pColorAttachments, spd.pColorAttachments + spd.colorAttachmentCount);

        patch_all_descriptions(ads, patch);
        patch_final_layout(ads, ias);
        patch_final_layout(ads, cas);
        spd.pResolveAttachments = nullptr;

        if (spd.pDepthStencilAttachment) {
          auto ia = spd.pDepthStencilAttachment;
          if (ia->attachment != VK_ATTACHMENT_UNUSED) {
            current_layouts[ia->attachment] = ia->layout;
            ads[ia->attachment].finalLayout = ia->layout;
          }
        }
        spd.pPreserveAttachments = nullptr;
        spd.preserveAttachmentCount = 0;

        nci.subpassCount = 1;
        nci.pSubpasses = &spd;
        nci.dependencyCount = 0;
        nci.pDependencies = nullptr;
        nci.pAttachments = ads.data();
        spd.pInputAttachments = ias.data();
        spd.pColorAttachments = cas.data();

        VkResult res = vkCreateRenderPass(rp->device, &nci, nullptr, &subpass_handles[1]);
        GAPID2_ASSERT(res == VK_SUCCESS, "Expected success on the render pass create");
      }

      {
        auto nci = *ci;
        auto spd = ci->pSubpasses[i];
        patch_type patch = patch_type::load;
        if (!is_last_subpass) {
          patch |= patch_type::store | patch_type::final_layout;
        }

        std::vector<VkAttachmentDescription> ads(ci->pAttachments, ci->pAttachments + ci->attachmentCount);
        std::vector<VkAttachmentReference> ias(spd.pInputAttachments, spd.pInputAttachments + spd.inputAttachmentCount);
        std::vector<VkAttachmentReference> cas(spd.pColorAttachments, spd.pColorAttachments + spd.colorAttachmentCount);

        patch_all_descriptions(ads, patch);

        if (spd.pDepthStencilAttachment) {
          auto ia = spd.pDepthStencilAttachment;
          if (ia->attachment != VK_ATTACHMENT_UNUSED) {
            current_layouts[ia->attachment] = ia->layout;
            ads[ia->attachment].finalLayout = ia->layout;
          }
        }
        spd.pPreserveAttachments = nullptr;
        spd.preserveAttachmentCount = 0;

        nci.subpassCount = 1;
        nci.pSubpasses = &spd;
        nci.dependencyCount = 0;
        nci.pDependencies = nullptr;
        nci.pAttachments = ads.data();
        spd.pInputAttachments = ias.data();
        spd.pColorAttachments = cas.data();

        VkResult res = vkCreateRenderPass(rp->device, &nci, nullptr, &subpass_handles[2]);
        GAPID2_ASSERT(res == VK_SUCCESS, "Expected success on the render pass create");
      }
      new_subpasses.push_back(subpasses{
          subpass_handles[0],
          subpass_handles[1],
          subpass_handles[2]});
    }
    split_renderpasses[render_pass] = renderpasses{std::move(new_subpasses)};
    return &split_renderpasses[render_pass];
  }

  void vkCmdBeginRenderPass(VkCommandBuffer commandBuffer, const VkRenderPassBeginInfo* pRenderPassBegin, VkSubpassContents contents) {
    if (commands_to_split_.empty()) {
      return super::vkCmdBeginRenderPass(commandBuffer, pRenderPassBegin, contents);
    }
    stage = current_stage::eFirstStage;
    current_render_pass_ = pRenderPassBegin->renderPass;
    auto new_rp = split_renderpass(pRenderPassBegin->renderPass)->subpasses[0].pre_split_render_pass;
    output_message(message_type::debug, std::format("Entering temporary renderpass {} instead of {}", reinterpret_cast<uintptr_t>(new_rp),
                                                    reinterpret_cast<uintptr_t>(current_render_pass_)));
    VkRenderPassBeginInfo rpb = *pRenderPassBegin;
    rpb.renderPass = new_rp;
    begin_info_allocator.reset();
    clone(state_block_, pRenderPassBegin[0], original_begin_info, &begin_info_allocator, _VkRenderPassBeginInfo_VkRenderPassSampleLocationsBeginInfoEXT_VkAttachmentSampleLocationsEXT_VkSampleLocationsInfoEXT_sampleLocationsPerPixel_valid, _VkRenderPassBeginInfo_VkRenderPassSampleLocationsBeginInfoEXT_VkSubpassSampleLocationsEXT_VkSampleLocationsInfoEXT_sampleLocationsPerPixel_valid, _VkRenderPassBeginInfo_pClearValues_valid, _VkRenderPassBeginInfo_VkClearValue_color_valid);

    return super::vkCmdBeginRenderPass(commandBuffer, &rpb, contents);
  }
  void vkCmdEndRenderPass(VkCommandBuffer commandBuffer) {
    if (current_render_pass_ == VK_NULL_HANDLE) {
      return super::vkCmdEndRenderPass(commandBuffer);
    }
    if (stage == current_stage::eFirstStage) {
      super::vkCmdEndRenderPass(commandBuffer);
      VkRenderPassBeginInfo begin_info = original_begin_info;
      begin_info.renderPass =
          split_renderpasses[current_render_pass_].subpasses[current_subpass_].post_split_render_pass;

      super::vkCmdBeginRenderPass(commandBuffer, &begin_info, VK_SUBPASS_CONTENTS_INLINE);
      stage = current_stage::eSecondStage;
    }
    if (stage == current_stage::eSecondStage) {
      super::vkCmdEndRenderPass(commandBuffer);
      VkRenderPassBeginInfo begin_info = original_begin_info;
      begin_info.renderPass =
          split_renderpasses[current_render_pass_].subpasses[current_subpass_].end_render_pass;

      super::vkCmdBeginRenderPass(commandBuffer, &begin_info, VK_SUBPASS_CONTENTS_INLINE);
      stage = current_stage::eLastStage;
    }
    current_render_pass_ = VK_NULL_HANDLE;
    return super::vkCmdEndRenderPass(commandBuffer);
  }
  void vkCmdNextSubpass(VkCommandBuffer commandBuffer, VkSubpassContents contents) {
    if (commands_to_split_.empty()) {
      return super::vkCmdNextSubpass(commandBuffer, contents);
    }
    if (stage == current_stage::eFirstStage) {
      super::vkCmdEndRenderPass(commandBuffer);
      VkRenderPassBeginInfo begin_info = original_begin_info;
      begin_info.renderPass =
          split_renderpasses[current_render_pass_].subpasses[current_subpass_].post_split_render_pass;

      super::vkCmdBeginRenderPass(commandBuffer, &begin_info, VK_SUBPASS_CONTENTS_INLINE);
      stage = current_stage::eSecondStage;
    }
    if (stage == current_stage::eSecondStage) {
      super::vkCmdEndRenderPass(commandBuffer);
      VkRenderPassBeginInfo begin_info = original_begin_info;
      begin_info.renderPass =
          split_renderpasses[current_render_pass_].subpasses[current_subpass_].end_render_pass;

      super::vkCmdBeginRenderPass(commandBuffer, &begin_info, VK_SUBPASS_CONTENTS_INLINE);
      stage = current_stage::eLastStage;
    }

    if (stage == current_stage::eLastStage) {
      super::vkCmdEndRenderPass(commandBuffer);
    }
    current_subpass_++;

    VkRenderPassBeginInfo begin_info = original_begin_info;
    begin_info.renderPass =
        split_renderpasses[current_render_pass_].subpasses[current_subpass_].pre_split_render_pass;
    return super::vkCmdBeginRenderPass(commandBuffer, &begin_info, VK_SUBPASS_CONTENTS_INLINE);
  }

  void vkCmdBindPipeline(VkCommandBuffer commandBuffer, VkPipelineBindPoint pipelineBindPoint, VkPipeline pipeline) {
    if (commands_to_split_.empty()) {
      return super::vkCmdBindPipeline(commandBuffer, pipelineBindPoint, pipeline);
    }
    return super::vkCmdBindPipeline(commandBuffer, pipelineBindPoint, pipeline);
  }
  void vkCmdExecuteCommands(VkCommandBuffer commandBuffer, uint32_t commandBufferCount, const VkCommandBuffer* pCommandBuffers) {
    if (commands_to_split_.empty()) {
      return super::vkCmdExecuteCommands(commandBuffer, commandBufferCount, pCommandBuffers);
    }
    return super::vkCmdExecuteCommands(commandBuffer, commandBufferCount, pCommandBuffers);
  }
  static const uint64_t no_command = ~static_cast<uint64_t>(0);

  std::vector<uint64_t> commands_to_split_;
  transform<command_buffer_recorder>* recorder;
  void (*on_command_buffer_split_)(VkCommandBuffer);
  enum class current_stage {
    eFirstStage,
    eSecondStage,
    eLastStage
  };

  VkRenderPassBeginInfo original_begin_info;
  temporary_allocator begin_info_allocator;

  VkRenderPass current_render_pass_ = VK_NULL_HANDLE;
  uint64_t current_subpass_ = 0;
  current_stage stage = current_stage::eFirstStage;

  VkPipeline fixed_pipelines = VK_NULL_HANDLE;
  std::unordered_map<VkPipeline, VkPipeline> rewritten_pipelines;
};

struct command_buffer_splitter_layers {
  command_buffer_splitter_layers(transform_base* base) {
    state_block_ = std::make_unique<transform<state_block>>(base);
    creation_tracker_ = std::make_unique<transform<creation_tracker<VkRenderPass, VkPipeline, VkShaderModule, VkDescriptorSetLayout, VkPipelineLayout>>>(base);
    creation_data_tracker_ = std::make_unique<transform<creation_data_tracker<VkRenderPass, VkPipeline, VkShaderModule, VkDescriptorSetLayout, VkPipelineLayout>>>(base);
    command_buffer_recorder_ = std::make_unique<transform<command_buffer_recorder>>(base);
    command_buffer_splitter_ = std::make_unique<transform<command_buffer_splitter>>(base);
    command_buffer_splitter_->recorder = command_buffer_recorder_.get();
  }

  std::unique_ptr<transform<state_block>> state_block_;
  std::unique_ptr<transform<creation_tracker<VkRenderPass, VkPipeline, VkShaderModule, VkDescriptorSetLayout, VkPipelineLayout>>> creation_tracker_;
  std::unique_ptr<transform<creation_data_tracker<VkRenderPass, VkPipeline, VkShaderModule, VkDescriptorSetLayout, VkPipelineLayout>>> creation_data_tracker_;
  std::unique_ptr<transform<command_buffer_recorder>> command_buffer_recorder_;
  std::unique_ptr<transform<command_buffer_splitter>> command_buffer_splitter_;
};

}  // namespace gapid2

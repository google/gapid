#include <unordered_set>
#include <vector>

#include "json.hpp"
#include "layer.h"

#define QUERIES_PER_POOL 128

struct timestamp_locations {
  std::unordered_set<uint64_t> render_passes;
};

std::unordered_map<uint64_t, std::unordered_map<uint64_t, timestamp_locations>> submit_indices;
static bool include_draw_calls = false;
static VkQueue rerecording_queue = VK_NULL_HANDLE;
std::unordered_map<VkQueue, VkDevice> queuesToDevices;

struct query {
  VkQueryPool query_pool;
  size_t num_queries_left;
  size_t num_queries_used;
};

struct queue_query_data {
  std::vector<query> renderpass_query_pools;
  std::vector<query> draw_query_pools;
};

std::unordered_map<VkQueue, queue_query_data>
    queries;

VkQueryPool create_pool_for_queue(VkQueue queue) {
  auto device = queuesToDevices[queue];
  VkQueryPoolCreateInfo qpci = {
      .sType = VK_STRUCTURE_TYPE_QUERY_POOL_CREATE_INFO,
      .pNext = nullptr,
      .flags = 0,
      .queryType = VK_QUERY_TYPE_TIMESTAMP,
      .queryCount = QUERIES_PER_POOL,
      .pipelineStatistics = 0};
  VkQueryPool pool = VK_NULL_HANDLE;
  if (VK_SUCCESS != vkCreateQueryPool(device, &qpci, nullptr, &pool)) {
    LogMessage(error, std::format("Could not create query pool for queue"));
    return VK_NULL_HANDLE;
  }
  return pool;
}

VkQueryPool reserve_pool_for(VkQueue queue, size_t* query_index, std::vector<query> queue_query_data::*type) {
  if (queries.count(queue) == 0) {
    queries[queue] = queue_query_data{};
  }
  auto& q = queries[queue];
  if ((q.*type).empty() || (q.*type).back().num_queries_left < 4) {
    (q.*type).push_back(query{
        create_pool_for_queue(queue),
        QUERIES_PER_POOL,
        QUERIES_PER_POOL});
  }
  query_index[0] = QUERIES_PER_POOL - (q.*type).back().num_queries_left;
  (q.*type).back().num_queries_left -= 2;
  return (q.*type).back().query_pool;
}

VkQueryPool reserve_pool_for_renderpass(VkQueue queue, size_t* query_index) {
  return reserve_pool_for(queue, query_index, &queue_query_data::renderpass_query_pools);
}

VkQueryPool reserve_pool_for_draw(VkQueue queue, size_t* query_index) {
  return reserve_pool_for(queue, query_index, &queue_query_data::draw_query_pools);
}

VKAPI_ATTR void VKAPI_CALL override_vkGetDeviceQueue(
    VkDevice device,
    uint32_t queueFamilyIndex,
    uint32_t queueIndex,
    VkQueue* pQueue) {
  vkGetDeviceQueue(device, queueFamilyIndex, queueIndex, pQueue);
  queuesToDevices[*pQueue] = device;
}

VKAPI_ATTR void VKAPI_CALL SetupLayer(LayerOptions* options) {
  const char* js = options->GetUserConfig();
  if (js) {
    auto setup = nlohmann::json::parse(js, nullptr, false);

    if (setup.contains("timestamp_locations")) {
      for (auto& val : setup["timestamp_locations"]) {
        std::unordered_map<uint64_t, timestamp_locations> locs;
        uint64_t i = val["submit_index"];
        auto& command_buffers = val["command_buffer_indices"];
        for (auto& cb : command_buffers) {
          uint64_t cb_idx = cb["command_buffer_index"];
          std::vector<uint64_t> indices = cb["renderpasses_indices"];
          timestamp_locations sl;
          for (auto& i : indices) {
            sl.render_passes.insert(i);
          }
          locs[cb_idx] = (std::move(sl));
        }
        submit_indices[i] = std::move(locs);
      }
    }
    if (setup.contains("include_draw_calls")) {
      include_draw_calls = setup["include_draw_calls"];
    }
  }
  options->CaptureAllCommands();
}

struct draw_query {
  VkQueryPool query_pool;
  size_t query_index;
};

struct renderpass_query {
  VkRenderPass renderpass;
  VkQueryPool query_pool;
  uint64_t renderpass_index;
  size_t query_index;
  std::vector<draw_query> draw_queries;
};

std::vector<std::pair<std::pair<VkCommandBuffer, uint64_t>, std::vector<renderpass_query>>> command_buffer_queries;

const timestamp_locations* current_submit;
uint64_t current_renderpass = 0;

VKAPI_ATTR VkResult VKAPI_CALL
override_vkQueueSubmit(
    VkQueue queue,
    uint32_t submitCount,
    const VkSubmitInfo* pSubmits,
    VkFence fence) {
  auto current_command_index = GetCommandIndex();
  if (submit_indices.count(current_command_index)) {
    rerecording_queue = queue;
    const auto& timestamp_locations = submit_indices[current_command_index];
    uint64_t submitted_idx = 0;
    for (size_t i = 0; i < submitCount; ++i) {
      for (size_t j = 0; j < pSubmits[i].commandBufferCount; ++j) {
        auto cb_idx = submitted_idx++;
        if (timestamp_locations.count(cb_idx)) {
          const auto& loc = timestamp_locations.find(cb_idx)->second;
          current_submit = &loc;
          rerecording_queue = queue;
          current_renderpass = 0;
          command_buffer_queries.push_back(std::make_pair(std::make_pair(pSubmits[i].pCommandBuffers[j], cb_idx), std::vector<renderpass_query>()));
          Rerecord_CommandBuffer(pSubmits[i].pCommandBuffers[j]);
        }
      }
    }
  }

  auto ret = vkQueueSubmit(queue, submitCount, pSubmits, fence);

  if (rerecording_queue != VK_NULL_HANDLE) {
    rerecording_queue = VK_NULL_HANDLE;
    vkQueueWaitIdle(queue);
    auto data = nlohmann::json();
    VkDevice device = queuesToDevices[queue];
    for (auto& it : command_buffer_queries) {
      auto renderpass_data = nlohmann::json();
      for (auto& it2 : it.second) {
        auto rp = nlohmann::json();
        rp["render_pass"] = reinterpret_cast<uintptr_t>(it2.renderpass);
        uint64_t d[2];
        vkGetQueryPoolResults(device, it2.query_pool, it2.query_index, 2, sizeof(uint64_t) * 2, d, sizeof(uint64_t), VK_QUERY_RESULT_64_BIT | VK_QUERY_RESULT_WAIT_BIT);
        rp["start_time"] = d[0];
        rp["end_time"] = d[1];
        rp["command_buffer"] = reinterpret_cast<uintptr_t>(it.first.first);
        rp["command_buffer_index"] = it.first.second;
        rp["render_pass_index"] = it2.renderpass_index;
        rp["submit_index"] = current_command_index;
        auto draw_datas = nlohmann::json();
        uint64_t draw_idx = 0;
        for (auto& it3 : it2.draw_queries) {
          auto draw_data = nlohmann::json();
          draw_data["draw_index"] = draw_idx++;
          vkGetQueryPoolResults(device, it3.query_pool, it3.query_index, 2, sizeof(uint64_t) * 2, d, sizeof(uint64_t), VK_QUERY_RESULT_64_BIT | VK_QUERY_RESULT_WAIT_BIT);
          draw_data["start_time"] = d[0];
          draw_data["end_time"] = d[1];
          draw_datas.push_back(draw_data);
        }
        rp["draws"] = draw_datas;
        data.push_back(rp);
      }
    }
    command_buffer_queries.clear();
    auto str = data.dump();
    SendJson(str.c_str(), str.size());
  }
  return ret;
}

size_t current_renderpass_timestamp_index = ~static_cast<size_t>(0);

VKAPI_ATTR void VKAPI_CALL override_vkCmdBeginRenderPass(
    VkCommandBuffer commandBuffer,
    const VkRenderPassBeginInfo* pRenderPassBegin,
    VkSubpassContents contents) {
  auto rp = pRenderPassBegin->renderPass;
  if (rerecording_queue != VK_NULL_HANDLE && current_submit->render_passes.count(current_renderpass)) {
    VkQueryPool pool = reserve_pool_for_renderpass(rerecording_queue, &current_renderpass_timestamp_index);
    vkCmdWriteTimestamp(commandBuffer, VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT, pool, current_renderpass_timestamp_index);
    command_buffer_queries.back().second.push_back(renderpass_query{
        rp,
        pool,
        current_renderpass,
        current_renderpass_timestamp_index,
        std::vector<draw_query>()});
  }
  current_renderpass++;
  vkCmdBeginRenderPass(commandBuffer, pRenderPassBegin, contents);
}

VKAPI_ATTR void VKAPI_CALL override_vkCmdEndRenderPass(
    VkCommandBuffer commandBuffer) {
  vkCmdEndRenderPass(commandBuffer);

  if (rerecording_queue != VK_NULL_HANDLE && current_renderpass_timestamp_index != ~static_cast<size_t>(0)) {
    auto& dat = command_buffer_queries.back().second.back();
    vkCmdWriteTimestamp(commandBuffer, VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT, dat.query_pool, dat.query_index + 1);
    current_renderpass_timestamp_index = ~static_cast<size_t>(0);
  }
}

VKAPI_ATTR void VKAPI_CALL override_vkCmdDraw(VkCommandBuffer commandBuffer, uint32_t vertexCount, uint32_t instanceCount, uint32_t firstVertex, uint32_t firstInstance) {
  size_t current_draw_index = 0;
  VkQueryPool pool = VK_NULL_HANDLE;
  if (include_draw_calls && rerecording_queue != VK_NULL_HANDLE && current_renderpass_timestamp_index != ~static_cast<size_t>(0)) {
    pool = reserve_pool_for_draw(rerecording_queue, &current_draw_index);
    vkCmdWriteTimestamp(commandBuffer, VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT, pool, current_draw_index);
  }

  vkCmdDraw(commandBuffer, vertexCount, instanceCount, firstVertex, firstInstance);

  if (include_draw_calls && rerecording_queue != VK_NULL_HANDLE && current_renderpass_timestamp_index != ~static_cast<size_t>(0)) {
    vkCmdWriteTimestamp(commandBuffer, VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT, pool, current_draw_index + 1);
    command_buffer_queries.back().second.back().draw_queries.push_back({pool, current_draw_index});
  }
}

VKAPI_ATTR void VKAPI_CALL override_vkCmdDrawIndexed(VkCommandBuffer commandBuffer, uint32_t indexCount, uint32_t instanceCount, uint32_t firstIndex, int32_t vertexOffset, uint32_t firstInstance) {
  size_t current_draw_index = 0;
  VkQueryPool pool = VK_NULL_HANDLE;
  if (include_draw_calls && rerecording_queue != VK_NULL_HANDLE && current_renderpass_timestamp_index != ~static_cast<size_t>(0)) {
    pool = reserve_pool_for_draw(rerecording_queue, &current_draw_index);
    vkCmdWriteTimestamp(commandBuffer, VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT, pool, current_draw_index);
  }

  vkCmdDrawIndexed(commandBuffer, indexCount, instanceCount, firstIndex, vertexOffset, firstInstance);

  if (include_draw_calls && rerecording_queue != VK_NULL_HANDLE && current_renderpass_timestamp_index != ~static_cast<size_t>(0)) {
    vkCmdWriteTimestamp(commandBuffer, VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT, pool, current_draw_index + 1);
    command_buffer_queries.back().second.back().draw_queries.push_back({pool, current_draw_index});
  }
}

VKAPI_ATTR void VKAPI_CALL override_vkCmdDrawIndirect(VkCommandBuffer commandBuffer, VkBuffer buffer, VkDeviceSize offset, uint32_t drawCount, uint32_t stride) {
  size_t current_draw_index = 0;
  VkQueryPool pool = VK_NULL_HANDLE;
  if (include_draw_calls && rerecording_queue != VK_NULL_HANDLE && current_renderpass_timestamp_index != ~static_cast<size_t>(0)) {
    pool = reserve_pool_for_draw(rerecording_queue, &current_draw_index);
    vkCmdWriteTimestamp(commandBuffer, VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT, pool, current_draw_index);
  }

  vkCmdDrawIndirect(commandBuffer, buffer, offset, drawCount, stride);

  if (include_draw_calls && rerecording_queue != VK_NULL_HANDLE && current_renderpass_timestamp_index != ~static_cast<size_t>(0)) {
    vkCmdWriteTimestamp(commandBuffer, VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT, pool, current_draw_index + 1);
    command_buffer_queries.back().second.back().draw_queries.push_back({pool, current_draw_index});
  }
}

VKAPI_ATTR void VKAPI_CALL override_vkCmdDrawIndexedIndirect(VkCommandBuffer commandBuffer, VkBuffer buffer, VkDeviceSize offset, uint32_t drawCount, uint32_t stride) {
  size_t current_draw_index = 0;
  VkQueryPool pool = VK_NULL_HANDLE;
  if (include_draw_calls && rerecording_queue != VK_NULL_HANDLE && current_renderpass_timestamp_index != ~static_cast<size_t>(0)) {
    pool = reserve_pool_for_draw(rerecording_queue, &current_draw_index);
    vkCmdWriteTimestamp(commandBuffer, VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT, pool, current_draw_index);
  }

  vkCmdDrawIndexedIndirect(commandBuffer, buffer, offset, drawCount, stride);

  if (include_draw_calls && rerecording_queue != VK_NULL_HANDLE && current_renderpass_timestamp_index != ~static_cast<size_t>(0)) {
    vkCmdWriteTimestamp(commandBuffer, VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT, pool, current_draw_index + 1);
    command_buffer_queries.back().second.back().draw_queries.push_back({pool, current_draw_index});
  }
}

VKAPI_ATTR void VKAPI_CALL override_vkCmdDrawIndirectCount(VkCommandBuffer commandBuffer, VkBuffer buffer, VkDeviceSize offset, VkBuffer countBuffer, VkDeviceSize countBufferOffset, uint32_t maxDrawCount, uint32_t stride) {
  size_t current_draw_index = 0;
  VkQueryPool pool = VK_NULL_HANDLE;
  if (include_draw_calls && rerecording_queue != VK_NULL_HANDLE && current_renderpass_timestamp_index != ~static_cast<size_t>(0)) {
    pool = reserve_pool_for_draw(rerecording_queue, &current_draw_index);
    vkCmdWriteTimestamp(commandBuffer, VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT, pool, current_draw_index);
  }

  vkCmdDrawIndirectCount(commandBuffer, buffer, offset, countBuffer, countBufferOffset, maxDrawCount, stride);

  if (include_draw_calls && rerecording_queue != VK_NULL_HANDLE && current_renderpass_timestamp_index != ~static_cast<size_t>(0)) {
    vkCmdWriteTimestamp(commandBuffer, VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT, pool, current_draw_index + 1);
    command_buffer_queries.back().second.back().draw_queries.push_back({pool, current_draw_index});
  }
}

VKAPI_ATTR void VKAPI_CALL override_vkCmdDrawIndexedIndirectCount(VkCommandBuffer commandBuffer, VkBuffer buffer, VkDeviceSize offset, VkBuffer countBuffer, VkDeviceSize countBufferOffset, uint32_t maxDrawCount, uint32_t stride) {
  size_t current_draw_index = 0;
  VkQueryPool pool = VK_NULL_HANDLE;
  if (include_draw_calls && rerecording_queue != VK_NULL_HANDLE && current_renderpass_timestamp_index != ~static_cast<size_t>(0)) {
    pool = reserve_pool_for_draw(rerecording_queue, &current_draw_index);
    vkCmdWriteTimestamp(commandBuffer, VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT, pool, current_draw_index);
  }

  vkCmdDrawIndexedIndirectCount(commandBuffer, buffer, offset, countBuffer, countBufferOffset, maxDrawCount, stride);

  if (rerecording_queue != VK_NULL_HANDLE && current_renderpass_timestamp_index != ~static_cast<size_t>(0)) {
    vkCmdWriteTimestamp(commandBuffer, VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT, pool, current_draw_index + 1);
    command_buffer_queries.back().second.back().draw_queries.push_back({pool, current_draw_index});
  }
}
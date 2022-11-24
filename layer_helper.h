#pragma once

#include <sstream>
#include <string>
#include <vector>

#include "common.h"

namespace gapid2 {

inline nlohmann::json get_layers() {
  return receive_message();
}

inline nlohmann::json get_user_config() {
  return receive_message();
}

}  // namespace gapid2
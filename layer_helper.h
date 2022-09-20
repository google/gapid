#pragma once

#include <sstream>
#include <string>
#include <vector>

namespace gapid2 {

inline std::vector<std::string> get_layers() {
  std::vector<std::string> ret;
  char* e = getenv("GAPID2_LAYERS");
  if (!e) {
    return ret;
  }
  std::stringstream ss(e);
  std::string elem;
  while (std::getline(ss, elem, ';')) {
    ret.push_back(elem);
  }
  return ret;
}

inline std::string get_user_config() {
  std::vector<std::string> ret;
  char* e = getenv("GAPID2_USER_CONFIG");
  if (!e) {
    return "";
  }
  return std::string(e);
}

}  // namespace gapid2
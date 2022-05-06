#pragma once
/*
 * Copyright (C) 2022 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>
#include "stype_header.h"

#define _QUOTE(x) #x
#define QUOTE(x) _QUOTE(x)
#define __FILE__LINE__ __FILE__ ":" QUOTE(__LINE__) " : "

#define FIXME(user, ...)                                        \
  message(__FILE__LINE__                                        \
          "\n"                                                  \
          " ------------------------------------------------\n" \
          "|  FIXME(" #user ") :  " #__VA_ARGS__                \
          "\n"                                                  \
          " -------------------------------------------------\n")

#define TODO(user, ...)                                        \
  message(__FILE__LINE__                                        \
          "\n"                                                  \
          " ------------------------------------------------\n" \
          "|  TODO(" #user ") :  " #__VA_ARGS__                \
          "\n"                                                  \
          " -------------------------------------------------\n")


namespace gapid2 {
template <typename T, typename... Ts>
constexpr bool args_contain() { return sizeof...(Ts) == 0 ? true : std::disjunction_v<std::is_same<T, Ts>...>; }

template<typename T>
const T* get_pNext(const void* v) {
  auto bis = reinterpret_cast<const VkBaseInStructure*>(v);
  while (bis) {
    if (bis->sType == get_stype<T>::sType) {
      return reinterpret_cast<const T*>(bis);
    }
    bis = bis->pNext;
  }
}

}
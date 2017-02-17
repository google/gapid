/*
 * Copyright (C) 2017 Google Inc.
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

#pragma once

#include <array>
#include <sstream>
#include <string>

#include "Interceptor.h"

//------------------------------------------------------------------------------
// Header only C++ wrapper interface around the basic extren "C" interface for
// convinince. The interface simplifies the usecase when linking agains the
// interceptor-lib with automated resource management and with adding a list
// of templatized functions for type safety and for intercepting multiple
// symbols
// with a single target function.
//------------------------------------------------------------------------------
class Interceptor {
public:
  Interceptor() { m_interceptor = ::InitializeInterceptor(); }

  ~Interceptor() { ::TerminateInterceptor(m_interceptor); }

  void *FindFunctionByName(const char *symbol_name) {
    return ::FindFunctionByName(m_interceptor, symbol_name);
  }

  template <typename Ret, typename... Args>
  bool InterceptFunction(Ret (*old_function)(Args...),
                         Ret (*new_function)(Args...),
                         Ret (**callback_function)(Args...),
                         std::string *error_message = nullptr);

  template <typename Ret, typename... Args>
  bool InterceptFunction(const char *symbol_name, Ret (*new_function)(Args...),
                         Ret (**callback_function)(Args...),
                         std::string *error_message = nullptr);

  template <typename DATA, typename FUN_TYPE> struct CallbackSignature;

  template <typename DATA, typename RET, typename... ARGS>
  struct CallbackSignature<DATA, RET(ARGS...)> {
    using type = RET (*)(DATA, RET (*)(ARGS...), ARGS...);
  };

  template <typename DATA, typename FUN_TYPE,
            typename CallbackSignature<DATA, FUN_TYPE>::type FUN,
            size_t FUN_COUNT>
  bool InterceptMultipleFunction(
      const std::array<std::pair<DATA, const char *>, FUN_COUNT> &functions,
      std::string *error_message = nullptr);

private:
  void *m_interceptor;

  // Helper method for collecting the error messages into a string stream
  static void ErrorCollector(void *baton, const char *message) {
    std::ostringstream *oss = static_cast<std::ostringstream *>(baton);
    (*oss) << message << '\n';
  }

  // Helper classes to implement the InterceptMultipleFunction method.
  template <typename DATA, size_t N, typename RET, typename... ARGS>
  struct SignleFunctionInterceptor {
    template <RET (*FUN)(DATA, RET (*)(ARGS...), ARGS...)>
    static bool Impl(Interceptor &inteptor, const char *symbol_name, DATA data,
                     std::string *error_message);

    template <RET (*FUN)(DATA, RET (*)(ARGS...), ARGS...)>
    static RET TrampolineFunction(ARGS... args);

    static DATA s_data;
    static RET (*s_callback)(ARGS...);
  };

  template <typename DATA, typename FUN_TYPE,
            typename CallbackSignature<DATA, FUN_TYPE>::type FUN,
            size_t FUN_COUNT, size_t N>
  struct MultiFunctionInterceptor;

  template <typename DATA, typename RET, typename... ARGS,
            RET (*FUN)(DATA, RET (*)(ARGS...), ARGS...), size_t FUN_COUNT,
            size_t N>
  struct MultiFunctionInterceptor<DATA, RET(ARGS...), FUN, FUN_COUNT, N> {
    static bool
    Impl(Interceptor &interceptor,
         const std::array<std::pair<DATA, const char *>, FUN_COUNT> &functions,
         std::string *error_message);
  };
};

template <typename Ret, typename... Args>
bool Interceptor::InterceptFunction(Ret (*old_function)(Args...),
                                    Ret (*new_function)(Args...),
                                    Ret (**callback_function)(Args...),
                                    std::string *error_message) {
  std::ostringstream error_oss;
  void *error_callback_baton = &error_oss;
  void (*error_callback)(void *, const char *) =
      error_message ? &ErrorCollector : nullptr;
  bool res =
      ::InterceptFunction(m_interceptor, reinterpret_cast<void *>(old_function),
                          reinterpret_cast<void *>(new_function),
                          reinterpret_cast<void **>(callback_function),
                          error_callback, error_callback_baton);
  if (error_message)
    *error_message += error_oss.str();
  return res;
}

template <typename Ret, typename... Args>
bool Interceptor::InterceptFunction(const char *symbol_name,
                                    Ret (*new_function)(Args...),
                                    Ret (**callback_function)(Args...),
                                    std::string *error_message) {
  std::ostringstream error_oss;
  void *error_callback_baton = &error_oss;
  void (*error_callback)(void *, const char *) =
      error_message ? &ErrorCollector : nullptr;
  bool res = ::InterceptSymbol(m_interceptor, symbol_name,
                               reinterpret_cast<void *>(new_function),
                               reinterpret_cast<void **>(callback_function),
                               error_callback, error_callback_baton);
  if (error_message)
    *error_message += error_oss.str();
  return res;
}

template <typename DATA, typename FUN_TYPE,
          typename Interceptor::CallbackSignature<DATA, FUN_TYPE>::type FUN,
          size_t FUN_COUNT>
bool Interceptor::InterceptMultipleFunction(
    const std::array<std::pair<DATA, const char *>, FUN_COUNT> &functions,
    std::string *error_message) {
  return MultiFunctionInterceptor<DATA, FUN_TYPE, FUN, FUN_COUNT,
                                  FUN_COUNT>::Impl(*this, functions,
                                                   error_message);
}

template <typename DATA, size_t N, typename RET, typename... ARGS>
template <RET (*FUN)(DATA, RET (*)(ARGS...), ARGS...)>
RET Interceptor::SignleFunctionInterceptor<
    DATA, N, RET, ARGS...>::TrampolineFunction(ARGS... args) {
  return FUN(s_data, s_callback, std::forward<ARGS>(args)...);
}

template <typename DATA, size_t N, typename RET, typename... ARGS>
template <RET (*FUN)(DATA, RET (*)(ARGS...), ARGS...)>
bool Interceptor::SignleFunctionInterceptor<DATA, N, RET, ARGS...>::Impl(
    Interceptor &interceptor, const char *symbol_name, DATA data,
    std::string *error_message) {
  s_data = data;
  return interceptor.InterceptFunction(symbol_name, &TrampolineFunction<FUN>,
                                       &s_callback, error_message);
}

template <typename DATA, typename RET, typename... ARGS,
          RET (*FUN)(DATA, RET (*)(ARGS...), ARGS...), size_t FUN_COUNT,
          size_t N>
bool Interceptor::
    MultiFunctionInterceptor<DATA, RET(ARGS...), FUN, FUN_COUNT, N>::Impl(
        Interceptor &interceptor,
        const std::array<std::pair<DATA, const char *>, FUN_COUNT> &functions,
        std::string *error_message) {
  bool res =
      MultiFunctionInterceptor<DATA, RET(ARGS...), FUN, FUN_COUNT, N - 1>::Impl(
          interceptor, functions, error_message);
  if (functions[N - 1].second)
    res &= SignleFunctionInterceptor<DATA, N - 1, RET, ARGS...>::template Impl<
        FUN>(interceptor, functions[N - 1].second, functions[N - 1].first,
             error_message);
  return res;
}

template <typename DATA, typename RET, typename... ARGS,
          RET (*FUN)(DATA, RET (*)(ARGS...), ARGS...), size_t FUN_COUNT>
struct Interceptor::MultiFunctionInterceptor<DATA, RET(ARGS...), FUN, FUN_COUNT,
                                             0> {
  static bool
  Impl(Interceptor &interceptor,
       const std::array<std::pair<DATA, const char *>, FUN_COUNT> &functions,
       std::string *error_message) {
    return true;
  }
};

template <typename DATA, size_t N, typename RET, typename... ARGS>
RET (*Interceptor::SignleFunctionInterceptor<DATA, N, RET, ARGS...>::s_callback)
(ARGS...) = nullptr;

template <typename DATA, size_t N, typename RET, typename... ARGS>
DATA Interceptor::SignleFunctionInterceptor<DATA, N, RET, ARGS...>::s_data;

import generator.vulkan as vulkan
import generator.args as args
import generator.standard as standard
import generator.generator as generator
import platform
import os


def main(args):
    vk = vulkan.load_vulkan(args)
    definition = vulkan.api_definition(vk,
                                       standard.version(),
                                       standard.exts(platform))

    with open(os.path.join(args.output_location, "layerer.h"), mode="w") as f:
        layerer = generator.generator(f)
        layerer.print('#pragma once')
        layerer.print('#include <vulkan/vulkan.h>')
        layerer.print('#include <filesystem>')
        layerer.print('#include <iostream>')
        layerer.print('#include <fstream>')
        layerer.print('#include <list>')
        layerer.print('#include "algorithm/sha1.hpp"')
        layerer.print('#include "common.h"')
        layerer.print('#include "transform_base.h"')
        layerer.print('#include "transform.h"')
        layerer.print('#include "indirect_functions.h"')
        layerer.print('#include "indirect_functions.h"')
        layerer.print('#include "handle_fixer.h"')
        layerer.print('#include "command_buffer_recorder.h"')
        layerer.print('''namespace gapid2 {
  const std::string version_string = "1";
  
  std::string inline get_file_sha(const std::string& str) {
    std::ifstream t(str);
    if (t.bad()) {
      return "";
    }
    std::stringstream buffer;
    buffer << t.rdbuf();
    digestpp::sha1 hasher;
    hasher.absorb(version_string);
  #ifndef NDEBUG
    hasher.absorb("Debug");
  #else
    hasher.absorb("RelWithDebInfo");
  #endif
    hasher.absorb(buffer.str());
    return hasher.hexdigest();
  }
  
''')

        for cmd in definition.commands.values():
            prms = [x.short_str() for x in cmd.args]
            args = [x.name for x in cmd.args]
            layerer.print(
                f'''
    {cmd.ret.short_str()} inline forward_{cmd.name}(void* fn, {", ".join(prms)}) {{
        return (*({cmd.ret.short_str()}(*)({", ".join(prms)}))(fn))({", ".join(args)});
    }}''')

        layerer.print('''
    class layerer: public transform_base {
      public:
        handle_fixer* fixer = nullptr;''')
        for cmd in definition.commands.values():
            prms = [x.short_str() for x in cmd.args]
            args = [x.name for x in cmd.args]
            layerer.print(
                f'''
        static {cmd.ret.short_str()} next_layer_{cmd.name}(void* data_, {", ".join(prms)}) {{
            return reinterpret_cast<transform_base*>(data_)->transform_base::{cmd.name}({", ".join(args)});
        }}''')

        layerer.print(
            f'''
        
        template<typename TT>
        static TT get_raw_handle(void* data_, TT in) {{
            auto fix = reinterpret_cast<layerer*>(data_)->fixer;
            if (fix) {{
              fix->fix_handle(&in);
            }}
            
            return in;
        }}''')

        layerer.print(
            '''        
        ~layerer() {
            for (auto& mod: modules) {
                FreeLibrary(mod);
            }
        }
        indirect_functions f;
        void RunUserSetup(HMODULE module);
        void* ResolveHelperFunction(const char* name, void** fout);
''')
        layerer.print('''        bool initializeLayers(std::vector<std::string> layers) {
          char cp[MAX_PATH];
          HMODULE hm = NULL;
          GetModuleHandleEx(GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS | 
                    GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT,
                    (LPCSTR) &get_file_sha, &hm);
          GetModuleFileName(hm, cp, MAX_PATH);
          std::filesystem::path fsp = cp;
          std::vector<std::string> layer_dlls;
          char cwd[MAX_PATH];
          GetCurrentDirectoryA(MAX_PATH, cwd);
          digestpp::sha1 hasher;
          hasher.absorb(cp);
          hasher.absorb(cwd);
          hasher.absorb(version_string);
#ifndef NDEBUG
          hasher.absorb("Debug");
#else
          hasher.absorb("RelWithDebInfo");
#endif
          std::filesystem::path build_path = 
            fsp.parent_path() / "scripts" / "build_file.bat";
          std::string work_path("gapid2_");
          work_path += hasher.hexdigest();
          for (auto& layer : layers) {
            if (!std::filesystem::exists(layer)) {
              GAPID2_ERROR("Could not find layer file");
            }
            if (layer.size() > 4 && 
                 layer[layer.size() - 4] == '.' && 
                 layer[layer.size() - 3] == 'd' && 
                 layer[layer.size() - 2] == 'l' && 
                 layer[layer.size() - 1] == 'l') {
                layer_dlls.push_back(layer);
                std::cout << "Using prebuilt dll: " << layer_dlls.back() << std::endl;
                continue;
            }
            auto file = std::filesystem::absolute(layer);
            std::string sha = get_file_sha(file.string());
            if (sha.empty()) {
              GAPID2_ERROR("Could not get sha for file");
            }
            char* t = getenv("TEMP");
            std::string dll(t);
            dll += std::string("\\\\") + work_path + "\\\\" + sha + ".dll";
            if (std::filesystem::exists(dll)) {
              std::cout << "Using existing layer " << dll << std::endl;
              layer_dlls.push_back(dll);
              continue;
            }
            std::string v = "cmd /c ";
            v += build_path.string();
            v += " ";
            v += file.string();
            v += " ";
            v += sha;
#ifndef NDEBUG
            v += " Debug";
#else
            v += " RelWithDebInfo";
#endif
            v += std::string(" ") + t + std::string("\\\\") + work_path + "\\\\";

            int ret = system(v.c_str());
            if (ret != 0) {
              GAPID2_ERROR("Could not build layer");
            }
            layer_dlls.push_back(dll);
            std::cout << "Built and readied layer " << layer_dlls.back() << std::endl;
          }                
            ''')
        for cmd in definition.commands.values():
            layerer.print(
                f'            f.fn_{cmd.name} = &next_layer_{cmd.name};')
            layerer.print(
                f'            f.{cmd.name}_user_data = this;')
        layerer.print('''
        for (const auto& layer: layer_dlls) {
          auto lib = LoadLibraryA(layer.c_str());
          if (!lib) {
              std::cerr << "Could not load library " << layer << std::endl;
              return false;
          }
          modules.push_back(lib);
          auto setup = (void (*)(void*, void* (*)(void*, const char*, void**), void*(tf)(void*, const char*)))GetProcAddress(lib, "SetupLayerInternal");
          if (!setup) {
              std::cerr << "Could not find library setup for " << layer << std::endl;
              return false;
          }
          std::cerr << "Setting up library " << layer << std::endl;
          setup(this, [](void* this__, const char* fn, void** fout) -> void* {
            layerer* this_ = reinterpret_cast<layerer*>(this__);''')
        for cmd in definition.commands.values():
            layerer.print(f'            if (!strcmp(fn, "{cmd.name}")) {{')
            layerer.print(
                f'              *fout = this_->f.{cmd.name}_user_data;')
            layerer.print(
                f'              return reinterpret_cast<void*>(this_->f.fn_{cmd.name});')
            layerer.print(f'            }}')
        layerer.print('''
            auto ret = this_->ResolveHelperFunction(fn, fout);
            if (!ret) {
                std::cerr << "Could not resolve function " << fn << std::endl;
            }
            return ret;
          }, [](void* this__, const char* tp) -> void* {''')
        for x in definition.types.values():
            if type(x) == vulkan.handle:
                layerer.print(
                    f'            if (!strcmp(tp, "{x.name}")) {{')
                layerer.print(
                    f'              auto r = layerer::get_raw_handle<{x.name}>;')
                layerer.print(f'              return r;')
                layerer.print(f'            }}')
        layerer.print(
            '''            std::cerr << "Could not resolve handle type " << tp << std::endl;''')
        layerer.print(
            '''            return nullptr;''')
        layerer.print('''          });''')

        for cmd in definition.commands.values():
            prms = [x.type.name for x in cmd.args]
            layerer.print(
                f'            auto f_{cmd.name} = ({cmd.ret.short_str()}(*)({", ".join(prms)}))GetProcAddress(lib, "override_{cmd.name}");')
            layerer.print(f'            if (f_{cmd.name}) {{')
            layerer.print(
                f'              f.{cmd.name}_user_data = f_{cmd.name};')
            layerer.print(
                f'              f.fn_{cmd.name} = &forward_{cmd.name};')
            layerer.print(
                f'              std::cerr << "Found function override_{cmd.name} in layer, setting up chain" << std::endl;')
            layerer.print(f'            }}')
        layerer.print(
            '''
          RunUserSetup(lib);
        }
        return true;
      }
''')
        for cmd in definition.commands.values():
            prms = [x.short_str() for x in cmd.args]
            args = [x.name for x in cmd.args]
            layerer.print(
                f'''
        {cmd.ret.short_str()} {cmd.name}({", ".join(prms)}) override {{
            return f.fn_{cmd.name}(f.{cmd.name}_user_data, {", ".join(args)});
        }}''')
        layerer.print('''
      std::vector<std::unique_ptr<transform<command_buffer_recorder>>> recorders;
      std::vector<HMODULE> modules;
  };
}
#include "layerer.inl"
''')


if __name__ == "__main__":
    main(args.get_args())

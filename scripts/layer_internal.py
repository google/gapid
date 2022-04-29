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
    with open(os.path.join(args.output_location, "layer_internal.inl"), mode="w") as transform_base:
        g = generator.generator(transform_base)
        g.print(standard.HEADER)

        g.print('\nvoid* user_data;')
        for cmd in definition.commands.values():
            prms = [x.short_str() for x in cmd.args]
            args = [x.name for x in cmd.args]
            g.print(f'void *call_{cmd.name}_user_data;')
            g.print(
                f'{cmd.ret.short_str()} (*call_{cmd.name})(void*, {", ".join(prms)});')
        for x in definition.types.values():
            if type(x) == vulkan.handle:
                g.print(
                    f'{x.name} (*get_raw_handle_{x.name})(void* data_, {x.name} in);')
        g.print(
            '''
extern "C" {
__declspec(dllexport) void SetupLayerInternal(void* user_data_, void* (fn)(void*, const char*, void**), void*(tf)(void*, const char*)) {
  user_data = user_data_;
''')
        for cmd in definition.commands.values():
            prms = [x.short_str() for x in cmd.args]
            g.print(
                f'  call_{cmd.name} = ({cmd.ret.short_str()}(*)(void*, {", ".join(prms)}))fn(user_data_, "{cmd.name}", &call_{cmd.name}_user_data);')
        for x in definition.types.values():
            if type(x) == vulkan.handle:
                g.print(
                    f'get_raw_handle_{x.name} = ({x.name}(*)(void*, {x.name}))tf(user_data_, "{x.name}");')

        g.print(
            '''
  SetupInternalPointers(user_data_, fn);
}
}
''')
        for cmd in definition.commands.values():
            prms = [x.short_str() for x in cmd.args]
            args = [x.name for x in cmd.args]
            g.print(
                f'''
inline {cmd.ret.short_str()} {cmd.name}({', '.join(prms)}) {{
  return (*call_{cmd.name})(call_{cmd.name}_user_data, {', '.join(args)});
}}''')
        for x in definition.types.values():
            if type(x) == vulkan.handle:
                g.print(
                    f'''
{x.name} get_raw_handle({x.name} in) {{
  return (*get_raw_handle_{x.name})(user_data, in);
}}''')
        g.print(
            '''
#undef VKAPI_ATTR
#undef VKAPI_CALL
#define VKAPI_CALL
#define VKAPI_ATTR extern "C" __declspec(dllexport)
''')


if __name__ == "__main__":
    main(args.get_args())

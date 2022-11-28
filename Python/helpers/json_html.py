from . import trace

def get_html_for_value(v, name):
    if type(v) == type(None):
        v = "nullptr"
    return "<div>" + name + ": " + str(v) + "</div>"

def get_html_for_list(l, name):
    summary = name
    skipName = False
    if len(l) == 0:
        return f"<div>{summary}: ----</div>"
    if (summary == ""):
        summary = len(l)
    out = "<div>"

    out = f'<details style="border-style: solid;border-width: 2px;"><summary>{summary}</summary>'
    out += '<div style="margin-left: 1em">'
    o = []
    for x in range(len(l)):
        b =  get_html_for_js_object(l[x], str(x))
        o.append(b)
    out += "".join(o)
    out += "</div>"
    out += "</details>"
    return out

def get_html_for_object(obj, name):
    summary = name
    skipName = False
    
    if 'name' in obj:
        if 'idx' in obj:
            summary = f"{obj['idx']}: {obj['name']}"
        else:
            summary = obj['name']
        skipName = True
    

    out = f'<details style="border-style: solid;border-width: 2px;"><summary>{summary}</summary>'
    out += '<div style="margin-left: 1em">'
    o = []
    for k, v in obj.items():
        if skipName and k == 'name':
            continue
        o.append('<div style="border-style: solid; border-width: 1px;">')
        o.append(get_html_for_js_object(v, k))
        o.append("</div>")
    out += "".join(o)
    out += "</div>"
    out += "</details>"
    return out

def get_html_for_js_object(js_obj, name=""):
    if type(js_obj) == dict:
        return get_html_for_object(js_obj, name)
    if type(js_obj) == list:
        return get_html_for_list(js_obj, name)
    else:
        return get_html_for_value(js_obj, name)
    

def get_html_for_commands(commands):
    return "".join(get_html_for_js_object(x) for x in commands)
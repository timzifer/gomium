import re, os, sys
def parse(d):
    sigs = {}
    for root, _, files in os.walk(d):
        for fn in files:
            if not fn.endswith('.h') or fn.endswith('_versions.h'): continue
            txt = open(os.path.join(root, fn), encoding='utf-8', errors='replace').read()
            for sm in re.finditer(r'typedef struct _(cef_\w+_t)\s*\{(.*?)\n\}\s*', txt, re.S):
                name, body = sm.group(1), sm.group(2)
                for mm in re.finditer(r'([\w \*]+?)\(CEF_CALLBACK\s*\*\s*(\w+)\)\s*\((.*?)\);', body, re.S):
                    ret, mname, args = mm.groups()
                    args = re.sub(r'\s+', ' ', args).strip()
                    ret = ret.strip()
                    sigs[(name, mname)] = f"{ret} ({args})"
    return sigs
old = parse(sys.argv[1]); new = parse(sys.argv[2])
focus = ['cef_app_t','cef_browser_process_handler_t','cef_render_process_handler_t','cef_client_t','cef_life_span_handler_t','cef_display_handler_t','cef_render_handler_t','cef_request_handler_t','cef_browser_t','cef_browser_host_t','cef_frame_t','cef_process_message_t','cef_list_value_t','cef_value_t','cef_binary_value_t','cef_task_runner_t','cef_request_t','cef_load_handler_t']
for k in sorted(set(old) & set(new)):
    if k[0] in focus and old[k] != new[k]:
        print(f"{k[0]}.{k[1]}")
        print(f"  ALT: {old[k]}")
        print(f"  NEU: {new[k]}")

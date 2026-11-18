import re
def members(path, struct):
    txt = open(path, encoding='utf-8', errors='replace').read()
    m = re.search(r'typedef struct (?:_' + struct[:-2] + r'_t\s*)?\{(.*?)\n\} ' + struct, txt, re.S)
    if not m: return None
    return set(re.findall(r'^\s*[\w \*]+?(\w+)\s*;\s*$', m.group(1), re.M))
for struct, oldf, newf in [
    ('cef_settings_t', 'G:/golang/gomium/include/internal/cef_types.h', 'cef151/include/internal/cef_types.h'),
    ('cef_browser_settings_t', 'G:/golang/gomium/include/internal/cef_types.h', 'cef151/include/internal/cef_types.h'),
    ('cef_window_info_t', 'G:/golang/gomium/include/internal/cef_types_win.h', 'cef151/include/internal/cef_types_win.h'),
]:
    o = members(oldf, struct); n = members(newf, struct)
    print('##', struct, '(old miss)' if o is None else '', '(new miss)' if n is None else '')
    if o and n:
        print('  + ' + ', '.join(sorted(n - o)))
        print('  - ' + ', '.join(sorted(o - n)))

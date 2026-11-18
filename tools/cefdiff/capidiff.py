import re, os, sys, json
def parse_dir(d):
    structs = {}   # name -> set(members)
    funcs = set()  # global CEF_EXPORT functions
    for root, _, files in os.walk(d):
        for fn in files:
            if not fn.endswith('.h') or fn.endswith('_versions.h'): continue
            txt = open(os.path.join(root, fn), encoding='utf-8', errors='replace').read()
            # global exports
            for m in re.finditer(r'CEF_EXPORT\s+[\w \*]+?(\w+)\s*\(', txt):
                funcs.add(m.group(1))
            # structs with members
            for sm in re.finditer(r'typedef struct _(cef_\w+_t)\s*\{(.*?)\n\}\s*\1?', txt, re.S):
                name, body = sm.group(1), sm.group(2)
                members = set()
                # function pointer members
                for mm in re.finditer(r'CEF_CALLBACK\s*\*\s*(\w+)\)', body):
                    members.add(mm.group(1))
                # plain data members (settings structs etc.)
                for mm in re.finditer(r'^\s*[\w \*]+?(\w+)\s*;\s*$', body, re.M):
                    members.add(mm.group(1))
                structs[name] = members
    return structs, funcs

old_s, old_f = parse_dir(sys.argv[1])
new_s, new_f = parse_dir(sys.argv[2])
removed_structs = sorted(set(old_s) - set(new_s))
added_structs   = sorted(set(new_s) - set(old_s))
common = sorted(set(old_s) & set(new_s))
changed = []
for s in common:
    add = new_s[s] - old_s[s]; rem = old_s[s] - new_s[s]
    if add or rem: changed.append((s, sorted(add), sorted(rem)))
print(f"OLD: {len(old_s)} structs, {len(old_f)} global funcs")
print(f"NEW: {len(new_s)} structs, {len(new_f)} global funcs")
print(f"structs removed: {len(removed_structs)}, added: {len(added_structs)}, changed: {len(changed)}, unchanged: {len(common)-len(changed)}")
print("\n== REMOVED STRUCTS =="); print(', '.join(removed_structs))
print("\n== ADDED STRUCTS =="); print(', '.join(added_structs))
print("\n== GLOBAL FUNCS removed =="); print(', '.join(sorted(old_f-new_f)))
print("\n== GLOBAL FUNCS added =="); print(', '.join(sorted(new_f-old_f)))
json.dump({'changed': [{'struct': s, 'added': a, 'removed': r} for s, a, r in changed]}, open('changed_structs.json','w'), indent=1)

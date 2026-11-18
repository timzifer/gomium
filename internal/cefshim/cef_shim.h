// Bind interface of the generated libcef shim (cef_shim.S / cef_shim.c).
// LoadLibcef resolves every symbol in cef_shim_symbols.go and stores the
// addresses through gomium_cef_bind; the trampolines jump through the bound
// pointers on every call.

#ifndef GOMIUM_CEF_SHIM_H_
#define GOMIUM_CEF_SHIM_H_

#include <stdint.h>

int gomium_cef_slot_count(void);
/* fn is a raw address (GetProcAddress result), passed as uintptr_t so the
   Go side never has to cast an integer through unsafe.Pointer. */
void gomium_cef_bind(int idx, uintptr_t fn);

#endif  // GOMIUM_CEF_SHIM_H_

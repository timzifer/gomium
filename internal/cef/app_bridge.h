#ifndef APP_BRIDGE_H
#define APP_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include <stdint.h>
#include "include/cef_api_hash.h"
#include "include/cef_version_info.h"
#include "include/capi/cef_app_capi.h"
#include "include/capi/cef_scheme_capi.h"
#include "v8.h"

struct _cef_app_t* cAppCreate();
void cAppFree(struct _cef_app_t* app);
void cAppSetIgnoreCertificateErrors(int ignore);
const char* cAppExpectedApiHash();
const char* cAppConfigureApiVersion();
const char* cAppCompiledCefVersion();
int cAppRuntimeVersionMatches();
void cAppRuntimeCefVersion(char* buf, size_t len);

#ifdef _WIN32
void cAppSetMainArgsInstance(cef_main_args_t* args, uintptr_t instance);
#endif

#endif
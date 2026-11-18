#ifndef SCHEME_BRIDGE_H
#define SCHEME_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_scheme_capi.h"
#include "include/capi/cef_request_context_capi.h"

int cSchemeRegistrarAdd(struct _cef_scheme_registrar_t* registrar, const char* name, int options);

struct _cef_scheme_handler_factory_t* cSchemeHandlerFactoryCreate();
void cSchemeHandlerFactoryFree(struct _cef_scheme_handler_factory_t* factory);

int cRequestContextRegisterSchemeHandlerFactory(struct _cef_request_context_t* context,
                                                const char* scheme,
                                                struct _cef_scheme_handler_factory_t* factory);

#endif

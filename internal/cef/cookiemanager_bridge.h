#ifndef COOKIE_MANAGER_BRIDGE_H
#define COOKIE_MANAGER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_cookie_capi.h"

struct _cef_cookie_manager_t* cCookieManagerGetGlobal();

struct _cef_cookie_visitor_t* cCookieVisitorCreate();
struct _cef_set_cookie_callback_t* cSetCookieCallbackCreate();
struct _cef_delete_cookies_callback_t* cDeleteCookiesCallbackCreate();
struct _cef_completion_callback_t* cCompletionCallbackCreate();

int cCookieManagerVisitAllCookies(struct _cef_cookie_manager_t* manager,
                                  struct _cef_cookie_visitor_t* visitor);
int cCookieManagerVisitUrlCookies(struct _cef_cookie_manager_t* manager,
                                  cef_string_t* url,
                                  int includeHttpOnly,
                                  struct _cef_cookie_visitor_t* visitor);
int cCookieManagerSetCookie(struct _cef_cookie_manager_t* manager,
                            cef_string_t* url,
                            struct _cef_cookie_t* cookie,
                            struct _cef_set_cookie_callback_t* callback);
int cCookieManagerDeleteCookies(struct _cef_cookie_manager_t* manager,
                                cef_string_t* url,
                                cef_string_t* cookie_name,
                                struct _cef_delete_cookies_callback_t* callback);
int cCookieManagerFlushStore(struct _cef_cookie_manager_t* manager,
                             struct _cef_completion_callback_t* callback);

#endif

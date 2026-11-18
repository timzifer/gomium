#include "cookiemanager_bridge.h"
#include "_cgo_export.h"

static int CEF_CALLBACK cookie_visit_trampoline(struct _cef_cookie_visitor_t* self,
                                                const struct _cef_cookie_t* cookie,
                                                int count,
                                                int total,
                                                int* deleteCookie) {
    return goCookieVisit(self, (struct _cef_cookie_t*)cookie, count, total);
}

static void CEF_CALLBACK set_cookie_complete_trampoline(struct _cef_set_cookie_callback_t* self,
                                                        int success) {
    goCookieAckComplete((void*)self, success);
}

static void CEF_CALLBACK delete_cookies_complete_trampoline(struct _cef_delete_cookies_callback_t* self,
                                                            int num_deleted) {
    goCookieAckComplete((void*)self, num_deleted);
}

static void CEF_CALLBACK completion_complete_trampoline(struct _cef_completion_callback_t* self) {
    goCookieAckComplete((void*)self, 1);
}

struct _cef_cookie_manager_t* cCookieManagerGetGlobal() { DEBUG_CALL;
    return cef_cookie_manager_get_global_manager(NULL);
}

struct _cef_cookie_visitor_t* cCookieVisitorCreate() { DEBUG_CALL;
    cef_cookie_visitor_t* visitor = (cef_cookie_visitor_t*)calloc(1, sizeof(cef_cookie_visitor_t));
    visitor -> base.size = sizeof(cef_cookie_visitor_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)visitor);
    visitor -> visit = cookie_visit_trampoline;
    return visitor;
}

struct _cef_set_cookie_callback_t* cSetCookieCallbackCreate() { DEBUG_CALL;
    cef_set_cookie_callback_t* callback = (cef_set_cookie_callback_t*)calloc(1, sizeof(cef_set_cookie_callback_t));
    callback -> base.size = sizeof(cef_set_cookie_callback_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)callback);
    callback -> on_complete = set_cookie_complete_trampoline;
    return callback;
}

struct _cef_delete_cookies_callback_t* cDeleteCookiesCallbackCreate() { DEBUG_CALL;
    cef_delete_cookies_callback_t* callback = (cef_delete_cookies_callback_t*)calloc(1, sizeof(cef_delete_cookies_callback_t));
    callback -> base.size = sizeof(cef_delete_cookies_callback_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)callback);
    callback -> on_complete = delete_cookies_complete_trampoline;
    return callback;
}

struct _cef_completion_callback_t* cCompletionCallbackCreate() { DEBUG_CALL;
    cef_completion_callback_t* callback = (cef_completion_callback_t*)calloc(1, sizeof(cef_completion_callback_t));
    callback -> base.size = sizeof(cef_completion_callback_t);
    initialize_cef_base_ref_counted((cef_base_ref_counted_t*)callback);
    callback -> on_complete = completion_complete_trampoline;
    return callback;
}

int cCookieManagerVisitAllCookies(struct _cef_cookie_manager_t* manager,
                                  struct _cef_cookie_visitor_t* visitor) {
    return manager->visit_all_cookies(manager, visitor);
}

int cCookieManagerVisitUrlCookies(struct _cef_cookie_manager_t* manager,
                                  cef_string_t* url,
                                  int includeHttpOnly,
                                  struct _cef_cookie_visitor_t* visitor) {
    return manager->visit_url_cookies(manager, url, includeHttpOnly, visitor);
}

int cCookieManagerSetCookie(struct _cef_cookie_manager_t* manager,
                            cef_string_t* url,
                            struct _cef_cookie_t* cookie,
                            struct _cef_set_cookie_callback_t* callback) {
    return manager->set_cookie(manager, url, cookie, callback);
}

int cCookieManagerDeleteCookies(struct _cef_cookie_manager_t* manager,
                                cef_string_t* url,
                                cef_string_t* cookie_name,
                                struct _cef_delete_cookies_callback_t* callback) {
    return manager->delete_cookies(manager, url, cookie_name, callback);
}

int cCookieManagerFlushStore(struct _cef_cookie_manager_t* manager,
                             struct _cef_completion_callback_t* callback) {
    return manager->flush_store(manager, callback);
}

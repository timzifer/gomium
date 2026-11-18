#include "requestcontext_bridge.h"

struct _cef_cookie_manager_t* cRequestContextGetCookieManager(struct _cef_request_context_t* context) {
    return context->get_cookie_manager(context, NULL);
}

int cRequestContextIsGlobal(struct _cef_request_context_t* context) {
    return context->is_global(context);
}

int cRequestContextIsSame(struct _cef_request_context_t* context, struct _cef_request_context_t* other) {
    // CEF releases the reference an argument carries (CppToC::Unwrap),
	// so the caller-owned |other| needs one of its own for the call.
	if (other) ((cef_base_ref_counted_t*)other)->add_ref((cef_base_ref_counted_t*)other);
	return context->is_same(context, other);
}

cef_string_userfree_t cRequestContextGetCachePath(struct _cef_request_context_t* context) {
    return context->get_cache_path(context);
}

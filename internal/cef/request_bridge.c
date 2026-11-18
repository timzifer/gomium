#include "request_bridge.h"

int cBridgeRequestIsReadOnly(struct _cef_request_t* request) {
    return request -> is_read_only(request);
}

cef_string_utf8_t* cBridgeRequestGetUrl(struct _cef_request_t* request) {
  cef_string_userfree_t source = request -> get_url(request);
  cef_string_utf8_t * output = cef_string_userfree_utf8_alloc();
  if (source == 0) {
    return output;
  }
  cef_string_to_utf8(source->str, source->length, output);
  cef_string_userfree_free(source);

  return output;

}

cef_string_utf8_t* cBridgeRequestGetMethod(struct _cef_request_t* request) {
  cef_string_userfree_t source = request -> get_method(request);
  cef_string_utf8_t * output = cef_string_userfree_utf8_alloc();
  if (source == 0) {
    return output;
  }
  cef_string_to_utf8(source->str, source->length, output);
  cef_string_userfree_free(source);

  return output;

}

cef_string_utf8_t* cBridgeRequestGetReferrer(struct _cef_request_t* request) {
  cef_string_userfree_t source = request -> get_referrer_url(request);
  cef_string_utf8_t * output = cef_string_userfree_utf8_alloc();
  if (source == 0) {
    return output;
  }
  cef_string_to_utf8(source->str, source->length, output);
  cef_string_userfree_free(source);

  return output;

}
void cBridgeRequestSetUrl(struct _cef_request_t* request, const cef_string_t* url) {
  request -> set_url(request, url);
}
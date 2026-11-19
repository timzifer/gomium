// Stand-ins for the vendored CEF headers.

typedef struct _cef_base_ref_counted_t {
  size_t size;
} cef_base_ref_counted_t;

///
/// A reference-counted struct.
///
typedef struct _cef_frame_t {
  ///
  /// Base structure.
  ///
  cef_base_ref_counted_t base;
} cef_frame_t;

typedef struct _cef_browser_t {
  cef_base_ref_counted_t base;
} cef_browser_t;

typedef struct _cef_preference_manager_t {
  cef_base_ref_counted_t base;
} cef_preference_manager_t;

// Reference-counted through its base, like cef_request_context_t.
typedef struct _cef_request_context_t {
  cef_preference_manager_t base;
} cef_request_context_t;

// A plain value struct: no references.
typedef struct _cef_rect_t {
  int x;
} cef_rect_t;

CEF_EXPORT int cef_post_frame(cef_frame_t* frame, cef_rect_t* rect);

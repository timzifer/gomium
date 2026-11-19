#define REF_CONSUMED
#define REF_BORROWED

int frame_is_same(cef_frame_t* self, cef_frame_t* that REF_BORROWED);
void frame_adopt(cef_frame_t* self, cef_frame_t* other REF_CONSUMED);
void context_adopt(cef_frame_t* self, cef_request_context_t* context REF_CONSUMED);
void frame_set_rect(cef_frame_t* self, cef_rect_t* rect);
void frame_unclassified(cef_frame_t* self, cef_browser_t* browser); // want: frame_unclassified: parameter 1
cef_frame_t* frame_create(void);

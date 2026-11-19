#ifndef BROWSER_BRIDGE_H
#define BROWSER_BRIDGE_H

#include "debug.h"
#include "memory.h"
#include <stdlib.h>
#include "include/capi/cef_client_capi.h"
#include "include/capi/cef_values_capi.h"


void cBridgeBrowserReload(struct _cef_browser_t *browser);
void cBridgeBrowserReloadIgnoreCache(struct _cef_browser_t *browser);
void cBridgeBrowserLoadUrl(struct _cef_browser_t *browser, const cef_string_t* url);


void cBridgeBrowserSendFocusEvent(struct _cef_browser_t *browser, int setFocus);
void cBridgeBrowserWasResized(struct _cef_browser_t *browser);
void cBridgeBrowserCloseBrowser(struct _cef_browser_t *browser, int force);
void cBridgeBrowserSendMouseMoveEvent(struct _cef_browser_t *browser, int x, int y, int mouseLeave, uint32_t modifiers);
void cBridgeBrowserSendMouseWheelEvent(struct _cef_browser_t *browser, int x, int y, int dx, int dy, uint32_t modifiers);
void cBridgeBrowserSendMouseClickEvent(struct _cef_browser_t *browser, int x, int y, int type, int mouseUp, int clickCount, uint32_t modifiers);
void cBridgeBrowserSendKeyEvent(struct _cef_browser_t *browser,	cef_key_event_type_t eventType, int nativeKeyCode, int windowsKeyCode, char16_t character, int isSystemKey, uint32_t modifiers);
void cBridgeBrowserExecuteJavascript(struct _cef_browser_t *browser, const cef_string_t* code, const cef_string_t* script_url, int start_line);

int cBridgeBrowserCanGoBack(struct _cef_browser_t *browser);
int cBridgeBrowserCanGoForward(struct _cef_browser_t *browser);
void cBridgeBrowserGoBack(struct _cef_browser_t *browser);
void cBridgeBrowserGoForward(struct _cef_browser_t *browser);

void cBridgeBrowserSetZoomLevel(struct _cef_browser_t *browser, double level);
double cBridgeBrowserGetZoomLevel(struct _cef_browser_t *browser);

void cBridgeBrowserFind(struct _cef_browser_t *browser, const cef_string_t* text, int forward, int matchCase, int findNext);
void cBridgeBrowserStopFinding(struct _cef_browser_t *browser, int clearSelection);

void cBridgeBrowserSetAudioMuted(struct _cef_browser_t *browser, int muted);
int cBridgeBrowserIsAudioMuted(struct _cef_browser_t *browser);

struct _cef_pdf_print_callback_t* cPdfPrintCallbackCreate();
void cBridgeBrowserPrintToPdf(struct _cef_browser_t *browser,
                              const cef_string_t* path,
                              int landscape,
                              int printBackground,
                              struct _cef_pdf_print_callback_t* callback REF_CONSUMED);

// The DevTools bridge builds the parameter dictionary in single steps so the
// Go side can keep its map[string]any shape. cBridgeBrowserExecuteDevToolsMethod
// hands the dictionary to CEF, which takes over the reference - only a
// dictionary that never reached it has to go through cBridgeDictionaryRelease.
struct _cef_dictionary_value_t* cBridgeDictionaryCreate();
void cBridgeDictionarySetString(struct _cef_dictionary_value_t *dictionary, const cef_string_t* key, const cef_string_t* value);
void cBridgeDictionarySetBool(struct _cef_dictionary_value_t *dictionary, const cef_string_t* key, int value);
void cBridgeDictionarySetInt(struct _cef_dictionary_value_t *dictionary, const cef_string_t* key, int value);
void cBridgeDictionaryRelease(struct _cef_dictionary_value_t *dictionary);

int cBridgeBrowserExecuteDevToolsMethod(struct _cef_browser_t *browser,
                                        const cef_string_t* method,
                                        struct _cef_dictionary_value_t *params REF_CONSUMED);

#endif

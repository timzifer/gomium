// Objective-C side of the macOS application bootstrap. CEF refuses to run
// under a plain NSApplication: its message pump sets and reads a
// "currently inside sendEvent:" flag on the shared application object, and
// the object has to answer that through CefAppProtocol.
//
// Compiled by cgo on macOS only (see the _darwin suffix); app_darwin.go
// carries the -framework Cocoa link flag.

#import <Cocoa/Cocoa.h>

// CefAppProtocol as declared in include/cef_application_mac.h. That header is
// C++ and pulls in the whole CefRefPtr machinery for one three-line protocol,
// so the protocol is spelled out here instead. The selectors are what CEF
// calls; they must not be renamed.
@protocol CefAppProtocol <NSObject>
- (BOOL)isHandlingSendEvent;
- (void)setHandlingSendEvent:(BOOL)handlingSendEvent;
@end

@interface GomiumApplication : NSApplication <CefAppProtocol>
@end

@implementation GomiumApplication {
  BOOL handlingSendEvent_;
}

- (BOOL)isHandlingSendEvent {
  return handlingSendEvent_;
}

- (void)setHandlingSendEvent:(BOOL)handlingSendEvent {
  handlingSendEvent_ = handlingSendEvent;
}

// The flag is saved and restored instead of simply cleared: CEF pumps nested
// event loops from inside sendEvent:, and each of them comes back through
// here. Clearing unconditionally would tell CEF the outer event is done while
// it is still being dispatched.
- (void)sendEvent:(NSEvent*)event {
  BOOL wasHandlingSendEvent = handlingSendEvent_;
  handlingSendEvent_ = YES;
  [super sendEvent:event];
  handlingSendEvent_ = wasHandlingSendEvent;
}

@end

// cAppInstallMacApplication makes sure NSApp exists and speaks
// CefAppProtocol. Returns 1 on success, 0 when the process already runs an
// NSApplication that does not implement the protocol - that one cannot be
// replaced, since AppKit keeps the first instance for the lifetime of the
// process.
int cAppInstallMacApplication(void) {
  @autoreleasepool {
    if (NSApp != nil) {
      return [NSApp conformsToProtocol:@protocol(CefAppProtocol)] ? 1 : 0;
    }
    [GomiumApplication sharedApplication];
    // Without a regular activation policy a binary started outside an .app
    // bundle gets no dock entry and no key window - its browser window would
    // come up behind everything else and never take focus.
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
    return 1;
  }
}

// cAppFinishMacApplicationLaunch completes the AppKit launch sequence. gomium
// drives CEF through an external message pump, so -[NSApplication run] is
// never called - and that is where AppKit would normally call
// finishLaunching. Without it the process stays half-launched: no key window,
// no menu bar, and a browser window that comes up behind everything.
//
// Idempotent, and a no-op for a host that runs its own NSApplication loop:
// AppKit sets NSApp.running once -run is under way, and calling
// finishLaunching a second time would post the launch notifications again.
//
// Taking focus is deliberately not part of this - see
// cAppActivateMacApplication. An offscreen render has no window to show and
// must not pull the user out of whatever they were doing.
void cAppFinishMacApplicationLaunch(void) {
  @autoreleasepool {
    if (NSApp == nil || [NSApp isRunning]) {
      return;
    }
    static BOOL finished = NO;
    if (!finished) {
      finished = YES;
      [NSApp finishLaunching];
    }
  }
}

// cAppActivateMacApplication brings the process to the front. A binary
// started from a terminal is not the active application, so its first window
// would come up behind the terminal and never become key.
void cAppActivateMacApplication(void) {
  @autoreleasepool {
    if (NSApp == nil) {
      return;
    }
    [NSApp activateIgnoringOtherApps:YES];
  }
}

// cAppInstallMacDefaultMenu puts a minimal main menu in place. Keyboard
// shortcuts on macOS are dispatched through the menu bar, not by the window:
// with no main menu, cmd+C / cmd+V / cmd+A never reach the web contents and a
// windowed browser looks broken. cmd+Q is included for the same reason - a
// bundle without it can only be killed from outside.
//
// Does nothing when a menu is already there, so a host with its own menu bar
// keeps it.
void cAppInstallMacDefaultMenu(const char* appName) {
  @autoreleasepool {
    if (NSApp == nil || [NSApp mainMenu] != nil) {
      return;
    }
    NSString* name = appName != NULL ? @(appName) : @"Application";

    NSMenu* mainMenu = [[NSMenu alloc] initWithTitle:@""];

    NSMenuItem* appItem = [[NSMenuItem alloc] initWithTitle:@"" action:NULL keyEquivalent:@""];
    NSMenu* appMenu = [[NSMenu alloc] initWithTitle:name];
    [appMenu addItemWithTitle:[@"Hide " stringByAppendingString:name]
                       action:@selector(hide:)
                keyEquivalent:@"h"];
    [appMenu addItem:[NSMenuItem separatorItem]];
    [appMenu addItemWithTitle:[@"Quit " stringByAppendingString:name]
                       action:@selector(terminate:)
                keyEquivalent:@"q"];
    [appItem setSubmenu:appMenu];
    [mainMenu addItem:appItem];

    // The Edit menu carries no logic of its own - the selectors are the
    // standard responder-chain ones, which is exactly what the browser's
    // native view answers.
    NSMenuItem* editItem = [[NSMenuItem alloc] initWithTitle:@"Edit" action:NULL keyEquivalent:@""];
    NSMenu* editMenu = [[NSMenu alloc] initWithTitle:@"Edit"];
    [editMenu addItemWithTitle:@"Undo" action:@selector(undo:) keyEquivalent:@"z"];
    [[editMenu addItemWithTitle:@"Redo" action:@selector(redo:) keyEquivalent:@"z"]
        setKeyEquivalentModifierMask:NSEventModifierFlagCommand | NSEventModifierFlagShift];
    [editMenu addItem:[NSMenuItem separatorItem]];
    [editMenu addItemWithTitle:@"Cut" action:@selector(cut:) keyEquivalent:@"x"];
    [editMenu addItemWithTitle:@"Copy" action:@selector(copy:) keyEquivalent:@"c"];
    [editMenu addItemWithTitle:@"Paste" action:@selector(paste:) keyEquivalent:@"v"];
    [editMenu addItemWithTitle:@"Select All" action:@selector(selectAll:) keyEquivalent:@"a"];
    [editItem setSubmenu:editMenu];
    [mainMenu addItem:editItem];

    [NSApp setMainMenu:mainMenu];
  }
}

// cAppPumpMacApplicationEvents drains the AppKit event queue.
//
// This is the half of the message loop cef_do_message_loop_work does not do.
// CEF's own loop and the NSApplication event queue are separate: cefclient
// keeps -[NSApplication run] going and only decides *when* to call
// CefDoMessageLoopWork from inside it. gomium runs the loop itself, so
// nothing drains NSApp - a windowed browser then paints once and afterwards
// ignores mouse, keyboard and its own close, which surfaces as "timed out
// waiting for browsers to close" on shutdown.
//
// Dequeuing with distantPast makes this non-blocking: it takes what is
// already queued and returns. The events go through -[GomiumApplication
// sendEvent:], which is what sets the CefAppProtocol flag CEF reads.
//
// AppKit is main-thread only, so a call from anywhere else does nothing
// rather than corrupting the queue.
int cAppPumpMacApplicationEvents(void) {
  if (![NSThread isMainThread]) {
    return 0;
  }
  int more = 0;
  @autoreleasepool {
    if (NSApp == nil) {
      return 0;
    }
    // Bounded: a page that produces events as fast as they are consumed
    // would otherwise keep the CEF loop from ever running again. Hitting
    // the bound is reported, so the caller schedules the next pass at once.
    more = 1;
    for (int i = 0; i < 128; i++) {
      NSEvent* event = [NSApp nextEventMatchingMask:NSEventMaskAny
                                          untilDate:[NSDate distantPast]
                                             inMode:NSDefaultRunLoopMode
                                            dequeue:YES];
      if (event == nil) {
        more = 0;
        break;
      }
      [NSApp sendEvent:event];
    }
    [NSApp updateWindows];
  }
  return more;
}

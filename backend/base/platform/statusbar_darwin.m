#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>

// Exported from Go (//export in statusbar_darwin.go / minipanel_darwin.go) to
// route menu bar clicks and mini panel interactions back to app logic.
extern void quotapanelStatusBarLeftClick(void);
extern void quotapanelStatusBarShow(void);
extern void quotapanelStatusBarRestart(void);
extern void quotapanelStatusBarQuit(void);
extern void quotapanelMiniPanelAction(const char *action, int accountID);

// QPStatusBarController owns the menu bar status item and its right-click menu.
@interface QPStatusBarController : NSObject
@property(strong) NSStatusItem *statusItem;
@property(strong) NSMenu *menu;
@end

@implementation QPStatusBarController

// Status item click: left button opens the quota mini panel, right button pops up the menu.
- (void)handleClick:(id)sender {
  NSEventType type = NSApp.currentEvent.type;
  if (type == NSEventTypeRightMouseUp || type == NSEventTypeRightMouseDown) {
    NSStatusBarButton *button = self.statusItem.button;
    [self.menu popUpMenuPositioningItem:nil
                             atLocation:NSMakePoint(0, button.bounds.size.height + 6)
                                 inView:button];
  } else {
    quotapanelStatusBarLeftClick();
  }
}

- (void)onShow:(id)sender {
  quotapanelStatusBarShow();
}

- (void)onRestart:(id)sender {
  quotapanelStatusBarRestart();
}

- (void)onQuit:(id)sender {
  quotapanelStatusBarQuit();
}

@end

static QPStatusBarController *gStatusBarController = nil;

// Build the menu; rebuilt on every call so labels can follow locale changes.
void qpStatusBarSetMenu(const char *show, const char *restart, const char *quit) {
  NSString *showTitle = show ? [NSString stringWithUTF8String:show] : @"Show";
  NSString *restartTitle = restart ? [NSString stringWithUTF8String:restart] : @"Restart";
  NSString *quitTitle = quit ? [NSString stringWithUTF8String:quit] : @"Quit";

  dispatch_async(dispatch_get_main_queue(), ^{
    if (gStatusBarController == nil) {
      return;
    }
    NSMenu *menu = [[NSMenu alloc] init];
    NSMenuItem *showItem = [[NSMenuItem alloc] initWithTitle:showTitle
                                                     action:@selector(onShow:)
                                              keyEquivalent:@""];
    showItem.target = gStatusBarController;
    [menu addItem:showItem];
    [menu addItem:[NSMenuItem separatorItem]];

    NSMenuItem *restartItem = [[NSMenuItem alloc] initWithTitle:restartTitle
                                                        action:@selector(onRestart:)
                                                 keyEquivalent:@""];
    restartItem.target = gStatusBarController;
    [menu addItem:restartItem];

    NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:quitTitle
                                                     action:@selector(onQuit:)
                                              keyEquivalent:@""];
    quitItem.target = gStatusBarController;
    [menu addItem:quitItem];

    gStatusBarController.menu = menu;
  });
}

// Create the menu bar status item on the main thread.
void qpStatusBarInit(const void *iconBytes, int length, const char *tooltip) {
  NSData *data = nil;
  if (iconBytes != NULL && length > 0) {
    data = [NSData dataWithBytes:iconBytes length:(NSUInteger)length];
  }
  NSString *tip = tooltip ? [NSString stringWithUTF8String:tooltip] : nil;

  dispatch_async(dispatch_get_main_queue(), ^{
    gStatusBarController = [[QPStatusBarController alloc] init];
    NSStatusItem *item =
        [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];
    NSStatusBarButton *button = item.button;
    if (data != nil) {
      NSImage *image = [[NSImage alloc] initWithData:data];
      [image setSize:NSMakeSize(18, 18)];
      // Template image adapts to light/dark menu bar automatically.
      image.template = YES;
      button.image = image;
    }
    button.toolTip = tip;
    button.target = gStatusBarController;
    button.action = @selector(handleClick:);
    [button sendActionOn:(NSEventMaskLeftMouseUp | NSEventMaskRightMouseUp)];
    gStatusBarController.statusItem = item;
  });
}

// ── Quota mini panel (popover hosting a web view) ───────────────────────────

// gMiniPanelShown mirrors the popover visibility so qpMiniPanelIsShown() can be
// answered without dispatching to the main thread (Go may call it from the main
// thread itself, where a dispatch_sync would deadlock).
static volatile int gMiniPanelShown = 0;

// QPMiniPanelController owns the popover shown when the status item is clicked.
// Content is HTML generated on the Go side; clicks are bridged back through the
// "quotapanel" WKScriptMessageHandler.
@interface QPMiniPanelController : NSObject <WKScriptMessageHandler>
@property(strong) NSPopover *popover;
@property(strong) WKWebView *webView;
@property(nonatomic) NSTimeInterval lastCloseTime;
@end

@implementation QPMiniPanelController

// Lazily build the popover and its web view.
- (void)ensurePopover {
  if (self.popover) {
    return;
  }

  WKWebViewConfiguration *config = [[WKWebViewConfiguration alloc] init];
  WKUserContentController *contentController = [[WKUserContentController alloc] init];
  [contentController addScriptMessageHandler:self name:@"quotapanel"];
  config.userContentController = contentController;

  WKWebView *web = [[WKWebView alloc] initWithFrame:NSMakeRect(0, 0, 340, 380)
                                      configuration:config];
  web.autoresizingMask = NSViewWidthSizable | NSViewHeightSizable;
  self.webView = web;

  NSViewController *controller = [[NSViewController alloc] init];
  controller.view = web;

  NSPopover *popover = [[NSPopover alloc] init];
  popover.behavior = NSPopoverBehaviorTransient;
  popover.animates = YES;
  popover.contentViewController = controller;
  popover.contentSize = NSMakeSize(340, 380);
  self.popover = popover;
}

// Show / hide the popover anchored under the status item.
- (void)toggleWithHTML:(NSString *)html height:(CGFloat)height {
  NSStatusBarButton *button =
      gStatusBarController ? gStatusBarController.statusItem.button : nil;
  if (button == nil) {
    return;
  }

  if (self.popover.isShown) {
    [self.popover close];
    return;
  }

  // A transient popover closes itself on the status button mouse-down, before
  // this action fires; ignore the immediate re-open from that same click so the
  // icon behaves as a real toggle.
  NSTimeInterval now = [NSDate timeIntervalSinceReferenceDate];
  if (now - self.lastCloseTime < 0.25) {
    return;
  }

  [self ensurePopover];
  if (height > 0) {
    self.popover.contentSize = NSMakeSize(340, height);
  }
  [self.webView loadHTMLString:html baseURL:nil];
  [self.popover showRelativeToRect:button.bounds ofView:button preferredEdge:NSRectEdgeMaxY];
  gMiniPanelShown = self.popover.isShown ? 1 : 0;
}

// Replace the panel content while it is visible.
- (void)reloadWithHTML:(NSString *)html {
  if (!self.popover.isShown || self.webView == nil) {
    return;
  }
  [self.webView loadHTMLString:html baseURL:nil];
}

// Bridge interactions from the page: "resize" adjusts the popover, everything
// else is forwarded to Go.
- (void)userContentController:(WKUserContentController *)userContentController
      didReceiveScriptMessage:(WKScriptMessage *)message {
  NSDictionary *body = message.body;
  if (![body isKindOfClass:[NSDictionary class]]) {
    return;
  }

  NSString *action = body[@"action"];
  if ([action isEqualToString:@"resize"]) {
    NSNumber *heightValue = body[@"height"];
    CGFloat height = heightValue ? [heightValue doubleValue] : 0;
    if (height <= 0) {
      return;
    }
    height = MAX(120.0, MIN(height, 640.0));
    self.popover.contentSize = NSMakeSize(340, height);
    return;
  }

  int accountID = [body[@"accountId"] intValue];
  quotapanelMiniPanelAction((char *)[action UTF8String], accountID);
}

- (void)popoverDidClose:(NSNotification *)notification {
  self.lastCloseTime = [NSDate timeIntervalSinceReferenceDate];
  gMiniPanelShown = 0;
}

@end

static QPMiniPanelController *gMiniPanelController = nil;

static QPMiniPanelController *qpMiniPanel(void) {
  if (gMiniPanelController == nil) {
    gMiniPanelController = [[QPMiniPanelController alloc] init];
    [[NSNotificationCenter defaultCenter] addObserver:gMiniPanelController
                                            selector:@selector(popoverDidClose:)
                                                name:NSPopoverDidCloseNotification
                                              object:nil];
  }
  return gMiniPanelController;
}

// Show the mini panel with the given HTML and initial height.
void qpMiniPanelShow(const char *html, int height) {
  NSString *content = html ? [NSString stringWithUTF8String:html] : @"";
  dispatch_async(dispatch_get_main_queue(), ^{
    [qpMiniPanel() toggleWithHTML:content height:(CGFloat)height];
  });
}

// Refresh the mini panel content (no-op when it is hidden).
void qpMiniPanelReload(const char *html) {
  NSString *content = html ? [NSString stringWithUTF8String:html] : @"";
  dispatch_async(dispatch_get_main_queue(), ^{
    [qpMiniPanel() reloadWithHTML:content];
  });
}

// Close the mini panel.
void qpMiniPanelHide(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    QPMiniPanelController *panel = qpMiniPanel();
    if (panel.popover.isShown) {
      [panel.popover close];
    }
  });
}

// Whether the mini panel is currently visible (safe from any thread; reads the
// flag maintained on the main thread instead of dispatching synchronously).
int qpMiniPanelIsShown(void) {
  return gMiniPanelShown;
}

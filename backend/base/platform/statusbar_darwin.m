#import <Cocoa/Cocoa.h>

// Exported from Go (//export in statusbar_darwin.go) to route menu bar clicks to app logic.
extern void quotapanelStatusBarLeftClick(void);
extern void quotapanelStatusBarShow(void);
extern void quotapanelStatusBarRestart(void);
extern void quotapanelStatusBarQuit(void);

// QPStatusBarController owns the menu bar status item and its right-click menu.
@interface QPStatusBarController : NSObject
@property(strong) NSStatusItem *statusItem;
@property(strong) NSMenu *menu;
@end

@implementation QPStatusBarController

// Status item click: left button shows the main window, right button pops up the menu.
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

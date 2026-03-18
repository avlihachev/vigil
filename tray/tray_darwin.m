#import <Cocoa/Cocoa.h>

extern void trayOnClick(void);

static NSStatusItem *statusItem = nil;
static NSMenu *contextMenu = nil;

@interface TrayClickTarget : NSObject
- (void)clicked:(id)sender;
@end

@implementation TrayClickTarget
- (void)clicked:(id)sender {
    NSEvent *event = [NSApp currentEvent];
    if (event.type == NSEventTypeRightMouseUp) {
        [statusItem setMenu:contextMenu];
        [statusItem.button performClick:nil];
        [statusItem setMenu:nil];
    } else {
        trayOnClick();
    }
}
@end

static TrayClickTarget *clickTarget = nil;

void tray_init(const char* title, const char* tooltip) {
    dispatch_async(dispatch_get_main_queue(), ^{
        statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
        statusItem.button.title = [NSString stringWithUTF8String:title];
        statusItem.button.toolTip = [NSString stringWithUTF8String:tooltip];

        clickTarget = [[TrayClickTarget alloc] init];
        [statusItem.button setTarget:clickTarget];
        [statusItem.button setAction:@selector(clicked:)];
        [statusItem.button sendActionOn:(NSEventMaskLeftMouseUp | NSEventMaskRightMouseUp)];

        contextMenu = [[NSMenu alloc] init];
        NSMenuItem *showItem = [contextMenu addItemWithTitle:@"Show / Hide"
                                                      action:@selector(clicked:)
                                               keyEquivalent:@""];
        [showItem setTarget:clickTarget];
        [contextMenu addItem:[NSMenuItem separatorItem]];
        NSMenuItem *quitItem = [contextMenu addItemWithTitle:@"Quit"
                                                      action:@selector(terminate:)
                                               keyEquivalent:@"q"];
        [quitItem setTarget:[NSApplication sharedApplication]];
    });
}

void tray_set_title(const char* title) {
    NSString *t = [NSString stringWithUTF8String:title];
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem != nil) {
            statusItem.button.title = t;
        }
    });
}

void tray_remove(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem != nil) {
            [[NSStatusBar systemStatusBar] removeStatusItem:statusItem];
            statusItem = nil;
        }
    });
}

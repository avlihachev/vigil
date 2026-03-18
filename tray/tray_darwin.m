#import <Cocoa/Cocoa.h>
#include <stdio.h>

extern void trayOnClick(void);

static NSStatusItem *statusItem = nil;

@interface TrayDelegate : NSObject
- (void)statusItemClicked:(id)sender;
- (void)quitApp:(id)sender;
@end

@implementation TrayDelegate
- (void)statusItemClicked:(id)sender {
    trayOnClick();
}
- (void)quitApp:(id)sender {
    [NSApp terminate:nil];
}
@end

static TrayDelegate *delegate = nil;

void tray_init(const char* title, const char* tooltip) {
    NSString *titleStr = [NSString stringWithUTF8String:title];
    NSString *tooltipStr = [NSString stringWithUTF8String:tooltip];

    // slight delay to let Wails finish NSApp setup
    dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(0.5 * NSEC_PER_SEC)), dispatch_get_main_queue(), ^{
        // hide dock icon
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];

        delegate = [[TrayDelegate alloc] init];

        statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
        [statusItem retain];
        statusItem.button.title = titleStr;
        statusItem.button.toolTip = tooltipStr;
        [statusItem.button setTarget:delegate];
        [statusItem.button setAction:@selector(statusItemClicked:)];
        [statusItem.button sendActionOn:NSEventMaskLeftMouseUp];

        NSMenu *menu = [[NSMenu alloc] init];
        NSMenuItem *showItem = [[NSMenuItem alloc] initWithTitle:@"Show / Hide"
                                                           action:@selector(statusItemClicked:)
                                                    keyEquivalent:@""];
        [showItem setTarget:delegate];
        [menu addItem:showItem];
        [menu addItem:[NSMenuItem separatorItem]];
        NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"Quit"
                                                           action:@selector(quitApp:)
                                                    keyEquivalent:@"q"];
        [quitItem setTarget:delegate];
        [menu addItem:quitItem];

        // right-click shows menu, left-click calls action
        statusItem.menu = nil;

        fprintf(stderr, "[tray] status item created: %s\n", [titleStr UTF8String]);
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
            [statusItem release];
            statusItem = nil;
        }
    });
}

#import <Cocoa/Cocoa.h>
#include <stdio.h>

extern void trayOnClick(void);
extern void trayOnQuit(void);

static NSStatusItem *statusItem = nil;

@interface TrayDelegate : NSObject {
    NSMenu *_contextMenu;
}
- (void)statusItemClicked:(id)sender;
- (void)quitApp:(id)sender;
- (void)setContextMenu:(NSMenu *)menu;
@end

@implementation TrayDelegate
- (void)setContextMenu:(NSMenu *)menu {
    _contextMenu = menu;
}
- (void)statusItemClicked:(id)sender {
    NSEvent *event = [NSApp currentEvent];
    BOOL isRight = (event.type == NSEventTypeRightMouseUp) ||
                   (event.modifierFlags & NSEventModifierFlagControl);
    if (isRight && _contextMenu) {
        NSPoint loc = NSMakePoint(0, statusItem.button.bounds.size.height + 4);
        [_contextMenu popUpMenuPositioningItem:nil atLocation:loc inView:statusItem.button];
    } else {
        trayOnClick();
    }
}
- (void)quitApp:(id)sender {
    trayOnQuit();
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

        // left click → toggle popup; right/ctrl click → context menu with Quit
        [delegate setContextMenu:menu];
        [statusItem.button setAction:@selector(statusItemClicked:)];
        [statusItem.button sendActionOn:NSEventMaskLeftMouseUp | NSEventMaskRightMouseUp];
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

// show popup: activate app so first click works, float above other windows
void tray_show_popup(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        for (NSWindow *win in [NSApp windows]) {
            [win setLevel:NSFloatingWindowLevel];
            [NSApp activateIgnoringOtherApps:YES];
            [win makeKeyAndOrderFront:nil];
            break;
        }
    });
}

// hide popup via orderOut (not [NSApp hide:]) so macOS doesn't auto-switch to Ghostty
// dispatch_sync ensures the window is hidden before ActivateSession runs its AppleScript
void tray_hide_popup(void) {
    dispatch_sync(dispatch_get_main_queue(), ^{
        for (NSWindow *win in [NSApp windows]) {
            [win orderOut:nil];
            break;
        }
    });
}

void traySetBadge(int count) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem == nil) return;

        if (count <= 0) {
            statusItem.button.image = nil;
            statusItem.button.title = @"◉";
            return;
        }

        // render a small colored square with the count number as the button image
        NSSize size = NSMakeSize(22, 18);
        NSImage *img = [[NSImage alloc] initWithSize:size];
        [img lockFocus];

        NSString *base = @"◉";
        NSDictionary *baseAttrs = @{
            NSFontAttributeName: [NSFont systemFontOfSize:13],
            NSForegroundColorAttributeName: [NSColor labelColor],
        };
        [base drawAtPoint:NSMakePoint(0, 2) withAttributes:baseAttrs];

        NSString *label = [NSString stringWithFormat:@"%d", count > 9 ? 9 : count];
        NSDictionary *attrs = @{
            NSFontAttributeName: [NSFont boldSystemFontOfSize:8],
            NSForegroundColorAttributeName: [NSColor whiteColor],
        };
        NSSize textSize = [label sizeWithAttributes:attrs];
        CGFloat badgeW = MAX(textSize.width + 4, 11);
        NSRect badgeRect = NSMakeRect(size.width - badgeW, size.height - 11, badgeW, 11);

        [[NSColor colorWithRed:0.94 green:0.50 blue:0.0 alpha:1.0] setFill];
        NSBezierPath *path = [NSBezierPath bezierPathWithRoundedRect:badgeRect xRadius:3 yRadius:3];
        [path fill];

        NSPoint textPoint = NSMakePoint(
            badgeRect.origin.x + (badgeW - textSize.width) / 2,
            badgeRect.origin.y + (11 - textSize.height) / 2
        );
        [label drawAtPoint:textPoint withAttributes:attrs];

        [img unlockFocus];
        [img setTemplate:NO];
        statusItem.button.image = img;
        statusItem.button.title = @"";
    });
}

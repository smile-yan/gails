//go:build darwin && !ios

package notifications

/*
#cgo CFLAGS:-x objective-c
#cgo LDFLAGS: -framework Foundation -framework Cocoa

#if __MAC_OS_X_VERSION_MAX_ALLOWED >= 110000
#cgo LDFLAGS: -framework UserNotifications
#endif

#import "./notifications_darwin.h"
*/
import "C"
import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/gailsapp/gails/pkg/application"
)

var (
	cIsNotificationAvailable     = func() bool { return bool(C.isNotificationAvailable()) }
	cCheckBundleIdentifier       = func() bool { return bool(C.checkBundleIdentifier()) }
	cEnsureDelegateInitialized   = func() bool { return bool(C.ensureDelegateInitialized()) }
	cRequestNotificationAuthorization = func(id int) { C.requestNotificationAuthorization(C.int(id)) }
	cCheckNotificationAuthorization   = func(id int) { C.checkNotificationAuthorization(C.int(id)) }

	requestNotificationAuthorizationTimeout = 180 * time.Second
	checkNotificationAuthorizationTimeout   = 15 * time.Second
	notificationTimeout                     = 5 * time.Second
	categoryTimeout                         = 5 * time.Second

	jsonMarshal = json.Marshal

	cSendNotification = func(id int, identifier, title, subtitle, body, dataJSON string) {
		cIdentifier := C.CString(identifier)
		cTitle := C.CString(title)
		cSubtitle := C.CString(subtitle)
		cBody := C.CString(body)
		defer C.free(unsafe.Pointer(cIdentifier))
		defer C.free(unsafe.Pointer(cTitle))
		defer C.free(unsafe.Pointer(cSubtitle))
		defer C.free(unsafe.Pointer(cBody))

		var cDataJSON *C.char
		if dataJSON != "" {
			cDataJSON = C.CString(dataJSON)
			defer C.free(unsafe.Pointer(cDataJSON))
		}

		C.sendNotification(C.int(id), cIdentifier, cTitle, cSubtitle, cBody, cDataJSON)
	}

	cSendNotificationWithActions = func(id int, identifier, title, subtitle, body, categoryID, dataJSON string) {
		cIdentifier := C.CString(identifier)
		cTitle := C.CString(title)
		cSubtitle := C.CString(subtitle)
		cBody := C.CString(body)
		cCategoryID := C.CString(categoryID)
		defer C.free(unsafe.Pointer(cIdentifier))
		defer C.free(unsafe.Pointer(cTitle))
		defer C.free(unsafe.Pointer(cSubtitle))
		defer C.free(unsafe.Pointer(cBody))
		defer C.free(unsafe.Pointer(cCategoryID))

		var cDataJSON *C.char
		if dataJSON != "" {
			cDataJSON = C.CString(dataJSON)
			defer C.free(unsafe.Pointer(cDataJSON))
		}

		C.sendNotificationWithActions(C.int(id), cIdentifier, cTitle, cSubtitle, cBody, cCategoryID, cDataJSON)
	}

	cRegisterNotificationCategory = func(id int, categoryID, actionsJSON string, hasReplyField bool, replyPlaceholder, replyButtonTitle string) {
		cCategoryID := C.CString(categoryID)
		cActionsJSON := C.CString(actionsJSON)
		defer C.free(unsafe.Pointer(cCategoryID))
		defer C.free(unsafe.Pointer(cActionsJSON))

		var cReplyPlaceholder, cReplyButtonTitle *C.char
		if hasReplyField {
			cReplyPlaceholder = C.CString(replyPlaceholder)
			cReplyButtonTitle = C.CString(replyButtonTitle)
			defer C.free(unsafe.Pointer(cReplyPlaceholder))
			defer C.free(unsafe.Pointer(cReplyButtonTitle))
		}

		C.registerNotificationCategory(C.int(id), cCategoryID, cActionsJSON, C.bool(hasReplyField),
			cReplyPlaceholder, cReplyButtonTitle)
	}

	cRemoveNotificationCategory = func(id int, categoryID string) {
		cCategoryID := C.CString(categoryID)
		defer C.free(unsafe.Pointer(cCategoryID))
		C.removeNotificationCategory(C.int(id), cCategoryID)
	}

	cRemoveAllPendingNotifications = func() { C.removeAllPendingNotifications() }
	cRemovePendingNotification       = func(identifier string) {
		cIdentifier := C.CString(identifier)
		defer C.free(unsafe.Pointer(cIdentifier))
		C.removePendingNotification(cIdentifier)
	}
	cRemoveAllDeliveredNotifications = func() { C.removeAllDeliveredNotifications() }
	cRemoveDeliveredNotification     = func(identifier string) {
		cIdentifier := C.CString(identifier)
		defer C.free(unsafe.Pointer(cIdentifier))
		C.removeDeliveredNotification(cIdentifier)
	}
)

type darwinNotifier struct {
	channels      map[int]chan notificationChannel
	channelsLock  sync.Mutex
	nextChannelID int
}

type notificationChannel struct {
	Success bool
	Error   error
}

type ChannelHandler interface {
	GetChannel(id int) (chan notificationChannel, bool)
}

const AppleDefaultActionIdentifier = "com.apple.UNNotificationDefaultActionIdentifier"

// Creates a new Notifications Service.
// Your app must be packaged and signed for this feature to work.
func New() *NotificationService {
	notificationServiceOnce.Do(func() {
		impl := &darwinNotifier{
			channels:      make(map[int]chan notificationChannel),
			nextChannelID: 0,
		}

		NotificationService_ = &NotificationService{
			impl: impl,
		}
	})

	return NotificationService_
}

func (dn *darwinNotifier) Startup(ctx context.Context, options application.ServiceOptions) error {
	if !cIsNotificationAvailable() {
		return fmt.Errorf("notifications are not available on this system")
	}
	if !cCheckBundleIdentifier() {
		return fmt.Errorf("notifications require a valid bundle identifier")
	}
	if !cEnsureDelegateInitialized() {
		return fmt.Errorf("failed to initialize notification center delegate")
	}
	return nil
}

func (dn *darwinNotifier) Shutdown() error {
	return nil
}

// isNotificationAvailable checks if notifications are available on the system.
func isNotificationAvailable() bool {
	return cIsNotificationAvailable()
}

func checkBundleIdentifier() bool {
	return cCheckBundleIdentifier()
}

// RequestNotificationAuthorization requests permission for notifications.
// Default timeout is 3 minutes
func (dn *darwinNotifier) RequestNotificationAuthorization() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestNotificationAuthorizationTimeout)
	defer cancel()

	id, resultCh := dn.registerChannel()

	cRequestNotificationAuthorization(id)

	select {
	case result := <-resultCh:
		return result.Success, result.Error
	case <-ctx.Done():
		dn.cleanupChannel(id)
		return false, fmt.Errorf("notification authorization timed out after 3 minutes: %w", ctx.Err())
	}
}

// CheckNotificationAuthorization checks current notification permission status.
func (dn *darwinNotifier) CheckNotificationAuthorization() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), checkNotificationAuthorizationTimeout)
	defer cancel()

	id, resultCh := dn.registerChannel()

	cCheckNotificationAuthorization(id)

	select {
	case result := <-resultCh:
		return result.Success, result.Error
	case <-ctx.Done():
		dn.cleanupChannel(id)
		return false, fmt.Errorf("notification authorization timed out after 15s: %w", ctx.Err())
	}
}

// SendNotification sends a basic notification with a unique identifier, title, subtitle, and body.
func (dn *darwinNotifier) SendNotification(options NotificationOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), notificationTimeout)
	defer cancel()

	var dataJSON string
	if options.Data != nil {
		jsonData, err := jsonMarshal(options.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal notification data: %w", err)
		}
		dataJSON = string(jsonData)
	}

	id, resultCh := dn.registerChannel()
	cSendNotification(id, options.ID, options.Title, options.Subtitle, options.Body, dataJSON)

	select {
	case result := <-resultCh:
		if !result.Success {
			if result.Error != nil {
				return result.Error
			}
			return fmt.Errorf("sending notification failed")
		}
		return nil
	case <-ctx.Done():
		dn.cleanupChannel(id)
		return fmt.Errorf("sending notification timed out: %w", ctx.Err())
	}
}

// SendNotificationWithActions sends a notification with additional actions and inputs.
// A NotificationCategory must be registered with RegisterNotificationCategory first. The `CategoryID` must match the registered category.
// If a NotificationCategory is not registered a basic notification will be sent.
func (dn *darwinNotifier) SendNotificationWithActions(options NotificationOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), notificationTimeout)
	defer cancel()

	var dataJSON string
	if options.Data != nil {
		jsonData, err := jsonMarshal(options.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal notification data: %w", err)
		}
		dataJSON = string(jsonData)
	}

	id, resultCh := dn.registerChannel()
	cSendNotificationWithActions(id, options.ID, options.Title, options.Subtitle, options.Body, options.CategoryID, dataJSON)

	select {
	case result := <-resultCh:
		if !result.Success {
			if result.Error != nil {
				return result.Error
			}
			return fmt.Errorf("sending notification failed")
		}
		return nil
	case <-ctx.Done():
		dn.cleanupChannel(id)
		return fmt.Errorf("sending notification timed out: %w", ctx.Err())
	}
}

// RegisterNotificationCategory registers a new NotificationCategory to be used with SendNotificationWithActions.
// Registering a category with the same name as a previously registered NotificationCategory will override it.
func (dn *darwinNotifier) RegisterNotificationCategory(category NotificationCategory) error {
	ctx, cancel := context.WithTimeout(context.Background(), categoryTimeout)
	defer cancel()

	actionsJSON, err := jsonMarshal(category.Actions)
	if err != nil {
		return fmt.Errorf("failed to marshal notification category: %w", err)
	}

	id, resultCh := dn.registerChannel()
	cRegisterNotificationCategory(id, category.ID, string(actionsJSON), category.HasReplyField,
		category.ReplyPlaceholder, category.ReplyButtonTitle)

	select {
	case result := <-resultCh:
		if !result.Success {
			if result.Error != nil {
				return result.Error
			}
			return fmt.Errorf("category registration failed")
		}
		return nil
	case <-ctx.Done():
		dn.cleanupChannel(id)
		return fmt.Errorf("category registration timed out: %w", ctx.Err())
	}
}

// RemoveNotificationCategory remove a previously registered NotificationCategory.
func (dn *darwinNotifier) RemoveNotificationCategory(categoryId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), categoryTimeout)
	defer cancel()

	id, resultCh := dn.registerChannel()
	cRemoveNotificationCategory(id, categoryId)

	select {
	case result := <-resultCh:
		if !result.Success {
			if result.Error != nil {
				return result.Error
			}
			return fmt.Errorf("category removal failed")
		}
		return nil
	case <-ctx.Done():
		dn.cleanupChannel(id)
		return fmt.Errorf("category removal timed out: %w", ctx.Err())
	}
}

// RemoveAllPendingNotifications removes all pending notifications.
func (dn *darwinNotifier) RemoveAllPendingNotifications() error {
	cRemoveAllPendingNotifications()
	return nil
}

// RemovePendingNotification removes a pending notification matching the unique identifier.
func (dn *darwinNotifier) RemovePendingNotification(identifier string) error {
	cRemovePendingNotification(identifier)
	return nil
}

// RemoveAllDeliveredNotifications removes all delivered notifications.
func (dn *darwinNotifier) RemoveAllDeliveredNotifications() error {
	cRemoveAllDeliveredNotifications()
	return nil
}

// RemoveDeliveredNotification removes a delivered notification matching the unique identifier.
func (dn *darwinNotifier) RemoveDeliveredNotification(identifier string) error {
	cRemoveDeliveredNotification(identifier)
	return nil
}

// RemoveNotification is a macOS stub that always returns nil.
// Use one of the following instead:
// RemoveAllPendingNotifications
// RemovePendingNotification
// RemoveAllDeliveredNotifications
// RemoveDeliveredNotification
// (Linux-specific)
func (dn *darwinNotifier) RemoveNotification(identifier string) error {
	return nil
}

// testCaptureResult exposes captureResult to tests without requiring cgo in test files.
func testCaptureResult(id int, success bool, errorMsg string) {
	var cErr *C.char
	if errorMsg != "" {
		cErr = C.CString(errorMsg)
		defer C.free(unsafe.Pointer(cErr))
	}
	captureResult(C.int(id), C.bool(success), cErr)
}

// testDidReceiveNotificationResponse exposes didReceiveNotificationResponse to tests.
func testDidReceiveNotificationResponse(payload *string, err *string) {
	var cPayload, cErr *C.char
	if payload != nil {
		cPayload = C.CString(*payload)
		defer C.free(unsafe.Pointer(cPayload))
	}
	if err != nil {
		cErr = C.CString(*err)
		defer C.free(unsafe.Pointer(cErr))
	}
	didReceiveNotificationResponse(cPayload, cErr)
}

//export captureResult
func captureResult(channelID C.int, success C.bool, errorMsg *C.char) {
	ns := getNotificationService()
	if ns == nil {
		return
	}

	handler, ok := ns.impl.(ChannelHandler)
	if !ok {
		return
	}

	resultCh, exists := handler.GetChannel(int(channelID))
	if !exists {
		return
	}

	var err error
	if errorMsg != nil {
		err = fmt.Errorf("%s", C.GoString(errorMsg))
	}

	resultCh <- notificationChannel{
		Success: bool(success),
		Error:   err,
	}

	close(resultCh)
}

//export didReceiveNotificationResponse
func didReceiveNotificationResponse(jsonPayload *C.char, err *C.char) {
	result := NotificationResult{}

	if err != nil {
		errMsg := C.GoString(err)
		result.Error = fmt.Errorf("notification response error: %s", errMsg)
		if ns := getNotificationService(); ns != nil {
			ns.handleNotificationResult(result)
		}
		return
	}

	if jsonPayload == nil {
		result.Error = fmt.Errorf("received nil JSON payload in notification response")
		if ns := getNotificationService(); ns != nil {
			ns.handleNotificationResult(result)
		}
		return
	}

	payload := C.GoString(jsonPayload)

	var response NotificationResponse
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		result.Error = fmt.Errorf("failed to unmarshal notification response: %w", err)
		if ns := getNotificationService(); ns != nil {
			ns.handleNotificationResult(result)
		}
		return
	}

	if response.ActionIdentifier == AppleDefaultActionIdentifier {
		response.ActionIdentifier = DefaultActionIdentifier
	}

	result.Response = response
	if ns := getNotificationService(); ns != nil {
		ns.handleNotificationResult(result)
	}
}

// Helper methods

func (dn *darwinNotifier) registerChannel() (int, chan notificationChannel) {
	dn.channelsLock.Lock()
	defer dn.channelsLock.Unlock()

	id := dn.nextChannelID
	dn.nextChannelID++

	resultCh := make(chan notificationChannel, 1)

	dn.channels[id] = resultCh
	return id, resultCh
}

func (dn *darwinNotifier) GetChannel(id int) (chan notificationChannel, bool) {
	dn.channelsLock.Lock()
	defer dn.channelsLock.Unlock()

	ch, exists := dn.channels[id]
	if exists {
		delete(dn.channels, id)
	}
	return ch, exists
}

func (dn *darwinNotifier) cleanupChannel(id int) {
	dn.channelsLock.Lock()
	defer dn.channelsLock.Unlock()

	if ch, exists := dn.channels[id]; exists {
		delete(dn.channels, id)
		close(ch)
	}
}

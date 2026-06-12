//go:build darwin && !ios

package notifications

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gailsapp/gails/pkg/application"
)

func setCFuncs(t *testing.T, replacements map[string]any) {
	t.Helper()
	orig := map[string]any{
		"cIsNotificationAvailable":          cIsNotificationAvailable,
		"cCheckBundleIdentifier":            cCheckBundleIdentifier,
		"cEnsureDelegateInitialized":        cEnsureDelegateInitialized,
		"cRequestNotificationAuthorization": cRequestNotificationAuthorization,
		"cCheckNotificationAuthorization":   cCheckNotificationAuthorization,
		"cSendNotification":                 cSendNotification,
		"cSendNotificationWithActions":      cSendNotificationWithActions,
		"cRegisterNotificationCategory":     cRegisterNotificationCategory,
		"cRemoveNotificationCategory":       cRemoveNotificationCategory,
		"cRemoveAllPendingNotifications":    cRemoveAllPendingNotifications,
		"cRemovePendingNotification":        cRemovePendingNotification,
		"cRemoveAllDeliveredNotifications":  cRemoveAllDeliveredNotifications,
		"cRemoveDeliveredNotification":      cRemoveDeliveredNotification,
	}
	for k, v := range replacements {
		switch k {
		case "cIsNotificationAvailable":
			cIsNotificationAvailable = v.(func() bool)
		case "cCheckBundleIdentifier":
			cCheckBundleIdentifier = v.(func() bool)
		case "cEnsureDelegateInitialized":
			cEnsureDelegateInitialized = v.(func() bool)
		case "cRequestNotificationAuthorization":
			cRequestNotificationAuthorization = v.(func(int))
		case "cCheckNotificationAuthorization":
			cCheckNotificationAuthorization = v.(func(int))
		case "cSendNotification":
			cSendNotification = v.(func(int, string, string, string, string, string))
		case "cSendNotificationWithActions":
			cSendNotificationWithActions = v.(func(int, string, string, string, string, string, string))
		case "cRegisterNotificationCategory":
			cRegisterNotificationCategory = v.(func(int, string, string, bool, string, string))
		case "cRemoveNotificationCategory":
			cRemoveNotificationCategory = v.(func(int, string))
		case "cRemoveAllPendingNotifications":
			cRemoveAllPendingNotifications = v.(func())
		case "cRemovePendingNotification":
			cRemovePendingNotification = v.(func(string))
		case "cRemoveAllDeliveredNotifications":
			cRemoveAllDeliveredNotifications = v.(func())
		case "cRemoveDeliveredNotification":
			cRemoveDeliveredNotification = v.(func(string))
		}
	}
	t.Cleanup(func() {
		for k, v := range orig {
			switch k {
			case "cIsNotificationAvailable":
				cIsNotificationAvailable = v.(func() bool)
			case "cCheckBundleIdentifier":
				cCheckBundleIdentifier = v.(func() bool)
			case "cEnsureDelegateInitialized":
				cEnsureDelegateInitialized = v.(func() bool)
			case "cRequestNotificationAuthorization":
				cRequestNotificationAuthorization = v.(func(int))
			case "cCheckNotificationAuthorization":
				cCheckNotificationAuthorization = v.(func(int))
			case "cSendNotification":
				cSendNotification = v.(func(int, string, string, string, string, string))
			case "cSendNotificationWithActions":
				cSendNotificationWithActions = v.(func(int, string, string, string, string, string, string))
			case "cRegisterNotificationCategory":
				cRegisterNotificationCategory = v.(func(int, string, string, bool, string, string))
			case "cRemoveNotificationCategory":
				cRemoveNotificationCategory = v.(func(int, string))
			case "cRemoveAllPendingNotifications":
				cRemoveAllPendingNotifications = v.(func())
			case "cRemovePendingNotification":
				cRemovePendingNotification = v.(func(string))
			case "cRemoveAllDeliveredNotifications":
				cRemoveAllDeliveredNotifications = v.(func())
			case "cRemoveDeliveredNotification":
				cRemoveDeliveredNotification = v.(func(string))
			}
		}
	})
}

func newDarwinNotifier(t *testing.T) *darwinNotifier {
	t.Helper()
	dn := &darwinNotifier{
		channels:      make(map[int]chan notificationChannel),
		nextChannelID: 0,
	}
	NotificationService_ = &NotificationService{impl: dn}
	t.Cleanup(func() { NotificationService_ = nil })
	return dn
}

func TestDarwinNotifier_New(t *testing.T) {
	resetNotificationService()
	ns1 := New()
	ns2 := New()
	if ns1 != ns2 {
		t.Fatal("New() should return the same singleton")
	}
	if ns1.impl == nil {
		t.Fatal("singleton impl is nil")
	}
}

func TestDarwinNotifier_Startup_NotAvailable(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cIsNotificationAvailable": func() bool { return false },
	})
	dn := newDarwinNotifier(t)
	if err := dn.Startup(context.Background(), application.ServiceOptions{}); err == nil {
		t.Fatal("expected error when notifications unavailable")
	}
}

func TestDarwinNotifier_Startup_NoBundle(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cIsNotificationAvailable": func() bool { return true },
		"cCheckBundleIdentifier":   func() bool { return false },
	})
	dn := newDarwinNotifier(t)
	if err := dn.Startup(context.Background(), application.ServiceOptions{}); err == nil {
		t.Fatal("expected error when bundle invalid")
	}
}

func TestDarwinNotifier_Startup_DelegateFail(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cIsNotificationAvailable":   func() bool { return true },
		"cCheckBundleIdentifier":     func() bool { return true },
		"cEnsureDelegateInitialized": func() bool { return false },
	})
	dn := newDarwinNotifier(t)
	if err := dn.Startup(context.Background(), application.ServiceOptions{}); err == nil {
		t.Fatal("expected error when delegate initialization fails")
	}
}

func TestDarwinNotifier_Startup_Success(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cIsNotificationAvailable":   func() bool { return true },
		"cCheckBundleIdentifier":     func() bool { return true },
		"cEnsureDelegateInitialized": func() bool { return true },
	})
	dn := newDarwinNotifier(t)
	if err := dn.Startup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatalf("Startup: %v", err)
	}
}

func TestDarwinNotifier_Shutdown(t *testing.T) {
	dn := newDarwinNotifier(t)
	if err := dn.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestDarwinNotifier_RequestNotificationAuthorization_Success(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cRequestNotificationAuthorization": func(id int) { testCaptureResult(id, true, "") },
	})
	dn := newDarwinNotifier(t)
	ok, err := dn.RequestNotificationAuthorization()
	if err != nil {
		t.Fatalf("RequestNotificationAuthorization: %v", err)
	}
	if !ok {
		t.Fatal("expected true")
	}
}

func setShortTimeouts(t *testing.T) {
	origRequest := requestNotificationAuthorizationTimeout
	origCheck := checkNotificationAuthorizationTimeout
	origNotification := notificationTimeout
	origCategory := categoryTimeout
	requestNotificationAuthorizationTimeout = 50 * time.Millisecond
	checkNotificationAuthorizationTimeout = 50 * time.Millisecond
	notificationTimeout = 50 * time.Millisecond
	categoryTimeout = 50 * time.Millisecond
	t.Cleanup(func() {
		requestNotificationAuthorizationTimeout = origRequest
		checkNotificationAuthorizationTimeout = origCheck
		notificationTimeout = origNotification
		categoryTimeout = origCategory
	})
}

func TestDarwinNotifier_RequestNotificationAuthorization_Timeout(t *testing.T) {
	setShortTimeouts(t)
	setCFuncs(t, map[string]any{
		"cRequestNotificationAuthorization": func(id int) {},
	})
	dn := newDarwinNotifier(t)
	start := time.Now()
	_, err := dn.RequestNotificationAuthorization()
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("timeout took too long")
	}
}

func TestDarwinNotifier_CheckNotificationAuthorization_Success(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cCheckNotificationAuthorization": func(id int) { testCaptureResult(id, true, "") },
	})
	dn := newDarwinNotifier(t)
	ok, err := dn.CheckNotificationAuthorization()
	if err != nil {
		t.Fatalf("CheckNotificationAuthorization: %v", err)
	}
	if !ok {
		t.Fatal("expected true")
	}
}

func TestDarwinNotifier_CheckNotificationAuthorization_Timeout(t *testing.T) {
	setShortTimeouts(t)
	setCFuncs(t, map[string]any{
		"cCheckNotificationAuthorization": func(id int) {},
	})
	dn := newDarwinNotifier(t)
	_, err := dn.CheckNotificationAuthorization()
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestDarwinNotifier_SendNotification_Success(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cSendNotification": func(id int, identifier, title, subtitle, body, dataJSON string) {
			testCaptureResult(id, true, "")
		},
	})
	dn := newDarwinNotifier(t)
	if err := dn.SendNotification(NotificationOptions{ID: "1", Title: "T", Subtitle: "S", Body: "B", Data: map[string]any{"k": "v"}}); err != nil {
		t.Fatalf("SendNotification: %v", err)
	}
}

func TestDarwinNotifier_SendNotification_MarshalError(t *testing.T) {
	dn := newDarwinNotifier(t)
	if err := dn.SendNotification(NotificationOptions{ID: "1", Title: "T", Data: map[string]any{"ch": make(chan int)}}); err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestDarwinNotifier_SendNotification_Timeout(t *testing.T) {
	setShortTimeouts(t)
	setCFuncs(t, map[string]any{
		"cSendNotification": func(id int, identifier, title, subtitle, body, dataJSON string) {},
	})
	dn := newDarwinNotifier(t)
	if err := dn.SendNotification(NotificationOptions{ID: "1", Title: "T"}); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestDarwinNotifier_SendNotification_Failure(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cSendNotification": func(id int, identifier, title, subtitle, body, dataJSON string) {
			testCaptureResult(id, false, "")
		},
	})
	dn := newDarwinNotifier(t)
	if err := dn.SendNotification(NotificationOptions{ID: "1", Title: "T"}); err == nil {
		t.Fatal("expected failure error")
	}
}

func TestDarwinNotifier_SendNotification_ErrorMessage(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cSendNotification": func(id int, identifier, title, subtitle, body, dataJSON string) {
			testCaptureResult(id, false, "boom")
		},
	})
	dn := newDarwinNotifier(t)
	if err := dn.SendNotification(NotificationOptions{ID: "1", Title: "T"}); err == nil || err.Error() != "boom" {
		t.Fatalf("expected boom error, got %v", err)
	}
}

func TestDarwinNotifier_SendNotificationWithActions_ErrorMessage(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cSendNotificationWithActions": func(id int, identifier, title, subtitle, body, categoryID, dataJSON string) {
			testCaptureResult(id, false, "boom")
		},
	})
	dn := newDarwinNotifier(t)
	if err := dn.SendNotificationWithActions(NotificationOptions{ID: "1", Title: "T", CategoryID: "cat"}); err == nil || err.Error() != "boom" {
		t.Fatalf("expected boom error, got %v", err)
	}
}

func setJSONMarshal(t *testing.T, f func(any) ([]byte, error)) {
	t.Helper()
	orig := jsonMarshal
	jsonMarshal = f
	t.Cleanup(func() { jsonMarshal = orig })
}

func TestDarwinNotifier_SendNotificationWithActions_Success(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cSendNotificationWithActions": func(id int, identifier, title, subtitle, body, categoryID, dataJSON string) {
			if dataJSON == "" {
				t.Fatal("expected dataJSON to be sent")
			}
			testCaptureResult(id, true, "")
		},
	})
	dn := newDarwinNotifier(t)
	if err := dn.SendNotificationWithActions(NotificationOptions{ID: "1", Title: "T", CategoryID: "cat", Data: map[string]any{"k": "v"}}); err != nil {
		t.Fatalf("SendNotificationWithActions: %v", err)
	}
}

func TestDarwinNotifier_SendNotificationWithActions_Failure(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cSendNotificationWithActions": func(id int, identifier, title, subtitle, body, categoryID, dataJSON string) {
			testCaptureResult(id, false, "")
		},
	})
	dn := newDarwinNotifier(t)
	if err := dn.SendNotificationWithActions(NotificationOptions{ID: "1", Title: "T", CategoryID: "cat"}); err == nil {
		t.Fatal("expected failure error")
	}
}

func TestDarwinNotifier_SendNotificationWithActions_Timeout(t *testing.T) {
	setShortTimeouts(t)
	setCFuncs(t, map[string]any{
		"cSendNotificationWithActions": func(id int, identifier, title, subtitle, body, categoryID, dataJSON string) {},
	})
	dn := newDarwinNotifier(t)
	if err := dn.SendNotificationWithActions(NotificationOptions{ID: "1", Title: "T"}); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestDarwinNotifier_SendNotificationWithActions_MarshalError(t *testing.T) {
	dn := newDarwinNotifier(t)
	if err := dn.SendNotificationWithActions(NotificationOptions{ID: "1", Title: "T", Data: map[string]any{"ch": make(chan int)}}); err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestDarwinNotifier_RegisterNotificationCategory_Success(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cRegisterNotificationCategory": func(id int, categoryID, actionsJSON string, hasReplyField bool, replyPlaceholder, replyButtonTitle string) {
			testCaptureResult(id, true, "")
		},
	})
	dn := newDarwinNotifier(t)
	cat := NotificationCategory{
		ID:               "cat",
		Actions:          []NotificationAction{{ID: "a", Title: "A"}},
		HasReplyField:    true,
		ReplyPlaceholder: "Reply...",
		ReplyButtonTitle: "Send",
	}
	if err := dn.RegisterNotificationCategory(cat); err != nil {
		t.Fatalf("RegisterNotificationCategory: %v", err)
	}
}

func TestDarwinNotifier_RegisterNotificationCategory_Failure(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cRegisterNotificationCategory": func(id int, categoryID, actionsJSON string, hasReplyField bool, replyPlaceholder, replyButtonTitle string) {
			testCaptureResult(id, false, "")
		},
	})
	dn := newDarwinNotifier(t)
	if err := dn.RegisterNotificationCategory(NotificationCategory{ID: "cat"}); err == nil {
		t.Fatal("expected failure error")
	}
}

func TestDarwinNotifier_RegisterNotificationCategory_ErrorMessage(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cRegisterNotificationCategory": func(id int, categoryID, actionsJSON string, hasReplyField bool, replyPlaceholder, replyButtonTitle string) {
			testCaptureResult(id, false, "boom")
		},
	})
	dn := newDarwinNotifier(t)
	if err := dn.RegisterNotificationCategory(NotificationCategory{ID: "cat"}); err == nil || err.Error() != "boom" {
		t.Fatalf("expected boom error, got %v", err)
	}
}

func TestDarwinNotifier_RegisterNotificationCategory_Timeout(t *testing.T) {
	setShortTimeouts(t)
	setCFuncs(t, map[string]any{
		"cRegisterNotificationCategory": func(id int, categoryID, actionsJSON string, hasReplyField bool, replyPlaceholder, replyButtonTitle string) {},
	})
	dn := newDarwinNotifier(t)
	if err := dn.RegisterNotificationCategory(NotificationCategory{ID: "cat"}); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestDarwinNotifier_RegisterNotificationCategory_MarshalError(t *testing.T) {
	setJSONMarshal(t, func(v any) ([]byte, error) { return nil, errors.New("marshal error") })
	dn := newDarwinNotifier(t)
	cat := NotificationCategory{
		ID:      "cat",
		Actions: []NotificationAction{{ID: "a", Title: "A", Destructive: true}},
	}
	if err := dn.RegisterNotificationCategory(cat); err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestDarwinNotifier_RemoveNotificationCategory_Success(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cRemoveNotificationCategory": func(id int, categoryID string) { testCaptureResult(id, true, "") },
	})
	dn := newDarwinNotifier(t)
	if err := dn.RemoveNotificationCategory("cat"); err != nil {
		t.Fatalf("RemoveNotificationCategory: %v", err)
	}
}

func TestDarwinNotifier_RemoveNotificationCategory_Failure(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cRemoveNotificationCategory": func(id int, categoryID string) { testCaptureResult(id, false, "") },
	})
	dn := newDarwinNotifier(t)
	if err := dn.RemoveNotificationCategory("cat"); err == nil {
		t.Fatal("expected failure error")
	}
}

func TestDarwinNotifier_RemoveNotificationCategory_ErrorMessage(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cRemoveNotificationCategory": func(id int, categoryID string) { testCaptureResult(id, false, "boom") },
	})
	dn := newDarwinNotifier(t)
	if err := dn.RemoveNotificationCategory("cat"); err == nil || err.Error() != "boom" {
		t.Fatalf("expected boom error, got %v", err)
	}
}

func TestDarwinNotifier_RemoveNotificationCategory_Timeout(t *testing.T) {
	setShortTimeouts(t)
	setCFuncs(t, map[string]any{
		"cRemoveNotificationCategory": func(id int, categoryID string) {},
	})
	dn := newDarwinNotifier(t)
	if err := dn.RemoveNotificationCategory("cat"); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestDarwinNotifier_RemoveAllPendingNotifications(t *testing.T) {
	called := false
	setCFuncs(t, map[string]any{
		"cRemoveAllPendingNotifications": func() { called = true },
	})
	dn := newDarwinNotifier(t)
	if err := dn.RemoveAllPendingNotifications(); err != nil {
		t.Fatalf("RemoveAllPendingNotifications: %v", err)
	}
	if !called {
		t.Fatal("cRemoveAllPendingNotifications was not called")
	}
}

func TestDarwinNotifier_RemovePendingNotification(t *testing.T) {
	called := false
	setCFuncs(t, map[string]any{
		"cRemovePendingNotification": func(identifier string) { called = true },
	})
	dn := newDarwinNotifier(t)
	if err := dn.RemovePendingNotification("1"); err != nil {
		t.Fatalf("RemovePendingNotification: %v", err)
	}
	if !called {
		t.Fatal("cRemovePendingNotification was not called")
	}
}

func TestDarwinNotifier_RemoveAllDeliveredNotifications(t *testing.T) {
	called := false
	setCFuncs(t, map[string]any{
		"cRemoveAllDeliveredNotifications": func() { called = true },
	})
	dn := newDarwinNotifier(t)
	if err := dn.RemoveAllDeliveredNotifications(); err != nil {
		t.Fatalf("RemoveAllDeliveredNotifications: %v", err)
	}
	if !called {
		t.Fatal("cRemoveAllDeliveredNotifications was not called")
	}
}

func TestDarwinNotifier_RemoveDeliveredNotification(t *testing.T) {
	called := false
	setCFuncs(t, map[string]any{
		"cRemoveDeliveredNotification": func(identifier string) { called = true },
	})
	dn := newDarwinNotifier(t)
	if err := dn.RemoveDeliveredNotification("1"); err != nil {
		t.Fatalf("RemoveDeliveredNotification: %v", err)
	}
	if !called {
		t.Fatal("cRemoveDeliveredNotification was not called")
	}
}

func TestDarwinNotifier_RemoveNotification(t *testing.T) {
	dn := newDarwinNotifier(t)
	if err := dn.RemoveNotification("1"); err != nil {
		t.Fatalf("RemoveNotification: %v", err)
	}
}

func TestCaptureResult_NilService(t *testing.T) {
	NotificationService_ = nil
	testCaptureResult(0, true, "")
}

func TestCaptureResult_NonChannelHandler(t *testing.T) {
	NotificationService_ = &NotificationService{impl: captureFakeNotifier{}}
	testCaptureResult(0, true, "")
}

func TestCaptureResult_MissingChannel(t *testing.T) {
	_ = newDarwinNotifier(t)
	testCaptureResult(999, true, "")
}

func TestCaptureResult_Success(t *testing.T) {
	dn := newDarwinNotifier(t)
	id, ch := dn.registerChannel()
	go testCaptureResult(id, true, "")
	result := <-ch
	if !result.Success || result.Error != nil {
		t.Fatalf("expected success, got %+v", result)
	}
}

func TestCaptureResult_Error(t *testing.T) {
	dn := newDarwinNotifier(t)
	id, ch := dn.registerChannel()
	go testCaptureResult(id, false, "boom")
	result := <-ch
	if result.Success || result.Error == nil {
		t.Fatalf("expected error, got %+v", result)
	}
}

func TestDidReceiveNotificationResponse_NilPayload(t *testing.T) {
	_ = newDarwinNotifier(t)
	called := false
	NotificationService_.OnNotificationResponse(func(result NotificationResult) {
		called = true
		if result.Error == nil {
			t.Fatal("expected error")
		}
	})
	testDidReceiveNotificationResponse(nil, nil)
	if !called {
		t.Fatal("callback was not invoked")
	}
}

func TestDidReceiveNotificationResponse_Error(t *testing.T) {
	_ = newDarwinNotifier(t)
	called := false
	NotificationService_.OnNotificationResponse(func(result NotificationResult) {
		called = true
		if result.Error == nil {
			t.Fatal("expected error")
		}
	})
	errMsg := "boom"
	testDidReceiveNotificationResponse(nil, &errMsg)
	if !called {
		t.Fatal("callback was not invoked")
	}
}

func TestDidReceiveNotificationResponse_BadJSON(t *testing.T) {
	_ = newDarwinNotifier(t)
	called := false
	NotificationService_.OnNotificationResponse(func(result NotificationResult) {
		called = true
		if result.Error == nil {
			t.Fatal("expected error")
		}
	})
	payload := "not json"
	testDidReceiveNotificationResponse(&payload, nil)
	if !called {
		t.Fatal("callback was not invoked")
	}
}

func TestDidReceiveNotificationResponse_AppleDefaultAction(t *testing.T) {
	_ = newDarwinNotifier(t)
	called := false
	NotificationService_.OnNotificationResponse(func(result NotificationResult) {
		called = true
		if result.Error != nil {
			t.Fatalf("unexpected error: %v", result.Error)
		}
		if result.Response.ActionIdentifier != DefaultActionIdentifier {
			t.Fatalf("expected %q, got %q", DefaultActionIdentifier, result.Response.ActionIdentifier)
		}
	})
	payload := `{"actionIdentifier":"com.apple.UNNotificationDefaultActionIdentifier","id":"1"}`
	testDidReceiveNotificationResponse(&payload, nil)
	if !called {
		t.Fatal("callback was not invoked")
	}
}

func TestDidReceiveNotificationResponse_Normal(t *testing.T) {
	_ = newDarwinNotifier(t)
	called := false
	NotificationService_.OnNotificationResponse(func(result NotificationResult) {
		called = true
		if result.Error != nil {
			t.Fatalf("unexpected error: %v", result.Error)
		}
		if result.Response.ID != "1" {
			t.Fatalf("expected id 1, got %q", result.Response.ID)
		}
	})
	payload := `{"id":"1","actionIdentifier":"action"}`
	testDidReceiveNotificationResponse(&payload, nil)
	if !called {
		t.Fatal("callback was not invoked")
	}
}

func TestIsNotificationAvailable(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cIsNotificationAvailable": func() bool { return true },
	})
	if !isNotificationAvailable() {
		t.Fatal("expected true")
	}
}

func TestCheckBundleIdentifier(t *testing.T) {
	setCFuncs(t, map[string]any{
		"cCheckBundleIdentifier": func() bool { return true },
	})
	if !checkBundleIdentifier() {
		t.Fatal("expected true")
	}
}

func TestDarwinNotifier_RegisterChannel_GetChannel_CleanupChannel(t *testing.T) {
	dn := newDarwinNotifier(t)
	id, ch := dn.registerChannel()
	if id != 0 {
		t.Fatalf("expected first id 0, got %d", id)
	}
	got, ok := dn.GetChannel(id)
	if !ok || got != ch {
		t.Fatal("GetChannel did not return registered channel")
	}
	_, exists := dn.GetChannel(id)
	if exists {
		t.Fatal("GetChannel should remove channel")
	}

	id2, _ := dn.registerChannel()
	dn.cleanupChannel(id2)
	_, exists = dn.channels[id2]
	if exists {
		t.Fatal("cleanupChannel should remove channel")
	}
}

type captureFakeNotifier struct{}

func (captureFakeNotifier) Startup(context.Context, application.ServiceOptions) error { return errors.New("startup error") }
func (captureFakeNotifier) Shutdown() error                                         { return errors.New("shutdown error") }
func (captureFakeNotifier) RequestNotificationAuthorization() (bool, error)         { return false, errors.New("auth error") }
func (captureFakeNotifier) CheckNotificationAuthorization() (bool, error)           { return false, errors.New("check error") }
func (captureFakeNotifier) SendNotification(NotificationOptions) error              { return errors.New("send error") }
func (captureFakeNotifier) SendNotificationWithActions(NotificationOptions) error   { return errors.New("send actions error") }
func (captureFakeNotifier) RegisterNotificationCategory(NotificationCategory) error { return errors.New("register error") }
func (captureFakeNotifier) RemoveNotificationCategory(string) error                 { return errors.New("remove cat error") }
func (captureFakeNotifier) RemoveAllPendingNotifications() error                    { return errors.New("remove all pending error") }
func (captureFakeNotifier) RemovePendingNotification(string) error                  { return errors.New("remove pending error") }
func (captureFakeNotifier) RemoveAllDeliveredNotifications() error                  { return errors.New("remove all delivered error") }
func (captureFakeNotifier) RemoveDeliveredNotification(string) error                { return errors.New("remove delivered error") }
func (captureFakeNotifier) RemoveNotification(string) error                         { return errors.New("remove error") }

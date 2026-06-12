package notifications

import (
	"context"
	"errors"
	"testing"

	"github.com/gailsapp/gails/pkg/application"
)

type fakeNotifier struct {
	startupCalled bool
	shutdownCalled bool
	requestAuthResult bool
	requestAuthErr error
	checkAuthResult bool
	checkAuthErr error
	sendNotificationCalled bool
	sendNotificationWithActionsCalled bool
	registerCategoryCalled bool
	removeCategoryCalled bool
	removeAllPendingCalled bool
	removePendingCalled bool
	removeAllDeliveredCalled bool
	removeDeliveredCalled bool
	removeNotificationCalled bool
}

func (f *fakeNotifier) Startup(ctx context.Context, options application.ServiceOptions) error {
	f.startupCalled = true
	return nil
}

func (f *fakeNotifier) Shutdown() error {
	f.shutdownCalled = true
	return nil
}

func (f *fakeNotifier) RequestNotificationAuthorization() (bool, error) {
	return f.requestAuthResult, f.requestAuthErr
}

func (f *fakeNotifier) CheckNotificationAuthorization() (bool, error) {
	return f.checkAuthResult, f.checkAuthErr
}

func (f *fakeNotifier) SendNotification(options NotificationOptions) error {
	f.sendNotificationCalled = true
	return nil
}

func (f *fakeNotifier) SendNotificationWithActions(options NotificationOptions) error {
	f.sendNotificationWithActionsCalled = true
	return nil
}

func (f *fakeNotifier) RegisterNotificationCategory(category NotificationCategory) error {
	f.registerCategoryCalled = true
	return nil
}

func (f *fakeNotifier) RemoveNotificationCategory(categoryID string) error {
	f.removeCategoryCalled = true
	return nil
}

func (f *fakeNotifier) RemoveAllPendingNotifications() error {
	f.removeAllPendingCalled = true
	return nil
}

func (f *fakeNotifier) RemovePendingNotification(identifier string) error {
	f.removePendingCalled = true
	return nil
}

func (f *fakeNotifier) RemoveAllDeliveredNotifications() error {
	f.removeAllDeliveredCalled = true
	return nil
}

func (f *fakeNotifier) RemoveDeliveredNotification(identifier string) error {
	f.removeDeliveredCalled = true
	return nil
}

func (f *fakeNotifier) RemoveNotification(identifier string) error {
	f.removeNotificationCalled = true
	return nil
}

func TestNotificationService_ServiceName(t *testing.T) {
	ns := &NotificationService{impl: &fakeNotifier{}}
	want := "github.com/gailsapp/gails/services/notifications"
	if got := ns.ServiceName(); got != want {
		t.Fatalf("ServiceName: want %q, got %q", want, got)
	}
}

func TestNotificationService_ServiceStartup(t *testing.T) {
	fake := &fakeNotifier{}
	ns := &NotificationService{impl: fake}
	if err := ns.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup: %v", err)
	}
	if !fake.startupCalled {
		t.Fatal("Startup was not called")
	}
}

func TestNotificationService_ServiceShutdown(t *testing.T) {
	fake := &fakeNotifier{}
	ns := &NotificationService{impl: fake}
	if err := ns.ServiceShutdown(); err != nil {
		t.Fatalf("ServiceShutdown: %v", err)
	}
	if !fake.shutdownCalled {
		t.Fatal("Shutdown was not called")
	}
}

func TestNotificationService_OnNotificationResponse(t *testing.T) {
	ns := &NotificationService{}
	called := false
	ns.OnNotificationResponse(func(result NotificationResult) { called = true })
	ns.handleNotificationResult(NotificationResult{})
	if !called {
		t.Fatal("callback was not invoked")
	}
}

func TestNotificationService_OnNotificationResponse_Nil(t *testing.T) {
	ns := &NotificationService{}
	// Should not panic when no callback is registered.
	ns.handleNotificationResult(NotificationResult{})
}

func TestNotificationService_RequestNotificationAuthorization(t *testing.T) {
	fake := &fakeNotifier{requestAuthResult: true}
	ns := &NotificationService{impl: fake}
	ok, err := ns.RequestNotificationAuthorization()
	if err != nil {
		t.Fatalf("RequestNotificationAuthorization: %v", err)
	}
	if !ok {
		t.Fatal("expected true")
	}
}

func TestNotificationService_CheckNotificationAuthorization(t *testing.T) {
	fake := &fakeNotifier{checkAuthResult: true}
	ns := &NotificationService{impl: fake}
	ok, err := ns.CheckNotificationAuthorization()
	if err != nil {
		t.Fatalf("CheckNotificationAuthorization: %v", err)
	}
	if !ok {
		t.Fatal("expected true")
	}
}

func TestNotificationService_SendNotification(t *testing.T) {
	fake := &fakeNotifier{}
	ns := &NotificationService{impl: fake}
	if err := ns.SendNotification(NotificationOptions{ID: "1", Title: "Hello"}); err != nil {
		t.Fatalf("SendNotification: %v", err)
	}
	if !fake.sendNotificationCalled {
		t.Fatal("SendNotification was not delegated")
	}
}

func TestNotificationService_SendNotification_Invalid(t *testing.T) {
	ns := &NotificationService{impl: &fakeNotifier{}}
	if err := ns.SendNotification(NotificationOptions{ID: "", Title: ""}); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestNotificationService_SendNotificationWithActions(t *testing.T) {
	fake := &fakeNotifier{}
	ns := &NotificationService{impl: fake}
	if err := ns.SendNotificationWithActions(NotificationOptions{ID: "1", Title: "Hello"}); err != nil {
		t.Fatalf("SendNotificationWithActions: %v", err)
	}
	if !fake.sendNotificationWithActionsCalled {
		t.Fatal("SendNotificationWithActions was not delegated")
	}
}

func TestNotificationService_SendNotificationWithActions_Invalid(t *testing.T) {
	ns := &NotificationService{impl: &fakeNotifier{}}
	if err := ns.SendNotificationWithActions(NotificationOptions{ID: "1", Title: ""}); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestNotificationService_RegisterNotificationCategory(t *testing.T) {
	fake := &fakeNotifier{}
	ns := &NotificationService{impl: fake}
	if err := ns.RegisterNotificationCategory(NotificationCategory{ID: "cat"}); err != nil {
		t.Fatalf("RegisterNotificationCategory: %v", err)
	}
	if !fake.registerCategoryCalled {
		t.Fatal("RegisterNotificationCategory was not delegated")
	}
}

func TestNotificationService_RemoveNotificationCategory(t *testing.T) {
	fake := &fakeNotifier{}
	ns := &NotificationService{impl: fake}
	if err := ns.RemoveNotificationCategory("cat"); err != nil {
		t.Fatalf("RemoveNotificationCategory: %v", err)
	}
	if !fake.removeCategoryCalled {
		t.Fatal("RemoveNotificationCategory was not delegated")
	}
}

func TestNotificationService_RemoveAllPendingNotifications(t *testing.T) {
	fake := &fakeNotifier{}
	ns := &NotificationService{impl: fake}
	if err := ns.RemoveAllPendingNotifications(); err != nil {
		t.Fatalf("RemoveAllPendingNotifications: %v", err)
	}
	if !fake.removeAllPendingCalled {
		t.Fatal("RemoveAllPendingNotifications was not delegated")
	}
}

func TestNotificationService_RemovePendingNotification(t *testing.T) {
	fake := &fakeNotifier{}
	ns := &NotificationService{impl: fake}
	if err := ns.RemovePendingNotification("1"); err != nil {
		t.Fatalf("RemovePendingNotification: %v", err)
	}
	if !fake.removePendingCalled {
		t.Fatal("RemovePendingNotification was not delegated")
	}
}

func TestNotificationService_RemoveAllDeliveredNotifications(t *testing.T) {
	fake := &fakeNotifier{}
	ns := &NotificationService{impl: fake}
	if err := ns.RemoveAllDeliveredNotifications(); err != nil {
		t.Fatalf("RemoveAllDeliveredNotifications: %v", err)
	}
	if !fake.removeAllDeliveredCalled {
		t.Fatal("RemoveAllDeliveredNotifications was not delegated")
	}
}

func TestNotificationService_RemoveDeliveredNotification(t *testing.T) {
	fake := &fakeNotifier{}
	ns := &NotificationService{impl: fake}
	if err := ns.RemoveDeliveredNotification("1"); err != nil {
		t.Fatalf("RemoveDeliveredNotification: %v", err)
	}
	if !fake.removeDeliveredCalled {
		t.Fatal("RemoveDeliveredNotification was not delegated")
	}
}

func TestNotificationService_RemoveNotification(t *testing.T) {
	fake := &fakeNotifier{}
	ns := &NotificationService{impl: fake}
	if err := ns.RemoveNotification("1"); err != nil {
		t.Fatalf("RemoveNotification: %v", err)
	}
	if !fake.removeNotificationCalled {
		t.Fatal("RemoveNotification was not delegated")
	}
}

func TestValidateNotificationOptions(t *testing.T) {
	if err := validateNotificationOptions(NotificationOptions{ID: "", Title: "T"}); err == nil {
		t.Fatal("expected error for empty ID")
	}
	if err := validateNotificationOptions(NotificationOptions{ID: "1", Title: ""}); err == nil {
		t.Fatal("expected error for empty title")
	}
	if err := validateNotificationOptions(NotificationOptions{ID: "1", Title: "T"}); err != nil {
		t.Fatalf("expected valid options: %v", err)
	}
}

func TestGetNotificationService(t *testing.T) {
	resetNotificationService()
	if got := getNotificationService(); got != nil {
		t.Fatalf("expected nil before New, got %v", got)
	}
	ns := New()
	if got := getNotificationService(); got != ns {
		t.Fatal("getNotificationService did not return the singleton")
	}
}

type errNotifier struct{}

func (errNotifier) Startup(context.Context, application.ServiceOptions) error { return errors.New("startup error") }
func (errNotifier) Shutdown() error { return errors.New("shutdown error") }
func (errNotifier) RequestNotificationAuthorization() (bool, error) { return false, errors.New("auth error") }
func (errNotifier) CheckNotificationAuthorization() (bool, error) { return false, errors.New("check error") }
func (errNotifier) SendNotification(NotificationOptions) error { return errors.New("send error") }
func (errNotifier) SendNotificationWithActions(NotificationOptions) error { return errors.New("send actions error") }
func (errNotifier) RegisterNotificationCategory(NotificationCategory) error { return errors.New("register error") }
func (errNotifier) RemoveNotificationCategory(string) error { return errors.New("remove cat error") }
func (errNotifier) RemoveAllPendingNotifications() error { return errors.New("remove all pending error") }
func (errNotifier) RemovePendingNotification(string) error { return errors.New("remove pending error") }
func (errNotifier) RemoveAllDeliveredNotifications() error { return errors.New("remove all delivered error") }
func (errNotifier) RemoveDeliveredNotification(string) error { return errors.New("remove delivered error") }
func (errNotifier) RemoveNotification(string) error { return errors.New("remove error") }

func TestNotificationService_Errors(t *testing.T) {
	ns := &NotificationService{impl: errNotifier{}}
	if err := ns.ServiceStartup(context.Background(), application.ServiceOptions{}); err == nil {
		t.Fatal("expected startup error")
	}
	if err := ns.ServiceShutdown(); err == nil {
		t.Fatal("expected shutdown error")
	}
	if _, err := ns.RequestNotificationAuthorization(); err == nil {
		t.Fatal("expected auth error")
	}
	if _, err := ns.CheckNotificationAuthorization(); err == nil {
		t.Fatal("expected check error")
	}
	if err := ns.SendNotification(NotificationOptions{ID: "1", Title: "T"}); err == nil {
		t.Fatal("expected send error")
	}
	if err := ns.SendNotificationWithActions(NotificationOptions{ID: "1", Title: "T"}); err == nil {
		t.Fatal("expected send actions error")
	}
	if err := ns.RegisterNotificationCategory(NotificationCategory{ID: "c"}); err == nil {
		t.Fatal("expected register error")
	}
	if err := ns.RemoveNotificationCategory("c"); err == nil {
		t.Fatal("expected remove category error")
	}
	if err := ns.RemoveAllPendingNotifications(); err == nil {
		t.Fatal("expected remove all pending error")
	}
	if err := ns.RemovePendingNotification("1"); err == nil {
		t.Fatal("expected remove pending error")
	}
	if err := ns.RemoveAllDeliveredNotifications(); err == nil {
		t.Fatal("expected remove all delivered error")
	}
	if err := ns.RemoveDeliveredNotification("1"); err == nil {
		t.Fatal("expected remove delivered error")
	}
	if err := ns.RemoveNotification("1"); err == nil {
		t.Fatal("expected remove error")
	}
}

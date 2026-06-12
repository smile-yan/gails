package dock

import (
	"context"
	"errors"
	"image/color"
	"testing"

	"github.com/gailsapp/gails/pkg/application"
)

type fakeDock struct {
	startupCalled bool
	shutdownCalled bool
	hideCalled bool
	showCalled bool
	setBadgeLabel string
	setCustomBadgeLabel string
	setCustomBadgeOptions BadgeOptions
	removeBadgeCalled bool
	badge *string
}

func (f *fakeDock) Startup(ctx context.Context, options application.ServiceOptions) error {
	f.startupCalled = true
	return nil
}

func (f *fakeDock) Shutdown() error {
	f.shutdownCalled = true
	return nil
}

func (f *fakeDock) HideAppIcon() {
	f.hideCalled = true
}

func (f *fakeDock) ShowAppIcon() {
	f.showCalled = true
}

func (f *fakeDock) SetBadge(label string) error {
	f.setBadgeLabel = label
	return nil
}

func (f *fakeDock) SetCustomBadge(label string, options BadgeOptions) error {
	f.setCustomBadgeLabel = label
	f.setCustomBadgeOptions = options
	return nil
}

func (f *fakeDock) RemoveBadge() error {
	f.removeBadgeCalled = true
	return nil
}

func (f *fakeDock) GetBadge() *string {
	return f.badge
}

func TestDockService_ServiceName(t *testing.T) {
	s := &DockService{impl: &fakeDock{}}
	want := "github.com/gailsapp/gails/pkg/services/dock"
	if got := s.ServiceName(); got != want {
		t.Fatalf("ServiceName: want %q, got %q", want, got)
	}
}

func TestDockService_ServiceStartup(t *testing.T) {
	fake := &fakeDock{}
	s := &DockService{impl: fake}
	if err := s.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup: %v", err)
	}
	if !fake.startupCalled {
		t.Fatal("Startup was not called")
	}
}

func TestDockService_ServiceShutdown(t *testing.T) {
	fake := &fakeDock{}
	s := &DockService{impl: fake}
	if err := s.ServiceShutdown(); err != nil {
		t.Fatalf("ServiceShutdown: %v", err)
	}
	if !fake.shutdownCalled {
		t.Fatal("Shutdown was not called")
	}
}

func TestDockService_HideAppIcon(t *testing.T) {
	fake := &fakeDock{}
	s := &DockService{impl: fake}
	s.HideAppIcon()
	if !fake.hideCalled {
		t.Fatal("HideAppIcon was not called")
	}
}

func TestDockService_ShowAppIcon(t *testing.T) {
	fake := &fakeDock{}
	s := &DockService{impl: fake}
	s.ShowAppIcon()
	if !fake.showCalled {
		t.Fatal("ShowAppIcon was not called")
	}
}

func TestDockService_SetBadge(t *testing.T) {
	fake := &fakeDock{}
	s := &DockService{impl: fake}
	if err := s.SetBadge("42"); err != nil {
		t.Fatalf("SetBadge: %v", err)
	}
	if fake.setBadgeLabel != "42" {
		t.Fatalf("expected label 42, got %q", fake.setBadgeLabel)
	}
}

func TestDockService_SetCustomBadge(t *testing.T) {
	fake := &fakeDock{}
	s := &DockService{impl: fake}
	opts := BadgeOptions{TextColour: color.RGBA{R: 255}, BackgroundColour: color.RGBA{B: 255}, FontName: "Arial", FontSize: 12, SmallFontSize: 10}
	if err := s.SetCustomBadge("7", opts); err != nil {
		t.Fatalf("SetCustomBadge: %v", err)
	}
	if fake.setCustomBadgeLabel != "7" {
		t.Fatalf("expected label 7, got %q", fake.setCustomBadgeLabel)
	}
	if fake.setCustomBadgeOptions.FontName != "Arial" {
		t.Fatal("options not forwarded")
	}
}

func TestDockService_RemoveBadge(t *testing.T) {
	fake := &fakeDock{}
	s := &DockService{impl: fake}
	if err := s.RemoveBadge(); err != nil {
		t.Fatalf("RemoveBadge: %v", err)
	}
	if !fake.removeBadgeCalled {
		t.Fatal("RemoveBadge was not called")
	}
}

func TestDockService_GetBadge(t *testing.T) {
	label := "99"
	fake := &fakeDock{badge: &label}
	s := &DockService{impl: fake}
	if got := s.GetBadge(); got != &label {
		t.Fatal("GetBadge did not return expected label")
	}
}

type errDock struct{}

func (errDock) Startup(ctx context.Context, options application.ServiceOptions) error {
	return errors.New("startup error")
}

func (errDock) Shutdown() error {
	return errors.New("shutdown error")
}

func (errDock) HideAppIcon() {}
func (errDock) ShowAppIcon()  {}
func (errDock) SetBadge(string) error { return nil }
func (errDock) SetCustomBadge(string, BadgeOptions) error { return nil }
func (errDock) RemoveBadge() error { return nil }
func (errDock) GetBadge() *string { return nil }

func TestDockService_ServiceStartup_Error(t *testing.T) {
	s := &DockService{impl: errDock{}}
	if err := s.ServiceStartup(context.Background(), application.ServiceOptions{}); err == nil {
		t.Fatal("expected startup error")
	}
}

func TestDockService_ServiceShutdown_Error(t *testing.T) {
	s := &DockService{impl: errDock{}}
	if err := s.ServiceShutdown(); err == nil {
		t.Fatal("expected shutdown error")
	}
}

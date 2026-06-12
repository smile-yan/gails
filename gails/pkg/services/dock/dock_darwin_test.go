//go:build darwin

package dock

import (
	"context"
	"testing"

	"github.com/gailsapp/gails/pkg/application"
)

func TestDarwinDock_New(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("New() returned nil")
	}
	if s.ServiceName() != "github.com/gailsapp/gails/pkg/services/dock" {
		t.Fatal("unexpected service name")
	}
}

func TestDarwinDock_NewWithOptions(t *testing.T) {
	s := NewWithOptions(BadgeOptions{FontName: "Arial"})
	if s == nil {
		t.Fatal("NewWithOptions() returned nil")
	}
}

func TestDarwinDock_StartupShutdown(t *testing.T) {
	d := &darwinDock{}
	if err := d.Startup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatalf("Startup: %v", err)
	}
	if err := d.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestDarwinDock_HideAppIcon(t *testing.T) {
	called := false
	orig := cHideDockIcon
	cHideDockIcon = func() { called = true }
	defer func() { cHideDockIcon = orig }()

	d := &darwinDock{}
	d.HideAppIcon()
	if !called {
		t.Fatal("cHideDockIcon was not called")
	}
}

func TestDarwinDock_ShowAppIcon(t *testing.T) {
	called := false
	orig := cShowDockIcon
	cShowDockIcon = func() { called = true }
	defer func() { cShowDockIcon = orig }()

	d := &darwinDock{}
	d.ShowAppIcon()
	if !called {
		t.Fatal("cShowDockIcon was not called")
	}
}

func TestDarwinDock_SetBadge(t *testing.T) {
	orig := cSetBadge
	cSetBadge = func(label *string) bool { return true }
	defer func() { cSetBadge = orig }()

	d := &darwinDock{}
	if err := d.SetBadge("42"); err != nil {
		t.Fatalf("SetBadge: %v", err)
	}
	if got := d.GetBadge(); got == nil || *got != "42" {
		t.Fatalf("expected badge 42, got %v", got)
	}
}

func TestDarwinDock_SetBadge_EmptyDefaultsToDot(t *testing.T) {
	orig := cSetBadge
	cSetBadge = func(label *string) bool { return true }
	defer func() { cSetBadge = orig }()

	d := &darwinDock{}
	if err := d.SetBadge(""); err != nil {
		t.Fatalf("SetBadge: %v", err)
	}
	if got := d.GetBadge(); got == nil || *got != "●" {
		t.Fatalf("expected default dot badge, got %v", got)
	}
}

func TestDarwinDock_SetBadge_Failure(t *testing.T) {
	orig := cSetBadge
	cSetBadge = func(label *string) bool { return false }
	defer func() { cSetBadge = orig }()

	d := &darwinDock{}
	if err := d.SetBadge("42"); err == nil {
		t.Fatal("expected error when cSetBadge returns false")
	}
}

func TestDarwinDock_SetCustomBadge(t *testing.T) {
	orig := cSetBadge
	cSetBadge = func(label *string) bool { return true }
	defer func() { cSetBadge = orig }()

	d := &darwinDock{}
	if err := d.SetCustomBadge("custom", BadgeOptions{}); err != nil {
		t.Fatalf("SetCustomBadge: %v", err)
	}
	if got := d.GetBadge(); got == nil || *got != "custom" {
		t.Fatalf("expected badge custom, got %v", got)
	}
}

func TestDarwinDock_RemoveBadge(t *testing.T) {
	orig := cSetBadge
	cSetBadge = func(label *string) bool { return true }
	defer func() { cSetBadge = orig }()

	d := &darwinDock{Badge: strPtr("old")}
	if err := d.RemoveBadge(); err != nil {
		t.Fatalf("RemoveBadge: %v", err)
	}
	if d.GetBadge() != nil {
		t.Fatal("expected badge to be nil after RemoveBadge")
	}
}

func TestDarwinDock_GetBadge(t *testing.T) {
	label := "badge"
	d := &darwinDock{Badge: &label}
	if got := d.GetBadge(); got != &label {
		t.Fatal("GetBadge did not return expected pointer")
	}
}

func strPtr(s string) *string { return &s }

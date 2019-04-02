package crabfs

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	mountPoint := "/tmp/crabfs"
	fs, err := New(mountPoint, "testdht")
	if err != nil {
		t.Error(err)
	}

	if fs.MountPoint != mountPoint {
		t.Errorf("Expected %s, but got %s", mountPoint, fs.MountPoint)
	}
}

func TestCloseCtx(t *testing.T) {
	mountPoint := "/tmp/crabfs"
	ctx, cancel := context.WithCancel(context.Background())

	fs, err := NewWithContext(ctx, mountPoint, "testdht")
	if err != nil {
		t.Error(err)
	}

	if fs.MountPoint != mountPoint {
		t.Errorf("Expected %s, but got %s", mountPoint, fs.MountPoint)
	}

	if fs.IsClosed() {
		t.Errorf("Expected fs to not be closed")
	}

	cancel()

	if !fs.IsClosed() {
		t.Errorf("Expected fs to be closed")
	}
}

func TestClose(t *testing.T) {
	mountPoint := "/tmp/crabfs"

	fs, err := New(mountPoint, "testdht")
	if err != nil {
		t.Error(err)
	}

	if fs.MountPoint != mountPoint {
		t.Errorf("Expected %s, but got %s", mountPoint, fs.MountPoint)
	}

	if fs.IsClosed() {
		t.Errorf("Expected fs to not be closed")
	}

	fs.Close()

	if !fs.IsClosed() {
		t.Errorf("Expected fs to be closed")
	}
}

func TestGetID(t *testing.T) {
	mountPoint := "/tmp/crabfs"

	fs, err := New(mountPoint, "testdht")
	if err != nil {
		t.Error(err)
	}

	if fs.MountPoint != mountPoint {
		t.Errorf("Expected %s, but got %s", mountPoint, fs.MountPoint)
	}

	if fs.GetHostID() == "" {
		t.Error("Invalid host id received")
	}
}

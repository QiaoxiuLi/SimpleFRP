package service

import (
	"path/filepath"
	"testing"
)

func TestPathEntryPreservesUnrelatedEntries(t *testing.T) {
	entry := filepath.Join(t.TempDir(), "SimpleFRP")
	value := "first;second;" + entry + ";third"
	if !pathContains(value, entry, ";") {
		t.Fatal("missing managed entry")
	}
	if got := removePathEntry(value, entry); got != "first;second;third" {
		t.Fatal("unrelated PATH entries changed")
	}
	if got := removePathEntry("first;second", entry); got != "first;second" {
		t.Fatal("absent entry changed PATH")
	}
}
func TestBinaryOwnershipIsConfinedToHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SIMPLEFRP_HOME", root)
	if !OwnsInstalledBinary(filepath.Join(root, "bin", "simplefrp")) {
		t.Fatal("owned binary rejected")
	}
	if OwnsInstalledBinary(filepath.Join(root, "..", "shared", "simplefrp")) {
		t.Fatal("shared binary claimed")
	}
}

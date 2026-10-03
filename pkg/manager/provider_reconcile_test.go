package manager

import (
	"testing"

	debridTypes "github.com/sirrobot01/decypharr/pkg/debrid/types"
	"github.com/sirrobot01/decypharr/pkg/storage"
)

func placement(provider, link string, status debridTypes.TorrentStatus) *storage.ProviderEntry {
	return &storage.ProviderEntry{
		Provider: provider,
		ID:       "1",
		Status:   status,
		Files:    map[string]*storage.ProviderFile{"a.mkv": {Id: "1", Link: link}},
	}
}

var configuredProviders = map[string]string{
	"RD Primary": "realdebrid",
	"TorBox":     "torbox",
}

func TestReconcileLeavesConfiguredActiveProviderAlone(t *testing.T) {
	e := &storage.Entry{
		ActiveProvider: "TorBox",
		Providers: map[string]*storage.ProviderEntry{
			"TorBox": placement("TorBox", "torbox://1/0", debridTypes.TorrentStatusDownloaded),
		},
	}
	if reconcileEntryProviders(e, configuredProviders) {
		t.Fatal("entry with a configured active provider must not change")
	}
	if e.ActiveProvider != "TorBox" {
		t.Fatalf("active provider changed to %q", e.ActiveProvider)
	}
}

func TestReconcileSwitchesToExistingConfiguredPlacement(t *testing.T) {
	e := &storage.Entry{
		ActiveProvider: "TorBox Secondary",
		Providers: map[string]*storage.ProviderEntry{
			"TorBox":           placement("TorBox", "torbox://1/0", debridTypes.TorrentStatusDownloaded),
			"TorBox Secondary": placement("TorBox Secondary", "torbox://1/0", debridTypes.TorrentStatusDownloaded),
		},
	}
	if !reconcileEntryProviders(e, configuredProviders) {
		t.Fatal("expected the entry to be re-pointed")
	}
	if e.ActiveProvider != "TorBox" {
		t.Fatalf("active provider = %q, want TorBox", e.ActiveProvider)
	}
}

func TestReconcileRelabelsPlacementOfRemovedProvider(t *testing.T) {
	e := &storage.Entry{
		ActiveProvider: "TorBox Secondary",
		Providers: map[string]*storage.ProviderEntry{
			"TorBox Secondary": placement("TorBox Secondary", "torbox://1/0", debridTypes.TorrentStatusDownloaded),
		},
	}
	if !reconcileEntryProviders(e, configuredProviders) {
		t.Fatal("expected the placement to be relabelled")
	}
	if e.ActiveProvider != "TorBox" {
		t.Fatalf("active provider = %q, want TorBox", e.ActiveProvider)
	}
	if _, ok := e.Providers["TorBox Secondary"]; ok {
		t.Fatal("old placement key must be gone")
	}
	if p := e.Providers["TorBox"]; p == nil || p.Provider != "TorBox" {
		t.Fatalf("placement not stored under TorBox: %+v", p)
	}
}

func TestReconcileRelabelsRealDebridPlacement(t *testing.T) {
	e := &storage.Entry{
		ActiveProvider: "RD Secondary",
		Providers: map[string]*storage.ProviderEntry{
			"RD Secondary": placement("RD Secondary", "https://real-debrid.com/d/ABC", debridTypes.TorrentStatusDownloaded),
		},
	}
	if !reconcileEntryProviders(e, configuredProviders) {
		t.Fatal("expected the placement to be relabelled")
	}
	if e.ActiveProvider != "RD Primary" {
		t.Fatalf("active provider = %q, want RD Primary", e.ActiveProvider)
	}
}

func TestReconcileGivesUpWhenNothingMatches(t *testing.T) {
	e := &storage.Entry{
		ActiveProvider: "AllDebrid",
		Providers: map[string]*storage.ProviderEntry{
			"AllDebrid": placement("AllDebrid", "https://alldebrid.com/f/x", debridTypes.TorrentStatusDownloaded),
		},
	}
	if reconcileEntryProviders(e, configuredProviders) {
		t.Fatal("nothing can serve this entry, it must be left unchanged")
	}
	if e.ActiveProvider != "AllDebrid" {
		t.Fatalf("active provider changed to %q", e.ActiveProvider)
	}
}

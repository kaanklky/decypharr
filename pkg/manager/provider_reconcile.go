package manager

import (
	"sort"
	"strings"

	debrid "github.com/sirrobot01/decypharr/pkg/debrid/common"
	debridTypes "github.com/sirrobot01/decypharr/pkg/debrid/types"
	"github.com/sirrobot01/decypharr/pkg/storage"
)

func placementKind(p *storage.ProviderEntry) string {
	for _, f := range p.Files {
		switch {
		case strings.HasPrefix(f.Link, "torbox://"):
			return "torbox"
		case strings.Contains(f.Link, "real-debrid.com"):
			return "realdebrid"
		}
	}
	return ""
}

func firstConfiguredOfKind(configured map[string]string, kind string) string {
	if kind == "" {
		return ""
	}
	names := make([]string, 0, len(configured))
	for name, k := range configured {
		if k == kind {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

func reconcileEntryProviders(e *storage.Entry, configured map[string]string) bool {
	if e.IsNZB() || e.ActiveProvider == "" {
		return false
	}
	if _, ok := configured[e.ActiveProvider]; ok {
		return false
	}
	keys := make([]string, 0, len(e.Providers))
	for k := range e.Providers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		p := e.Providers[k]
		if p.Status != debridTypes.TorrentStatusDownloaded {
			continue
		}
		if _, ok := configured[p.Provider]; !ok {
			continue
		}
		if e.ActivatePlacement(p.Provider) == nil {
			return true
		}
	}
	for _, k := range keys {
		p := e.Providers[k]
		if p.Status != debridTypes.TorrentStatusDownloaded {
			continue
		}
		if _, ok := configured[p.Provider]; ok {
			continue
		}
		target := firstConfiguredOfKind(configured, placementKind(p))
		if target == "" {
			continue
		}
		if _, taken := e.Providers[target]; taken {
			continue
		}
		delete(e.Providers, k)
		p.Provider = target
		e.Providers[target] = p
		return e.ActivatePlacement(target) == nil
	}
	return false
}

func (m *Manager) reconcileProviders() {
	configured := make(map[string]string)
	m.clients.Range(func(name string, c debrid.Client) bool {
		if c != nil {
			configured[name] = c.Config().Provider
		}
		return true
	})
	if len(configured) == 0 {
		return
	}
	var fixed []*storage.Entry
	err := m.storage.ForEachBatch(500, func(batch []*storage.Entry) error {
		for _, e := range batch {
			if reconcileEntryProviders(e, configured) {
				fixed = append(fixed, e)
			}
		}
		return nil
	})
	if err != nil {
		m.logger.Warn().Err(err).Msg("Failed to scan entries for unconfigured providers")
		return
	}
	if len(fixed) == 0 {
		return
	}
	if err := m.storage.BatchAddOrUpdate(fixed); err != nil {
		m.logger.Error().Err(err).Int("entries", len(fixed)).Msg("Failed to save entries re-pointed at a configured provider")
		return
	}
	m.logger.Info().Int("entries", len(fixed)).Msg("Re-pointed entries whose active provider is no longer configured")
}

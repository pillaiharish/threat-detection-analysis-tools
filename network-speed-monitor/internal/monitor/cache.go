package monitor

import (
	"network-speed-monitor/internal/models"
	"sync"
)

var (
	cache     models.Stat
	cacheLock sync.Mutex
)

// UpdateCache updates the stats cache
func UpdateCache(stat models.Stat) {
	cacheLock.Lock()
	cache = stat
	cacheLock.Unlock()
}

// GetCache retrieves the cached stats
func GetCache() models.Stat {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	return cache
}

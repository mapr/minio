package cmd

import (
	"github.com/minio/minio-go/v7/pkg/set"
	"github.com/minio/minio/cmd/logger"
	"github.com/minio/minio/pkg/dsync"
)

type listsDsyncLockers []dsync.NetLocker

type MapRServerPools struct {
	*FSObjects
	// Distributed locker clients.
	lockers listsDsyncLockers

	// Distributed lock owner (constant per running instance).
	lockOwner string
}

func NewMapRServerPools(endpointServerPools EndpointServerPools) (ObjectLayer, error) {
	var erasureLockers listsDsyncLockers
	var path string

	logger.Info("Hosts for distributed mode:")
	for _, ep := range endpointServerPools {
		for _, endpoint := range ep.Endpoints {
			logger.Info("%s:%s", endpoint.Host, endpoint.Path)
			erasureLockers = append(erasureLockers, newLockAPI(endpoint))
			if endpoint.IsLocal {
				path = endpoint.Path
			}
		}
	}

	fs, err := NewFSObjectLayer(path)

	if err != nil {
		return nil, err
	}

	mutex := newNSLock(globalIsDistErasure)
	fs.(*FSObjects).nsMutex = mutex

	objectApi := &MapRServerPools{
		FSObjects: fs.(*FSObjects),
		lockers:   erasureLockers,
	}
	objectApi.FSObjects.GetLockers = objectApi.GetLockers()

	return objectApi, err
}

// GetAllLockers return a list of all lockers for all sets.
func (s *MapRServerPools) GetAllLockers() []dsync.NetLocker {
	var allLockers []dsync.NetLocker
	lockEpSet := set.NewStringSet()
	for _, locker := range s.lockers {
		if locker == nil || !locker.IsOnline() {
			// Skip any offline lockers.
			continue
		}
		if lockEpSet.Contains(locker.String()) {
			// Skip duplicate lockers.
			continue
		}
		lockEpSet.Add(locker.String())
		allLockers = append(allLockers, locker)
	}
	return allLockers
}

func (s *MapRServerPools) GetLockers() func() ([]dsync.NetLocker, string) {
	return func() ([]dsync.NetLocker, string) {
		lockers := make([]dsync.NetLocker, len(s.lockers))
		copy(lockers, s.lockers)
		return lockers, s.lockOwner
	}
}

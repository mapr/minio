package cmd

import (
	"sync"
	"net/http"
	"time"
	"io/ioutil"
	"encoding/json"
)

type UidGidPair struct {
	uid int
	gid int
}

/// Implements TenantMapper interface by maintaining map of tenants in memory and
/// periodically updating it from the specified file
type LocalTenantMapper struct {
	tenantMap map[string]UidGidPair
	tenantMapMutex *sync.RWMutex /// Used to synchronise access to the tenantMap
}

/// Creates new TenantMapper which periodically (every refreshPeriodSeconds second)
/// loads data from tenantListFilename
func newLocalTenantMapper(tenantFilename string, refreshPeriodSeconds int) (TenantMapper, error) {

	localTenantMapper := &LocalTenantMapper{
		tenantMapMutex: &sync.RWMutex{},
	}

	tickerChannel := time.NewTicker(time.Duration(refreshPeriodSeconds) * time.Second)

	err := localTenantMapper.readTenantMappingFile(tenantFilename);
	if err != nil {
		return nil, err
	}

	go func () {
		for {
			<- tickerChannel.C
			localTenantMapper.readTenantMappingFile(tenantFilename)
		}
	}()

	return localTenantMapper, nil
}

/// Parses the HTTP request and handles both AWSAccessKeyID query param
/// and Authorization header to map it to the UID/GID from the tenantMap
func (self LocalTenantMapper) MapCredentials(request *http.Request) (uid, gid int, err error) {
	self.tenantMapMutex.RLock()
	defer self.tenantMapMutex.RUnlock()

	accessKeyId, err := getRequestAccessKeyId(request);
	if err != nil {
		return 0, 0, err
	}

	if uidGidPair, ok := self.tenantMap[accessKeyId]; ok {
		return uidGidPair.uid, uidGidPair.gid, nil
	}

	return 0, 0, errInvalidAccessKeyID
}

func (self *LocalTenantMapper) readTenantMappingFile(tenantFilename string) error {
	self.tenantMapMutex.Lock()
	defer self.tenantMapMutex.Unlock()

	data, err := ioutil.ReadFile(tenantFilename)
	if err != nil {
		// TODO(RostakaGmfun): any error handling here?
		return err
	}

	var unmarshalled interface{}
	err = json.Unmarshal(data, &unmarshalled)

	tenants, ok := unmarshalled.(map[string]UidGidPair)

	if !ok {
		// TODO(RostakaGmfun): any error handling here?
		return errInvalidArgument
	}

	self.tenantMap = tenants

	return nil
}

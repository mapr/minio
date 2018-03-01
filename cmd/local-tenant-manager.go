package cmd

import (
	"encoding/json"
	"io/ioutil"
	"sync"
	"time"
	"syscall"
)

type TenantInfo struct {
	uid int
	gid int
	secretKey string
}

/// Implements TenantManager interface by maintaining map of tenants in memory and
/// periodically updating it from the specified file
type LocalTenantManager struct {
	tenantMap      map[string]TenantInfo
	tenantMapMutex *sync.RWMutex /// Used to synchronise access to the tenantMap
}

/// Creates new TenantMapper which periodically (every refreshPeriodSeconds second)
/// loads data from tenantListFilename
func newLocalTenantManager(tenantFilename string, refreshPeriodSeconds int) (TenantManager, error) {

	localTenantManager := &LocalTenantManager{
		tenantMap: make(map[string]TenantInfo),
		tenantMapMutex: &sync.RWMutex{},
	}

	if tenantFilename != "" {
		localTenantManager.readTenantMappingFile(tenantFilename)
	}

	if refreshPeriodSeconds > 0 {
		tickerChannel := time.NewTicker(time.Duration(refreshPeriodSeconds) * time.Second)

		go func() {
			for {
				<-tickerChannel.C
				localTenantManager.readTenantMappingFile(tenantFilename)
			}
		}()
	}

	return localTenantManager, nil
}

func (self *LocalTenantManager) GetSecretKey(accessKey string) (string, error) {
	self.tenantMapMutex.RLock()
	defer self.tenantMapMutex.RUnlock()

	if tenantInfo, ok := self.tenantMap[accessKey]; ok {
		return tenantInfo.secretKey, nil
	}

	return "", errInvalidAccessKeyID

}

/// Parses the HTTP request and handles both AWSAccessKeyID query param
/// and Authorization header to map it to the UID/GID from the tenantMap
func (self LocalTenantManager) GetUidGid(accessKey string) (uid, gid int, err error) {
	self.tenantMapMutex.RLock()
	defer self.tenantMapMutex.RUnlock()

	if tenantInfo, ok := self.tenantMap[accessKey]; ok {
		return tenantInfo.uid, tenantInfo.gid, nil
	}

	// Use default credentials in case given accessKey was not found in the tenant list
	defaultCredentials := globalServerConfig.GetCredential()
	if defaultCredentials.AccessKey == accessKey {
		return syscall.Geteuid(), syscall.Getegid(), nil
	}

	return 0, 0, errInvalidAccessKeyID
}

func (self *LocalTenantManager) readTenantMappingFile(tenantFilename string) error {
	self.tenantMapMutex.Lock()
	defer self.tenantMapMutex.Unlock()

	data, err := ioutil.ReadFile(tenantFilename)
	if err != nil {
		// TODO(RostakaGmfun): any error handling here?
		return err
	}

	var unmarshalled interface{}
	err = json.Unmarshal(data, &unmarshalled)

	tenants, ok := unmarshalled.(map[string]TenantInfo)

	if !ok {
		// TODO(RostakaGmfun): any error handling here?
		return errInvalidArgument
	}

	self.tenantMap = tenants

	return nil
}

func (self *LocalTenantManager) addTenant(accessKey string, secretKey string, uid int, gid int) error {
	self.tenantMapMutex.Lock()
	defer self.tenantMapMutex.Unlock()

	self.tenantMap[accessKey] = TenantInfo{
		secretKey: secretKey,
		uid: uid,
		gid: gid,
	}

	return nil
}

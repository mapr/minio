package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type TenantInfo struct {
	uid int
	gid int
}

type TenantCredential struct {
	tenant string /// tenant name which owns this credential
	secretKey string
}

/// Implements TenantManager interface by maintaining map of tenants in memory and
/// periodically updating it from the specified file
type LocalTenantManager struct {
	tenants      map[string]TenantInfo /// tenantName -> (uid, gid)
	credentials    map[string]TenantCredential /// accessKey -> (secretKey, tenantName)
	mutex *sync.RWMutex /// Used to synchronise access to the tenantMap
}

/// Creates new TenantMapper which periodically (every refreshPeriodSeconds second)
/// loads data from tenantListFilename
func newLocalTenantManager(tenantFilename string, refreshPeriodSeconds int) (TenantManager, error) {

	localTenantManager := &LocalTenantManager{
		tenants:      make(map[string]TenantInfo),
		credentials:      make(map[string]TenantCredential),
		mutex: &sync.RWMutex{},
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
	self.mutex.RLock()
	defer self.mutex.RUnlock()

	if cred, ok := self.credentials[accessKey]; ok {
		return cred.secretKey, nil
	}

	defaultCredentials := globalServerConfig.GetCredential()
	if defaultCredentials.AccessKey == accessKey {
		return defaultCredentials.SecretKey, nil
	}

	return "", errInvalidAccessKeyID

}

/// Parses the HTTP request and handles both AWSAccessKeyID query param
/// and Authorization header to map it to the UID/GID from the tenantMap
func (self LocalTenantManager) GetUidGid(accessKey string) (uid, gid int, err error) {
	self.mutex.RLock()
	defer self.mutex.RUnlock()

	if cred, ok := self.credentials[accessKey]; ok {
		tenant, _ := self.tenants[cred.tenant]
		return tenant.uid, tenant.gid, nil
	}

	// Use default credentials in case given accessKey was not found in the tenant list
	defaultCredentials := globalServerConfig.GetCredential()
	if defaultCredentials.AccessKey == accessKey {
		return syscall.Geteuid(), syscall.Getegid(), nil
	}

	return 0, 0, errInvalidAccessKeyID
}

func (self *LocalTenantManager) GetUidGidByName(tenantName string) (uid int, gid int, err error) {
	self.mutex.RLock()
	defer self.mutex.RUnlock()


	if tenant, ok := self.tenants[tenantName]; ok {
		return tenant.uid, tenant.gid, nil
	}

	return 0, 0, errInvalidArgument
}

func (self *LocalTenantManager) readTenantMappingFile(tenantFilename string) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	data, err := ioutil.ReadFile(tenantFilename)
	if err != nil {
		// TODO(RostakaGmfun): any error handling here?
		return err
	}

	var unmarshalled interface{}
	err = json.Unmarshal(data, &unmarshalled)

	//tenants := make(map[string]TenantInfo)

	tenantsConfig, ok := unmarshalled.(map[string]interface{})
	if !ok {
		return errInvalidArgument
	}

	tenants, ok := tenantsConfig["tenants"]
	if !ok {
		fmt.Println("Wrong tenants.json format: tenants array not found")
		return errInvalidArgument
	}

	credentials, ok := tenantsConfig["credentials"]
	if !ok {
		fmt.Println("Wrong tenants.json format: credentials array not found")
		return errInvalidArgument
	}

	for _, iface := range tenants.([]interface{}) {
		tenantInfo := iface.(map[string]interface{})
		var tenant TenantInfo
		name, ok := tenantInfo["name"]
		if !ok {
			fmt.Println("Not name field present")
			continue
		}

		uid, ok := tenantInfo["uid"]
		if !ok {
			fmt.Println("Not uid field present")
			continue
		}
		tenant.uid, _ = strconv.Atoi(uid.(string))

		gid, ok := tenantInfo["gid"]
		if !ok {
			fmt.Println("Not gid field present")
			continue
		}
		tenant.gid, _ = strconv.Atoi(gid.(string))
		self.tenants[name.(string)] = tenant
	}
	fmt.Println(self.tenants)

	for _, iface := range credentials.([]interface{}) {
		cred := iface.(map[string]interface{})
		var tenantCred TenantCredential
		accessKey, ok := cred["accessKey"]
		if !ok {
			fmt.Println("Not accessKey field present")
			continue
		}
		secretKey, ok := cred["secretKey"]
		if !ok {
			fmt.Println("Not secretKey field present")
			continue
		}
		tenantCred.secretKey = secretKey.(string)
		tenantName, ok := cred["tenant"]
		if !ok {
			fmt.Println("Not tenant field present")
			continue
		}
		tenantCred.tenant = tenantName.(string)
		self.credentials[accessKey.(string)] = tenantCred
	}
	fmt.Println(self.credentials)

	return nil
}

func (self *LocalTenantManager) GetTenantName(accessKey string) (string, error) {
	self.mutex.RLock()
	cred, ok := self.credentials[accessKey]
	self.mutex.RUnlock()
	if ok {
		return cred.tenant, nil
	}

	defaultCredentials := globalServerConfig.GetCredential()
	if defaultCredentials.AccessKey == accessKey {
		return "default", nil
	}

	return "", errInvalidAccessKeyID
}

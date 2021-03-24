package cmd

import (
	"encoding/json"
	"fmt"
	configEnv "github.com/minio/minio/cmd/config"
	ldap2 "github.com/minio/minio/cmd/config/identity/ldap"
	"github.com/minio/minio/cmd/logger"
	"io/ioutil"
	"os"
	"strings"
)

/// This structure represetns separate configuration file
/// It was made separate to avoid clashes with Minio's config versioning
type MapRMinioConfig struct {
	Version          string   `json:"version",omitempty`          /// Path to the Minio data root directory
	FsPath           string   `json:"fsPath",omitempty`           /// Path to the Minio data root directory
	Port             string   `json:"port",omitempty`             /// Port for server
	DistributedHosts string   `json:"distributedHosts",omitempty` /// Hosts with path for distributed mode
	AccessKey        string   `json:"accessKey",omitempty`        /// Minio accessKey
	SecretKey        string   `json:"secretKey",omitempty`        /// Minio secretKey
	OldAccessKey     string   `json:"oldAccessKey",omitempty`     /// Old Minio accessKey
	OldSecretKey     string   `json:"oldSecretKey",omitempty`     /// Old Minio secretKey
	DeploymentMode   string   `json:"deploymentMode",omitempty`   /// Security scenario to use
	Domain           string   `json:"domain",omitempty`           /// Domain for virtual-hosted–style
	LogPath          string   `json:"logPath",omitempty`          /// Path to the log file
	LogLevel         int      `json:"logLevel",omitempty`         /// Logger verbosity
	Ldap             MapRLdap `json:"ldap",omitempty`             /// MapR's LDAP config
}

type MapRLdap struct {
	ServerAddr         string `json:"serverAddr",omitempty`
	UsernameFormat     string `json:"usernameFormat",omitempty`
	UserDNSearchBaseDN string `json:"userDNSearchBaseDN",omitempty`
	UserDNSearchFilter string `json:"userDNSearchFilter",omitempty`
	GroupSearchFilter  string `json:"groupSearchFilter",omitempty`
	GroupSearchBaseDn  string `json:"groupSearchBaseDn",omitempty`
	LookUpBindDN       string `json:"lookUpBindDN",omitempty`
	LookUpBindPassword string `json:"lookUpBindPassword",omitempty`
	StsExpiry          string `json:"stsExpiry",omitempty`
	TlsSkipVerify      string `json:"tlsSkipVerify",omitempty`
	ServerStartTls     string `json:"serverStartTls",omitempty`
	SeverInsecure      string `json:"severInsecure",omitempty`
}

func parseMapRMinioConfig(maprfsConfigPath string) (config MapRMinioConfig, err error) {
	fmt.Println("Reading MapR Minio config", maprfsConfigPath)

	patchedConfig := false

	data, err := ioutil.ReadFile(maprfsConfigPath)
	if err != nil {
		fmt.Println("Failed to read", maprfsConfigPath)
		return
	}

	version, err := getConfigVersion(data)
	if err != nil {
		return
	}

	if version == "" {
		data, err = migrateConfigToV2(data)
		if err != nil {
			return
		}
		patchedConfig = true
	}

	if json.Unmarshal(data, &config) != nil {
		return
	}

	if patchedConfig {
		config.saveConfig(maprfsConfigPath)
	}

	return
}

func (config MapRMinioConfig) setEnvsIfNecessary() error {
	if err := setEnvIfNecessary(configEnv.EnvDomain, config.Domain); err != nil {
		return err
	}

	ldap := config.Ldap

	if err := setEnvIfNecessary(ldap2.EnvServerAddr, ldap.ServerAddr); err != nil {
		return err
	}
	if err := setEnvIfNecessary(ldap2.EnvUsernameFormat, ldap.UsernameFormat); err != nil {
		return err
	}
	if err := setEnvIfNecessary(ldap2.EnvUserDNSearchBaseDN, ldap.UserDNSearchBaseDN); err != nil {
		return err
	}
	if err := setEnvIfNecessary(ldap2.EnvUserDNSearchFilter, ldap.UserDNSearchFilter); err != nil {
		return err
	}
	if err := setEnvIfNecessary(ldap2.EnvGroupSearchFilter, ldap.GroupSearchFilter); err != nil {
		return err
	}
	if err := setEnvIfNecessary(ldap2.EnvGroupSearchBaseDN, ldap.GroupSearchBaseDn); err != nil {
		return err
	}
	if err := setEnvIfNecessary(ldap2.EnvLookupBindDN, ldap.LookUpBindDN); err != nil {
		return err
	}
	if err := setEnvIfNecessary(ldap2.EnvLookupBindPassword, ldap.LookUpBindPassword); err != nil {
		return err
	}
	if err := setEnvIfNecessary(ldap2.EnvSTSExpiry, ldap.StsExpiry); err != nil {
		return err
	}
	if err := setEnvIfNecessary(ldap2.EnvTLSSkipVerify, ldap.TlsSkipVerify); err != nil {
		return err
	}
	if err := setEnvIfNecessary(ldap2.EnvServerInsecure, ldap.SeverInsecure); err != nil {
		return err
	}
	if err := setEnvIfNecessary(ldap2.EnvServerStartTLS, ldap.ServerStartTls); err != nil {
		return err
	}

	return nil
}

func (config MapRMinioConfig) updateConfig(maprfsConfigPath string) error {
	if maprfsConfigPath != "" && (config.OldAccessKey != "" || config.OldSecretKey != "") {
		logger.Info("Updating config " + maprfsConfigPath)
		newMinioConfig := config
		newMinioConfig.OldAccessKey = ""
		newMinioConfig.OldSecretKey = ""

		if err := newMinioConfig.saveConfig(maprfsConfigPath); err != nil {
			logger.FatalIf(err, "Failed to update config")
		}
	}

	return nil
}

func (config MapRMinioConfig) saveConfig(maprfsConfigPath string) error {
	data, err := json.MarshalIndent(config, "", "\t")
	if err == nil {
		err = ioutil.WriteFile(maprfsConfigPath, data, 644)
	}

	return err
}

func setEnvIfNecessary(variableName, variableValue string) error {
	if v := os.Getenv(variableName); v == "" {
		return os.Setenv(variableName, variableValue)
	}

	return nil
}

func getConfigVersion(data []byte) (version string, err error) {
	var config map[string]interface{}
	if err = json.Unmarshal(data, &config); err != nil {
		return "", err
	}

	if versionValue := config["version"]; versionValue != nil {
		version = fmt.Sprintf("%v", versionValue)
	} else {
		version = ""
	}

	return
}

func migrateConfigToV2(data []byte) (newData []byte, err error) {
	fmt.Println("Migrating config to V2...")

	configString := string(data)
	configString = strings.ReplaceAll(configString, "\"usernameSearchBaseDn\"", "\"userDNSearchBaseDN\"")
	configString = strings.ReplaceAll(configString, "\"usernameSearchFilter\"", "\"userDNSearchFilter\"")

	newConfig := MapRMinioConfig{}
	err = json.Unmarshal([]byte(configString), &newConfig)
	if err != nil {
		fmt.Println("Failed to parse config")
		return nil, err
	}

	newConfig.Version = "2"

	return json.Marshal(newConfig)
}

package cmd

import (
	"encoding/json"
	"fmt"
	configEnv "github.com/minio/minio/cmd/config"
	ldap2 "github.com/minio/minio/cmd/config/identity/ldap"
	"github.com/minio/minio/cmd/logger"
	"io/ioutil"
	"os"
)

/// This structure represetns separate configuration file
/// It was made separate to avoid clashes with Minio's config versioning
type MapRMinioConfig struct {
	FsPath         string   `json:"fsPath",omitempty`         /// Path to the Minio data root directory
	AccessKey      string   `json:"accessKey",omitempty`      /// Minio accessKey
	SecretKey      string   `json:"secretKey",omitempty`      /// Minio secretKey
	OldAccessKey   string   `json:"oldAccessKey",omitempty`   /// Old Minio accessKey
	OldSecretKey   string   `json:"oldSecretKey",omitempty`   /// Old Minio secretKey
	DeploymentMode string   `json:"deploymentMode",omitempty` /// Security scenario to use
	Domain         string   `json:"domain",omitempty`         /// Domain for virtual-hosted–style
	LogPath        string   `json:"logPath",omitempty`        /// Path to the log file
	LogLevel       int      `json:"logLevel",omitempty`       /// Logger verbosity
	Ldap           MapRLdap `json:"ldap",omitempty`           /// MapR's LDAP config
}

type MapRLdap struct {
	ServerAddr           string `json:"serverAddr",omitempty`
	UsernameFormat       string `json:"usernameFormat",omitempty`
	UsernameSearchFilter string `json:"usernameSearchFilter",omitempty`
	GroupSearchFilter    string `json:"groupSearchFilter",omitempty`
	GroupSearchBaseDn    string `json:"groupSearchBaseDn",omitempty`
	UsernameSearchBaseDn string `json:"usernameSearchBaseDn",omitempty`
	GroupNameAttribute   string `json:"groupNameAttribute",omitempty`
	StsExpiry            string `json:"stsExpiry",omitempty`
	TlsSkipVerify        string `json:"tlsSkipVerify",omitempty`
	ServerStartTls       string `json:"serverStartTls",omitempty`
	SeverInsecure        string `json:"severInsecure",omitempty`
}

func parseMapRMinioConfig(maprfsConfigPath string) (config MapRMinioConfig, err error) {
	fmt.Println("Reading MapR Minio config", maprfsConfigPath)
	data, err := ioutil.ReadFile(maprfsConfigPath)
	if err != nil {
		fmt.Println("Failed to read", maprfsConfigPath)
		return config, err
	}

	err = json.Unmarshal(data, &config)

	return config, err
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
	if err := setEnvIfNecessary(ldap2.EnvUsernameSearchFilter, ldap.UsernameSearchFilter); err != nil {
		return err
	}
	if err := setEnvIfNecessary(ldap2.EnvGroupSearchFilter, ldap.GroupSearchFilter); err != nil {
		return err
	}
	if err := setEnvIfNecessary(ldap2.EnvGroupSearchBaseDN, ldap.GroupSearchBaseDn); err != nil {
		return err
	}
	if err := setEnvIfNecessary(ldap2.EnvUsernameSearchBaseDN, ldap.UsernameSearchBaseDn); err != nil {
		return err
	}
	if err := setEnvIfNecessary(ldap2.EnvGroupNameAttribute, ldap.GroupNameAttribute); err != nil {
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

		data, err := json.MarshalIndent(newMinioConfig, "", "\t")
		if err == nil {
			err = ioutil.WriteFile(maprfsConfigPath, data, 644)
		}

		if err != nil {
			logger.FatalIf(err, "Failed to update config")
		}
	}

	return nil
}

func setEnvIfNecessary(variableName, variableValue string) error {
	if v := os.Getenv(variableName); v == "" {
		return os.Setenv(variableName, variableValue)
	}

	return nil
}

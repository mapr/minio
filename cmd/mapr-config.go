package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/minio/minio-go/v7/pkg/set"
	"github.com/minio/minio/cmd/logger"
	"io/ioutil"
)

/// This structure represetns separate configuration file
/// It was made separate to avoid clashes with Minio's config versioning
type MapRMinioConfig struct {
	FsPath         string `json:"fsPath",omitempty`         /// Path to the Minio data root directory
	AccessKey      string `json:"accessKey",omitempty`      /// Minio accessKey
	SecretKey      string `json:"secretKey",omitempty`      /// Minio secretKey
	OldAccessKey   string `json:"oldAccessKey",omitempty`   /// Old Minio accessKey
	OldSecretKey   string `json:"oldSecretKey",omitempty`   /// Old Minio secretKey
	DeploymentMode string `json:"deploymentMode",omitempty` /// Security scenario to use
	LogPath        string `json:"logPath",omitempty`        /// Path to the log file
	LogLevel       int    `json:"logLevel",omitempty`       /// Logger verbosity
}

func parseMapRMinioConfig(maprfsConfigPath string) (config MapRMinioConfig, err error) {
	fmt.Println("Reading MapR Minio config", maprfsConfigPath)
	data, err := ioutil.ReadFile(maprfsConfigPath)
	if err != nil {
		fmt.Println("Failed to read", maprfsConfigPath)
		return config, err
	}

	err = json.Unmarshal(data, &config)

	if err != nil {
		return config, err
	}

	if config.DeploymentMode == "" {
		config.DeploymentMode = "FS"
	}

	if !isSupportedDeploymentMode(config.DeploymentMode) {
		logger.FatalIf(errInvalidArgument, "Unsupported deployment mode specified"+config.DeploymentMode)
		return config, errInvalidArgument
	}

	return config, err
}

func isSupportedDeploymentMode(mode string) bool {
	supportedDeploymentModes := set.StringSet{
		"FS": {},
		"S3": {},
	}

	_, ok := supportedDeploymentModes[mode]
	return ok
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

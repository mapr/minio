package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/minio/minio/cmd/logger"
	"github.com/minio/minio-go/pkg/set"
)


/// This structure represetns separate configuration file
/// It was made separate to avoid clashes with Minio's config versioning
type MapRMinioConfig struct {
	FsPath string `json:fsPath` /// Path to the Minio data root directory
	TenantsFile string `json:tenantsFile` /// Path to the tenants.json file
	DeploymentMode string  `json:deploymentMode` /// Security scenario to use
	LogPath string `json:logPath` /// Path to the log file
	LogLevel int `json:logLevel` /// Logger verbosity
}

func parseMapRMinioConfig(maprfsConfig string) (config MapRMinioConfig, err error) {
	fmt.Println("Reading MapR Minio config", maprfsConfig)
	data, err := ioutil.ReadFile(maprfsConfig)
	if err != nil {
		fmt.Println("Failed to read", maprfsConfig)
		return config, err
	}

	err = json.Unmarshal(data, &config)

	if err != nil {
		return config, err
	}

	fmt.Println(config)

	if config.DeploymentMode == "" {
		config.DeploymentMode = "mixed"
	}

	if !isSupportedDeploymentMode(config.DeploymentMode) {
		logger.FatalIf(errInvalidArgument, "Unsupported deployment mode specified" + config.DeploymentMode)
		return config, errInvalidArgument
	}

	return config, err
}

func isSupportedDeploymentMode(mode string) bool {
	supportedDeploymentModes := set.StringSet {
		"fs_only": {},
		"mixed": {},
		"s3_only": {},
	}

	_, ok := supportedDeploymentModes[mode]
	return ok
}
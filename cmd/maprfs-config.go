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
	SecurityScenario string  `json:securityScenario` /// Security scenario to use
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

	if config.SecurityScenario == "" {
		config.SecurityScenario = "hybrid"
	}

	if !isSupportedSecurityScenario(config.SecurityScenario) {
		logger.FatalIf(errInvalidArgument, "Unsupported security scenario specified" + config.SecurityScenario)
		return config, errInvalidArgument
	}

	return config, err
}

func isSupportedSecurityScenario(scenario string) bool {
	supportedSecurityScenarios := set.StringSet {
		"fs_only": {},
		"hybrid": {},
		"s3_only": {},
	}

	_, ok := supportedSecurityScenarios[scenario]
	return ok
}

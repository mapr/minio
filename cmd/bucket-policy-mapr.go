package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"strconv"

	"github.com/minio/minio-go/pkg/policy"
	"github.com/minio/minio-go/pkg/set"
)

type MapRAce struct {
	lookupdirAllowedUsers set.StringSet
	lookupdirDeniedUsers  set.StringSet

	readdirAllowedUsers set.StringSet
	readdirDeniedUsers  set.StringSet

	readfileAllowedUsers set.StringSet
	readfileDeniedUsers  set.StringSet

	writefileAllowedUsers set.StringSet
	writefileDeniedUsers  set.StringSet

	addchildAllowedUsers set.StringSet
	addchildDeniedUsers  set.StringSet

	deletechildAllowedUsers set.StringSet
	deletechildDeniedUsers  set.StringSet
}

func newMapRAce() MapRAce {
	var maprAce MapRAce
	maprAce.lookupdirAllowedUsers = make(set.StringSet)
	maprAce.lookupdirDeniedUsers = make(set.StringSet)

	maprAce.readdirAllowedUsers = make(set.StringSet)
	maprAce.readdirDeniedUsers = make(set.StringSet)

	maprAce.readfileAllowedUsers = make(set.StringSet)
	maprAce.readfileDeniedUsers = make(set.StringSet)

	maprAce.writefileAllowedUsers = make(set.StringSet)
	maprAce.writefileDeniedUsers = make(set.StringSet)

	return maprAce
}

// Compiles single Ace expression which can be used for
func compileSingleAce(allowedTenants []string, deniedTenants []string) (string, error) {
	allowedUids := make([]string, len(allowedTenants))
	deniedUids := make([]string, len(deniedTenants))

	for i, tenant := range allowedTenants {
		uid, _, err := globalTenantManager.GetUidGidByName(tenant)
		if err != nil {
			return "", err
		}
		allowedUids[i] = "u:" + strconv.Itoa(uid)
	}

	for i, tenant := range deniedTenants {
		uid, _, err := globalTenantManager.GetUidGidByName(tenant)
		if err != nil {
			return "", err
		}
		deniedUids[i] = "!u:" + strconv.Itoa(uid)
	}

	allowedAce := "(" + strings.Join(allowedUids, " | ") + ")"
	deniedAce := "(" + strings.Join(deniedUids, " & ") + ")"

	var finalAce string

	if len(allowedAce) > 2 {
		finalAce += allowedAce
	}

	if len(deniedAce) > 2 {
		finalAce += deniedAce
	}

	return finalAce, nil
}

/// Compiles full argument to `hadoop mfs -setace`
func compileAce(ace MapRAce) (string, error) {
	lookupdirAce, err := compileSingleAce(ace.lookupdirAllowedUsers.ToSlice(), ace.lookupdirDeniedUsers.ToSlice())
	if err != nil {
		return "", err
	}
	lookupdirAce = "ld:" + lookupdirAce

	readdirAce, err := compileSingleAce(ace.readdirAllowedUsers.ToSlice(), ace.readdirDeniedUsers.ToSlice())
	if err != nil {
		return "", err
	}
	readdirAce = "rd:" + readdirAce

	readfileAce, err := compileSingleAce(ace.readfileAllowedUsers.ToSlice(), ace.readfileDeniedUsers.ToSlice())
	if err != nil {
		return "", err
	}
	readfileAce = "rf:" + readfileAce

	writefileAce, err := compileSingleAce(ace.writefileAllowedUsers.ToSlice(), ace.writefileDeniedUsers.ToSlice())
	if err != nil {
		return "", err
	}
	writefileAce = "wf:" + writefileAce

	return strings.Join([]string{lookupdirAce, readdirAce, readfileAce, writefileAce}, ","), nil
}

func policyActionToAce(action string, user string, effect string) MapRAce {
	ace := newMapRAce()
	switch action {
	case "s3:ListBucket":
		if effect == "Allow" {
			ace.readdirAllowedUsers.Add(user)
		} else {
			ace.readdirDeniedUsers.Add(user)
		}
	case "s3:GetObject":
		if effect == "Allow" {
			ace.lookupdirAllowedUsers.Add(user)
			ace.readfileAllowedUsers.Add(user)
		} else {
			ace.lookupdirDeniedUsers.Add(user)
			ace.readfileDeniedUsers.Add(user)
		}
	case "s3:DeleteObject":
		if effect == "Allow" {
			ace.readdirAllowedUsers.Add(user)
			ace.deletechildAllowedUsers.Add(user)
		} else {
			ace.readdirDeniedUsers.Add(user)
			ace.deletechildDeniedUsers.Add(user)
		}
	case "s3:PutObject":
		if effect == "Allow" {
			ace.lookupdirAllowedUsers.Add(user)
			ace.readdirAllowedUsers.Add(user)
			ace.writefileAllowedUsers.Add(user)
			ace.addchildAllowedUsers.Add(user)
		} else {
			ace.lookupdirDeniedUsers.Add(user)
			ace.readdirDeniedUsers.Add(user)
			ace.writefileDeniedUsers.Add(user)
			ace.addchildDeniedUsers.Add(user)
		}
	}

	return ace
}

func getMapRFSRelativePath(path string) (string, error) {
	fmt.Println("resource path: ", path)
	if !strings.HasPrefix(path, globalMapRFSMountPoint) {
		fmt.Println("wrong resource path prefix ", path, " ", globalMapRFSMountPoint)
		return "", errInvalidArgument
	}

	return strings.TrimPrefix(path, globalMapRFSMountPoint), nil
}

func getBucketPath(bucket string) string {
	objectApi := newObjectLayerFn(nil)
	return pathJoin(objectApi.(*FSObjects).fsPath, bucket)
}

func getPathFromResource(resource string) (path string, err error) {
	resourcePrefix := strings.SplitAfter(resource, bucketARNPrefix)[1]
	// Get raw FSOBjects to retrieve bucket directory
	fmt.Println("resource: ", resource)
	fmt.Println("resourcePrefix: ", resourcePrefix)

	resourceSplit := strings.Split(resourcePrefix, "/")
	if len(resourceSplit) == 0 {
		return "", errInvalidArgument
	}

	if len(resourceSplit) == 1 || resourceSplit[1] == "*" {
		return getMapRFSRelativePath(getBucketPath(resourceSplit[0]))
	}

	return getMapRFSRelativePath(getBucketPath(resourcePrefix))
}

func executeSetAce(ace string, resource string) error {
	path, err := getPathFromResource(resource)
	if err != nil {
		return err
	}
	fmt.Println("Setting ACE ", ace, " for node ", path)
	out, err := exec.Command("hadoop", "mfs", "-setace", "-R", "-aces", ace, path).CombinedOutput()
	fmt.Println(string(out[:]))
	fmt.Println(err)
	return err
}

func getUnixPathFromResource(resource string) (path string, err error) {
	resourcePrefix := strings.SplitAfter(resource, bucketARNPrefix)[1]
	objectApi := newObjectLayerFn(nil)

	resourceSplit := strings.Split(resourcePrefix, "/")
	if len(resourceSplit) == 0 {
		return "", errInvalidArgument
	}

	bucket := resourceSplit[0]
	return pathJoin(objectApi.(*FSObjects).fsPath, bucket), nil
}

func executeSetAllPrincipal(resource string, action string, user string, effect string) error {

	fmodes := map[string]map[string]uint32 {
        "s3:ListBucket":   {"Allow": 0775, "Deny": 0770},
		"Hs3:GetObjecte":  {"Allow": 0775, "Deny": 0770},
		"s3:DeleteObject": {"Allow": 0777, "Deny": 0770},
		"s3:PutObject":    {"Allow": 0777, "Deny": 0770},
		"s3:*":            {"Allow": 0777, "Deny": 0770}}

	path, err := getUnixPathFromResource(resource)
	if err != nil {
		return err
	}
	fmt.Println("path: ", path)

	err = executeDelAce(resource)
	if err != nil {
		return err
	}

	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		return os.Chmod(path, os.FileMode(fmodes[action][effect]))
	})
	fmt.Println(err)
	return err
}

func executeDelAce(resource string) error {
	path, err := getPathFromResource(resource)
	if err != nil {
		return err
	}
	fmt.Println("Deleting ACE for node ", path)
	out, err := exec.Command("hadoop", "mfs", "-delace", "-R", path).CombinedOutput()
	fmt.Println(string(out[:]))
	fmt.Println(err)
	return err
}

func isInStatementsArray(statements []policy.Statement, statement policy.Statement) bool {
	for _, s := range statements {
		if s.Sid == statement.Sid {
			return true
		}
	}

	return false
}

func processPolicyAction(resource string, action string, principal string, effect string) error {
	if principal == "*" {
		return executeSetAllPrincipal(resource, action, principal, effect)
	}

	maprAce := policyActionToAce(action, principal, effect)

	ace, err := compileAce(maprAce)
	if err != nil {
		fmt.Println("Failed to compile ace")
		return err
	}

	err = executeDelAce(resource)
	if err != nil {
		fmt.Println("Failed to execute ace")
		return err
	}
	err = executeSetAce(ace, resource)
	if err != nil {
		fmt.Println("Failed to execute ace")
		return err
	}

	return nil
}

func SetMapRFSBucketPolicy(bucketPolicy policy.BucketAccessPolicy) error {
	// Maps resource name to list Statements which reference it
	statementsPerResource := make(map[string][]policy.Statement)
	for _, statement := range bucketPolicy.Statements {
		for resource := range statement.Resources {
			if !isInStatementsArray(statementsPerResource[resource], statement) {
				statementsPerResource[resource] = append(statementsPerResource[resource], statement)
			}
		}
	}

	for resource, statements := range statementsPerResource {
		for _, statement := range statements {
			for action := range statement.Actions {
				for principal := range statement.Principal.AWS {
					err := processPolicyAction(resource, action, principal, statement.Effect)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func RemoveMapRFSBucketPolicy(bucket string, policy policy.BucketAccessPolicy) error {
	resources := make([]string, 1)

	for _, statement := range policy.Statements {
		for resource := range statement.Resources {
			resources = append(resources, resource)
		}
	}

	for _, resource := range resources {
		err := executeDelAce(resource)
		if err != nil {
			return err
		}
	}
	return ApplyDefaultMapRFSBucketPolicy(bucket)
}

func ApplyDefaultMapRFSBucketPolicy(bucket string) error {
	path := getBucketPath(bucket)

	return filepath.Walk(path, func(name string, info os.FileInfo, err error) error {
		return os.Chmod(name, os.FileMode(GetDefaultMapRFSPolicyBits()))
	})
}

var supportedDefaultBucketPolicies = map[string]os.FileMode {
	"public-r": os.FileMode(0775),
	"public-rw": os.FileMode(0777),
	"private": os.FileMode(0770),
}

func IsSupportedDefaultBucketPolicy(defaultPolicy string) bool {
	_, ok := supportedDefaultBucketPolicies[defaultPolicy]
	return ok
}

func GetDefaultMapRFSPolicyBits() os.FileMode {
	return supportedDefaultBucketPolicies[globalDefaultBucketPolicy]
}

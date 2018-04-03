package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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

func processPolicyAction(action string, user string, effect string) MapRAce {
	ace := newMapRAce()
	switch action {
	case "s3:ListBucket":
		if effect == "Allow" {
			ace.lookupdirAllowedUsers.Add(user)
		} else {
			ace.lookupdirDeniedUsers.Add(user)
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
		fallthrough
	case "s3:PutObject":
		if effect == "Allow" {
			ace.lookupdirAllowedUsers.Add(user)
			ace.readdirAllowedUsers.Add(user)
			ace.readfileAllowedUsers.Add(user)
			ace.writefileAllowedUsers.Add(user)
		} else {
			ace.lookupdirDeniedUsers.Add(user)
			ace.readdirDeniedUsers.Add(user)
			ace.readfileDeniedUsers.Add(user)
			ace.writefileDeniedUsers.Add(user)
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

func getPathFromResource(resource string) (path string, err error) {
	resourcePrefix := strings.SplitAfter(resource, bucketARNPrefix)[1]
	// Get raw FSOBjects to retrieve bucket directory
	objectApi := newObjectLayerFn(nil)
	fmt.Println("resource: ", resource)
	fmt.Println("resourcePrefix: ", resourcePrefix)

	resourceSplit := strings.Split(resourcePrefix, "/")
	if len(resourceSplit) == 0 {
		return "", errInvalidArgument
	}
	bucket := resourceSplit[0]

	if len(resourceSplit) == 1 || resourceSplit[1] == "*" {
		return getMapRFSRelativePath(pathJoin(objectApi.(*FSObjects).fsPath, bucket))
	}

	return getMapRFSRelativePath(pathJoin(objectApi.(*FSObjects).fsPath, resourcePrefix))
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
	var out []byte

	fmodes := map[string]map[string]uint32{
		"s3:ListBucket":   {"Allow": 0775, "Deny": 0770},
		"Hs3:GetObjecte":  {"Allow": 0775, "Deny": 0770},
		"s3:DeleteObject": {"Allow": 0777, "Deny": 0770},
		"s3:PutObject":    {"Allow": 0777, "Deny": 0770},
		"s3:*":            {"Allow": 0777, "Deny": 0770}}

	//	fmt.Println("resource: ", resource)
	//	fmt.Println("action: ", action)
	//	fmt.Println("user: ", user)
	//	fmt.Println("effect: ", effect)

	path, err := getUnixPathFromResource(resource)
	if err != nil {
		return err
	}
	fmt.Println("path: ", path)

	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		return os.Chmod(path, os.FileMode(fmodes[action][effect]))
	})

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

func generateAceFromPolicy(bucketPolicy policy.BucketAccessPolicy) error {
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

					if principal == "*" {
						return executeSetAllPrincipal(resource, action, principal, statement.Effect)
					}

					maprAce := processPolicyAction(action, principal, statement.Effect)

					ace, err := compileAce(maprAce)
					if err != nil {
						fmt.Println("Failed to compile ace")
						return err
					}

					err = executeSetAce(ace, resource)
					if err != nil {
						fmt.Println("Failed to execute ace")
						return err
					}
				}
			}
		}
	}
	return nil
}

func deleteAce(bucketPolicy policy.BucketAccessPolicy) error {
	return nil
}

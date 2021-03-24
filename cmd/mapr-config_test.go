package cmd

import (
	"encoding/json"
	"testing"
)

func TestCorrectGetConfigVersion(t *testing.T) {
	config := "{\"version\":\"2\"}"
	data := []byte(config)

	version, err := getConfigVersion(data)
	if err != nil {
		t.Error(err)
	}

	if expected := "2"; version != expected {
		t.Fatal("Invalid version: ", version, "expected", expected)
	}
}

func TestInvalidGetConfigVersion(t *testing.T) {
	config := "{}"
	data := []byte(config)

	version, err := getConfigVersion(data)
	if err != nil {
		t.Error(err)
	}

	if expected := ""; version != expected {
		t.Fatal("Invalid version: ", version, "expected empty")
	}
}

func TestMigrateToV2(t *testing.T) {
	config := "{\n  \"fsPath\": \"~/minio-test/data\",\n  \"deploymentMode\": \"S3\",\n  \"accessKey\": \"minioadmin\",\n  \"secretKey\": \"minioadmin\",\n  \"oldAccessKey\": \"\",\n  \"oldSecretKey\": \"\",\n  \"port\": \"9000\",\n  \"logPath\": \"\",\n  \"logLevel\": 4,\n  \"ldap\": {\n    \"serverAddr\": \"1\",\n    \"usernameFormat\": \"2\",\n    \"usernameSearchFilter\": \"3\",\n    \"groupSearchFilter\": \"4\",\n    \"groupSearchBaseDn\": \"5\",\n    \"usernameSearchBaseDn\": \"6\",\n    \"groupNameAttribute\": \"7\",\n    \"stsExpiry\": \"8\",\n    \"tlsSkipVerify\": \"9\",\n    \"serverStartTls\": \"10\",\n    \"severInsecure\": \"11\"\n  }\n}\n"

	data, err := migrateConfigToV2([]byte(config))
	if err != nil {
		t.Error(err)
	}
	var newConfig map[string]interface{}
	if err = json.Unmarshal(data, &newConfig); err != nil {
		t.Error(err)
	}

	if version, expected := newConfig["version"], "2"; version != expected {
		t.Fatal("Invalid version: ", version, "expected", expected)
	}

	ldap := newConfig["ldap"].(map[string]interface{})

	if groupNameAttribute := ldap["groupNameAttribute"]; groupNameAttribute != nil {
		t.Fatal("Expected not contain \"groupNameAttribute\" but was", groupNameAttribute)
	}

	if userDNSearchBaseDN, expected := ldap["userDNSearchBaseDN"], "6"; userDNSearchBaseDN != expected {
		t.Fatal("Invalid \"userDNSearchBaseDN\": ", userDNSearchBaseDN, "expected", expected)
	}

	if userDNSearchFilter, expected := ldap["userDNSearchFilter"], "3"; userDNSearchFilter != expected {
		t.Fatal("Invalid \"userDNSearchFilter\": ", userDNSearchFilter, "expected", expected)
	}

	if lookUpBindDN, expected := ldap["lookUpBindDN"], ""; lookUpBindDN != expected {
		t.Fatal("Invalid \"lookUpBindDN\": ", lookUpBindDN, "expected empty")
	}

	if lookUpBindPassword, expected := ldap["lookUpBindPassword"], ""; lookUpBindPassword != expected {
		t.Fatal("Invalid \"lookUpBindPassword\": ", lookUpBindPassword, "expected empty")
	}
}

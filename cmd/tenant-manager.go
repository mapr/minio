package cmd

import (
	"github.com/minio/minio-go/pkg/policy"
)

type TenantManager interface {
	/// Retrieve secret key for the given accessKey
	GetSecretKey(accessKey string) (secretKey string, err error)

	/// Maps AWS credentials from HTTP request to UNIX uid and gid
	GetUidGid(accessKey string) (uid int, gid int, err error)

	/// Retrieve tenant UUID - a secure hash which depends on teneant meta info
	GetTenantUUID(accessKey string) (tenantUUID string, err error)

	/// Get human-readable tenant name - used to identify tenants in IAM bucket policies
	GetTenantName(accessKey string) (tenantName string, err error)

	/// Retrieve list of bucket policies which reference given tenant and given bucket name
	GetAssociatedBucketPolicies(tenantName string, bucketName string) ([]policy.BucketAccessPolicy, error)
}

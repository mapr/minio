package cmd

type TenantManager interface {
	/// Retrieve secret key for the given accessKey
	GetSecretKey(accessKey string) (secretKey string, err error)

	/// Maps AWS credentials from HTTP request to UNIX uid and gid
	GetUidGid(accessKey string) (uid int, gid int, err error)

	/// Retrieve UID/GID pair by tenant name
	GetUidGidByName(tenantName string) (uid int, gid int, err error)

	/// Get human-readable tenant name - used to identify tenants in IAM bucket policies
	GetTenantName(accessKey string) (tenantName string, err error)

	/// Look for tenant name by UID
	GetTenantNameByUid(uid int) (string, error)
}

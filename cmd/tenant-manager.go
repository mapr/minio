package cmd

type TenantManager interface {
	/// Retrieve secret key for the given accessKey
	GetSecretKey(accessKey string) (secretKey string, err error)

	/// Maps AWS credentials from HTTP request to UNIX uid and gid
	GetUidGid(accessKey string) (uid int, gid int, err error)
}

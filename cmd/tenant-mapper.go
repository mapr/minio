package cmd

import (
	"net/http"
)

type TenantMapper interface {
	/// Maps AWS credentials from HTTP request to UNIX uid and gid
	MapCredentials(*http.Request) (uid, gid int, err error)
}

package remote

import (
	"strings"

	"z-server-backup-tools/backend/model"
	"z-server-backup-tools/backend/util"
)

// BuildRemoteCommand builds a remote argv line for the given OS type.
func BuildRemoteCommand(osType, srv string, argv ...string) string {
	osType = model.NormalizeOSType(osType)
	srv = util.NormalizeRemotePathForOS(srv, osType)
	quote := util.QuoteWindowsArg
	if model.IsLinuxOS(osType) {
		quote = util.QuoteShellArg
	}
	parts := []string{quote(srv)}
	for _, a := range argv {
		parts = append(parts, quote(a))
	}
	return strings.Join(parts, " ")
}

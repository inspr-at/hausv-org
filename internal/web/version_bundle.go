package web

import "github.com/inspr-at/hausv-org/internal/versionbundle"

// Raw Go builds also fail closed before serving an altered embedded bundle.
// Supported builds verify source files and tracked provenance before compiling.
func init() {
	if err := versionbundle.Verify(Assets, versionbundle.Directory); err != nil {
		panic(err)
	}
}

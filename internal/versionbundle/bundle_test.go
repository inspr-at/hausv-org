package versionbundle

import (
	"io/fs"
	"os"
	"testing"
	"testing/fstest"
)

func TestCompleteBundleClosure(t *testing.T) {
	source := os.DirFS("../web/assets/versioning")
	read := func() fstest.MapFS {
		m := fstest.MapFS{}
		entries, err := fs.ReadDir(source, ".")
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			b, err := fs.ReadFile(source, e.Name())
			if err != nil {
				t.Fatal(err)
			}
			m[e.Name()] = &fstest.MapFile{Data: b}
		}
		return m
	}
	if err := Verify(read(), "."); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"display.json", "version.js", "presentation.js", "version-interaction.js", "auto-animate.js", "auto-animate-license.js", "package.json", "manifest.json"} {
		t.Run(name, func(t *testing.T) {
			missing := read()
			delete(missing, name)
			if Verify(missing, ".") == nil {
				t.Fatal("missing payload accepted")
			}
			altered := read()
			altered[name].Data = append(altered[name].Data, ' ')
			if Verify(altered, ".") == nil {
				t.Fatal("altered payload accepted")
			}
		})
	}
	extra := read()
	extra["untracked.js"] = &fstest.MapFile{Data: []byte("extra")}
	if Verify(extra, ".") == nil {
		t.Fatal("extra payload accepted")
	}
	nested := read()
	nested["extra/file"] = &fstest.MapFile{}
	if Verify(nested, ".") == nil {
		t.Fatal("extra directory accepted")
	}
}

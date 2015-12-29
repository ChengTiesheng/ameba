package docker2rkt

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/appc/spec/schema"
	"github.com/containerops/ameba/docker/docker2rkt/convert"
	"github.com/containerops/ameba/docker/docker2rkt/convert/parse"
)

// Input parameters:
// dockerImgPath: A local Docker file path or a Docker registry URL.

// Output parameters:
// string: ACI image path.
// *schema.ImageManifest: ACI manifest converted from docker.
func Docker2rkt(dockerImgPath string) (string, *schema.ImageManifest, error) {

	var aciLayerPaths []string

	aciLayerPaths, err := convert.ConvertFile(dockerImgPath, ".", os.TempDir())
	if err != nil {
		return "", nil, fmt.Errorf("conversion error: %v", err)
	}

	manifest, err := parse.GetManifest(aciLayerPaths[len(aciLayerPaths)-1])
	if err != nil {
		return "", nil, err
	}

	aciPath := aciLayerPaths[len(aciLayerPaths)-1]
	aciPath, _ = filepath.Abs(aciPath)
	return aciPath, manifest, nil
}

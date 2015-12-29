package convert

import (
	"archive/tar"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/appc/spec/aci"
	"github.com/appc/spec/pkg/acirenderer"
	"github.com/appc/spec/schema"
	"github.com/appc/spec/schema/types"
	"github.com/containerops/ameba/docker/docker2rkt/convert/build"
	"github.com/containerops/ameba/docker/docker2rkt/convert/parse"
	"github.com/containerops/ameba/docker/docker2rkt/convert/utils"
)

const (
	hashPrefix = "sha512-"
)

type aciInfo struct {
	path          string
	key           string
	ImageManifest *schema.ImageManifest
}

// ConversionStore is an simple implementation of the acirenderer.ACIRegistry
// interface. It stores the Docker layers converted to ACI so we can take
// advantage of acirenderer to generate a squashed ACI Image.
type ConversionStore struct {
	acis map[string]*aciInfo
}

func NewConversionStore() *ConversionStore {
	return &ConversionStore{acis: make(map[string]*aciInfo)}
}

func (ms *ConversionStore) WriteACI(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	cr, err := aci.NewCompressedReader(f)
	if err != nil {
		return "", err
	}
	defer cr.Close()

	h := sha512.New()
	r := io.TeeReader(cr, h)

	// read the file so we can get the hash
	if _, err := io.Copy(ioutil.Discard, r); err != nil {
		return "", fmt.Errorf("error reading ACI: %v", err)
	}

	im, err := aci.ManifestFromImage(f)
	if err != nil {
		return "", err
	}

	key := ms.HashToKey(h)
	ms.acis[key] = &aciInfo{path: path, key: key, ImageManifest: im}
	return key, nil
}

func (ms *ConversionStore) GetImageManifest(key string) (*schema.ImageManifest, error) {
	aci, ok := ms.acis[key]
	if !ok {
		return nil, fmt.Errorf("aci with key: %s not found", key)
	}
	return aci.ImageManifest, nil
}

func (ms *ConversionStore) GetACI(name types.ACIdentifier, labels types.Labels) (string, error) {
	for _, aci := range ms.acis {
		// we implement this function to comply with the interface so don't
		// bother implementing a proper label check
		if aci.ImageManifest.Name.String() == name.String() {
			return aci.key, nil
		}
	}
	return "", fmt.Errorf("aci not found")
}

func (ms *ConversionStore) ReadStream(key string) (io.ReadCloser, error) {
	img, ok := ms.acis[key]
	if !ok {
		return nil, fmt.Errorf("stream for key: %s not found", key)
	}
	f, err := os.Open(img.path)
	if err != nil {
		return nil, fmt.Errorf("error opening aci: %s", img.path)
	}

	tr, err := aci.NewCompressedReader(f)
	if err != nil {
		return nil, err
	}

	return tr, nil
}

func (ms *ConversionStore) ResolveKey(key string) (string, error) {
	return key, nil
}

func (ms *ConversionStore) HashToKey(h hash.Hash) string {
	s := h.Sum(nil)
	return fmt.Sprintf("%s%x", hashPrefix, s)
}

// It returns the list of generated ACI paths.
func ConvertFile(filePath string, outputDir string, tmpDir string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer f.Close()

	return convertRun(f, outputDir, tmpDir)
}

// GetIndexName returns the docker index server from a docker URL.
func GetIndexName(dockerURL string) string {
	index, _ := parse.SplitReposName(dockerURL)
	return index
}

// GetDockercfgAuth reads a ~/.dockercfg file and returns the username and password
// of the given docker index server.
func GetDockercfgAuth(indexServer string) (string, string, error) {
	return parse.GetAuthInfo(indexServer)
}

func convertRun(file *os.File, outputDir string, tmpDir string) ([]string, error) {
	ancestry, appName, err := build.GetImageInfo(file)
	if err != nil {
		return nil, err
	}

	layersOutputDir := outputDir

	layersOutputDir, err = ioutil.TempDir(tmpDir, "docker2aci-")
	if err != nil {
		return nil, fmt.Errorf("error creating dir: %v", err)
	}
	defer os.RemoveAll(layersOutputDir)

	conversionStore := NewConversionStore()

	var images acirenderer.Images
	var aciLayerPaths []string
	var curPwl []string
	for i := len(ancestry) - 1; i >= 0; i-- {
		layerID := ancestry[i]

		aciPath, manifest, err := build.BuildACI(file, appName, i, layerID, layersOutputDir, tmpDir, curPwl)
		if err != nil {
			return nil, fmt.Errorf("error building layer: %v", err)
		}

		key, err := conversionStore.WriteACI(aciPath)
		if err != nil {
			return nil, fmt.Errorf("error inserting in the conversion store: %v", err)
		}

		images = append(images, acirenderer.Image{Im: manifest, Key: key, Level: uint16(i)})
		aciLayerPaths = append(aciLayerPaths, aciPath)
		curPwl = manifest.PathWhitelist
	}

	// acirenderer expects images in order from upper to base layer
	images = utils.ReverseImages(images)

	squashedImagePath, err := SquashLayers(images, conversionStore, appName, outputDir)
	if err != nil {
		return nil, fmt.Errorf("error squashing image: %v", err)
	}
	aciLayerPaths = []string{squashedImagePath}

	return aciLayerPaths, nil
}

// SquashLayers receives a list of ACI layer file names ordered from base image
// to application image and squashes them into one ACI
func SquashLayers(images []acirenderer.Image, aciRegistry acirenderer.ACIRegistry, appName string, outputDir string) (path string, err error) {
	renderedACI, err := acirenderer.GetRenderedACIFromList(images, aciRegistry)
	if err != nil {
		return "", fmt.Errorf("error rendering squashed image: %v", err)
	}
	manifests, err := getManifests(renderedACI, aciRegistry)
	if err != nil {
		return "", fmt.Errorf("error getting manifests: %v", err)
	}

	squashedFilename := getSquashedFilename(appName)
	squashedImagePath := filepath.Join(outputDir, squashedFilename)

	squashedTempFile, err := ioutil.TempFile(outputDir, "docker2aci-squashedFile-")
	if err != nil {
		return "", err
	}
	defer func() {
		if err == nil {
			err = squashedTempFile.Close()
		} else {
			// remove temp file on error
			// we ignore its error to not mask the real error
			os.Remove(squashedTempFile.Name())
		}
	}()

	if err := writeSquashedImage(squashedTempFile, renderedACI, aciRegistry, manifests); err != nil {
		return "", fmt.Errorf("error writing squashed image: %v", err)
	}

	if err := parse.ValidateACI(squashedTempFile.Name()); err != nil {
		return "", fmt.Errorf("error validating image: %v", err)
	}

	if err := os.Rename(squashedTempFile.Name(), squashedImagePath); err != nil {
		return "", err
	}

	return squashedImagePath, nil
}

func getSquashedFilename(appName string) string {
	squashedFilename := strings.Replace(appName, "/", "-", -1)
	squashedFilename += ".aci"

	return squashedFilename
}

func getManifests(renderedACI acirenderer.RenderedACI, aciRegistry acirenderer.ACIRegistry) ([]schema.ImageManifest, error) {
	var manifests []schema.ImageManifest

	for _, aci := range renderedACI {
		im, err := aciRegistry.GetImageManifest(aci.Key)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, *im)
	}

	return manifests, nil
}

func writeSquashedImage(outputFile *os.File, renderedACI acirenderer.RenderedACI, aciProvider acirenderer.ACIProvider, manifests []schema.ImageManifest) error {
	var tarWriterTarget io.WriteCloser = outputFile

	outputWriter := tar.NewWriter(tarWriterTarget)
	defer outputWriter.Close()

	finalManifest := mergeManifests(manifests)

	if err := parse.WriteManifest(outputWriter, finalManifest); err != nil {
		return err
	}

	if err := parse.WriteRootfsDir(outputWriter); err != nil {
		return err
	}

	type hardLinkEntry struct {
		firstLinkCleanName string
		firstLinkHeader    tar.Header
		keepOriginal       bool
		walked             bool
	}
	// map aciFileKey -> cleanTarget -> hardLinkEntry
	hardLinks := make(map[string]map[string]hardLinkEntry)

	// first pass: read all the entries and build the hardLinks map in memory
	// but don't write on disk
	for _, aciFile := range renderedACI {
		rs, err := aciProvider.ReadStream(aciFile.Key)
		if err != nil {
			return err
		}
		defer rs.Close()

		hardLinks[aciFile.Key] = map[string]hardLinkEntry{}

		squashWalker := func(t *utils.TarFile) error {
			cleanName := filepath.Clean(t.Name())
			// the rootfs and the squashed manifest are added separately
			if cleanName == "manifest" || cleanName == "rootfs" {
				return nil
			}
			_, keep := aciFile.FileMap[cleanName]
			if keep && t.Header.Typeflag == tar.TypeLink {
				cleanTarget := filepath.Clean(t.Linkname())
				if _, ok := hardLinks[aciFile.Key][cleanTarget]; !ok {
					_, keepOriginal := aciFile.FileMap[cleanTarget]
					hardLinks[aciFile.Key][cleanTarget] = hardLinkEntry{cleanName, *t.Header, keepOriginal, false}
				}
			}
			return nil
		}

		tr := tar.NewReader(rs)
		if err := utils.Walk(*tr, squashWalker); err != nil {
			return err
		}
	}

	// second pass: write on disk
	for _, aciFile := range renderedACI {
		rs, err := aciProvider.ReadStream(aciFile.Key)
		if err != nil {
			return err
		}
		defer rs.Close()

		squashWalker := func(t *utils.TarFile) error {
			cleanName := filepath.Clean(t.Name())
			// the rootfs and the squashed manifest are added separately
			if cleanName == "manifest" || cleanName == "rootfs" {
				return nil
			}
			_, keep := aciFile.FileMap[cleanName]

			if link, ok := hardLinks[aciFile.Key][cleanName]; ok {
				if keep != link.keepOriginal {
					return fmt.Errorf("logic error: should we keep file %q?", cleanName)
				}
				if keep {
					if err := outputWriter.WriteHeader(t.Header); err != nil {
						return fmt.Errorf("error writing header: %v", err)
					}
					if _, err := io.Copy(outputWriter, t.TarStream); err != nil {
						return fmt.Errorf("error copying file into the tar out: %v", err)
					}
				} else {
					// The current file does not remain but there is a hard link pointing to
					// it. Write the current file but with the filename of the first hard link
					// pointing to it. That first hard link will not be written later, see
					// variable "alreadyWritten".
					link.firstLinkHeader.Size = t.Header.Size
					link.firstLinkHeader.Typeflag = t.Header.Typeflag
					link.firstLinkHeader.Linkname = ""

					if err := outputWriter.WriteHeader(&link.firstLinkHeader); err != nil {
						return fmt.Errorf("error writing header: %v", err)
					}
					if _, err := io.Copy(outputWriter, t.TarStream); err != nil {
						return fmt.Errorf("error copying file into the tar out: %v", err)
					}
				}
			} else if keep {
				alreadyWritten := false
				if t.Header.Typeflag == tar.TypeLink {
					cleanTarget := filepath.Clean(t.Linkname())
					if link, ok := hardLinks[aciFile.Key][cleanTarget]; ok {
						if !link.keepOriginal {
							if link.walked {
								t.Header.Linkname = link.firstLinkCleanName
							} else {
								alreadyWritten = true
							}
						}
						link.walked = true
						hardLinks[aciFile.Key][cleanTarget] = link
					}
				}

				if !alreadyWritten {
					if err := outputWriter.WriteHeader(t.Header); err != nil {
						return fmt.Errorf("error writing header: %v", err)
					}
					if _, err := io.Copy(outputWriter, t.TarStream); err != nil {
						return fmt.Errorf("error copying file into the tar out: %v", err)
					}
				}
			}
			return nil
		}

		tr := tar.NewReader(rs)
		if err := utils.Walk(*tr, squashWalker); err != nil {
			return err
		}
	}

	return nil
}

func mergeManifests(manifests []schema.ImageManifest) schema.ImageManifest {
	// FIXME(iaguis) we take app layer's manifest as the final manifest for now
	manifest := manifests[0]

	manifest.Dependencies = nil

	layerIndex := -1
	for i, l := range manifest.Labels {
		if l.Name.String() == "layer" {
			layerIndex = i
		}
	}

	if layerIndex != -1 {
		manifest.Labels = append(manifest.Labels[:layerIndex], manifest.Labels[layerIndex+1:]...)
	}

	nameWithoutLayerID := types.MustACIdentifier(stripLayerID(manifest.Name.String()))

	manifest.Name = *nameWithoutLayerID

	// once the image is squashed, we don't need a pathWhitelist
	manifest.PathWhitelist = nil

	return manifest
}

// striplayerID strips the layer ID from an app name:
//
// myregistry.com/organization/app-name-85738f8f9a7f1b04b5329c590ebcb9e425925c6d0984089c43a022de4f19c281
// myregistry.com/organization/app-name
func stripLayerID(layerName string) string {
	n := strings.LastIndex(layerName, "-")
	return layerName[:n]
}

package convert

import (
	"encoding/json"
	"strings"
)

type ManifestDesc struct {
	Name          string
	Repository    string
	Tag           string
	Architecture  string
	SchemaVersion float64
	ImgId         []string
	ImgTarsum     []string
	ImgJSON       []string
	//Signatures    []interface{}
}

/*
type Signature struct {
	crv       string
	kid       string
	kty       string
	x         string
	y         []string
	alg       []string
	signature string
	protected string
}
*/

func (m *ManifestDesc) JSON2manifest(data []byte) error {

	return nil
}

func (m *ManifestDesc) Manifest2JSON(data []byte) error {
	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}

	m.Name = strings.Split(manifest["name"].(string), "/")[0]
	m.Repository = strings.Split(manifest["name"].(string), "/")[1]
	m.Tag = manifest["tag"].(string)
	m.Architecture = manifest["architecture"].(string)
	m.SchemaVersion = manifest["schemaVersion"].(float64)
	//m.Signatures = manifest["signatures"].([]interface{})

	for k := len(manifest["history"].([]interface{})) - 1; k >= 0; k-- {
		v := manifest["history"].([]interface{})[k]

		imgJSON := v.(map[string]interface{})["v1Compatibility"].(string)
		m.ImgJSON = append(m.ImgJSON, imgJSON)

		var img map[string]interface{}
		if err := json.Unmarshal([]byte(imgJSON), &img); err != nil {
			return err
		}
		m.ImgId = append(m.ImgId, img["id"].(string))

		blobSum := manifest["fsLayers"].([]interface{})[k].(map[string]interface{})["blobSum"].(string)
		tarsum := strings.Split(blobSum, ":")[1]
		m.ImgTarsum = append(m.ImgTarsum, tarsum)
	}

	return nil
}

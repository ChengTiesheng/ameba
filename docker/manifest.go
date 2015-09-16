package docker

import (
	"encoding/json"
)

type Manifest struct {
	Name          string            `json:"name"`
	Tag           string            `json:"tag"`
	Architecture  string            `json:"architecture"`
	FSLayers      []BlobSum         `json:"fslayers"`
	History       []V1Compatibility `json:"history"`
	V1JSON        []V1JSON          ``
	SchemaVersion int64             `json:"schemaVersion"`
	Signatures    []Signatures      `json:"signatures"`
}

type BlobSum struct {
	BlobSum string `json:"blobsum"`
}

type V1Compatibility struct {
	Compatibility string `json:"v1Compatibility"`
}

type V1JSON struct {
	Id              string `json:"id"`
	Parent          string `json:"parent"`
	Os              string `json:"os"`
	Version         string `json:"docker_version"`
	Author          string `json:"author"`
	Container       string `json:"container"`
	ContainerConfig Config `json:"container_config"`
	Config          Config `json:"config"`
	Architecture    string `json:"architecture"`
	Size            int64  `json:"size"`
	Created         string `json:"created"`
}

type Config struct {
	Hostname        string                 `json:"Hostname"`
	Domainname      string                 `json:"Domainname"`
	User            string                 `json:"User"`
	Memory          int64                  `json:"Memory"`
	MemorySwap      int64                  `json:"MemorySwap"`
	CpuShares       int64                  `json:"CpuShares"`
	CpuSet          string                 `json:"CpuSet"`
	AttachStdin     bool                   `json:"AttachStdin"`
	AttachStdout    bool                   `json:"AttachStdout"`
	AttachStderr    bool                   `json"AttachStderr"`
	PortSpecs       map[string]interface{} `json:"PortSpecs"`
	ExposedPorts    map[string]interface{} `json:"ExposedPorts"`
	Tty             bool                   `json:"Tty"`
	OpenStdin       bool                   `json:"OpenStdin"`
	StdinOnce       bool                   `json:"StdinOnce"`
	Env             map[string]interface{} `json:"Env"`
	CMD             []string               `json:"Cmd"`
	Image           string                 `json:"Image"`
	Volumes         map[string]interface{} `json:"Volumes"`
	WorkingDir      string                 `json:"WorkingDir"`
	Entrypoint      map[string]interface{} `json:"Entrypoint"`
	NetworkDisabled bool                   `json:"NetworkDisabled"`
	MacAddress      string                 `json:"MacAddress"`
	OnBuild         map[string]interface{} `json:"OnBuild"`
	Labels          map[string]interface{} `json:"Labels"`
}

type Signatures struct {
	Header    JWKHeader `json:"header"`
	Signature string    `json:"signature"`
	Protected string    `json:"protected"`
}

type JWKHeader struct {
	JWK JWK    `json:"jwk"`
	Alg string `json:"alg"`
}

type JWK struct {
	Crv string `json:"crv"`
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func (m *Manifest) Decode(data string) error {
	if err := json.Unmarshal([]byte(data), m); err != nil {
		return err
	} else {
		for _, c := range m.History {
			v1 := new(V1JSON)

			if err := json.Unmarshal([]byte(c.Compatibility), v1); err != nil {
				return err
			} else {
				m.V1JSON = append(m.V1JSON, *v1)
			}
		}
	}

	return nil
}

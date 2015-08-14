package docker

type Manifest struct {
	Name          string            `json:"name"`
	Tag           string            `json:"tag"`
	Architecture  string            `json:"architecture"`
	FSLayers      []BlobSum         `json:"fslayers"`
	History       []V1Compatibility `json:"history"`
	SchemaVersion string            `json:"schemaVersion"`
	Signatures    []Signatures      `json:"signatures"`
}

type BlobSum struct {
	BlobSum string `json:"blobsum"`
}

type V1Compatibility struct {
	Id           string `json:"id"`
	Parent       string `json:"parent"`
	Os           string `json:"os"`
	Version      string `json:"docker_version"`
	Container    string `json:"container"`
	Architecture string `json:"architecture"`
	Size         string `json:"size"`
	Created      string `json:"created"`
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

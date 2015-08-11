package main

import (
	"fmt"
	"io/ioutil"

	"github.com/containerops/ameba/convert"
)

//only for test
func main() {
	var data []byte
	var err error

	if data, err = ioutil.ReadFile("manifest_test.json"); err != nil {
		fmt.Println("Read file failed:", err.Error())
		return
	}

	m := new(convert.ManifestDesc)
	if err := m.Manifest2JSON(data); err != nil {
		fmt.Println("Convert failed:", err.Error())
		return
	}
}

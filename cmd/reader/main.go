package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/arisu-archive/assets-dumper/pkg/resourceapi"
	"github.com/arisu-archive/assets-dumper/pkg/resources"
	"github.com/arisu-archive/bluearchive-data-sync/pkg/javax"
)

func main() {
	valid, err := os.Open("./tmp/valid/patch.file.hash")
	if err != nil {
		panic(err)
	}
	defer valid.Close()

	dec, err := javax.NewDecoder(valid)
	if err != nil {
		panic(err)
	}

	rs, err := dec.ReadObject()
	if err != nil {
		panic(err)
	}

	x := rs.(*javax.HashMap)
	validHashMap := make(map[string]string)
	for key, value := range x.Data {
		validHashMap[key.(string)] = value.(string)
	}

	invalid, err := os.Open("./tmp/outdated/patch.file.hash")
	if err != nil {
		panic(err)
	}
	defer invalid.Close()

	dec, err = javax.NewDecoder(invalid)
	if err != nil {
		panic(err)
	}

	rs, err = dec.ReadObject()
	if err != nil {
		panic(err)
	}

	x = rs.(*javax.HashMap)
	invalidHashMap := make(map[string]string)
	for key, value := range x.Data {
		invalidHashMap[key.(string)] = value.(string)
	}

	compare(validHashMap, invalidHashMap)

	// Now, patch the outdated hashmap with the valid hashmap
	rc, _ := resources.NewClient(resourceapi.ServerGlobal)

	res, _ := rc.ListResources(context.Background(), "Preload/**")
	for _, r := range res {
		fmt.Println(r.Path)
		invalidHashMap[r.Path] = r.Hash
	}

	compare(validHashMap, invalidHashMap)

	out := bytes.NewBuffer(nil)
	enc, _ := javax.NewEncoder(out)

	h := javax.NewHashMap(make(map[any]any))
	for key, value := range invalidHashMap {
		h.Set(key, value)
	}
	enc.WriteObject(h)
	os.MkdirAll("./tmp/patched", 0o755)
	os.WriteFile("./tmp/patched/patch.file.hash", out.Bytes(), 0o644)

	patched, err := os.Open("./tmp/patched/patch.file.hash")
	if err != nil {
		panic(err)
	}
	defer patched.Close()

	dec, err = javax.NewDecoder(patched)
	if err != nil {
		panic(err)
	}

	rs, err = dec.ReadObject()
	if err != nil {
		panic(err)
	}

	x = rs.(*javax.HashMap)
	patchedHashMap := make(map[string]string)
	for key, value := range x.Data {
		patchedHashMap[key.(string)] = value.(string)
	}

	compare(validHashMap, patchedHashMap)
}

func compare(validHashMap, invalidHashMap map[string]string) {
	// Compare the two maps: 1. Missing in invalid 2. Different in invalid
	missing := make(map[string]string)
	different := make(map[string]string)
	for key, value := range validHashMap {
		if !strings.Contains(key, "Preload/") {
			continue
		}
		if _, ok := invalidHashMap[key]; !ok {
			missing[key] = value
		} else if invalidHashMap[key] != value {
			different[key] = value
		}
	}

	// Print the missing and different keys
	fmt.Println("Missing:")
	for key := range missing {
		fmt.Println(key)
	}
	fmt.Println("Different:")
	for key := range different {
		fmt.Println(key, different[key])
	}
}

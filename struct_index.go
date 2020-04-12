package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

type Index struct {
	Products map[string]*Product
}

func NewIndex() Index {
	resp, err := HTTPGet(ReleasesURL + "/index.json")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	etag := resp.Header.Get("Etag")
	etag = strings.Trim(etag, "\"")
	if etag == "" {
		panic("no etag found")
	}

	// TODO: cache expiration or purge
	cacheFilePath := path.Join(tmpDir, etag, etag+indexSuffix)

	b, err := ioutil.ReadFile(cacheFilePath)
	if err != nil {
		b, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		if err = os.MkdirAll(path.Dir(cacheFilePath), 0700); err != nil {
			panic(err)
		}
		if err = ioutil.WriteFile(cacheFilePath, b, 0600); err != nil {
			panic(err)
		}
	}

	// this intermediary `products` var is so the Index only gets core Products unless -all
	var products map[string]*Product
	if err = json.Unmarshal(b, &products); err != nil {
		panic(err)
	}

	opts := GetOptions()
	index := Index{
		Products: make(map[string]*Product),
	}
	for n, p := range products {
		if !opts.all && !InArray(CoreProducts, n) {
			continue
		}
		index.Products[n] = p
	}

	for _, v := range index.Products {
		if err = v.sortVersions(); err != nil {
			panic(err)
		}
	}
	return index
}

func (i *Index) GetProductVersion(name string, version string) (*Product, *Version, error) {
	p, err := i.GetProduct(name)
	if err != nil {
		return nil, nil, err
	}
	v, err := p.GetVersion(version)
	if err != nil {
		return p, nil, err
	}
	return p, v, nil
}

func (i *Index) GetProduct(name string) (*Product, error) {
	product, ok := i.Products[name]
	if !ok {
		return nil, errors.New("invalid product name")
	}
	return product, nil
}

func (i *Index) LatestVersion(product string) string { // TODO: remove me? Product.LatestVersion()
	p, ok := i.Products[product]
	if !ok {
		return ""
	}
	return p.Sorted[len(p.Sorted)-1].Original()
}

// func (i *Index) LatestBuild(product, os, arch string) *Build {
// 	p, ok := i.Products[product]
// 	if !ok {
// 		return nil
// 	}
// 	v, ok := p.Versions[p.Sorted[len(p.Sorted)-1].Original()]
// 	if !ok {
// 		return nil
// 	}
// 	return v.GetBuildForLocal()
// }

func (i *Index) ListVersions(product string) []string {
	p, ok := i.Products[product]
	if !ok {
		return nil
	}
	versions := make([]string, len(p.Sorted))
	for idx, v := range p.Sorted {
		versions[idx] = v.Original()
	}
	return versions
}

func (i *Index) ListProducts() []string {
	products := make([]string, len(i.Products))
	var idx int
	for k, _ := range i.Products {
		products[idx] = k
		idx++
	}
	return products
}

package metadata

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
)

func bindata_read(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	return buf.Bytes(), nil
}

var _db_sqlite_migrations_metadata_1637447083_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xa4\x56\x4d\x73\xdb\x36\x13\x3e\xbf\xfe\x15\x3b\xb9\x44\x9a\x57\xd4\xb9\xd3\x4c\x0f\x6e\xdc\xb8\x9e\x89\x9d\x8c\x62\x35\x39\x1a\x24\x96\x24\x2a\x10\x8b\x2e\x40\xc9\xcc\xaf\xef\x2c\x40\xaa\xb2\x65\xc9\xe9\xf4\x24\x0a\xd8\x7d\x76\xf7\xd9\x2f\x14\x05\xfc\xbf\x33\x0d\xab\x88\xb0\xf6\x17\x15\xa3\x7c\x45\x55\x5a\x84\x16\x95\x46\x0e\x30\xbb\x00\x00\x28\x0a\x58\x61\x45\xac\x81\x6a\x88\xad\x09\xe3\x3d\x90\x83\xd8\x8a\x8e\xc7\x24\xc8\x59\xca\xb8\x88\x0d\x32\x38\x8a\xe0\x7a\x6b\x17\x2f\xa1\x20\x58\x15\x22\xf4\x5e\x8b\xd9\x09\xf0\x3c\xbe\x68\x6c\x1c\xed\xdc\x6b\x86\x7e\xb5\x54\x6d\x9e\xa3\x99\x8c\x96\x75\x93\x64\x99\xc4\x7e\x04\xe5\x47\xbc\x3d\xc6\xdf\xfb\xfb\x8a\xa1\x9b\x1a\x02\xc6\x45\x52\x1f\xc1\x5a\x15\xa0\x44\x74\xa0\xd1\x62\x44\x9d\xb8\x50\x1e\x17\x50\xf6\x11\x1e\x9e\x31\xf1\x00\xca\xe9\x83\xd3\x64\xef\x01\x14\x23\x84\x68\xac\x15\x57\x19\x2d\x6e\x95\xab\x32\x95\x13\xec\x49\x9f\xee\x07\x8f\xb5\x55\x0d\x98\x90\x93\x30\x78\x14\x98\xd1\x3f\x74\x91\x87\xe5\x5e\xb8\x45\xf8\x8e\x4c\xb0\x55\xb6\x47\x51\x51\x7d\xa4\x4e\x45\x53\x29\x6b\x07\xf0\x4c\x1d\x89\xb9\x48\x80\x26\xb6\xc8\x09\x7f\x85\x0d\x50\xfe\xbc\x32\x3c\x81\x69\xf4\xe8\xb4\x71\xcd\x94\x7f\xcf\x18\xd0\x55\xc9\xbc\x82\xc8\xca\x58\xb9\x0d\x56\x85\x56\x58\xbf\x53\x1d\x66\x57\xe2\xde\xe9\x53\x61\x89\xac\xe0\xd4\xc6\x62\x0e\x22\xdd\x38\x39\x8e\xf8\x18\x5f\x20\x42\x71\x83\x31\x4b\x50\x0d\xd6\xb8\x0d\xcc\xb6\xca\x1a\x0d\xf5\xe8\xfc\x47\x39\x1b\xbf\xbf\x0c\x9d\x88\xcc\x73\x01\x18\xb7\x39\x03\xfd\x91\x1a\x21\x28\x3b\x13\xcc\x77\x94\x68\xca\x21\x62\x48\x12\xe3\xc9\x89\x48\x3e\x23\x77\x26\x04\x43\x2e\x25\xbf\x23\x8d\x50\x9a\x98\x55\xd3\xbf\x93\xaa\xeb\x80\x0c\x37\x57\x12\x0e\xed\x1c\x66\xe6\x7b\x73\xa6\x1a\xae\x99\x7a\x7f\xa4\xd2\x9c\x53\x49\x46\x26\xd6\x0e\xcc\x9c\xe1\x23\x5b\x39\xd6\x69\xce\xe8\xdc\xe4\xf6\xfc\x40\xdc\xa9\x28\x95\xd7\xbb\xe0\xb1\x32\xb5\x41\x9d\x3a\xca\xc1\x57\x36\x11\x79\x99\x7e\x7e\xcf\xe5\xcb\xd4\x3b\x1d\xe0\x96\xf4\xbd\xe9\x70\xc2\x8a\x94\xb0\x1c\x2a\xc6\x10\x21\x60\x45\x4e\x27\x76\x4d\xe3\x88\x31\x77\xc2\x65\x55\x61\x08\xa2\x97\xae\xde\xb7\xca\x35\x98\xfe\xd6\x06\xad\x0e\x53\x57\xec\x0b\x88\xa0\x0f\x4f\xd4\x88\x0f\xb4\x16\x90\xfd\x1d\x0e\xe3\x50\x01\x3e\x5f\x7e\x13\xc1\xeb\xbb\xf5\xf2\x19\x52\xe8\xcb\x62\xf4\x8d\x31\x90\xed\xa3\x21\x77\x06\x66\xaf\x7f\x4b\xda\xd4\xa6\x52\x22\x0f\x71\x0a\xbc\x23\x2d\xdf\x90\x46\xdb\x11\xbf\xd9\xed\x24\x0d\x33\xc6\xbf\x7a\x23\x3c\x8c\x4d\xfc\x8f\x8f\x10\x7a\xef\x89\x63\xae\x7b\x95\x94\xce\xa0\xe6\xf0\xff\x25\x6a\x95\x94\xce\xa0\xde\xaa\x3f\x89\x41\xe3\xd6\x54\x08\xae\xef\x4a\xe4\xe7\xad\xfa\xbe\x55\x3c\xb5\x6a\x9a\xef\xf3\x71\x1c\x6e\xbb\xa4\x7c\xb2\x9c\x6f\x8d\xfb\x0f\xd8\x49\xf9\x74\x2b\x5f\x7e\xcb\xab\x31\xa4\xd1\x09\x9d\xf2\xd2\x00\xc2\x03\x3e\x46\x74\x1a\xf5\x34\x77\xf3\xbc\x3f\xaa\x31\x69\xb6\x42\x63\x6d\x1c\xea\x49\x06\x42\x4b\xbd\xd5\xd0\xaa\x2d\xc2\x06\x87\x30\xad\xb2\x9a\xac\xa5\x9d\x8c\xd0\x9a\xb8\xfb\x79\xc4\xf8\xdf\x1f\xbf\xdd\x5d\x7d\x5a\x2d\x37\x38\xec\xa6\x05\x56\x14\xf0\xb5\x45\x46\xc8\x77\xe2\x5d\xa0\x0e\x53\x8b\x06\xaf\xaa\x34\xb0\x94\xb5\xd0\x7b\x8f\x5c\xa9\x80\x8b\xd4\x13\x23\x06\x74\x6a\x98\x70\x24\xe8\x8a\x5c\x54\xe3\x9a\x7c\xfb\xcb\x5b\x49\x28\xab\x2a\x0a\x93\xb8\x6c\x96\x0b\x78\x73\xfd\xe9\xe3\xe5\xdd\xf5\xd2\x6f\x9a\xe5\x16\x59\x86\xdb\x9b\xf9\x93\x2d\xb3\xc1\x21\x59\xc8\x8b\x66\x8c\xb0\x94\x5a\x70\x05\x76\x3e\x0e\xb0\xbe\xff\x50\xfc\x04\x21\xb2\x71\xcd\x11\x4f\x5f\x4f\xcc\x03\x13\x40\x36\x95\x8c\x0c\x61\x7d\x62\x50\x23\x9b\x2d\x6a\xa8\x99\x3a\xf1\x7a\x82\xa1\x54\xa7\xb9\xdd\x85\x82\x11\x25\xaa\x4d\xda\x56\x15\xea\xbc\xaf\xb6\xb9\x9a\x57\x87\x49\xf3\xea\x71\x82\x7f\x79\xa2\x8d\xbd\x3b\x4d\xb1\x30\xe6\x2c\x1d\x8e\x19\x8c\x8a\xc7\x82\x38\x1a\x36\xf2\x20\x91\x34\x61\x84\x72\x80\x55\x16\xba\x13\x43\x4a\x4a\xab\xc4\x10\x0b\xac\x6b\xe2\x08\x4d\x2f\x9d\xad\xe2\x81\x81\x3d\xd7\x5f\x8c\x04\x20\x17\x19\x02\xac\x29\x91\xd3\x32\x67\x54\x7a\xaa\x03\x72\x45\x45\x9d\xb7\x46\xb9\x98\xd6\x58\xd8\x87\x61\xd2\x34\xf6\x14\x82\x91\x27\xa5\x34\x49\x7a\x2d\x45\x92\x7c\xe5\x20\xd7\x2e\x3d\x57\x9e\x07\x71\x53\x1f\xc6\xfc\x74\xa6\xc3\xee\xd5\x14\xee\xc7\xb9\x48\x9a\x28\x33\x73\x24\xd1\x70\x88\x13\xec\x6c\x2c\x44\xe2\xf1\x2d\xb7\xfe\x72\x7f\xb9\x4a\xf9\x5f\xc8\xec\x99\x4f\x30\x95\xf2\xe9\x51\x4c\x35\xa0\xab\x28\xbd\x4c\x52\x24\xa3\xe1\x59\xc0\x29\x9e\xb1\x56\x27\xc7\x8f\xfa\x3d\x17\x00\x9b\x4e\xf1\x90\x4a\x79\x26\x9d\xb4\xd8\xbf\x13\xe6\x17\xf3\x77\x17\x87\x0f\xf3\x2b\xda\xb9\x0b\xcd\xe4\x9f\x3e\xcc\xdf\xfd\x1d\x00\x00\xff\xff\x7e\x7f\x3e\xa6\xbd\x0b\x00\x00")

func db_sqlite_migrations_metadata_1637447083_sql() ([]byte, error) {
	return bindata_read(
		_db_sqlite_migrations_metadata_1637447083_sql,
		"../../db/sqlite/migrations/metadata/1637447083.sql",
	)
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		return f()
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() ([]byte, error){
	"../../db/sqlite/migrations/metadata/1637447083.sql": db_sqlite_migrations_metadata_1637447083_sql,
}
// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for name := range node.Children {
		rv = append(rv, name)
	}
	return rv, nil
}

type _bintree_t struct {
	Func func() ([]byte, error)
	Children map[string]*_bintree_t
}
var _bintree = &_bintree_t{nil, map[string]*_bintree_t{
	"..": &_bintree_t{nil, map[string]*_bintree_t{
		"..": &_bintree_t{nil, map[string]*_bintree_t{
			"db": &_bintree_t{nil, map[string]*_bintree_t{
				"sqlite": &_bintree_t{nil, map[string]*_bintree_t{
					"migrations": &_bintree_t{nil, map[string]*_bintree_t{
						"metadata": &_bintree_t{nil, map[string]*_bintree_t{
							"1637447083.sql": &_bintree_t{db_sqlite_migrations_metadata_1637447083_sql, map[string]*_bintree_t{
							}},
						}},
					}},
				}},
			}},
		}},
	}},
}}

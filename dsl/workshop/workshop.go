package workshop

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/dsl/repo"
	"github.com/yaoapp/gou/dsl/utils"
)

// Tidy scan the source and update workshop.yao then auto-generation the workshop.sum.yao file
func Tidy(root string) error { return nil }

// Format scan the source and format DSL code
func Format(root string) error { return nil }

// OpenWorkshop open and parse the workshop dsl
func OpenWorkshop(root string) (*Workshop, error) {

	cfg, err := GetConfig()
	if err != nil {
		return nil, err
	}

	file := path.Join(root, "workshop.yao")
	exists, err := utils.FileExists(file)
	if err != nil {
		return nil, err
	}

	if !exists {
		return &Workshop{
			file:    file,
			Require: []*Package{},
			Replace: map[string]string{},
			Mapping: map[string]*Package{},
		}, nil
	}

	data, err := utils.FileGetJSON(file)
	if err != nil {
		return nil, err
	}

	workshop := &Workshop{Mapping: map[string]*Package{}}
	err = jsoniter.Unmarshal(data, workshop)
	if err != nil {
		return nil, err
	}

	workshop.file = file
	workshop.cfg = cfg
	err = workshop.SetMapping()
	if err != nil {
		return nil, err
	}

	return workshop, nil
}

// Get the url from the given remote repo
// url:
//   github.com/yaoapp/demo-crm
//   github.com/yaoapp/demo-crm@v0.9.1
//   github.com/yaoapp/demo-crm@e86eab4c8490
//   github.com/yaoapp/demo-wms/cloud@e86eab4c8490
//   github.com/yaoapp/demo-wms/edge@e86eab4c8490
func (workshop *Workshop) Get(url, alias string, process func(total uint64, pkg *Package, message string)) error {

	// Lock the file
	err := workshop.lock()
	if err != nil {
		return err
	}
	defer workshop.unlock()

	// Create a new package
	pkg, err := workshop.Package(url, alias)
	if err != nil {
		return err
	}

	// If has checkout the package, return
	if _, has := workshop.Mapping[pkg.Unique]; has {
		return nil
	}

	// Add the package
	err = workshop.Add(pkg, process, false)
	if err != nil {
		return err
	}

	// Save to file
	return workshop.Save()
}

// SetMapping mapping alias and package
func (workshop *Workshop) SetMapping() error {

	for i, pkg := range workshop.Require {

		// Check name
		if _, has := workshop.Mapping[pkg.Alias]; has {
			return fmt.Errorf(
				"\"%s\" and \"%s\" has the same name \"%s\", please change it",
				workshop.Mapping[pkg.Alias].URL, pkg.URL, pkg.Alias,
			)
		}

		// Replace
		pkgpath := filepath.Join(pkg.Addr, pkg.Path)
		if path, has := workshop.Replace[pkgpath]; has {

			localpath := path
			if !filepath.IsAbs(path) {
				absPath, err := filepath.Abs(filepath.Join(filepath.Dir(workshop.file), path))
				if err != nil {
					return err
				}
				localpath = absPath
			}

			if _, err := os.Stat(localpath); err != nil {
				return err
			}

			if _, err := os.Stat(filepath.Join(localpath, "app.yao")); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("%s is not YAO application", localpath)
				}
				return err
			}
			workshop.Require[i].Replaced = true
			workshop.Require[i].LocalPath = localpath
		}
		workshop.Mapping[pkg.Alias] = workshop.Require[i]
		workshop.Mapping[pkg.Unique] = workshop.Require[i]
	}
	return nil
}

// Map workshop to map[string]interface{}
func (workshop Workshop) Map() map[string]interface{} {
	require := []map[string]interface{}{}
	for _, pkg := range workshop.Require {
		require = append(require, pkg.Map())
	}
	return map[string]interface{}{
		"file":    workshop.file,
		"mapping": workshop.Mapping,
		"replace": workshop.Replace,
		"require": require,
	}
}

// Has the package
func (workshop *Workshop) Has(name string) bool {
	_, ok := workshop.Mapping[name]
	return ok
}

// Add add a package to workshop.ayo
func (workshop *Workshop) Add(pkg *Package, process func(total uint64, pkg *Package, message string), indirect bool) error {

	// Download the package
	_, err := workshop.Download(pkg, process)
	if err != nil {
		return err
	}

	pkg.Indirect = indirect
	workshop.Require = append(workshop.Require, pkg)
	workshop.Mapping[pkg.Alias] = pkg
	workshop.Mapping[pkg.Unique] = pkg

	// add Dependencies
	deps, err := pkg.Dependencies()
	if err != nil {
		return err
	}

	for _, dep := range deps {

		if workshop.Has(dep.Unique) {
			continue
		}

		err := workshop.Add(dep, process, true)
		if err != nil {
			return err
		}
	}

	return nil
}

// Del delete a repo from workshop.yao
func (workshop *Workshop) Del(repo string) error {
	return nil
}

// Download and unzip a package
func (workshop *Workshop) Download(pkg *Package, process func(total uint64, pkg *Package, message string)) (string, error) {

	root, err := Root()
	if err != nil {
		return "", err
	}

	option := pkg.Option(workshop.cfg)
	option["cache"] = filepath.Join(root, "cache")

	// Download package
	dest, err := pkg.Download(root, option, process)
	if err != nil {
		return "", err
	}

	return dest, nil
}

// Package create a new package
func (workshop *Workshop) Package(url, alias string) (*Package, error) {
	pkg := &Package{}
	if !strings.Contains(url, "@") {

		uri := strings.Split(url, "/")
		if len(uri) < 3 {
			return nil, fmt.Errorf("package url should be a git repo. \"domain/org/repo/path\", but got: %s", url)
		}
		option := map[string]interface{}{}
		if opt, has := workshop.cfg[uri[0]]; has {
			option = opt
		}

		// Get the latest version
		repo, err := repo.NewRepo(url, option)
		if err != nil {
			return nil, err
		}

		rel, err := repo.Latest()
		if err != nil {
			return nil, err
		}

		url = fmt.Sprintf("%s@%s", url, rel)
	}

	err := pkg.Set(url, alias)
	if err != nil {
		return nil, err
	}
	return pkg, nil
}

// Save save the workshop to the file
func (workshop *Workshop) Save() error {

	content, err := workshop.Bytes()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(workshop.file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	err = f.Truncate(0)
	if err != nil {
		return err
	}

	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, "%s", content)
	if err != nil {
		return err
	}
	return nil
}

// Bytes format and the workshop content
func (workshop *Workshop) Bytes() ([]byte, error) {

	packages := []*Package{}
	indirects := []*Package{}

	for _, pkg := range workshop.Require {
		if pkg.Indirect {
			indirects = append(indirects, pkg)
			continue
		}
		packages = append(packages, pkg)
	}

	for _, pkg := range indirects {
		packages = append(packages, pkg)
	}

	require, err := jsoniter.MarshalIndent(map[string]interface{}{"require": packages}, "", "  ")
	if err != nil {
		return nil, err
	}

	replace, err := jsoniter.MarshalIndent(map[string]interface{}{"replace": workshop.Replace}, "", "  ")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString("{")
	buf.Write(require[1 : len(require)-2])
	buf.WriteString(",")
	buf.Write(replace[1 : len(replace)-2])
	buf.WriteString("\n}")

	return buf.Bytes(), nil
}

// lock the workshop.yao file
func (workshop *Workshop) lock() error {
	lockfile := fmt.Sprintf("%s.lock", workshop.file)
	exists, err := utils.FileExists(lockfile)
	if exists {
		return fmt.Errorf("%s has been locked. Maybe another process running\n try: rm %s", workshop.file, lockfile)
	}

	if err != nil {
		return err
	}

	file, err := os.OpenFile(lockfile, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	return file.Close()
}

// unlock the workshop.yao file
func (workshop *Workshop) unlock() error {
	lockfile := fmt.Sprintf("%s.lock", workshop.file)
	return os.Remove(lockfile)
}

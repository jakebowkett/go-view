package view

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type View struct {
	mu        sync.RWMutex
	templates *template.Template
	funcMap   map[string]interface{}
	filePaths map[string]string
	onLoad    func(ext string, tmpl []byte) (newTmpl []byte)
}

func New(funcMap map[string]interface{}) *View {
	return &View{
		templates: template.New("").Funcs(funcMap),
		funcMap:   funcMap,
		filePaths: make(map[string]string),
	}
}

func (v *View) List() []string {
	var ss []string
	for s, _ := range v.filePaths {
		ss = append(ss, s)
	}
	return ss
}

func (v *View) OnLoad(callback func(ext string, tmpl []byte) (newTmpl []byte)) {
	v.onLoad = callback
}

func (v *View) MustAddDir(alias, dirPath string, exts []string, recursive bool) {
	if err := v.AddDir(alias, dirPath, exts, recursive); err != nil {
		panic(err)
	}
}

func (v *View) MustAddTemplate(alias, filePath string) {
	if err := v.AddTemplate(alias, filePath); err != nil {
		panic(err)
	}
}

func (v *View) AddDir(alias, dirPath string, exts []string, recursive bool) error {

	dirPath, err := filepath.Abs(dirPath)
	if err != nil {
		return err
	}

	dir, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, info := range dir {

		alias := alias + "/" + info.Name()
		alias = strings.Trim(alias, "/")
		dirPath := filepath.Join(dirPath, info.Name())

		if info.IsDir() && recursive {
			v.AddDir(alias, dirPath, exts, recursive)
		}

		if !info.Mode().IsRegular() {
			continue
		}

		ext := filepath.Ext(info.Name())
		if len(exts) > 0 && !in(exts, ext) {
			continue
		}

		err := v.AddTemplate(alias, dirPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func in(ss []string, s string) bool {
	for i := range ss {
		if ss[i] == s {
			return true
		}
	}
	return false
}

func (v *View) AddTemplate(alias, filePath string) error {

	filePath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	if !info.Mode().IsRegular() {
		return errors.New(fmt.Sprintf("%s is not a file", filePath))
	}

	f, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	if v.onLoad != nil {
		f = v.onLoad(filepath.Ext(filePath), f)
	}

	// Parse the currently named template.
	tmpl, err := v.templates.New(alias).Parse(string(f))
	if err != nil {
		return err
	}
	v.templates = tmpl
	v.filePaths[alias] = filePath

	return nil
}

func (v *View) Render(alias string, data interface{}) ([]byte, error) {

	// if filepath.Ext(alias) == "" {
	// 	return nil, errors.New(fmt.Sprintf("template alias %q has no file extension", alias))
	// }

	v.mu.RLock()
	defer v.mu.RUnlock()

	tmpl := v.templates.Lookup(alias)
	if tmpl == nil {
		return nil, errors.New(fmt.Sprintf("couldn't find template %q", alias))
	}

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (v *View) Refresh() (dropped []string) {

	// Ensure dropped is non-nil for
	// calls to v.delete
	dropped = []string{}

	v.mu.Lock()
	defer v.mu.Unlock()

	v.templates = template.New("").Funcs(v.funcMap)

	for alias, filePath := range v.filePaths {

		info, err := os.Stat(filePath)
		if err != nil {
			dropped = append(dropped, alias)
			continue
		}

		if !info.Mode().IsRegular() {
			dropped = append(dropped, alias)
			continue
		}

		f, err := ioutil.ReadFile(filePath)
		if err != nil {
			dropped = append(dropped, alias)
			continue
		}

		if v.onLoad != nil {
			f = v.onLoad(filepath.Ext(filePath), f)
		}

		_, err = v.templates.New(alias).Parse(string(f))
		if err != nil {
			dropped = append(dropped, alias)
		}

	}

	return dropped
}

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
)

type View struct {
	templates   *template.Template
	filePaths   map[string]string
	BeforeParse func(ext, tmpl string) (newTmpl string)
}

func New(funcMap map[string]interface{}) *View {
	return &View{
		templates: template.New("").Funcs(funcMap),
		filePaths: make(map[string]string),
	}
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

		if info.IsDir() {
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

	if v.BeforeParse != nil {
		v.BeforeParse(filepath.Ext(filePath), string(f))
	}

	// Parse the currently named template.
	tmpl, err := v.templates.New(alias).Parse(string(f))
	if err != nil {
		return err
	}
	v.templates = tmpl
	v.filePaths = append(v.filePaths, filePath)

	return nil
}

func (v *View) Render(alias string, data interface{}) ([]byte, error) {

	if filepath.Ext(alias) == "" {
		return nil, errors.New(fmt.Sprintf("template alias %q has no file extension"))
	}

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

	for alias, tmplName := range v.filePaths {

	}

}

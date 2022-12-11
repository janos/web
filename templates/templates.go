// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package templates

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"golang.org/x/exp/slog"
)

// Error is a common error type that holds
// information about error message and template name.
type Error struct {
	Err      error
	Template string
}

func (e *Error) Error() string {
	if e.Template == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %s", e.Err.Error(), e.Template)
}

// FileReadFunc returns the content of file referenced
// by filename. It hes the same signature as ioutil.ReadFile
// function.
type FileReadFunc func(filename string) ([]byte, error)

// ErrUnknownTemplate will be returned by Render function if
// the template does not exist.
var ErrUnknownTemplate = fmt.Errorf("unknown template")

// Options holds parameters for creating Templates.
type Options struct {
	fileFindFunc     func(filename string) string
	fileReadFunc     FileReadFunc
	fileReadOnRender bool
	contentType      string
	files            map[string][]string
	strings          map[string][]string
	functions        template.FuncMap
	delimOpen        string
	delimClose       string
	logger           *slog.Logger
}

// Option sets parameters used in New function.
type Option func(*Options)

// WithContentType sets the content type HTTP header that
// will be written on Render and Response functions.
func WithContentType(contentType string) Option {
	return func(o *Options) { o.contentType = contentType }
}

// WithBaseDir sets the directory in which template files
// are stored.
func WithBaseDir(dir string) Option {
	return func(o *Options) {
		o.fileFindFunc = func(f string) string {
			return filepath.Join(dir, f)
		}
	}
}

// WithFileFindFunc sets the function that will return the
// file path on disk based on filename provided from files
// defind using WithTemplateFromFile or WithTemplateFromFiles.
func WithFileFindFunc(fn func(filename string) string) Option {
	return func(o *Options) { o.fileFindFunc = fn }
}

// WithFileReadFunc sets the function that will return the
// content of template given the filename.
func WithFileReadFunc(fn FileReadFunc) Option {
	return func(o *Options) { o.fileReadFunc = fn }
}

// WithFileReadOnRender forces template files to be read and
// parsed every time Render or Respond functions are called.
// This is useful for quickly reloading template files,
// but with a performance cost. This functionality
// is disabled by default.
func WithFileReadOnRender(yes bool) Option {
	return func(o *Options) { o.fileReadOnRender = yes }
}

// WithTemplateFromFiles adds a template parsed from files.
func WithTemplateFromFiles(name string, files ...string) Option {
	return func(o *Options) { o.files[name] = files }
}

// WithTemplatesFromFiles adds a map of templates parsed from files.
func WithTemplatesFromFiles(ts map[string][]string) Option {
	return func(o *Options) {
		for name, files := range ts {
			o.files[name] = files
		}
	}
}

// WithTemplateFromStrings adds a template parsed from string.
func WithTemplateFromStrings(name string, strings ...string) Option {
	return func(o *Options) { o.strings[name] = strings }
}

// WithTemplatesFromStrings adds a map of templates parsed from strings.
func WithTemplatesFromStrings(ts map[string][]string) Option {
	return func(o *Options) {
		for name, strings := range ts {
			o.strings[name] = strings
		}
	}
}

// WithFunction adds a function to templates.
func WithFunction(name string, fn any) Option {
	return func(o *Options) { o.functions[name] = fn }
}

// WithFunctions adds function map to templates.
func WithFunctions(fns template.FuncMap) Option {
	return func(o *Options) {
		for name, fn := range fns {
			o.functions[name] = fn
		}
	}
}

// WithDelims sets the delimiters used in templates.
func WithDelims(open, close string) Option {
	return func(o *Options) {
		o.delimOpen = open
		o.delimClose = close
	}
}

// WithLogger sets the function that will perform message logging.
// Default is slog.Default().
func WithLogger(l *slog.Logger) Option {
	return func(o *Options) { o.logger = l }
}

// Templates structure holds parsed templates.
type Templates struct {
	templates   map[string]*template.Template
	parseFiles  func(name string) (*template.Template, error)
	contentType string
	logger      *slog.Logger
}

// New creates a new instance of Templates and parses
// provided files and strings.
func New(opts ...Option) (t *Templates, err error) {
	functions := template.FuncMap{}
	for name, fn := range defaultFunctions {
		functions[name] = fn
	}
	o := &Options{
		fileFindFunc: func(f string) string {
			return f
		},
		fileReadFunc: ioutil.ReadFile,
		files:        map[string][]string{},
		functions:    functions,
		delimOpen:    "{{",
		delimClose:   "}}",
		logger:       slog.Default(),
	}
	for _, opt := range opts {
		opt(o)
	}

	t = &Templates{
		templates:   map[string]*template.Template{},
		contentType: o.contentType,
		logger:      o.logger,
	}
	for name, strings := range o.strings {
		tpl, err := parseStrings(template.New("").Funcs(o.functions).Delims(o.delimOpen, o.delimClose), strings...)
		if err != nil {
			return nil, err
		}
		t.templates[name] = tpl
	}

	parse := func(files []string) (tpl *template.Template, err error) {
		fs := []string{}
		for _, f := range files {
			fs = append(fs, o.fileFindFunc(f))
		}
		return parseFiles(o.fileReadFunc, template.New("").Funcs(o.functions).Delims(o.delimOpen, o.delimClose), fs...)
	}

	if o.fileReadOnRender {
		t.parseFiles = func(name string) (tpl *template.Template, err error) {
			files, ok := o.files[name]
			if !ok {
				return nil, &Error{Err: ErrUnknownTemplate, Template: name}
			}
			return parse(files)
		}
	} else {
		for name, files := range o.files {
			tpl, err := parse(files)
			if err != nil {
				return nil, err
			}
			t.templates[name] = tpl
		}
	}
	return
}

// RespondTemplateWithStatus executes a named template with provided data into buffer,
// then writes the the status and body to the response writer.
// A panic will be raised if the template does not exist or fails to execute.
func (t Templates) RespondTemplateWithStatus(w http.ResponseWriter, name, templateName string, data any, status int) {
	tpl := t.mustTemplate(name)
	buf := bytes.Buffer{}
	if err := tpl.ExecuteTemplate(&buf, templateName, data); err != nil {
		panic(err)
	}
	if t.contentType != "" {
		w.Header().Set("Content-Type", t.contentType)
	}
	if status > 0 {
		w.WriteHeader(status)
	}
	if _, err := buf.WriteTo(w); err != nil {
		t.logger.Debug("templates: respond template with status", "name", name, "template", templateName, "status", status, slog.ErrorKey, err)
	}
}

// RespondWithStatus executes a template with provided data into buffer,
// then writes the the status and body to the response writer.
// A panic will be raised if the template does not exist or fails to execute.
func (t Templates) RespondWithStatus(w http.ResponseWriter, name string, data any, status int) {
	tpl := t.mustTemplate(name)
	buf := bytes.Buffer{}
	if err := tpl.Execute(&buf, data); err != nil {
		panic(err)
	}
	if t.contentType != "" {
		w.Header().Set("Content-Type", t.contentType)
	}
	if status > 0 {
		w.WriteHeader(status)
	}
	if _, err := buf.WriteTo(w); err != nil {
		t.logger.Debug("templates: respond with status", "name", name, "status", status, slog.ErrorKey, err)
	}
}

// RespondTemplate executes a named template with provided data into buffer,
// then writes the the body to the response writer.
// A panic will be raised if the template does not exist or fails to execute.
func (t Templates) RespondTemplate(w http.ResponseWriter, name, templateName string, data any) {
	t.RespondTemplateWithStatus(w, name, templateName, data, 0)
}

// Respond executes template with provided data into buffer,
// then writes the the body to the response writer.
// A panic will be raised if the template does not exist or fails to execute.
func (t Templates) Respond(w http.ResponseWriter, name string, data any) {
	t.RespondWithStatus(w, name, data, 0)
}

// RenderTemplate executes a named template and returns the string.
func (t Templates) RenderTemplate(name, templateName string, data any) (s string, err error) {
	tpl := t.mustTemplate(name)
	buf := bytes.Buffer{}
	if err := tpl.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Render executes a template and returns the string.
func (t Templates) Render(name string, data any) (s string, err error) {
	tpl := t.mustTemplate(name)
	buf := bytes.Buffer{}
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (t Templates) mustTemplate(name string) (tpl *template.Template) {
	tpl, ok := t.templates[name]
	if ok {
		return tpl
	}
	if t.parseFiles != nil {
		tpl, err := t.parseFiles(name)
		if err != nil {
			panic(err)
		}
		return tpl
	}
	panic(&Error{Err: ErrUnknownTemplate, Template: name})
}

func parseFiles(fn FileReadFunc, t *template.Template, filenames ...string) (*template.Template, error) {
	for _, filename := range filenames {
		b, err := fn(filename)
		if err != nil {
			return nil, fmt.Errorf("read template file %s: %v", filename, err)
		}
		_, err = t.Parse(string(b))
		if err != nil {
			return nil, fmt.Errorf("parse template file %s: %v", filename, err)
		}
	}
	return t, nil
}

func parseStrings(t *template.Template, strings ...string) (*template.Template, error) {
	for _, str := range strings {
		_, err := t.Parse(str)
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

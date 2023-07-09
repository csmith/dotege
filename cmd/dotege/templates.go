package main

import (
	"io/ioutil"
	"log"
	"path"
	"sort"
	"strings"
	"text/template"
)

var templateFuncs = template.FuncMap{
	"replace": func(from, to, input string) string { return strings.Replace(input, from, to, -1) },
	"split":   func(sep, input string) []string { return strings.Split(input, sep) },
	"join":    func(sep string, input []string) string { return strings.Join(input, sep) },
	"sortlines": func(input string) string {
		lines := strings.Split(input, "\n")
		sort.Strings(lines)
		return strings.Join(lines, "\n")
	},
}

type Template struct {
	source      string
	destination string
	content     string
	template    *template.Template
}

func CreateTemplate(source, destination string) *Template {
	log.Printf("Registered template from %s, writing to %s", source, destination)
	tmpl, err := template.New(path.Base(source)).Funcs(templateFuncs).ParseFiles(source)
	if err != nil {
		log.Fatalf("Unable to parse template: %v", err)
	}

	buf, _ := ioutil.ReadFile(destination)
	return &Template{
		source:      source,
		destination: destination,
		content:     string(buf),
		template:    tmpl,
	}
}

type Templates []*Template

func (t Templates) Generate(context interface{}) (updated bool) {
	for _, tmpl := range t {
		log.Printf("Checking for updates to %s", tmpl.source)
		builder := &strings.Builder{}
		err := tmpl.template.Execute(builder, context)
		if err != nil {
			panic(err)
		}
		if tmpl.content != builder.String() {
			updated = true
			log.Printf("Writing updated template to %s", tmpl.destination)
			tmpl.content = builder.String()
			err = ioutil.WriteFile(tmpl.destination, []byte(builder.String()), 0666)
			if err != nil {
				log.Fatalf("Unable to write template: %v", err)
			}
		} else {
			log.Printf("Not writing template to %s as content is the same", tmpl.destination)
		}
	}
	return
}

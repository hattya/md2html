//
// md2html :: md2html.go
//
//   Copyright (c) 2020 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

const (
	highlightJS = "https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@10/build"
	mathJax     = "https://cdn.jsdelivr.net/npm/mathjax@3/es5/tex-mml-chtml.js"
)

var (
	hl      = flag.Bool("hl", true, "use highlight.js")
	hllang  = csv{}
	hlstyle = flag.String("hlstyle", "github", "highlight.js style")
	lang    = flag.String("lang", "en", "HTML lang attribute")
	math    = flag.Bool("m", false, "use MathJax")
	title   = flag.String("title", "", "document title")
)

func main() {
	flag.Var(&hllang, "hllang", "comma separated list of highlight.js langauges")
	flag.Parse()

	src, err := open(0)
	if err != nil {
		exit(err)
	}
	defer src.Close()

	dst, err := open(1)
	if err != nil {
		exit(err)
	}
	defer dst.Close()

	exit(convert(src, dst))
}

func open(fd int) (f *os.File, err error) {
	switch fd {
	case 0:
		if name := flag.Arg(fd); name != "" {
			return os.Open(name)
		}
		f = os.Stdin
	case 1:
		if name := flag.Arg(fd); name != "" {
			return os.Create(name)
		}
		f = os.Stdout
	default:
		err = os.ErrInvalid
	}
	return
}

func exit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v: %v\n", os.Args[0], err)
		os.Exit(1)
	}
	os.Exit(0)
}

func convert(r io.Reader, w io.Writer) (err error) {
	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithHeadingAttribute(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
		goldmark.WithExtensions(
			extension.GFM,
			emoji.Emoji,
		),
	)
	src, err := ioutil.ReadAll(r)
	if err != nil {
		return
	}
	doc := md.Parser().Parse(text.NewReader(src))

	fmt.Fprintln(w, `<!DOCTYPE html>`)
	fmt.Fprintf(w, "<html lang=\"%s\">\n", *lang)
	fmt.Fprintln(w, `<head>`)
	fmt.Fprintln(w, `<meta charset="UTF-8">`)
	if *title == "" {
		ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if n.Type() == ast.TypeBlock && n.Kind().String() == "Heading" {
				*title = string(n.Text(src))
				return ast.WalkStop, nil
			}
			return ast.WalkContinue, nil
		})
	}
	fmt.Fprintf(w, "<title>%s</title>\n", *title)
	// highlight.js
	if *hl && *hlstyle != "" {
		fmt.Fprintf(w, "<link rel=\"stylesheet\" href=\"%s/styles/%s.min.css\">\n", highlightJS, *hlstyle)
		fmt.Fprintf(w, "<script src=\"%s/highlight.min.js\"></script>\n", highlightJS)
		for _, lang := range hllang {
			fmt.Fprintf(w, "<script src=\"%s/languages/%s.min.js\"></script>\n", highlightJS, lang)
		}
		fmt.Fprintln(w, `<script>hljs.initHighlightingOnLoad();</script>`)
	}
	// MathJax
	if *math {
		fmt.Fprintln(w, `<script src="https://polyfill.io/v3/polyfill.min.js?features=es6"></script>`)
		fmt.Fprintf(w, "<script id=\"MathJax-script\" async src=\"%s\"></script>\n", mathJax)
	}
	fmt.Fprintln(w, `</head>`)
	fmt.Fprintln(w, `<body>`)
	if err = md.Renderer().Render(w, src, doc); err != nil {
		return
	}
	fmt.Fprintln(w, `</body>`)
	fmt.Fprintln(w, `</html>`)
	return
}

type csv []string

func (csv *csv) Set(s string) error {
	for _, v := range strings.Split(s, ",") {
		*csv = append(*csv, strings.TrimSpace(v))
	}
	return nil
}

func (csv *csv) Get() interface{} { return []string(*csv) }

func (csv *csv) String() string { return strings.Join(*csv, ",") }

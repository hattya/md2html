//
// md2html :: md2html.go
//
//   Copyright (c) 2020-2022 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

const (
	highlightJS = "https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11/build"
	mathJax     = "https://cdn.jsdelivr.net/npm/mathjax@3/es5/tex-mml-chtml.js"
	mermaid     = "https://cdn.jsdelivr.net/npm/mermaid@9/dist/mermaid.min.js"
)

var base string

var (
	diag    = flag.Bool("diag", false, "use Mermaid")
	embed   = flag.Bool("embed", false, "embed local files")
	hl      = flag.Bool("hl", true, "use highlight.js")
	hllang  = csv{}
	hlstyle = flag.String("hlstyle", "github", "highlight.js style")
	lang    = flag.String("lang", "en", "HTML lang attribute")
	math    = flag.Bool("m", false, "use MathJax")
	style   = flag.String("style", "", "style sheet")
	title   = flag.String("title", "", "document title")
)

func init() {
	var err error
	if base, err = os.Getwd(); err != nil {
		exit(err)
	}

	flag.Var(&hllang, "hllang", "comma separated list of highlight.js langauges")
}

func main() {
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
			base = filepath.Dir(name)
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
			extension.CJK,
			extension.GFM,
			emoji.Emoji,
			new(md2html),
		),
	)
	src, err := readAll(r)
	if err != nil {
		return
	}
	doc := md.Parser().Parse(text.NewReader(src))

	fmt.Fprintln(w, `<!DOCTYPE html>`)
	fmt.Fprintf(w, "<html lang=\"%s\">\n", *lang)
	fmt.Fprintln(w, `<head>`)
	fmt.Fprintln(w, `<meta charset="UTF-8">`)
	fmt.Fprintf(w, "<title>%s</title>\n", *title)
	if *style != "" {
		if *embed {
			var f *os.File
			if f, err = os.Open(filepath.Join(base, *style)); err != nil {
				return
			}
			defer f.Close()
			var b []byte
			if b, err = readAll(f); err != nil {
				return
			}
			fmt.Fprintln(w, `<style>`)
			w.Write(b)
			fmt.Fprintln(w, `</style>`)
		} else {
			fmt.Fprintf(w, "<link rel=\"stylesheet\" href=\"%s\">\n", *style)
		}
	}
	// highlight.js
	if *hl && *hlstyle != "" {
		fmt.Fprintf(w, "<link rel=\"stylesheet\" href=\"%s/styles/%s.min.css\">\n", highlightJS, *hlstyle)
		fmt.Fprintf(w, "<script src=\"%s/highlight.min.js\"></script>\n", highlightJS)
		for _, lang := range hllang {
			fmt.Fprintf(w, "<script src=\"%s/languages/%s.min.js\"></script>\n", highlightJS, lang)
		}
		fmt.Fprintln(w, `<script>hljs.highlightAll();</script>`)
	}
	// MathJax
	if *math {
		fmt.Fprintln(w, `<script src="https://polyfill.io/v3/polyfill.min.js?features=es6"></script>`)
		fmt.Fprintf(w, "<script id=\"MathJax-script\" async src=\"%s\"></script>\n", mathJax)
	}
	// Mermaid
	if *diag {
		fmt.Fprintf(w, "<script src=\"%s\"></script>\n", mermaid)
		fmt.Fprintln(w, `<script>mermaid.initialize({ startOnLoad: true });</script>`)
	}
	fmt.Fprintln(w, `</head>`)
	fmt.Fprintln(w, `<body>`)
	fmt.Fprintln(w, `<article class="markdown">`)
	if err = md.Renderer().Render(w, src, doc); err != nil {
		return
	}
	fmt.Fprintln(w, `</article>`)
	fmt.Fprintln(w, `</body>`)
	fmt.Fprintln(w, `</html>`)
	return
}

func readAll(r io.Reader) ([]byte, error) {
	var buf []byte
	br := bufio.NewReader(r)
	for {
		l, err := br.ReadBytes('\n')
		// convert CRLF → LF
		if len(l) > 1 && l[len(l)-2] == '\r' {
			l = l[:len(l)-1]
			l[len(l)-1] = '\n'
		}
		buf = append(buf, l...)

		if err != nil {
			if err == io.EOF {
				return buf, nil
			}
			return nil, err
		}
	}
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

//
// md2html :: extension.go
//
//   Copyright (c) 2020-2022 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"mime"
	"os"
	"path/filepath"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type md2html struct {
}

func (ext *md2html) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(new(astTransformer), 999),
		),
	)
}

type astTransformer struct {
}

func (tr *astTransformer) Transform(doc *ast.Document, r text.Reader, pc parser.Context) {
	if *embed {
		tr.embed(doc)
	}
	if *title == "" {
		tr.title(doc, r)
	}
}

func (tr *astTransformer) embed(doc *ast.Document) {
	ast.Walk(doc, func(n ast.Node, entering bool) (ws ast.WalkStatus, err error) {
		ws = ast.WalkContinue
		if n.Kind() == ast.KindImage && entering {
			img := n.(*ast.Image)
			src := filepath.Join(base, string(img.Destination))

			t := mime.TypeByExtension(filepath.Ext(src))
			if t == "" {
				fmt.Fprintf(os.Stderr, "detect %s: unknown media type\n", src)
				return
			}
			var b []byte
			if b, err = ioutil.ReadFile(src); err != nil {
				fmt.Fprintln(os.Stderr, err)
				err = nil
				return
			}
			scheme := []byte("data:" + t + ";base64,")
			data := make([]byte, len(scheme)+base64.StdEncoding.EncodedLen(len(b)))
			copy(data, scheme)
			base64.StdEncoding.Encode(data[len(scheme):], b)

			img.Destination = data
		}
		return
	})
}

func (tr *astTransformer) title(doc *ast.Document, r text.Reader) {
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if n.Kind() == ast.KindHeading {
			*title = string(n.Text(r.Source()))
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
}

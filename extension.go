//
// md2html :: extension.go
//
//   Copyright (c) 2020-2024 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
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
	md.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(new(nodeRenderer), 500),
		),
	)
}

type astTransformer struct {
}

func (tr *astTransformer) Transform(doc *ast.Document, r text.Reader, pc parser.Context) {
	if *embed {
		tr.embed(doc)
	}
	if *diag {
		tr.mermaid(doc, r)
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
			if b, err = os.ReadFile(src); err != nil {
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

func (tr *astTransformer) mermaid(doc *ast.Document, r text.Reader) {
	var list []*ast.FencedCodeBlock
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if n.Kind() == ast.KindFencedCodeBlock && entering {
			fcb := n.(*ast.FencedCodeBlock)
			if bytes.Equal(fcb.Language(r.Source()), []byte("mermaid")) {
				list = append(list, fcb)
			}
		}
		return ast.WalkContinue, nil
	})

	for _, fcb := range list {
		mb := new(mermaidBlock)
		mb.SetLines(fcb.Lines())
		if parent := fcb.Parent(); parent != nil {
			parent.ReplaceChild(parent, fcb, mb)
		}
	}
}

func (tr *astTransformer) title(doc *ast.Document, r text.Reader) {
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if n.Kind() == ast.KindHeading {
			*title = string(n.Lines().Value(r.Source()))
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
}

type nodeRenderer struct {
}

func (r *nodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindMermaidBlock, r.renderMermaidBlock)
}

func (r *nodeRenderer) renderMermaidBlock(w util.BufWriter, src []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString(`<div class="mermaid">`)
		for i := 0; i < n.Lines().Len(); i++ {
			l := n.Lines().At(i)
			w.Write(util.EscapeHTML(l.Value(src)))
		}
	} else {
		w.WriteString("</div>\n")
	}
	return ast.WalkContinue, nil
}

var kindMermaidBlock = ast.NewNodeKind("MermaidBlock")

type mermaidBlock struct {
	ast.BaseBlock
}

func (n *mermaidBlock) Kind() ast.NodeKind { return kindMermaidBlock }
func (n *mermaidBlock) IsRaw() bool        { return true }

func (n *mermaidBlock) Dump(src []byte, lv int) {
	ast.DumpHelper(n, src, lv, nil, nil)
}

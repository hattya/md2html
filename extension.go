//
// md2html :: extension.go
//
//   Copyright (c) 2020-2026 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"

	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer"
	"github.com/yuin/goldmark/v2/renderer/html"
	"github.com/yuin/goldmark/v2/text"
	"github.com/yuin/goldmark/v2/util"
)

type astTransformer struct {
}

func (tr *astTransformer) Transform(doc *ast.Document, r text.Reader, pc parser.Context) {
	var list []*ast.CodeBlock
	var h bool
	ast.Walk(doc, func(n ast.Node, entering bool) (ws ast.WalkStatus, err error) {
		ws = ast.WalkContinue
		if !entering {
			return
		}
		switch n := n.(type) {
		case *ast.Image:
			if *embed {
				src := filepath.Join(base, n.Destination.Value(r.Source()))

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

				n.Destination = text.NewSingleLineValue(data, r.Decoder())
			}
		case *ast.CodeBlock:
			if *diag {
				if l, ok := n.Language(r.Source()); ok && l == "mermaid" {
					list = append(list, n)
				}
			}
		case *ast.Heading:
			if *title == "" && !h {
				var b bytes.Buffer
				ast.Walk(n, func(n ast.Node, entering bool) (ws ast.WalkStatus, err error) {
					if t, ok := n.(*ast.Text); ok && entering {
						t.Value.WriteTo(&b, r.Source())
					}
					return ast.WalkContinue, nil
				})
				*title = b.String()
				h = true
				ws = ast.WalkSkipChildren
			}
		}
		return
	})

	for _, cb := range list {
		mb := newMermaidBlock()
		mb.SetPos(cb.Pos())
		mb.SetSource(cb.Source())
		mb.SetBlankPreviousLines(cb.HasBlankPreviousLines())
		if p := cb.Parent(); p != nil {
			p.ReplaceChild(cb, mb)
		}
		mb.Value = cb.Value
	}
}

type nodeRenderer struct {
}

func (r *nodeRenderer) Render(w io.Writer, src []byte, n ast.Node, entering bool, rc renderer.Context) (ast.WalkStatus, error) {
	bw := w.(util.BufWriter)
	if entering {
		bw.WriteString(`<pre class="mermaid">`)
		n.(*mermaidBlock).Value.WriteTo(html.ContextTextWriter(rc), src)
	} else {
		bw.WriteString("</pre>\n")
	}
	return ast.WalkContinue, nil
}

var kindMermaidBlock = ast.NewNodeKind("MermaidBlock")

type mermaidBlock struct {
	ast.BaseBlock

	Value text.Lines
}

func newMermaidBlock() *mermaidBlock {
	n := new(mermaidBlock)
	n.Init(n)
	return n
}

func (n *mermaidBlock) Kind() ast.NodeKind { return kindMermaidBlock }

func (n *mermaidBlock) Dump(_ []byte) *ast.NodeDump {
	return ast.NewNodeDump(n, nil)
}

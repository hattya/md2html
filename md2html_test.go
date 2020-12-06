//
// md2html :: md2html_test.go
//
//   Copyright (c) 2020 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hattya/go.diff"
)

var (
	saveHL      bool
	saveHLStyle string
	saveLang    string
	saveTitle   string
)

func init() {
	saveHL = *hl
	saveHLStyle = *hlstyle
	saveLang = *lang
	saveTitle = *title
}

func TestOpen(t *testing.T) {
	dir, err := tempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	md := filepath.Join(dir, "a.md")
	if err := touch(md); err != nil {
		t.Fatal(err)
	}
	html := filepath.Join(dir, "a.html")

	for _, tt := range []struct {
		args  []string
		files []string
	}{
		{[]string{}, []string{os.Stdin.Name(), os.Stdout.Name()}},
		{[]string{md}, []string{md, os.Stdout.Name()}},
		{[]string{md, html}, []string{md, html}},
	} {
		if err := flag.CommandLine.Parse(tt.args); err != nil {
			t.Fatal(err)
		}
		for i, e := range tt.files {
			switch g, err := open(i); {
			case err != nil:
				t.Error(err)
			case g.Name() != e:
				t.Error("unexpected file:", g.Name())
			}
		}
	}

	if _, err := open(3); err != os.ErrInvalid {
		t.Fatal("unexpected error:", err)
	}
}

func TestConvert(t *testing.T) {
	src, err := ioutil.ReadFile(filepath.Join("testdata", "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	t.Run("deafult", func(t *testing.T) {
		if err := try(src, "default.html"); err != nil {
			t.Error(err)
		}
	})
	t.Run("lang", func(t *testing.T) {
		*lang = "ja"
		if err := try(src, "lang-ja.html"); err != nil {
			t.Error(err)
		}
	})
	t.Run("title", func(t *testing.T) {
		*title = "test"
		if err := try(src, "title-test.html"); err != nil {
			t.Error(err)
		}
	})
	t.Run("highlight.js", func(t *testing.T) {
		*hl = false
		if err := try(src, "highlight-disabled.html"); err != nil {
			t.Error(err)
		}
		*hl = true
		*hlstyle = ""
		if err := try(src, "highlight-disabled.html"); err != nil {
			t.Error(err)
		}
		*hlstyle = "monokai"
		if err := try(src, "highlight-monokai.html"); err != nil {
			t.Error(err)
		}
	})
}

func try(src []byte, name string) error {
	defer func() {
		*hl = saveHL
		*hlstyle = saveHLStyle
		*lang = saveLang
		*title = saveTitle
	}()

	var dst bytes.Buffer
	if err := convert(bytes.NewReader(src), &dst); err != nil {
		return err
	}
	if err := verify(dst.String(), filepath.Join("testdata", name)); err != nil {
		return err
	}
	return nil
}

func verify(out, html string) (err error) {
	golden, err := ioutil.ReadFile(html)
	if err != nil {
		return
	}
	golden = bytes.ReplaceAll(golden, []byte("${highlight.js}"), []byte(highlightJS))

	a := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	b := strings.Split(strings.TrimSuffix(string(golden), "\n"), "\n")
	var Δ []string
	format := func(sign string, lines []string, i, j int) {
		for ; i < j; i++ {
			Δ = append(Δ, sign+lines[i])
		}
	}
	switch {
	case len(golden) == 0:
		format("-", a, 0, len(a))
	default:
		cl := diff.Strings(a, b)
		if len(cl) > 0 {
			lno := 0
			for _, c := range cl {
				format(" ", a, lno, c.A)
				format("-", a, c.A, c.A+c.Del)
				format("+", b, c.B, c.B+c.Ins)
				lno = c.A + c.Del
			}
			format(" ", a, lno, len(a))
		}
	}
	if len(Δ) > 0 {
		err = fmt.Errorf("diff of %v\n%v", html, strings.Join(Δ, "\n"))
	}
	return
}

func tempDir() (string, error) {
	return ioutil.TempDir("", "md2html")
}

func touch(s ...string) error {
	return ioutil.WriteFile(filepath.Join(s...), []byte{}, 0666)
}

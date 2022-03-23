//
// md2html :: md2html_test.go
//
//   Copyright (c) 2020-2022 Akinori Hattori <hattya@gmail.com>
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
	"reflect"
	"strings"
	"testing"

	"github.com/hattya/go.diff"
)

var (
	saveEmbed   bool
	saveHL      bool
	saveHLLang  []string
	saveHLStyle string
	saveLang    string
	saveMath    bool
	saveStyle   string
	saveTitle   string
)

func init() {
	saveEmbed = *embed
	saveHL = *hl
	saveHLLang = make([]string, len(hllang))
	copy(saveHLLang, hllang)
	saveHLStyle = *hlstyle
	saveLang = *lang
	saveMath = *math
	saveStyle = *style
	saveTitle = *title
}

func TestCSV(t *testing.T) {
	var v csv

	if g, e := v.Get(), []string(nil); !reflect.DeepEqual(g, e) {
		t.Errorf("expected %#v, got %#v", e, g)
	}
	if g, e := v.String(), ""; g != e {
		t.Errorf("expected %q, got %q", e, g)
	}

	if err := v.Set(" foo , bar "); err != nil {
		t.Error(err)
	}
	if g, e := v.Get(), []string{"foo", "bar"}; !reflect.DeepEqual(g, e) {
		t.Errorf("expected %#v, got %#v", e, g)
	}
	if g, e := v.String(), "foo,bar"; g != e {
		t.Errorf("expected %q, got %q", e, g)
	}
}

func TestOpen(t *testing.T) {
	dir := t.TempDir()
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
			f, err := open(i)
			switch {
			case err != nil:
				t.Error(err)
			case f.Name() != e:
				t.Error("unexpected file:", f.Name())
			}
			if i < len(tt.args) {
				f.Close()
			}
		}
	}

	if _, err := open(3); err != os.ErrInvalid {
		t.Fatal("unexpected error:", err)
	}
}

func TestReadAll(t *testing.T) {
	for _, data := range [][]byte{
		[]byte("1\n\n3\n"),
		[]byte("1\r\n\r\n3\r\n"),
		[]byte("1\n\r\n3\n"),
		[]byte("1\r\n\n3\r\n"),
	} {
		if g, err := readAll(bytes.NewReader(data)); err != nil {
			t.Error(err)
		} else if e := []byte("1\n\n3\n"); !reflect.DeepEqual(g, e) {
			t.Errorf("expected %q, got %q", e, g)
		}
	}

	for _, data := range [][]byte{
		[]byte("1\n\n3"),
		[]byte("1\r\n\r\n3"),
		[]byte("1\n\r\n3"),
		[]byte("1\r\n\n3"),
	} {
		if g, err := readAll(bytes.NewReader(data)); err != nil {
			t.Error(err)
		} else if e := []byte("1\n\n3"); !reflect.DeepEqual(g, e) {
			t.Errorf("expected %q, got %q", e, g)
		}
	}
}

func TestConvert(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	base = filepath.Join(wd, "testdata")

	src, err := ioutil.ReadFile(filepath.Join("testdata", "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	t.Run("default", func(t *testing.T) {
		if err := try(src, "default.html"); err != nil {
			t.Error(err)
		}
	})
	t.Run("embed", func(t *testing.T) {
		src, err := ioutil.ReadFile(filepath.Join("testdata", "embed.md"))
		if err != nil {
			t.Fatal(err)
		}

		*embed = true
		*style = "style-lf.css"
		if err := try(src, "embed.html"); err != nil {
			t.Error(err)
		}

		*embed = true
		*style = "style-crlf.css"
		if err := try(src, "embed.html"); err != nil {
			t.Error(err)
		}

		*embed = true
		*style = "_.css"
		if err := try(nil, "embed.html"); err == nil {
			t.Error("expected error")
		}
	})
	t.Run("lang", func(t *testing.T) {
		*lang = "ja"
		if err := try(src, "lang-ja.html"); err != nil {
			t.Error(err)
		}
	})
	t.Run("style", func(t *testing.T) {
		*style = "style.css"
		if err := try(src, "style.html"); err != nil {
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
		if err := try(src, "hl-false.html"); err != nil {
			t.Error(err)
		}

		*hl = true
		*hlstyle = ""
		if err := try(src, "hl-false.html"); err != nil {
			t.Error(err)
		}

		*hl = true
		hllang = []string{"vim"}
		if err := try(src, "hllang-vim.html"); err != nil {
			t.Error(err)
		}

		*hl = true
		*hlstyle = "monokai"
		if err := try(src, "hlstyle-monokai.html"); err != nil {
			t.Error(err)
		}
	})
	t.Run("MathJax", func(t *testing.T) {
		*math = true
		if err := try(src, "math.html"); err != nil {
			t.Error(err)
		}
	})
}

func try(src []byte, name string) error {
	defer func() {
		*embed = saveEmbed
		*hl = saveHL
		hllang = make([]string, len(saveHLLang))
		copy(hllang, saveHLLang)
		*hlstyle = saveHLStyle
		*lang = saveLang
		*math = saveMath
		*style = saveStyle
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
	for k, v := range map[string]string{
		"${highlight.js}": highlightJS,
		"${MathJax}":      mathJax,
	} {
		golden = bytes.ReplaceAll(golden, []byte(k), []byte(v))
	}

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

func touch(s ...string) error {
	return ioutil.WriteFile(filepath.Join(s...), []byte{}, 0o666)
}

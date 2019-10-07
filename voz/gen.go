package voz

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	readability "github.com/go-shiori/go-readability"
	"github.com/gocolly/colly"
)

const header = `
<?xml version='1.0' encoding='utf-8'?>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>Unknown</title>
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
  </head>
  <body>
`

const footer = `
</body></html>
`

const tpl = `
<?xml version='1.0' encoding='utf-8'?>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>{{.Name}}</title>
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
  </head>
  <body>
     <h1>Table of Contents</h1>
	 <p style="text-indent:0pt">
	 {{range $i, $e := .Chap}}<a href="chap_{{add $i 1}}.html">Chap {{add $i 1}}</a><br/>{{end}}
     </p>
   </body>
</html>

`

func add(x, y int) int {
	return x + y
}

const ebookConvertCmd = `ebook-convert` // require Calibre

type voz struct {
	// Name is the subject of the thread
	Name string
	// WorkingDir is the path of the working folder, which is a
	// subfolder of the default directory for temporary files.
	WorkingDir string
	colly      *colly.Collector
	numChap    int
}

func New(name string) (*voz, error) {
	r := make([]byte, 6)
	if _, err := rand.Read(r); err != nil {
		return nil, err
	}

	dir, err := ioutil.TempDir("", name+hex.EncodeToString(r))
	if err != nil {
		return nil, err
	}

	return &voz{
		WorkingDir: dir,
		Name:       name,
		numChap:    0,
		colly:      colly.NewCollector(),
	}, nil
}

func (v *voz) Run(urls []string) (string, error) {
	if err := v.readability(urls); err != nil {
		return "", err
	}
	if err := v.genToC(); err != nil {
		return "", err
	}
	if err := v.genMobi(); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/[voz-f17]%s.mobi", v.WorkingDir, v.Name), nil
}

func (v *voz) Remove() {
	os.RemoveAll(v.WorkingDir)
}

func (v *voz) readability(urls []string) error {
	for i, url := range urls {
		v.colly = colly.NewCollector()

		v.colly.OnRequest(func(r *colly.Request) {
			log.Println("Visiting", r.URL.String())
		})

		v.colly.OnResponse(func(r *colly.Response) {
			article, err := readability.FromReader(bytes.NewReader(r.Body), url)
			if err != nil {
				log.Println(err)
			}
			dstHTMLFile, _ := os.Create(fmt.Sprintf("%s/chap_%d.html", v.WorkingDir, i+1))
			defer dstHTMLFile.Close()
			dstHTMLFile.WriteString(header + article.Content + footer)
			v.numChap++
		})

		v.colly.Visit(url)
	}
	return nil
}

func (v *voz) genToC() error {
	funcs := template.FuncMap{"add": add}
	t := template.Must(template.New("index.html").Funcs(funcs).Parse(tpl))
	story := struct {
		Name string
		Chap []int
	}{
		Name: v.Name,
		Chap: make([]int, v.numChap),
	}

	ftpl, _ := os.Create(fmt.Sprintf("%s/toc.html", v.WorkingDir))
	defer ftpl.Close()
	err := t.Execute(ftpl, story)
	return err
}

func (v *voz) genMobi() error {
	cmd := exec.Command(ebookConvertCmd,
		fmt.Sprintf("%s/toc.html", v.WorkingDir),
		fmt.Sprintf("%s/[voz-f17]%s.mobi", v.WorkingDir, v.Name),
		"--output-profile", "kindle_voyage",
		"--authors", "Vozer")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

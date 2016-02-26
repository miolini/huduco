package main

import "fmt"
import "io"
import "log"
import "net/http"
import "gopkg.in/flosch/pongo2.v3"
import "strings"

func main() {
	app := NewApp()
	log.Printf("run server")
	if err := http.ListenAndServe(":8080", app); err != nil {
		log.Fatal(err)
	}
}

type App struct {
	tplImport *pongo2.Template
	httpClient *http.Client
}

func NewApp() *App {
	var err error
	app := &App{}
	app.tplImport, err = pongo2.FromString(tplImport)
	if err != nil {
		panic(err)
	}
	app.httpClient = &http.Client{}
	return app
}

func (app *App) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		log.Printf("panic: %s", r)
	}()
	log.Printf("%s %s", req.Method, req.URL.Path)
	if req.URL.Path == "/" {
		app.handleIndex(rw, req)
		return
	}
	app.handleImport(rw, req)
}

func (app *App) handleIndex(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(rw, "<h1>Welcome!</h1><h3>Usage</h3><pre>$ go get hudu.co/github_username/github_project.git_tag</pre><h3>Example</h3><pre>$ go get hudu.co/meshbird/meshbird.v0.2</pre><h3>Author</h3><a href=\"https://twitter.com/miolini\" target=\"_blank\">Artem Andreenko</a>")
}

func (app *App) handleImport(rw http.ResponseWriter, req *http.Request) {
	var scm, host, path, tag string
	parts := strings.Split(req.URL.Path[1:], "/")
	log.Printf("parts: %d %#v", len(parts), parts)
	path = parts[0] + "/" + parts[1]
	pos := strings.Index(path, ".")
	if pos == -1 {
		panic("bad sember")
	}
	tag = path[pos+1:]
	path = path[:pos]
	scm = "git"
	host = "github.com"

	if len(parts) == 2 {
		rw.Header().Set("Content-Type", "text/html")
		err := app.tplImport.ExecuteWriter(pongo2.Context{"scm": scm, "host":host, "path": path, "tag": tag}, rw)
		checkErr(err, "template execute err: %s", err)
		return
	}
	proxyUrl := fmt.Sprintf("https://github.com/%s.git/%s", path, strings.Join(parts[2:], "/"))
	if req.URL.RawQuery != "" {
		proxyUrl += "?" + req.URL.RawQuery
	}
	log.Printf("proxy url: %s", proxyUrl)
	log.Printf("proxy request headers: %V", req.Header)
	proxyReq, err := http.NewRequest(req.Method, proxyUrl, nil)
	checkErr(err, "proxy request create err: %s", err)
	proxyReq.Header = req.Header
	if req.Method == "POST" {
		proxyReq.Body = req.Body
	}
	proxyRes, err := app.httpClient.Do(proxyReq)
	checkErr(err, "do proxy request %s err: %s", proxyUrl, err)
	defer proxyRes.Body.Close()
	log.Printf("proxy response headers: %v", proxyRes.Header)
	for name, values := range proxyRes.Header {
		for _, value := range values {
			rw.Header().Add(name, value)
		}
	}
	rw.WriteHeader(proxyRes.StatusCode)
	_, err = io.Copy(rw, proxyRes.Body)
	checkErr(err, "copy proxy response %s err: %s", proxyUrl, err)
}

func checkErr(err interface{}, msg string, args...interface{}) {
	if err != nil {
		panic(fmt.Errorf(msg, args...))
	}
}

var tplImport = `
<html>
<head>
<meta name="go-import" content="hudu.co/{{path}}.{{tag}} {{scm}} https://hudu.co/{{path}}.{{tag}}">
<meta name="go-source" content="hudu.co/{{path}}.{{tag}} _ https://{{host}}/{{path}}/tree/{{tag}}{/dir} https://{{host}}/{{path}}/blob/{{tag}}{/dir}/{file}#L{line}">
</head>
<body>
go get hudu.co/{{scm}}/{{host}}/{{path}}/{{tag}}
</body>
</html>
`
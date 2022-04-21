package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

const targetTemplate = `
<!DOCTYPE html>
<head>
<style>
#log {
	font-family: monospace;
}
</style>
<script>
function appendLog(text) {
	console.dir(text)
	const p = document.createElement('p');
	p.innerText = performance.now() + ": " + text;
	document.querySelector('#log').appendChild(p);
}

new PerformanceObserver((entryList) => {
	for (const entry of entryList.getEntries()) {
		appendLog("LCP candidate: "+entry.startTime)
	}
}).observe({type: 'largest-contentful-paint', buffered: true});

window.addEventListener('load', () => {
	const [e] = performance.getEntriesByType('navigation')
	appendLog("load. respEnd-reqStart: " + (e.responseEnd - e.requestStart));

	const reses = performance.getEntriesByType('resource');
	for (let e of reses) {
		appendLog("load "+e.name+" respEnd-reqStart: " + (e.responseEnd - e.requestStart));
	}
	// appendLog(entry.toJSON())
})
</script>
<body>
<div id=log>
<p>request time: {{ .Now }}
</div>
<img src="/subresource{{ .Rand }}.jpg" width=1000>
`

func makeCachable(w http.ResponseWriter) {
	w.Header().Add("Cache-Control", "public, max-age=605800")
}

func targetHandlerFunc() func(w http.ResponseWriter, req *http.Request) {
	t, err := template.New("").Parse(targetTemplate)
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, req *http.Request) {
		writeError := func(err error) {
			http.Error(w, fmt.Sprintf("err: %v", err), 500)
		}

		var buf bytes.Buffer
		err := t.Execute(&buf, struct {
			Now  time.Time
			Rand int
		}{
			time.Now(),
			rand.Intn(1000),
		})
		if err != nil {
			writeError(err)
			return
		}

		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		makeCachable(w)
		_, _ = buf.WriteTo(w)
	}
}

func imageHandlerFunc() func(w http.ResponseWriter, req *http.Request) {
	bs, err := ioutil.ReadFile("./static/cat.jpg")
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, req *http.Request) {
		time.Sleep(1 * time.Second)
		w.Header().Add("Content-Type", "image/jpeg")
		makeCachable(w)
		_, _ = w.Write(bs)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	m := mux.NewRouter()

	fs := http.FileServer(http.Dir("./static"))
	m.Handle("/", fs)

	m.HandleFunc("/target{num:[0-9]+}", targetHandlerFunc())
	m.HandleFunc("/subresource{num:[0-9]+}.jpg", imageHandlerFunc())

	loggedRouter := handlers.LoggingHandler(os.Stdout, m)
	if err := http.ListenAndServe(":8000", loggedRouter); err != nil {
		panic(err)
	}
}

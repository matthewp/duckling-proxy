package quack

import (
	"context"
	"crypto/tls"
	"embed"
	"os/signal"
	"strconv"
	"text/template"

	"crawler.club/ce"
	"git.sr.ht/~adnano/go-gemini"
	"git.sr.ht/~adnano/go-gemini/certificate"
	"github.com/LukeEmmet/html2gemini"
	flag "github.com/spf13/pflag"

	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var version = "0.2.1"

//go:embed templates/*
var templatecontent embed.FS

type TemplateRenderer = func(*gemini.Request, gemini.ResponseWriter, Page)

type WebPipeHandler struct {
	RenderTemplate TemplateRenderer
}

func fatal(format string, a ...interface{}) {
	urlError(format, a...)
	os.Exit(1)
}

func urlError(format string, a ...interface{}) {
	format = "Error: " + strings.TrimRight(format, "\n") + "\n"
	fmt.Fprintf(os.Stderr, format, a...)
}

func info(format string, a ...interface{}) {
	format = "Info: " + strings.TrimRight(format, "\n") + "\n"
	fmt.Fprintf(os.Stderr, format, a...)
}

func check(e error) {
	if e != nil {
		panic(e)
		os.Exit(1)
	}
}

type Page struct {
	Title   string
	Url     string
	Gemtext string
	Version string
}

func htmlToGmi(inputHtml string) (string, error) {

	//convert html to gmi
	options := html2gemini.NewOptions()
	options.PrettyTables = *prettyTables
	options.CitationStart = *citationStart
	options.LinkEmitFrequency = *linkEmitFrequency
	options.CitationMarkers = *citationMarkers
	options.NumberedLinks = *numberedLinks
	options.EmitImagesAsLinks = *emitImagesAsLinks

	//dont use an extra line to separate header from body, but
	//do separate each row visually
	options.PrettyTablesOptions.HeaderLine = false
	options.PrettyTablesOptions.RowLine = true

	//pretty tables option is somewhat experimental
	//and the column positions not always correct
	//so use invisible borders of spaces for now
	options.PrettyTablesOptions.CenterSeparator = " "
	options.PrettyTablesOptions.ColumnSeparator = " "
	options.PrettyTablesOptions.RowSeparator = " "

	ctx := html2gemini.NewTraverseContext(*options)

	return html2gemini.FromString(inputHtml, *ctx)

}

//func (h WebPipeHandler) Handle(r gemini.Request) *gemini.Response {
func (h WebPipeHandler) Handle(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
	url := r.URL.String()
	if r.URL.Scheme != "http" && r.URL.Scheme != "https" {
		//any other schemes are not implemented by this proxy
		w.WriteHeader(53, "Scheme not supported: "+r.URL.Scheme)
		return
	}

	info("Retrieve: %s", r.URL.String())

	//see https://medium.com/@nate510/don-t-use-go-s-default-http-client-4804cb19f779
	//also https://gist.github.com/ijt/950790/fca88967337b9371bb6f7155f3304b3ccbf3946f

	connectTimeout := time.Second * time.Duration(*maxConnectTime)
	clientTimeout := time.Second * time.Duration(*maxDownloadTime)

	//create custom transport with timeout
	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: connectTimeout,
		}).Dial,
		TLSHandshakeTimeout: connectTimeout,
	}

	//create custom client with timeout
	var netClient = &http.Client{
		Timeout:   clientTimeout,
		Transport: netTransport,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		w.WriteHeader(43, "Could not connect to remote HTTP host")
		return
	}

	//set user agent if specified
	if *userAgent != "" {
		req.Header.Add("User-Agent", *userAgent)
	}

	response, err := netClient.Do(req)
	if err != nil {
		w.WriteHeader(43, "Remote host did not respond with valid HTTP")
		return
	}

	defer response.Body.Close()

	//final response (may have redirected)
	if url != response.Request.URL.String() {
		//notify of target location on stderr
		//see https://stackoverflow.com/questions/16784419/in-golang-how-to-determine-the-final-url-after-a-series-of-redirects
		info("Redirected to: %s", response.Request.URL.String())

		//tell the client to get it from a different location otherwise the client
		//wont know the baseline for link refs
		w.WriteHeader(30, response.Request.URL.String())
		return
	}

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		abandonMsg := fmt.Sprintf("Download abandoned after %d seconds: %s", *maxDownloadTime, response.Request.URL.String())
		info(abandonMsg)
		w.WriteHeader(43, abandonMsg)
		return
	}

	if response.StatusCode == 200 {
		contentType := response.Header.Get("Content-Type")

		info("Content-Type: %s", contentType)

		var body io.ReadCloser
		if !*unfiltered && strings.Contains(contentType, "text/html") {

			info("Converting to text/gemini: %s", r.URL.String())

			doc := ce.ParsePro(url, string(contents), "127.0.0.1", false)

			var transformedHtml string = doc.Html
			gmi, err := htmlToGmi(transformedHtml)

			if err != nil {
				w.WriteHeader(42, "HTML to GMI conversion failure")
				return
			}

			contentType = "text/gemini"
			w.WriteHeader(20, contentType)
			h.RenderTemplate(r, w, Page{
				Title:   doc.Title,
				Url:     url,
				Gemtext: gmi,
				Version: version,
			})
			return

		} else {
			//let everything else through with the same content type
			body = ioutil.NopCloser(strings.NewReader(string(contents)))
		}

		w.WriteHeader(20, contentType)
		io.Copy(w, body)
		return

	} else if response.StatusCode == 404 {
		w.WriteHeader(51, "Not found")
		return
		//return &gemini.Response{51, "Not found", nil, nil}
	} else {
		w.WriteHeader(50, "Failure: HTTP status: "+response.Status)
		return
		//return &gemini.Response{50, "Failure: HTTP status: " + response.Status, nil, nil}
	}
}

type Proxy struct {
	Mux            *gemini.Mux
	Handler        WebPipeHandler
	RenderTemplate TemplateRenderer
}

func NewProxy() *Proxy {
	tmpl := template.Must(template.ParseFS(templatecontent, "templates/*"))

	return &Proxy{
		Handler: WebPipeHandler{
			RenderTemplate: CreateDefaultRenderTemplate(tmpl),
		},
	}
}

func (p *Proxy) SetRenderTemplate(renderTemplate func(*gemini.Request, gemini.ResponseWriter, Page)) {
	p.Handler.RenderTemplate = renderTemplate
}

func CreateDefaultRenderTemplate(tmpl *template.Template) TemplateRenderer {
	return func(r *gemini.Request, w gemini.ResponseWriter, p Page) {
		tmpl.Execute(w, p)
	}
}

func (p *Proxy) Start() {
	flag.Parse()

	if *verFlag {
		fmt.Println("Duckling Proxy v" + version)
		return
	}

	info("Starting Duckling Proxy v%s on %s port: %d", version, *address, *port)

	certificates := &certificate.Store{}
	var scope string = "*"
	certificates.Register(scope)

	var pubkeybytes []byte
	var privkeybytes []byte
	if os.Getenv("CERT") != "" {
		pubkeybytes = []byte(os.Getenv("CERT"))
		privkeybytes = []byte(os.Getenv("KEY"))
	} else {
		c, err := ioutil.ReadFile(*serverCert)
		if err != nil {
			log.Fatal(err)
		}
		pubkeybytes = c
		k, err := ioutil.ReadFile(*serverKey)
		if err != nil {
			log.Fatal(err)
		}
		privkeybytes = k
	}

	cert, err := tls.X509KeyPair(pubkeybytes, privkeybytes)
	if err != nil {
		log.Fatal(err)
	}
	certificates.Add(scope, cert)

	baseHandler := gemini.HandlerFunc(func(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
		if r.URL.Scheme == "gemini" {
			if p.Mux != nil {
				p.Mux.ServeGemini(ctx, w, r)
				return
			}
		}
		p.Handler.Handle(ctx, w, r)
	})

	addr := ":" + strconv.Itoa(*port)
	if *address != "127.0.0.1" {
		addr = *address + addr
	}

	server := &gemini.Server{
		Addr:           addr,
		Handler:        gemini.LoggingMiddleware(baseHandler),
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   1 * time.Minute,
		GetCertificate: certificates.Get,
	}

	// Listen for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	errch := make(chan error)
	go func() {
		ctx := context.Background()
		errch <- server.ListenAndServe(ctx)
	}()

	select {
	case err := <-errch:
		log.Fatal(err)
	case <-c:
		// Shutdown the server
		log.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := server.Shutdown(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}
}

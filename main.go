package quack

import (
	"context"
	"embed"
	"text/template"

	"crawler.club/ce"
	"git.sr.ht/~adnano/go-gemini"
	"github.com/LukeEmmet/html2gemini"

	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed templates/*
var templatecontent embed.FS

type TemplateRenderer = func(*gemini.Request, gemini.ResponseWriter, Page)

type WebPipeHandler struct {
	RenderTemplate TemplateRenderer
	ProxyOptions   *ProxyOptions
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

type ProxyOptions struct {
	ConversionOptions *ConversionOptions
	UserAgent         string
	MaxDownloadTime   int
	MaxConnectTime    int
}

type ConversionOptions struct {
	PrettyTables      bool
	CitationStart     int
	LinkEmitFrequency int
	CitationMarkers   bool
	NumberedLinks     bool
	EmitImagesAsLinks bool
}

var defaultConversionOpts = &ConversionOptions{
	CitationStart:     1,
	CitationMarkers:   false,
	NumberedLinks:     false,
	PrettyTables:      false,
	EmitImagesAsLinks: true,
	LinkEmitFrequency: 2,
}

func htmlToGmi(opts *ConversionOptions, inputHtml string) (string, error) {

	//convert html to gmi
	options := html2gemini.NewOptions()
	options.PrettyTables = opts.PrettyTables
	options.CitationStart = opts.CitationStart
	options.LinkEmitFrequency = opts.LinkEmitFrequency
	options.CitationMarkers = opts.CitationMarkers
	options.NumberedLinks = opts.NumberedLinks
	options.EmitImagesAsLinks = opts.EmitImagesAsLinks

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

	connectTimeout := time.Second * time.Duration(h.ProxyOptions.MaxConnectTime)
	clientTimeout := time.Second * time.Duration(h.ProxyOptions.MaxDownloadTime)

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
	if h.ProxyOptions.UserAgent != "" {
		req.Header.Add("User-Agent", h.ProxyOptions.UserAgent)
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
		abandonMsg := fmt.Sprintf("Download abandoned after %d seconds: %s", h.ProxyOptions.MaxConnectTime, response.Request.URL.String())
		info(abandonMsg)
		w.WriteHeader(43, abandonMsg)
		return
	}

	if response.StatusCode == 200 {
		contentType := response.Header.Get("Content-Type")

		info("Content-Type: %s", contentType)

		var body io.ReadCloser
		if strings.Contains(contentType, "text/html") {

			info("Converting to text/gemini: %s", r.URL.String())

			doc := ce.ParsePro(url, string(contents), "127.0.0.1", false)

			var transformedHtml string = doc.Html
			gmi, err := htmlToGmi(h.ProxyOptions.ConversionOptions, transformedHtml)

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

func NewProxy(opts *ProxyOptions) *Proxy {
	tmpl := template.Must(template.ParseFS(templatecontent, "templates/*")).Lookup("view.gmi.tmpl")

	if opts == nil {
		opts = &ProxyOptions{
			ConversionOptions: defaultConversionOpts,
		}
	} else if opts.ConversionOptions == nil {
		opts.ConversionOptions = defaultConversionOpts
	}

	return &Proxy{
		Handler: WebPipeHandler{
			ProxyOptions:   opts,
			RenderTemplate: CreateDefaultRenderTemplate(tmpl),
		},
	}
}

func (p *Proxy) SetRenderTemplate(renderTemplate func(*gemini.Request, gemini.ResponseWriter, Page)) {
	p.Handler.RenderTemplate = renderTemplate
}

type MiddlewareOptions struct {
	Handler        gemini.Handler
	RenderTemplate *TemplateRenderer
	ProxyOptions   *ProxyOptions
}

func (p *Proxy) Middleware(opts MiddlewareOptions) gemini.HandlerFunc {
	if opts.RenderTemplate != nil {
		p.Handler.RenderTemplate = *opts.RenderTemplate
	}

	hh := opts.Handler
	return gemini.HandlerFunc(func(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
		if r.URL.Scheme == "gemini" {
			hh.ServeGemini(ctx, w, r)
			return
		}
		p.Handler.Handle(ctx, w, r)
	})
}

func (p *Proxy) DefaultMiddleware() gemini.HandlerFunc {
	return gemini.HandlerFunc(func(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
		if r.URL.Scheme == "gemini" {
			if p.Mux != nil {
				p.Mux.ServeGemini(ctx, w, r)
				return
			}
		}
		p.Handler.Handle(ctx, w, r)
	})
}

func Middleware(opts MiddlewareOptions) gemini.HandlerFunc {
	return NewProxy(nil).Middleware(opts)
}

func CreateDefaultRenderTemplate(tmpl *template.Template) TemplateRenderer {
	return func(r *gemini.Request, w gemini.ResponseWriter, p Page) {
		tmpl.Execute(w, p)
	}
}

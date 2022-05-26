# Quack Proxy 🦆

Quack proxy is a Gemini proxy that proxies HTTP content into the Gemini protocol. It is a fork of [Duckling proxy](https://github.com/LukeEmmet/duckling-proxy) adding the following features:

* When possible, Quack extracts the *article* from the page and avoids rendering sidebars and other content that doesn't translate well to gemtext.
* Quack is available as *middleware* to [go-gemini](https://git.sr.ht/~adnano/go-gemini) so it can be used from your own server.

Quack proxy is cross platform and written in Go.

## Usage

You will need to configure your Gemini client to point to the server when there is a need to access any <code>http://</code> or <code>https://</code> requests.

### As middleware

You can also use quack-proxy as middleware with [go-gemini](https://git.sr.ht/~adnano/go-gemini).

```go
mux := &gemini.Mux{}

middleware := quack.Middleware(quack.MiddlewareOptions{
  Handler:        mux,
})

mux.HandleFunc("/", func(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
  w.WriteHeader(gemini.StatusSuccess, "text/gemini")
  w.Write([]byte("Hello world!"))
})

server := &gemini.Server{
  Addr:           ":1965",
  Handler:        gemini.LoggingMiddleware(middleware),
  ReadTimeout:    30 * time.Second,
  WriteTimeout:   1 * time.Minute,
  GetCertificate: setupCerts().Get,
}

ctx := context.Background()
server.ListenAndServe(ctx)
```

## Supported clients

The following clients support per-scheme proxies and can be configured to use Duckling proxy.

* [Amfora](https://github.com/makeworld-the-better-one/amfora) - supports per scheme proxies since v1.5.0
* [AV-98](https://tildegit.org/solderpunk/AV-98)  - Merge [pull request #24](https://tildegit.org/solderpunk/AV-98/pulls/24) then use `set http_proxy machine:port` to access. 
* [diohsc](https://repo.or.cz/diohsc.git) - edit diohscrc config file
* [gemget](https://github.com/makeworld-the-better-one/gemget) - use -p option
* [GemiNaut](https://github.com/LukeEmmet/GemiNaut) - since 0.8.8, which also has its own native html to gemini conversion - update in settings
* [Lagrange](https://git.skyjake.fi/skyjake/lagrange) - set proxy in preferences (use 127.0.0.1:port, not localhost:port for localhost)
* [Telescope](https://telescope.omarpolo.com/) - set proxy in the config file add: ```proxy "https" via "gemini://127.0.0.1:1965"```, and similarly for http
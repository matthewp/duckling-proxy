# Quack Proxy 🦆

Quack proxy is a Gemini proxy that proxies HTTP content into the Gemini protocol. It is a fork of [Duckling proxy](https://github.com/LukeEmmet/duckling-proxy) adding the following features:

* When possible, Quack extracts the *article* from the page and avoids rendering sidebars and other content that doesn't translate well to gemtext.
* Quack is available as *middleware* to [go-gemini](https://git.sr.ht/~adnano/go-gemini) so it can be used from your own server.

Quack proxy is cross platform and written in Go.

## Usage

### Full server

```go
import (
  quack "github.com/matthewp/quack-proxy"
)
```

You can use Quack a couple of ways. To start a server straight away using CLI flags, like with Duckling, just call Start():

```go
func main() {
  quack.Start()
}
```

The following flags are inherited from Duckling:

```
Usage:
  -a, --address string          Bind to address
                                 (default "127.0.0.1")
  -m, --citationMarkers         Use footnote style citation markers
  -s, --citationStart int       Start citations from this index (default 1)
  -e, --emitImagesAsLinks       Emit links to included images
  -l, --linkEmitFrequency int   Emit gathered links through the document after this number of paragraphs (default 2)
  -T, --maxConnectTime int      Max connect time (s)
                                 (default 5)
  -t, --maxDownloadTime int     Max download time (s)
                                 (default 10)
  -n, --numberedLinks           Number the links
  -p, --port int                Server port (default 1965)
  -r, --prettyTables            Pretty tables - works with most simple tables
  -c, --serverCert string       serverCert path.
  -k, --serverKey string        serverKey path.
      --unfiltered              Do not filter text/html to text/gemini
  -u, --userAgent string        User agent for HTTP requests
  -v, --version                 Find out what version of Duckling Proxy you're running
  
```

You will need to configure your Gemini client to point to the server when there is a need to access any <code>http://</code> or <code>https://</code> requests.

## Supported clients

The following clients support per-scheme proxies and can be configured to use Duckling proxy.

* [Amfora](https://github.com/makeworld-the-better-one/amfora) - supports per scheme proxies since v1.5.0
* [AV-98](https://tildegit.org/solderpunk/AV-98)  - Merge [pull request #24](https://tildegit.org/solderpunk/AV-98/pulls/24) then use `set http_proxy machine:port` to access. 
* [diohsc](https://repo.or.cz/diohsc.git) - edit diohscrc config file
* [gemget](https://github.com/makeworld-the-better-one/gemget) - use -p option
* [GemiNaut](https://github.com/LukeEmmet/GemiNaut) - since 0.8.8, which also has its own native html to gemini conversion - update in settings
* [Lagrange](https://git.skyjake.fi/skyjake/lagrange) - set proxy in preferences (use 127.0.0.1:port, not localhost:port for localhost)
* [Telescope](https://telescope.omarpolo.com/) - set proxy in the config file add: ```proxy "https" via "gemini://127.0.0.1:1965"```, and similarly for http

Let me know if your client supports per scheme proxies and I'll add it to the list.
package quack

import (
	flag "github.com/spf13/pflag"
)

var (
	citationStart     = flag.IntP("citationStart", "s", 1, "Start citations from this index")
	citationMarkers   = flag.BoolP("citationMarkers", "m", false, "Use footnote style citation markers")
	numberedLinks     = flag.BoolP("numberedLinks", "n", false, "Number the links")
	prettyTables      = flag.BoolP("prettyTables", "r", false, "Pretty tables - works with most simple tables")
	emitImagesAsLinks = flag.BoolP("emitImagesAsLinks", "e", true, "Emit links to included images")
	linkEmitFrequency = flag.IntP("linkEmitFrequency", "l", 2, "Emit gathered links through the document after this number of paragraphs")
	serverCert        = flag.StringP("serverCert", "c", "", "serverCert path. ")
	serverKey         = flag.StringP("serverKey", "k", "", "serverKey path. ")
	userAgent         = flag.StringP("userAgent", "u", "", "User agent for HTTP requests\n")
	maxDownloadTime   = flag.IntP("maxDownloadTime", "t", 10, "Max download time (s)\n")
	maxConnectTime    = flag.IntP("maxConnectTime", "T", 5, "Max connect time (s)\n")
	port              = flag.IntP("port", "p", 1965, "Server port")
	address           = flag.StringP("address", "a", "127.0.0.1", "Bind to address\n")
	unfiltered        = flag.BoolP("unfiltered", "", false, "Do not filter text/html to text/gemini")
	verFlag           = flag.BoolP("version", "v", false, "Find out what version of Duckling Proxy you're running")
)

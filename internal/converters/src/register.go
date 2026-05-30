package converters

import "github.com/MitudruDutta/distill/internal/convert"

// Default returns a Registry wired with the built-in converters. Lower priority
// values are tried first: specific structured parsers (0) beat markup/fenced
// converters (5), which beat the plain-text catch-all (10).
func Default() *convert.Registry {
	reg := &convert.Registry{}

	// Generic catch-all (tried last).
	reg.Register(PlainText{}, 10)

	// Markup and language-tagged config fences.
	reg.Register(HTML{}, 5)
	reg.Register(Fenced{Lang: "xml", Exts: []string{".xml"}, Mimes: []string{"text/xml", "application/xml"}}, 5)
	reg.Register(Fenced{Lang: "yaml", Exts: []string{".yaml", ".yml"}, Mimes: []string{"application/yaml", "text/yaml"}}, 5)
	reg.Register(Fenced{Lang: "toml", Exts: []string{".toml"}}, 5)
	reg.Register(Fenced{Lang: "ini", Exts: []string{".ini", ".cfg", ".conf"}}, 5)

	// Specific structured parsers.
	reg.Register(CSV{}, 0)
	reg.Register(JSON{}, 0)
	reg.Register(Ipynb{}, 0)
	reg.Register(Feed{}, 0)
	reg.Register(DOCX{}, 0)
	reg.Register(PPTX{}, 0)
	reg.Register(ODF{}, 0)
	reg.Register(XLSX{}, 0)
	reg.Register(Image{}, 0)
	reg.Register(Media{}, 0)
	reg.Register(EML{}, 0)
	reg.Register(EPUB{}, 0)
	reg.Register(PDF{}, 0)
	reg.Register(Archive{Reg: reg}, 0)

	return reg
}

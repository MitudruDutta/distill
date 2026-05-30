package converters

import (
	"encoding/xml"
	"errors"
	"io"
	"path"
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/MitudruDutta/distill/internal/convert"
)

// EPUB converts an e-book by reading its OPF spine order and converting each
// XHTML document to Markdown via the HTML pipeline.
type EPUB struct{}

var errNotEPUB = errors.New("distill: not a valid EPUB")

func (EPUB) Accepts(info convert.StreamInfo) bool {
	return info.Extension == ".epub" || info.Mimetype == "application/epub+zip"
}

func (EPUB) Convert(r io.Reader, _ convert.StreamInfo) (convert.Result, error) {
	zr, err := openZip(r)
	if err != nil {
		return convert.Result{}, err
	}

	cdata, err := zipEntry(zr, "META-INF/container.xml")
	if err != nil {
		return convert.Result{}, err
	}
	var container struct {
		Rootfiles []struct {
			FullPath string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	if err := xml.Unmarshal(cdata, &container); err != nil || len(container.Rootfiles) == 0 {
		return convert.Result{}, errNotEPUB
	}
	opfPath := container.Rootfiles[0].FullPath

	opfData, err := zipEntry(zr, opfPath)
	if err != nil {
		return convert.Result{}, err
	}
	var pkg struct {
		Manifest []struct {
			ID   string `xml:"id,attr"`
			Href string `xml:"href,attr"`
		} `xml:"manifest>item"`
		Spine []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"spine>itemref"`
	}
	if err := xml.Unmarshal(opfData, &pkg); err != nil {
		return convert.Result{}, err
	}

	href := make(map[string]string, len(pkg.Manifest))
	for _, it := range pkg.Manifest {
		href[it.ID] = it.Href
	}

	base := path.Dir(opfPath)
	var b strings.Builder
	for _, ref := range pkg.Spine {
		h := href[ref.IDRef]
		if h == "" {
			continue
		}
		data, err := zipEntry(zr, path.Join(base, h))
		if err != nil {
			continue
		}
		md, err := htmltomarkdown.ConvertString(string(data))
		if err != nil {
			continue
		}
		if s := strings.TrimSpace(md); s != "" {
			b.WriteString(s)
			b.WriteString("\n\n")
		}
	}
	return convert.Result{Markdown: strings.TrimSpace(b.String())}, nil
}

package convert

// StreamInfo holds optional metadata about an input stream. Any field may be
// empty depending on how the input was opened.
type StreamInfo struct {
	Mimetype  string
	Extension string // lowercased, includes leading dot, e.g. ".csv"
	Charset   string
	Filename  string
	LocalPath string
	URL       string
}

// Merge returns a copy of si with the non-empty fields of other applied on top.
func (si StreamInfo) Merge(other StreamInfo) StreamInfo {
	if other.Mimetype != "" {
		si.Mimetype = other.Mimetype
	}
	if other.Extension != "" {
		si.Extension = other.Extension
	}
	if other.Charset != "" {
		si.Charset = other.Charset
	}
	if other.Filename != "" {
		si.Filename = other.Filename
	}
	if other.LocalPath != "" {
		si.LocalPath = other.LocalPath
	}
	if other.URL != "" {
		si.URL = other.URL
	}
	return si
}

// Table is a structured table extracted from a document, used by the JSON
// sidecar in later phases.
type Table struct {
	Header []string   `json:"header,omitempty"`
	Rows   [][]string `json:"rows,omitempty"`
}

// Result is the output of a conversion.
type Result struct {
	Markdown string   `json:"markdown"`
	Title    string   `json:"title,omitempty"`
	Headings []string `json:"headings,omitempty"`
	Tables   []Table  `json:"tables,omitempty"`
}

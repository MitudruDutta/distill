package converters

import (
	"strings"
	"testing"

	"github.com/MitudruDutta/distill/internal/convert"
	"github.com/xuri/excelize/v2"
)

func TestXLSXRendersSheetAsTable(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	for cell, val := range map[string]string{"A1": "name", "B1": "age", "A2": "Ada", "B2": "36"} {
		if err := f.SetCellValue("Sheet1", cell, val); err != nil {
			t.Fatal(err)
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	res, err := (XLSX{}).Convert(buf, convert.StreamInfo{Extension: ".xlsx"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## Sheet1", "| name | age |", "| Ada | 36 |"} {
		if !strings.Contains(res.Markdown, want) {
			t.Fatalf("missing %q in:\n%s", want, res.Markdown)
		}
	}
}

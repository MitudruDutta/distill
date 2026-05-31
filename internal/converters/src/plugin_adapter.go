package converters

import (
	"io"
	"strings"

	"github.com/MitudruDutta/distill/internal/convert"
	"github.com/MitudruDutta/distill/internal/plugin"
)

// pluginConverter adapts a discovered plugin.Plugin to convert.Converter.
type pluginConverter struct{ p plugin.Plugin }

func (pc pluginConverter) Accepts(info convert.StreamInfo) bool {
	for _, e := range pc.p.Capabilities.Extensions {
		if strings.EqualFold(info.Extension, e) {
			return true
		}
	}
	for _, m := range pc.p.Capabilities.Mimetypes {
		if info.Mimetype == m {
			return true
		}
	}
	return false
}

func (pc pluginConverter) Convert(r io.Reader, _ convert.StreamInfo) (convert.Result, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return convert.Result{}, err
	}
	out, err := pc.p.Convert(data)
	if err != nil {
		return convert.Result{}, err
	}
	return convert.Result{Markdown: string(out)}, nil
}

// RegisterPlugins loads configured plugin manifests, discovers their
// capabilities, and registers each ahead of the built-in converters (priority
// -10, so a plugin can override a built-in format). It returns the discovered
// plugins and any per-plugin discovery errors (non-fatal). A fatal config-load
// error is returned separately.
func RegisterPlugins(reg *convert.Registry) (plugins []plugin.Plugin, discoverErrs []error, err error) {
	manifests, err := plugin.LoadManifests()
	if err != nil {
		return nil, nil, err
	}
	plugins, discoverErrs = plugin.Discover(manifests)
	for _, p := range plugins {
		reg.Register(pluginConverter{p: p}, -10)
	}
	return plugins, discoverErrs, nil
}

package config

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"github.com/romsar/gonertia/v2"
)

type InertiaConfig struct {
	inertia        *gonertia.Inertia
	isDevelopment  bool
	viteHost       string
}

func NewInertiaConfig(isDevelopment bool, viteHost string) (*InertiaConfig, error) {
	if viteHost == "" {
		viteHost = "localhost"
	}
	// Create template with custom functions
	tmpl := template.New("").Funcs(template.FuncMap{
		"marshal": func(v interface{}) template.JS {
			data, _ := json.Marshal(v)
			return template.JS(data)
		},
	})

	// Parse the app template from file
	templateFile := filepath.Join("src", "web", "templates", "app.html")
	var err error
	tmpl, err = tmpl.ParseFiles(templateFile)
	if err != nil {
		return nil, err
	}
	
	// Get the parsed template by name (should be "app.html")
	tmpl = tmpl.Lookup("app.html")
	if tmpl == nil {
		return nil, fmt.Errorf("template 'app.html' not found")
	}

	// Prepare options for Inertia initialization
	opts := []gonertia.Option{}
	
	// Set version for asset versioning in production
	if !isDevelopment {
		// In production, read version from manifest
		manifestPath := "src/web/static/build/manifest.json"
		if data, err := os.ReadFile(manifestPath); err == nil {
			var manifest map[string]interface{}
			if err := json.Unmarshal(data, &manifest); err == nil {
				// Use the manifest hash as version
				opts = append(opts, gonertia.WithVersion(string(data[:8])))
			}
		}
	}

	// Initialize Inertia with template and options
	i, err := gonertia.NewFromTemplate(tmpl, opts...)
	if err != nil {
		return nil, err
	}

	// Share template data after initialization
	i.ShareTemplateData("isDevelopment", isDevelopment)
	i.ShareTemplateData("viteHost", viteHost)

	return &InertiaConfig{
		inertia:       i,
		isDevelopment: isDevelopment,
		viteHost:      viteHost,
	}, nil
}

func (ic *InertiaConfig) Middleware() func(http.Handler) http.Handler {
	return ic.inertia.Middleware
}

func (ic *InertiaConfig) Render(w http.ResponseWriter, r *http.Request, component string, props gonertia.Props) error {
	return ic.inertia.Render(w, r, component, props)
}

func (ic *InertiaConfig) Location(w http.ResponseWriter, r *http.Request, url string, status ...int) {
	ic.inertia.Location(w, r, url, status...)
}

func (ic *InertiaConfig) Back(w http.ResponseWriter, r *http.Request, status ...int) {
	ic.inertia.Back(w, r, status...)
}

func (ic *InertiaConfig) ShareTemplateData(key string, value interface{}) {
	ic.inertia.ShareTemplateData(key, value)
}

func (ic *InertiaConfig) ShareProp(key string, value interface{}) {
	ic.inertia.ShareProp(key, value)
}

func (ic *InertiaConfig) SharedProps() gonertia.Props {
	return ic.inertia.SharedProps()
}

func (ic *InertiaConfig) FlashMessage(r *http.Request, key string, message interface{}) {
	ic.inertia.ShareProp(key, message)
}
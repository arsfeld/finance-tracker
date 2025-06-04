package handlers

import (
	"html/template"
	"io"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

// TemplateRenderer wraps the template functionality
type TemplateRenderer struct {
	templates *template.Template
}

// NewTemplateRenderer creates a new template renderer
func NewTemplateRenderer() *TemplateRenderer {
	// Load all templates
	templates := template.New("")
	
	// Add custom functions
	templates = templates.Funcs(template.FuncMap{
		"dict": func(values ...interface{}) map[string]interface{} {
			if len(values)%2 != 0 {
				return nil
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil
				}
				dict[key] = values[i+1]
			}
			return dict
		},
	})

	// Parse template files with absolute path
	templateFiles, err := filepath.Glob("src/web/templates/*.html")
	if err != nil {
		log.Error().Err(err).Msg("Failed to find template files")
		// Fallback - try relative path
		templateFiles, err = filepath.Glob("./src/web/templates/*.html")
		if err != nil {
			log.Error().Err(err).Msg("Failed to find template files with relative path")
			return &TemplateRenderer{templates: templates}
		}
	}

	log.Info().Strs("template_files", templateFiles).Msg("Found template files")

	if len(templateFiles) > 0 {
		templates, err = templates.ParseFiles(templateFiles...)
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse template files")
			return &TemplateRenderer{templates: templates}
		}
		log.Info().Msg("Templates parsed successfully")
	} else {
		log.Warn().Msg("No template files found")
	}

	return &TemplateRenderer{
		templates: templates,
	}
}

// Render renders a template with the given data
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}) error {
	// Dashboard, transactions, accounts, analytics templates use base.html
	if name == "dashboard.html" || name == "transactions.html" || name == "accounts.html" || 
	   name == "analytics.html" || name == "account-detail.html" {
		// These templates define {{define "content"}} blocks
		// We need to execute the base template which includes the content
		
		// First, we need to make sure the specific template is parsed
		// The base template expects a "content" block to be defined
		
		// Create a new template set with base + specific template
		tmpl := template.New("base.html")
		tmpl = tmpl.Funcs(template.FuncMap{
			"dict": func(values ...interface{}) map[string]interface{} {
				if len(values)%2 != 0 {
					return nil
				}
				dict := make(map[string]interface{}, len(values)/2)
				for i := 0; i < len(values); i += 2 {
					key, ok := values[i].(string)
					if !ok {
						return nil
					}
					dict[key] = values[i+1]
				}
				return dict
			},
		})
		
		// Parse base template
		_, err := tmpl.ParseFiles("src/web/templates/base.html", "src/web/templates/"+name)
		if err != nil {
			// Try relative path
			_, err = tmpl.ParseFiles("./src/web/templates/base.html", "./src/web/templates/"+name)
			if err != nil {
				log.Error().Err(err).Str("template", name).Msg("Failed to parse templates")
				return err
			}
		}
		
		// Execute base template
		err = tmpl.Execute(w, data)
		if err != nil {
			log.Error().Err(err).Str("template", name).Msg("Failed to execute base template")
		}
		return err
	}
	
	// For standalone templates (login, register), execute directly
	err := t.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Error().Err(err).Str("template", name).Msg("Failed to render template")
	}
	return err
}

// PageData represents common data passed to all pages
type PageData struct {
	Title         string
	User          *UserContext
	Organization  *OrgContext
	CSRFToken     string
	Flash         *Flash
	Data          interface{}
}

// UserContext represents user information in templates
type UserContext struct {
	ID    string
	Email string
}

// OrgContext represents organization information in templates
type OrgContext struct {
	ID   string
	Name string
	Role string
}

// Flash represents flash messages
type Flash struct {
	Type    string // success, error, warning, info
	Message string
}
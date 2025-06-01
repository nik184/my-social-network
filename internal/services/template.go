package services

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"
)

// TemplateData holds data for template rendering
type TemplateData struct {
	PageTitle   string
	CurrentPage string
	PageScript  template.JS
	Data        interface{}
}

// TemplateService manages HTML templates
type TemplateService struct {
	templates   map[string]*template.Template
	templateDir string
}

// NewTemplateService creates a new template service
func NewTemplateService(templateDir string) (*TemplateService, error) {
	service := &TemplateService{
		templates:   make(map[string]*template.Template),
		templateDir: templateDir,
	}
	
	// Load templates
	err := service.loadTemplates(templateDir)
	if err != nil {
		return nil, err
	}
	
	return service, nil
}

// loadTemplates loads all templates from the template directory
func (ts *TemplateService) loadTemplates(templateDir string) error {
	// Base layout with components
	layoutPath := filepath.Join(templateDir, "layouts", "base.html")
	navPath := filepath.Join(templateDir, "components", "navigation.html")
	modalsPath := filepath.Join(templateDir, "components", "modals.html")
	
	// Page templates
	pages := []string{"network", "profile", "friends", "friend-profile"}
	
	for _, page := range pages {
		pagePath := filepath.Join(templateDir, "pages", page+".html")
		
		tmpl, err := template.ParseFiles(layoutPath, navPath, modalsPath, pagePath)
		if err != nil {
			return err
		}
		
		ts.templates[page] = tmpl
	}
	
	return nil
}

// loadPageScript loads the JavaScript for a specific page
func (ts *TemplateService) loadPageScript(page string) (template.JS, error) {
	scriptPath := filepath.Join(ts.templateDir, "scripts", page+".js")
	content, err := ioutil.ReadFile(scriptPath)
	if err != nil {
		return "", err
	}
	return template.JS(content), nil
}

// RenderPage renders a page template with data
func (ts *TemplateService) RenderPage(w http.ResponseWriter, page string, data TemplateData) error {
	tmpl, exists := ts.templates[page]
	if !exists {
		http.Error(w, "Template not found", http.StatusNotFound)
		return nil
	}
	
	// Load page script if not already provided
	if data.PageScript == "" {
		script, err := ts.loadPageScript(page)
		if err == nil {
			data.PageScript = script
		}
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.ExecuteTemplate(w, "base.html", data)
}
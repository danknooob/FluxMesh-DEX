package swagger

import (
	"net/http"
	"os"
	"path/filepath"
)

// Handler serves the Swagger UI HTML page at /docs and the spec at /docs/swagger.yaml.
type Handler struct {
	specPath string
}

func New(specPath string) *Handler {
	return &Handler{specPath: specPath}
}

// UI serves an HTML page that loads Swagger UI from CDN pointing to the local spec.
func (h *Handler) UI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerHTML))
}

// Spec serves the raw OpenAPI YAML file.
func (h *Handler) Spec(w http.ResponseWriter, r *http.Request) {
	abs, err := filepath.Abs(h.specPath)
	if err != nil {
		http.Error(w, "spec not found", http.StatusInternalServerError)
		return
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		http.Error(w, "spec not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(data)
}

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>FluxMesh DEX — API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; background: #fafafa; }
    #swagger-ui .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: '/docs/swagger.yaml',
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: 'BaseLayout',
    });
  </script>
</body>
</html>`

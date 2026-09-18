package server

import (
	"net/http"
)

const openAPISpecJSON = `{
  "openapi": "3.0.3",
  "info": {
    "title": "StreamGO API",
    "description": "High-Performance Media Streaming and Telegram Bot Backend written in Go",
    "version": "1.0.0"
  },
  "servers": [
    {
      "url": "/",
      "description": "Current Server"
    }
  ],
  "paths": {
    "/": {
      "get": {
        "tags": ["System"],
        "summary": "Root Service Info",
        "description": "Returns basic service metadata, status, and running version.",
        "responses": {
          "200": {
            "description": "Successful Response",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "service": { "type": "string", "example": "StreamGO" },
                    "status": { "type": "string", "example": "running" },
                    "version": { "type": "string", "example": "1.0.0" }
                  }
                }
              }
            }
          }
        }
      }
    },
    "/health": {
      "get": {
        "tags": ["System"],
        "summary": "Health Check",
        "description": "Returns server uptime, current time, and real-time MongoDB connectivity status.",
        "responses": {
          "200": {
            "description": "Health status response",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "status": { "type": "string", "example": "ok" },
                    "database": { "type": "string", "example": "healthy" },
                    "uptime": { "type": "string", "example": "1h23m45s" },
                    "time": { "type": "string", "example": "2026-09-18T11:22:33Z" }
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}`

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>StreamGO - Swagger UI</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
  <link rel="icon" type="image/png" href="https://fastapi.tiangolo.com/img/favicon.png">
  <style>
    html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
    .topbar { display: none !important; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
  <script>
    window.onload = function() {
      window.ui = SwaggerUIBundle({
        url: "/openapi.json",
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset
        ],
        plugins: [
          SwaggerUIBundle.plugins.DownloadUrl
        ],
        layout: "StandaloneLayout",
        persistAuthorization: true,
        displayRequestDuration: true
      });
    };
  </script>
</body>
</html>`

const redocHTML = `<!DOCTYPE html>
<html>
<head>
  <title>StreamGO - ReDoc</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
  <style>body { margin: 0; padding: 0; }</style>
</head>
<body>
  <redoc spec-url='/openapi.json'></redoc>
  <script src="https://cdn.jsdelivr.net/npm/redoc@next/bundles/redoc.standalone.js"></script>
</body>
</html>`

func (s *Server) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(openAPISpecJSON))
}

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(swaggerUIHTML))
}

func (s *Server) handleRedoc(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(redocHTML))
}

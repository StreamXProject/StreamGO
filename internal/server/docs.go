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
          "200": { "description": "Successful Response" }
        }
      }
    },
    "/health": {
      "get": {
        "tags": ["System"],
        "summary": "Health Check",
        "description": "Returns server uptime, current time, Telegram client state, and real-time MongoDB connectivity status.",
        "responses": {
          "200": { "description": "Health status response" }
        }
      }
    },
    "/tracks": {
      "get": {
        "tags": ["Tracks"],
        "summary": "Browse / List Tracks",
        "description": "Paginated track list with sorting, topic filtering, and channel filtering.",
        "parameters": [
          { "name": "page", "in": "query", "schema": { "type": "integer", "default": 1 } },
          { "name": "per_page", "in": "query", "schema": { "type": "integer", "default": 20 } },
          { "name": "sort", "in": "query", "schema": { "type": "string", "enum": ["updated_at", "play_count", "created_at"] } },
          { "name": "topic", "in": "query", "schema": { "type": "string" } },
          { "name": "channel_id", "in": "query", "schema": { "type": "integer" } }
        ],
        "responses": {
          "200": { "description": "Paginated list of browse items" }
        }
      }
    },
    "/tracks/{id}": {
      "get": {
        "tags": ["Tracks"],
        "summary": "Get Track by ID",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "Full track entity" },
          "404": { "description": "Track not found" }
        }
      }
    },
    "/tracks/random": {
      "get": {
        "tags": ["Tracks"],
        "summary": "Get Random Tracks",
        "parameters": [
          { "name": "limit", "in": "query", "schema": { "type": "integer", "default": 20 } },
          { "name": "channel_id", "in": "query", "schema": { "type": "integer" } }
        ],
        "responses": {
          "200": { "description": "Randomized selection of tracks" }
        }
      }
    },
    "/search": {
      "get": {
        "tags": ["Tracks"],
        "summary": "Search Tracks",
        "description": "Search tracks by title, artist, performer, or album.",
        "parameters": [
          { "name": "q", "in": "query", "required": true, "schema": { "type": "string" } },
          { "name": "limit", "in": "query", "schema": { "type": "integer", "default": 50 } }
        ],
        "responses": {
          "200": { "description": "Search results matching query" }
        }
      }
    },
    "/topics": {
      "get": {
        "tags": ["Topics"],
        "summary": "List Unique Topics",
        "description": "Aggregates unique topic names with total track counts and artwork.",
        "parameters": [
          { "name": "limit", "in": "query", "schema": { "type": "integer", "default": 100 } }
        ],
        "responses": {
          "200": { "description": "Aggregated topic list" }
        }
      }
    },
    "/topics/{name}/tracks": {
      "get": {
        "tags": ["Topics"],
        "summary": "Browse Topic Tracks",
        "parameters": [
          { "name": "name", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "page", "in": "query", "schema": { "type": "integer", "default": 1 } },
          { "name": "per_page", "in": "query", "schema": { "type": "integer", "default": 20 } }
        ],
        "responses": {
          "200": { "description": "Paginated tracks belonging to topic" }
        }
      }
    },
    "/channelids": {
      "get": {
        "tags": ["Topics"],
        "summary": "Get Indexed Channel IDs",
        "responses": {
          "200": { "description": "List of indexed Telegram source channels" }
        }
      }
    },
    "/tracks/{id}/stream": {
      "get": {
        "tags": ["Streaming"],
        "summary": "Stream Track Audio",
        "description": "Streams raw audio byte stream supporting HTTP 206 Range headers.",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "Range", "in": "header", "schema": { "type": "string", "example": "bytes=0-1048575" } }
        ],
        "responses": {
          "200": { "description": "Complete audio file" },
          "206": { "description": "Partial byte range of audio" },
          "416": { "description": "Range Not Satisfiable" }
        }
      },
      "head": {
        "tags": ["Streaming"],
        "summary": "Inspect Audio Headers",
        "description": "Returns audio headers (Content-Length, Accept-Ranges, Content-Type) without body.",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "Audio headers available" }
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

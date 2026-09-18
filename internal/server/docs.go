package server

import (
	"net/http"
)

const openAPISpecJSON = `{
  "openapi": "3.0.3",
  "info": {
    "title": "StreamGO API",
    "description": "High-Performance Media Streaming, Discovery, and Telegram MiniApp Backend written in Go",
    "version": "1.0.0"
  },
  "servers": [
    {
      "url": "/",
      "description": "Current Server"
    }
  ],
  "components": {
    "securitySchemes": {
      "bearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT / v1.<b64url>.<sig>"
      }
    }
  },
  "paths": {
    "/": {
      "get": {
        "tags": ["System"],
        "summary": "Root Service Info",
        "description": "Returns basic service metadata, status, and running version.",
        "responses": { "200": { "description": "Successful Response" } }
      }
    },
    "/health": {
      "get": {
        "tags": ["System"],
        "summary": "Health Check",
        "description": "Returns server uptime, current time, Telegram client state, and real-time MongoDB connectivity status.",
        "responses": { "200": { "description": "Health status response" } }
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
        "responses": { "200": { "description": "Paginated list of browse items" } }
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
        "responses": { "200": { "description": "Randomized selection of tracks" } }
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
        "responses": { "200": { "description": "Search results matching query" } }
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
        "responses": { "200": { "description": "Aggregated topic list" } }
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
        "responses": { "200": { "description": "Paginated tracks belonging to topic" } }
      }
    },
    "/channelids": {
      "get": {
        "tags": ["Topics"],
        "summary": "Get Indexed Channel IDs",
        "responses": { "200": { "description": "List of indexed Telegram source channels" } }
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
        "responses": { "200": { "description": "Audio headers available" } }
      }
    },
    "/artists": {
      "get": {
        "tags": ["Artists"],
        "summary": "List Artists",
        "description": "Paginated directory of artists indexed from indexed tracks.",
        "parameters": [
          { "name": "page", "in": "query", "schema": { "type": "integer", "default": 1 } },
          { "name": "per_page", "in": "query", "schema": { "type": "integer", "default": 20 } }
        ],
        "responses": { "200": { "description": "Paginated list of artists" } }
      }
    },
    "/artists/{id}": {
      "get": {
        "tags": ["Artists"],
        "summary": "Get Artist Details",
        "description": "Returns artist profile, follower count, top tracks, and discography.",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "Artist detail with top tracks and albums" },
          "404": { "description": "Artist not found" }
        }
      }
    },
    "/artists/{id}/tracks": {
      "get": {
        "tags": ["Artists"],
        "summary": "Get Artist Tracks",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "limit", "in": "query", "schema": { "type": "integer", "default": 50 } }
        ],
        "responses": { "200": { "description": "Top tracks for artist" } }
      }
    },
    "/albums": {
      "get": {
        "tags": ["Albums"],
        "summary": "List Albums",
        "parameters": [
          { "name": "page", "in": "query", "schema": { "type": "integer", "default": 1 } },
          { "name": "per_page", "in": "query", "schema": { "type": "integer", "default": 20 } },
          { "name": "artist", "in": "query", "schema": { "type": "string" } }
        ],
        "responses": { "200": { "description": "Paginated album list" } }
      }
    },
    "/albums/{id}": {
      "get": {
        "tags": ["Albums"],
        "summary": "Get Album Details",
        "description": "Returns album metadata and complete tracklist.",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "Album details and ordered tracks" },
          "404": { "description": "Album not found" }
        }
      }
    },
    "/albums/{id}/tracks": {
      "get": {
        "tags": ["Albums"],
        "summary": "Get Album Tracks",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": { "200": { "description": "Album tracks" } }
      }
    },
    "/favourites": {
      "get": {
        "tags": ["Favourites"],
        "summary": "List User Favourites",
        "security": [{ "bearerAuth": [] }],
        "parameters": [
          { "name": "page", "in": "query", "schema": { "type": "integer", "default": 1 } },
          { "name": "limit", "in": "query", "schema": { "type": "integer", "default": 20 } }
        ],
        "responses": { "200": { "description": "User favorite tracks" } }
      },
      "post": {
        "tags": ["Favourites"],
        "summary": "Add Track to Favourites",
        "security": [{ "bearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["track_id"],
                "properties": { "track_id": { "type": "string" } }
              }
            }
          }
        },
        "responses": { "200": { "description": "Favourite saved" } }
      }
    },
    "/favourites/{id}": {
      "delete": {
        "tags": ["Favourites"],
        "summary": "Remove Track from Favourites",
        "security": [{ "bearerAuth": [] }],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": { "200": { "description": "Removed from favourites" } }
      }
    },
    "/favourites/ids": {
      "get": {
        "tags": ["Favourites"],
        "summary": "Get Favourite Track IDs",
        "security": [{ "bearerAuth": [] }],
        "responses": { "200": { "description": "List of liked track IDs" } }
      }
    },
    "/history": {
      "get": {
        "tags": ["Favourites"],
        "summary": "Get Listening History",
        "security": [{ "bearerAuth": [] }],
        "parameters": [
          { "name": "limit", "in": "query", "schema": { "type": "integer", "default": 50 } }
        ],
        "responses": { "200": { "description": "Recent listening history" } }
      }
    },
    "/listening-events": {
      "post": {
        "tags": ["Favourites"],
        "summary": "Record Playback Telemetry Events",
        "security": [{ "bearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": { "events": { "type": "array", "items": { "type": "object" } } }
              }
            }
          }
        },
        "responses": { "200": { "description": "Events recorded" } }
      }
    },
    "/playlists": {
      "get": {
        "tags": ["Playlists"],
        "summary": "List User Playlists",
        "security": [{ "bearerAuth": [] }],
        "responses": { "200": { "description": "List of playlists owned by user" } }
      },
      "post": {
        "tags": ["Playlists"],
        "summary": "Create Custom Playlist",
        "security": [{ "bearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["title"],
                "properties": {
                  "title": { "type": "string" },
                  "description": { "type": "string" },
                  "cover_url": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": { "200": { "description": "Created playlist" } }
      }
    },
    "/playlists/{id}": {
      "get": {
        "tags": ["Playlists"],
        "summary": "Get Playlist by ID",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": { "200": { "description": "Playlist detail with tracks" } }
      },
      "patch": {
        "tags": ["Playlists"],
        "summary": "Update Playlist Metadata",
        "security": [{ "bearerAuth": [] }],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": { "200": { "description": "Updated playlist" } }
      },
      "delete": {
        "tags": ["Playlists"],
        "summary": "Delete Playlist",
        "security": [{ "bearerAuth": [] }],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": { "200": { "description": "Playlist deleted" } }
      }
    },
    "/playlists/{id}/tracks": {
      "get": {
        "tags": ["Playlists"],
        "summary": "Get Playlist Tracks",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": { "200": { "description": "Ordered tracks in playlist" } }
      },
      "post": {
        "tags": ["Playlists"],
        "summary": "Add Tracks to Playlist",
        "security": [{ "bearerAuth": [] }],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "track_id": { "type": "string" },
                  "track_ids": { "type": "array", "items": { "type": "string" } }
                }
              }
            }
          }
        },
        "responses": { "200": { "description": "Tracks added" } }
      }
    },
    "/playlists/{id}/reorder": {
      "post": {
        "tags": ["Playlists"],
        "summary": "Reorder Playlist Tracks",
        "security": [{ "bearerAuth": [] }],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["track_ids"],
                "properties": { "track_ids": { "type": "array", "items": { "type": "string" } } }
              }
            }
          }
        },
        "responses": { "200": { "description": "Playlist tracks reordered" } }
      }
    },
    "/auth/telegram": {
      "post": {
        "tags": ["Authentication"],
        "summary": "Telegram WebApp Login",
        "description": "Verifies Telegram initData HMAC-SHA256 signature and returns user session token.",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["init_data"],
                "properties": { "init_data": { "type": "string" } }
              }
            },
            "application/x-www-form-urlencoded": {
              "schema": {
                "type": "object",
                "required": ["init_data"],
                "properties": { "init_data": { "type": "string" } }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "Authentication successful" },
          "401": { "description": "Invalid Telegram signature" }
        }
      }
    },
    "/auth/telegram/widget": {
      "post": {
        "tags": ["Authentication"],
        "summary": "Telegram Widget Login",
        "description": "Authenticates user using Telegram OAuth Login Widget callback data.",
        "responses": { "200": { "description": "Authentication successful" } }
      }
    },
    "/auth/me": {
      "get": {
        "tags": ["Authentication"],
        "summary": "Current User Profile",
        "security": [{ "bearerAuth": [] }],
        "responses": {
          "200": { "description": "Current user profile" },
          "401": { "description": "Unauthorized" }
        }
      }
    },
    "/cover/{id}": {
      "get": {
        "tags": ["Media"],
        "summary": "Track Cover Redirect",
        "description": "Redirects to the best available album artwork or thumbnail for the track.",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": { "302": { "description": "Redirect to cover image URL" } }
      }
    },
    "/tracks/{id}/lyrics": {
      "get": {
        "tags": ["Media"],
        "summary": "Track Lyrics",
        "description": "Returns synchronized or plain lyrics for a track in JSON or text/plain.",
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "format", "in": "query", "schema": { "type": "string", "enum": ["plain", "json"] } }
        ],
        "responses": {
          "200": { "description": "Lyrics content" },
          "404": { "description": "Lyrics not found" }
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

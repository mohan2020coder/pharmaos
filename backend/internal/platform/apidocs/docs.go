package apidocs

import _ "embed"

//go:embed openapi.yaml
var spec []byte

func Spec() []byte { return spec }

func UI() []byte {
	return []byte(`<!doctype html><html><head><meta charset="utf-8"><title>PharmaOS API</title>
<style>body{font-family:system-ui;margin:3rem;color:#172b4d}code{background:#f4f5f7;padding:.2rem .4rem}</style></head>
<body><h1>PharmaOS API v1</h1><p>OpenAPI document: <a href="/docs/openapi.yaml"><code>/docs/openapi.yaml</code></a></p>
<p>Import the specification into Swagger UI, Redoc, or your API client.</p></body></html>`)
}

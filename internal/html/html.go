package html

import (
	"html/template"
	"net/http"

	"github.com/johanbrandhorst/reload"
)

const templateName = "index.html"

var indexTemplate = template.Must(template.New(templateName).Parse(
	`<!doctype html>
<html lang="en" class="h-full bg-gray-100">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta http-equiv="X-UA-Compatible" content="ie=edge" />
    <title>Trail tools</title>
    <link rel="stylesheet" href="index.css" />
    <link rel="icon" href="favicon.svg" />
    <script src="index.js" defer></script>
  </head>
  <body class="h-full">
    <div id="root" class="h-full"></div>
  </body>
  {{- if .Watch }}
  {{ .ReloadScript }}
  {{- end }}
</html>
`))

func ServeIndexHTML(watch bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		indexTemplate.ExecuteTemplate(w, templateName, map[string]any{
			"Watch":        watch,
			"ReloadScript": reload.Script,
		})
	})
}

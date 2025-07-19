// copied from github.com/function61/passitron
package f61ui

import (
	"net/http"
	"strings"
)

func IndexHTMLHandler(assetsPath string) http.HandlerFunc {
	index := strings.ReplaceAll(template, "[$assets_path]", assetsPath)

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write([]byte(index)); err != nil {
			panic(err)
		}
	}
}

const template = `<!doctype html>

<html>
<head>
	<meta charset="UTF-8" />
	<meta name="google" content="notranslate" />
	<title></title>
	<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/3.4.0/css/bootstrap.min.css" />
	<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/3.4.0/css/bootstrap-theme.min.css" />
	<link rel="stylesheet" href="[$assets_path]/f61ui/style.css" />
	<script defer src="[$assets_path]/build.js"></script>
	<script>
	// defer does not support inline scripts, so hook an event that guarantees
	// that previous deferred scripts are executed. https://stackoverflow.com/a/41395202
	window.addEventListener('DOMContentLoaded', function() {
		main.main(document.getElementById('app'), {
			assetsDir: '[$assets_path]/f61ui',
		});
	});
	</script>
</head>
<body>

<div id="app" class="container"></div>

<div id="appDialogs"></div>
	
</body>
</html>
`

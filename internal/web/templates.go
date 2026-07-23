package web

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var StaticFS embed.FS

var funcs = template.FuncMap{
	"yen": func(minor int64) string {
		neg := minor < 0
		v := minor
		if neg {
			v = -v
		}
		whole := v / 100
		s := formatThousands(whole)
		if neg {
			return "-¥" + s
		}
		return "¥" + s
	},
	"catSelected": func(catID *int64, id int64) bool {
		return catID != nil && *catID == id
	},
	"dict": func(pairs ...any) (map[string]any, error) {
		if len(pairs)%2 != 0 {
			return nil, fmt.Errorf("dict: odd number of arguments")
		}
		m := map[string]any{}
		for i := 0; i < len(pairs); i += 2 {
			key, ok := pairs[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict: keys must be strings")
			}
			m[key] = pairs[i+1]
		}
		return m, nil
	},
}

func formatThousands(n int64) string {
	s := fmt.Sprintf("%d", n)
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	var out []byte
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

var tmpl = template.Must(template.New("").Funcs(funcs).ParseFS(templateFS, "templates/*.html"))

func Render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

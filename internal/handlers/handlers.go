package handlers

import (
	"encoding/json"
	"html/template"

	"kakeibo/internal/config"
	"kakeibo/internal/repo"
)

type Handlers struct {
	Store *repo.Store
	Cfg   config.Config
}

func toJS(v any) template.JS {
	b, _ := json.Marshal(v)
	return template.JS(b)
}

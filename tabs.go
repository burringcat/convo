package main

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"net/http"
)
type Tab struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	Nodes []string `json:"nodes"`
}
func (t *Tab) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

var tabs []Tab
func ReadTabs() {
	for slug, tabConfig := range Config.Tabs {
		tabs = append(tabs, Tab{slug, tabConfig.Name, tabConfig.Nodes})
	}
}
func InitTabs() {
	sql := "INSERT IGNORE INTO tabs (name, slug) VALUES (?, ?)"
	nodeSql := "INSERT IGNORE INTO nodes (tab_id, slug) VALUES (?, ?)"
	for _, tab := range tabs {
		result, err := DB.db.Exec(sql, tab.Name, tab.Slug)
		if err != nil {
			panic(err)
		}
		tabId, err := result.LastInsertId()
		if err != nil {
			panic(err)
		}
		for _, node := range tab.Nodes {
			_, err := DB.db.Exec(nodeSql, tabId, node)
			if err != nil {
				panic(err)
			}
		}
	}

}
func TabsRouter() chi.Router {
	r := chi.NewRouter()
	r.Get("/", ListTabs)
	return r
}
func ListTabs(w http.ResponseWriter, r *http.Request) {
	var renderers []render.Renderer
	for i, _ := range tabs {
		renderers = append(renderers, &tabs[i])
	}
	render.RenderList(w, r, renderers)
	return
}
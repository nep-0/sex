package sex

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"sort"
)

type RouteInfo struct {
	Path         string            `json:"path"`
	ExposeType   string            `json:"expose_type"`
	ResourceType string            `json:"resource_type"`
	Source       string            `json:"source"`
	Headers      map[string]string `json:"headers"`
	Watch        bool              `json:"watch"`
	Timeout      string            `json:"timeout"`
}

func RegisterUI(mux *http.ServeMux, cfg ParsedConfig) error {
	sub, err := fs.Sub(uiFS, "web/ui")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))
	mux.HandleFunc("/api/routes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		routes := make([]RouteInfo, 0, len(cfg.Routes))
		for _, route := range cfg.Routes {
			routes = append(routes, RouteInfo{
				Path:         route.Path,
				ExposeType:   route.ExposeType,
				ResourceType: route.ResourceType,
				Source:       route.Source,
				Headers:      route.Headers,
				Watch:        route.Watch,
				Timeout:      route.Timeout.String(),
			})
		}
		sort.Slice(routes, func(i, j int) bool {
			return routes[i].Path < routes[j].Path
		})
		_ = json.NewEncoder(w).Encode(routes)
	})
	return nil
}

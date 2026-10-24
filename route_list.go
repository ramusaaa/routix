package routix

import (
	"encoding/json"
	"io"
	"os"
	"sort"
)

const RouteListFileEnv = "ROUTIX_ROUTE_LIST_FILE"

func (r *Router) WriteRoutes(w io.Writer) error {
	routes := r.Routes()
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})
	return json.NewEncoder(w).Encode(routes)
}

func (r *Router) writeRouteList() (bool, error) {
	path := os.Getenv(RouteListFileEnv)
	if path == "" {
		return false, nil
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return true, err
	}
	if err := r.WriteRoutes(file); err != nil {
		file.Close()
		return true, err
	}
	return true, file.Close()
}

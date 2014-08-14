package api

import "github.com/qor/qor/resource"

type Resource struct {
	Name string
	resource.Resource
	indexAttrs []string
	showAttrs  []string
}

func (r *Resource) IndexAttrs(columns ...string) {
	r.indexAttrs = columns
}

func (r *Resource) ShowAttrs(columns ...string) {
	r.showAttrs = columns
}

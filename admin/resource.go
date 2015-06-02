package admin

import (
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/qor/qor"
	"github.com/qor/qor/resource"
	"github.com/qor/qor/roles"
	"github.com/qor/qor/utils"
)

type Resource struct {
	resource.Resource
	admin         *Admin
	Config        *Config
	Metas         []*Meta
	actions       []*Action
	scopes        map[string]*Scope
	filters       map[string]*Filter
	searchAttrs   []string
	indexAttrs    []string
	newAttrs      []string
	editAttrs     []string
	showAttrs     []string
	cachedMetas   *map[string][]*Meta
	SearchHandler func(keyword string, context *qor.Context) *gorm.DB
}

func (res *Resource) Meta(meta *Meta) {
	meta.base = res
	meta.updateMeta()
	res.Metas = append(res.Metas, meta)
}

func (res Resource) GetAdmin() *Admin {
	return res.admin
}

func (res Resource) ToParam() string {
	return utils.ToParamString(res.Name)
}

func (res *Resource) ConvertObjectToMap(context qor.Contextor, value interface{}, metas []*Meta) interface{} {
	var metaors []resource.Metaor
	for _, meta := range metas {
		metaors = append(metaors, meta)
	}
	return resource.ConvertObjectToMap(context, value, metaors)
}

func (res *Resource) Decode(contextor qor.Contextor, value interface{}) (errs []error) {
	return resource.Decode(contextor, value, res)
}

func (res *Resource) IndexAttrs(columns ...string) {
	res.indexAttrs = columns
}

func (res *Resource) NewAttrs(columns ...string) {
	res.newAttrs = columns
}

func (res *Resource) EditAttrs(columns ...string) {
	res.editAttrs = columns
}

func (res *Resource) ShowAttrs(columns ...string) {
	res.showAttrs = columns
}

func (res *Resource) SearchAttrs(columns ...string) []string {
	if len(columns) > 0 {
		res.searchAttrs = columns
		res.SearchHandler = func(keyword string, context *qor.Context) *gorm.DB {
			db := context.GetDB()
			var conditions []string
			var keywords []interface{}
			scope := db.NewScope(res.Value)

			for _, column := range columns {
				if field, ok := scope.FieldByName(column); ok {
					conditions = append(conditions, fmt.Sprintf("upper(%v) like upper(?)", scope.Quote(field.DBName)))
				}

				keywords = append(keywords, "%"+keyword+"%")
			}
			return context.GetDB().Where(strings.Join(conditions, " OR "), keywords...)
		}
	}
	return res.searchAttrs
}

func (res *Resource) getCachedMetas(cacheKey string, fc func() []resource.Metaor) []*Meta {
	if res.cachedMetas == nil {
		res.cachedMetas = &map[string][]*Meta{}
	}

	if values, ok := (*res.cachedMetas)[cacheKey]; ok {
		return values
	} else {
		values := fc()
		var metas []*Meta
		for _, value := range values {
			metas = append(metas, value.(*Meta))
		}
		(*res.cachedMetas)[cacheKey] = metas
		return metas
	}
}

func (res *Resource) GetMetas(_attrs ...[]string) []resource.Metaor {
	var attrs []string
	for _, value := range _attrs {
		if value != nil {
			attrs = value
			break
		}
	}

	if attrs == nil {
		scope := &gorm.Scope{Value: res.Value}
		structFields := scope.GetModelStruct().StructFields
		attrs = []string{}

	StructFields:
		for _, field := range structFields {
			for _, meta := range res.Metas {
				if field.Name == meta.Alias {
					attrs = append(attrs, meta.Name)
					continue StructFields
				}
			}

			if field.IsForeignKey {
				continue
			}

			for _, value := range []string{"CreatedAt", "UpdatedAt", "DeletedAt"} {
				if value == field.Name {
					continue StructFields
				}
			}

			attrs = append(attrs, field.Name)
		}
	}

	primaryKey := res.PrimaryFieldName()

	metas := []resource.Metaor{}
	for _, attr := range attrs {
		var meta *Meta
		for _, m := range res.Metas {
			if m.GetName() == attr {
				meta = m
				break
			}
		}

		if meta == nil {
			meta = &Meta{}
			meta.Name = attr
			meta.base = res
			if attr == primaryKey {
				meta.Type = "hidden"
			}
			meta.updateMeta()
		}

		metas = append(metas, meta)
	}

	return metas
}

func (res *Resource) IndexMetas() []*Meta {
	return res.getCachedMetas("index_metas", func() []resource.Metaor {
		return res.GetMetas(res.indexAttrs, res.showAttrs)
	})
}

func (res *Resource) NewMetas() []*Meta {
	return res.getCachedMetas("new_metas", func() []resource.Metaor {
		return res.GetMetas(res.newAttrs, res.editAttrs)
	})
}

func (res *Resource) EditMetas() []*Meta {
	return res.getCachedMetas("edit_metas", func() []resource.Metaor {
		return res.GetMetas(res.editAttrs)
	})
}

func (res *Resource) ShowMetas() []*Meta {
	return res.getCachedMetas("show_metas", func() []resource.Metaor {
		return res.GetMetas(res.showAttrs, res.editAttrs)
	})
}

func (res *Resource) AllMetas() []*Meta {
	return res.getCachedMetas("all_metas", func() []resource.Metaor {
		return res.GetMetas()
	})
}

func (res *Resource) AllowedMetas(attrs []*Meta, context *Context, roles ...roles.PermissionMode) []*Meta {
	var metas = []*Meta{}
	for _, meta := range attrs {
		for _, role := range roles {
			if meta.HasPermission(role, context.Context) {
				metas = append(metas, meta)
				break
			}
		}
	}
	return metas
}

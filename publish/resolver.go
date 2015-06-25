package publish

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

type Resolver struct {
	Records      []interface{}
	Dependencies map[string]*Dependency
	DB           *Publish
}

type Dependency struct {
	Type        reflect.Type
	PrimaryKeys []string
}

func IncludeValue(value string, values []string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func (resolver *Resolver) AddDependency(dependency *Dependency) {
	name := dependency.Type.String()
	var newPrimaryKeys []string

	// append primary keys to dependency
	if dep, ok := resolver.Dependencies[name]; ok {
		for _, primaryKey := range dependency.PrimaryKeys {
			if !IncludeValue(primaryKey, dep.PrimaryKeys) {
				newPrimaryKeys = append(newPrimaryKeys, primaryKey)
				dep.PrimaryKeys = append(dep.PrimaryKeys, primaryKey)
			}
		}
	} else {
		resolver.Dependencies[name] = dependency
		newPrimaryKeys = dependency.PrimaryKeys
	}

	if len(newPrimaryKeys) > 0 {
		resolver.GetDependencies(dependency, newPrimaryKeys)
	}
}

func (resolver *Resolver) GetDependencies(dependency *Dependency, primaryKeys []string) {
	value := reflect.New(dependency.Type)
	fromScope := resolver.DB.DB.NewScope(value.Interface())

	draftDB := resolver.DB.DraftDB().Unscoped()
	for _, field := range fromScope.Fields() {
		if relationship := field.Relationship; relationship != nil {
			if IsPublishableModel(field.Field.Interface()) {
				toType := modelType(field.Field.Interface())
				toScope := draftDB.NewScope(reflect.New(toType).Interface())
				draftTable := DraftTableName(toScope.TableName())
				var dependencyKeys []string
				var rows *sql.Rows
				var err error

				if relationship.Kind == "belongs_to" || relationship.Kind == "has_many" {
					sql := fmt.Sprintf("%v IN (?) and publish_status = ?", relationship.ForeignDBName)
					rows, err = draftDB.Table(draftTable).Select(toScope.PrimaryKey()).Where(sql, primaryKeys, DIRTY).Rows()
				} else if relationship.Kind == "has_one" {
					fromTable := fromScope.TableName()
					fromPrimaryKey := fromScope.PrimaryKey()
					toTable := toScope.TableName()
					toPrimaryKey := toScope.PrimaryKey()

					sql := fmt.Sprintf("%v.%v IN (select %v.%v from %v where %v.%v IN (?)) and %v.publish_status = ?",
						toTable, toPrimaryKey, fromTable, relationship.ForeignDBName, fromTable, fromTable, fromPrimaryKey, toTable)

					rows, err = draftDB.Table(draftTable).Select(toTable+"."+toPrimaryKey).Where(sql, primaryKeys, DIRTY).Rows()
				} else if relationship.Kind == "many_to_many" {
					relationship.JoinTableHandler.Table(draftDB)
				}

				if rows != nil && err == nil {
					for rows.Next() {
						var primaryKey interface{}
						rows.Scan(&primaryKey)
						dependencyKeys = append(dependencyKeys, fmt.Sprintf("%v", primaryKey))
					}

					dependency := Dependency{Type: toType, PrimaryKeys: dependencyKeys}
					resolver.AddDependency(&dependency)
				}
			}
		}
	}
}

func (resolver *Resolver) GenerateDependencies() {
	for _, record := range resolver.Records {
		if IsPublishableModel(record) {
			scope := resolver.DB.DB.NewScope(record)
			dependency := Dependency{Type: modelType(record), PrimaryKeys: []string{fmt.Sprintf("%v", scope.PrimaryKeyValue())}}
			resolver.AddDependency(&dependency)
		}
	}
}

func (resolver *Resolver) Publish() error {
	resolver.GenerateDependencies()
	tx := resolver.DB.DB.Begin()

	for _, dependency := range resolver.Dependencies {
		value := reflect.New(dependency.Type).Elem()
		productionScope := resolver.DB.ProductionDB().NewScope(value.Addr().Interface())
		productionTable := productionScope.TableName()
		primaryKey := productionScope.PrimaryKey()
		draftTable := DraftTableName(productionTable)

		var columns []string
		for _, field := range productionScope.Fields() {
			if field.IsNormal {
				columns = append(columns, field.DBName)
			}
		}

		var productionColumns []string
		for _, column := range columns {
			productionColumns = append(productionColumns, fmt.Sprintf("%v.%v", productionTable, column))
		}

		var draftColumns []string
		for _, column := range columns {
			draftColumns = append(draftColumns, fmt.Sprintf("%v.%v", draftTable, column))
		}

		if len(dependency.PrimaryKeys) > 0 {
			deleteSql := fmt.Sprintf("DELETE FROM %v WHERE %v.%v IN (?)", productionTable, productionTable, primaryKey)
			tx.Exec(deleteSql, dependency.PrimaryKeys)

			publishSql := fmt.Sprintf("INSERT INTO %v (%v) SELECT %v FROM %v WHERE %v.%v IN (?)",
				productionTable, strings.Join(productionColumns, " ,"), strings.Join(draftColumns, " ,"),
				draftTable, draftTable, primaryKey)
			tx.Exec(publishSql, dependency.PrimaryKeys)

			updateStateSql := fmt.Sprintf("UPDATE %v SET publish_status = ? WHERE %v.%v IN (?)", draftTable, draftTable, primaryKey)
			tx.Exec(updateStateSql, bool(PUBLISHED), dependency.PrimaryKeys)
		}
	}

	if err := tx.Error; err == nil {
		return tx.Commit().Error
	} else {
		tx.Rollback()
		return err
	}
}

func (resolver *Resolver) Discard() error {
	resolver.GenerateDependencies()
	tx := resolver.DB.DB.Begin()

	for _, dependency := range resolver.Dependencies {
		value := reflect.New(dependency.Type).Elem()
		productionScope := resolver.DB.ProductionDB().NewScope(value.Addr().Interface())
		productionTable := productionScope.TableName()
		primaryKey := productionScope.PrimaryKey()
		draftTable := DraftTableName(productionTable)

		var columns []string
		for _, field := range productionScope.Fields() {
			if field.IsNormal {
				columns = append(columns, field.DBName)
			}
		}

		var productionColumns []string
		for _, column := range columns {
			productionColumns = append(productionColumns, fmt.Sprintf("%v.%v", productionTable, column))
		}

		var draftColumns []string
		for _, column := range columns {
			draftColumns = append(draftColumns, fmt.Sprintf("%v.%v", draftTable, column))
		}

		deleteSql := fmt.Sprintf("DELETE FROM %v WHERE %v.%v IN (?)", draftTable, draftTable, primaryKey)
		tx.Exec(deleteSql, dependency.PrimaryKeys)

		discardSql := fmt.Sprintf("INSERT INTO %v (%v) SELECT %v FROM %v WHERE %v.%v IN (?)",
			draftTable, strings.Join(draftColumns, " ,"), strings.Join(productionColumns, " ,"),
			productionTable, productionTable, primaryKey)
		tx.Exec(discardSql, dependency.PrimaryKeys)
	}

	if err := tx.Error; err == nil {
		return tx.Commit().Error
	} else {
		tx.Rollback()
		return err
	}
}

// schema emits reviewable PostgreSQL DDL and the source manifest without connecting to a database.
// Never replace an applied migration: use -out with a scratch directory and write a forward delta.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	models "github.com/wxlbd/ruoyi-mall-go/internal/schema"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
func run() error {
	out := flag.String("out", "/private/tmp/ruoyi-schema", "scratch output directory")
	flag.Parse()
	db, err := gorm.Open(postgres.New(postgres.Config{}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		return err
	}
	var sql strings.Builder
	sql.WriteString("-- Source: internal/model; generated with pinned GORM PostgreSQL dialect.\n-- Forward-only baseline; never edit after application.\nBEGIN;\n")
	var tables []map[string]any
	evidenceHashes := map[string]string{}
	cache := new(sync.Map)
	for _, entry := range models.Models {
		s, err := schema.Parse(entry.Model, cache, schema.NamingStrategy{})
		if err != nil {
			return err
		}
		var fields []map[string]any
		var cols []string
		var keys []string
		for _, column := range s.DBNames {
			f := s.FieldsByDBName[column]
			if f.DBName == "" {
				continue
			}
			copyField := *f
			switch string(copyField.DataType) {
			case "int(11)":
				copyField.DataType = "integer"
			case "tinyint":
				copyField.DataType = "smallint"
			case "longtext":
				copyField.DataType = "text"
			case "datetime":
				copyField.DataType = "timestamptz"
			}
			dataType := db.Migrator().FullDataTypeOf(&copyField).SQL
			if f.FieldType == reflect.TypeOf(types.BitBool(false)) {
				dataType = "smallint"
				if f.NotNull {
					dataType += " NOT NULL"
				}
				if f.HasDefaultValue {
					dataType += " DEFAULT " + f.DefaultValue
				}
			}
			def := fmt.Sprintf("%q %s", f.DBName, dataType)
			if f.FieldType == reflect.TypeOf(types.BitBool(false)) {
				def += fmt.Sprintf(" CHECK (%q IN (0,1))", f.DBName)
			}
			cols = append(cols, def)
			if f.PrimaryKey {
				keys = append(keys, fmt.Sprintf("%q", f.DBName))
			}
			amount := "not monetary"
			if strings.Contains(f.DBName, "price") || strings.Contains(f.DBName, "amount") {
				amount = "integer minor units (fen); see source comment/service; percentages and rates excluded"
			}
			codec := "native scalar"
			ft := f.FieldType.String()
			if strings.Contains(ft, "ListFromCSV") {
				codec = "pkg/types.ListFromCSV Scanner/Valuer: CSV text, JSON accepted on read"
			}
			if f.Serializer != nil || strings.Contains(ft, "JSON") || strings.Contains(ft, "PayClientConfig") {
				codec = "JSON serializer/Scanner/Valuer per source type; storage follows dialect/type tag"
			}
			if strings.Contains(ft, "BitBool") {
				codec = "pkg/types.BitBool: smallint 0/1; soft-delete clauses bind integer Valuer"
			}
			fields = append(fields, map[string]any{"column": f.DBName, "go_field": f.Name, "go_type": ft, "postgres_definition": dataType, "nullable": !f.NotNull && !f.PrimaryKey, "default": f.DefaultValue, "gorm_tag": string(f.Tag), "source": entry.Source, "embedded_path": f.BindNames, "amount_unit": amount, "codec": codec})
		}
		if len(keys) > 0 {
			cols = append(cols, "PRIMARY KEY ("+strings.Join(keys, ",")+")")
		}
		fmt.Fprintf(&sql, "\n-- %s (%T)\nCREATE TABLE %q (\n  %s\n);\n", entry.Source, entry.Model, s.Table, strings.Join(cols, ",\n  "))
		tenant := "global table (no tenant_id in source model); review service authorization"
		if s.LookUpField("TenantID") != nil {
			tenant = "tenant_id; trusted context required by TenantPlugin; raw SQL must scope explicitly"
		}
		soft := "none in source model"
		if s.LookUpField("Deleted") != nil {
			soft = "deleted smallint 0/1 via BitBool softDelete flag"
		}
		refs := []string{}
		filepath.WalkDir("internal", func(path string, d os.DirEntry, e error) error {
			if e == nil && !d.IsDir() && strings.HasSuffix(path, ".go") && !strings.Contains(path, "/query/") && !strings.Contains(path, "/model/") {
				b, _ := os.ReadFile(path)
				if strings.Contains(string(b), reflect.TypeOf(entry.Model).Elem().Name()) || strings.Contains(string(b), s.Table) {
					refs = append(refs, path)
					evidenceHashes[path] = fmt.Sprintf("%x", sha256.Sum256(b))
				}
			}
			return e
		})
		sourceBytes, err := os.ReadFile(entry.Source)
		if err != nil {
			return err
		}
		reviewed := []string{}
		for _, path := range []string{"migrations/000002_constraints.up.sql", "migrations/000005_payment_identity.up.sql"} {
			b, _ := os.ReadFile(path)
			for _, line := range strings.Split(string(b), "\n") {
				if strings.Contains(line, " ON "+s.Table+"(") {
					reviewed = append(reviewed, line)
				}
			}
		}
		tables = append(tables, map[string]any{"source_sha256": fmt.Sprintf("%x", sha256.Sum256(sourceBytes)), "reviewed_index_definitions": reviewed, "table": s.Table, "source": entry.Source, "model": reflect.TypeOf(entry.Model).String(), "tenant_strategy": tenant, "soft_delete": soft, "fields": fields, "query_service_references": refs, "constraints": "primary key, source NOT NULL/defaults, BitBool domain; reviewed candidates in 000002 and data-baseline.md", "unresolved": "Business uniqueness and operational enablement beyond reviewed constraints require service-level acceptance; schema presence does not enable module."})
	}
	sql.WriteString("COMMIT;\n")
	sort.Slice(tables, func(i, j int) bool { return tables[i]["table"].(string) < tables[j]["table"].(string) })
	routerPaths, err := filepath.Glob("internal/api/router/*.go")
	if err != nil {
		return err
	}
	for _, path := range routerPaths {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		evidenceHashes[path] = fmt.Sprintf("%x", sha256.Sum256(b))
	}
	manifest := map[string]any{"query_router_source_sha256": evidenceHashes, "format": "JSON is valid YAML 1.2", "source_commit": "4a240b8b15d54f77d439b631417d35630c27ad4a plus current batch-ab model patches", "license": "repository LICENSE; no external SQL used", "generator": "cmd/schema with gorm.io/driver/postgres v1.6.0 and gorm.io/gorm v1.31.1", "tables": tables, "excluded_models": []string{"internal/model/statistics.go legacy DTOs: no TableName, no query or service references; active statistics in model/product and model/trade"}}
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(*out, 0755); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(*out, "schema-manifest.yaml"), append(b, 10), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(*out, "000001_baseline.up.sql"), []byte(sql.String()), 0644)
}

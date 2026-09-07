package database

import (
	"context"
	"errors"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"log"
	"reflect"
	"regexp"
)

var ErrTenantRequired = errors.New("trusted positive tenant context required")
var ErrTenantUnsafeSQL = errors.New("unscoped SQL, joins, table overrides and tenant mutation are forbidden")

type policyReadKey struct{}

// PolicyReadContext is a read-only, audited startup scope for loading tenant-qualified RBAC policy.
func PolicyReadContext(ctx context.Context) context.Context {
	log.Print("tenant platform read: load RBAC policy")
	return context.WithValue(ctx, policyReadKey{}, true)
}

var policyTables = map[string]bool{"system_role": true, "system_role_menu": true, "system_user_role": true, "system_users": true, "system_menu": true}

// Shared reference data is explicitly listed. All writes need a separate platform workflow.
var sharedReadTables = map[string]bool{"system_tenant": true, "system_tenant_package": true, "system_menu": true, "system_dict_type": true, "system_dict_data": true, "system_area": true}

type TenantPlugin struct{}

func (*TenantPlugin) Name() string { return "TenantPlugin" }
func (*TenantPlugin) Initialize(db *gorm.DB) error {
	registrations := []error{
		db.Callback().Query().Before("gorm:query").Register("tenant:query", func(tx *gorm.DB) { tenantGuard(tx, "read") }),
		db.Callback().Row().Before("gorm:row").Register("tenant:row", func(tx *gorm.DB) { tenantGuard(tx, "read") }),
		db.Callback().Create().Before("audit:before_create").Before("gorm:create").Register("tenant:create", func(tx *gorm.DB) { tenantGuard(tx, "create") }),
		db.Callback().Update().Before("gorm:update").Register("tenant:update", func(tx *gorm.DB) { tenantGuard(tx, "update") }),
		db.Callback().Delete().Before("gorm:delete").Register("tenant:delete", func(tx *gorm.DB) { tenantGuard(tx, "delete") }),
		db.Callback().Raw().Before("gorm:raw").Register("tenant:raw", func(tx *gorm.DB) { tx.AddError(ErrTenantUnsafeSQL) }),
	}
	for _, err := range registrations {
		if err != nil {
			return err
		}
	}
	return nil
}
func tenantGuard(db *gorm.DB, operation string) {
	st := db.Statement
	if db.Error != nil {
		return
	}
	if st.SQL.Len() > 0 {
		db.AddError(ErrTenantUnsafeSQL)
		return
	}
	if operation == "read" && st.Context.Value(policyReadKey{}) == true {
		if policyTables[st.Table] {
			return
		}
		if st.TableExpr != nil && ((st.Table == "srm" && st.TableExpr.SQL == "system_role_menu srm") || (st.Table == "sur" && st.TableExpr.SQL == "system_user_role sur")) {
			return
		}
	}
	if unsafeExpression(reflect.ValueOf(st.TableExpr)) {
		db.AddError(ErrTenantUnsafeSQL)
		return
	}
	for _, selection := range st.Selects {
		if unsafeSQL(selection) {
			db.AddError(ErrTenantUnsafeSQL)
			return
		}
	}
	for _, item := range st.Clauses {
		if unsafeExpression(reflect.ValueOf(item.Expression)) {
			db.AddError(ErrTenantUnsafeSQL)
			return
		}
	}
	if unsafeExpression(reflect.ValueOf(st.Dest)) {
		db.AddError(ErrTenantUnsafeSQL)
		return
	}
	if st.Schema == nil || st.Table != st.Schema.Table || len(st.Joins) > 0 {
		db.AddError(ErrTenantUnsafeSQL)
		return
	}
	if from, ok := st.Clauses["FROM"].Expression.(clause.From); ok && (len(from.Joins) > 0 || len(from.Tables) > 0) {
		db.AddError(ErrTenantUnsafeSQL)
		return
	}
	field := st.Schema.LookUpField("TenantID")
	if field == nil {
		if operation == "read" && sharedReadTables[st.Table] {
			return
		}
		db.AddError(ErrTenantUnsafeSQL)
		return
	}
	tenant, ok := pkgContext.TenantID(st.Context)
	if !ok {
		db.AddError(ErrTenantRequired)
		return
	}
	if operation == "create" {
		if _, exists := st.Clauses["ON CONFLICT"]; exists {
			db.AddError(ErrTenantUnsafeSQL)
			return
		}
		var setTenant func(reflect.Value)
		setTenant = func(v reflect.Value) {
			for v.IsValid() && (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) {
				v = v.Elem()
			}
			if !v.IsValid() {
				return
			}
			switch v.Kind() {
			case reflect.Slice, reflect.Array:
				for i := 0; i < v.Len(); i++ {
					setTenant(v.Index(i))
				}
			case reflect.Struct:
				value, _ := field.ValueOf(st.Context, v)
				id, ok := value.(int64)
				if !ok || (id != 0 && id != tenant) {
					db.AddError(ErrTenantUnsafeSQL)
					return
				}
				if err := field.Set(st.Context, v, tenant); err != nil {
					db.AddError(err)
				}
			default:
				db.AddError(ErrTenantUnsafeSQL)
			}
		}
		setTenant(st.ReflectValue)
		return
	}
	if operation == "update" {
		// Updates/Save must never move an existing record to a different tenant.
		st.Omits = append(st.Omits, field.DBName)
		if assignments, ok := st.Clauses["SET"].Expression.(clause.Set); ok {
			for _, assignment := range assignments {
				if assignment.Column.Name == field.DBName || assignment.Column.Name == field.Name {
					db.AddError(ErrTenantUnsafeSQL)
					return
				}
			}
		}
		v := reflect.Indirect(reflect.ValueOf(st.Dest))
		if v.IsValid() && v.Kind() == reflect.Map {
			for _, key := range v.MapKeys() {
				if key.Kind() == reflect.String && (key.String() == "tenant_id" || key.String() == "TenantID") {
					db.AddError(ErrTenantUnsafeSQL)
					return
				}
			}
		} else if v.IsValid() && v.Kind() == reflect.Struct && v.Type() == st.Schema.ModelType {
			value, _ := field.ValueOf(st.Context, v)
			if id, ok := value.(int64); !ok || (id != 0 && id != tenant) {
				db.AddError(ErrTenantUnsafeSQL)
				return
			}
		}
	}
	st.AddClause(clause.Where{Exprs: []clause.Expression{clause.Eq{Column: clause.Column{Table: st.Table, Name: field.DBName}, Value: tenant}}})
}

var unsafeSQLKeyword = regexp.MustCompile(`(?i)\b(select|union|join)\b|;`)

func unsafeSQL(sql string) bool { return unsafeSQLKeyword.MatchString(sql) }
func unsafeExpression(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	if v.CanInterface() {
		switch expr := v.Interface().(type) {
		case *gorm.DB:
			return true
		case clause.Expr:
			if unsafeSQL(expr.SQL) {
				return true
			}
			for _, arg := range expr.Vars {
				if unsafeExpression(reflect.ValueOf(arg)) {
					return true
				}
			}
			return false
		case clause.NamedExpr:
			if unsafeSQL(expr.SQL) {
				return true
			}
			for _, arg := range expr.Vars {
				if unsafeExpression(reflect.ValueOf(arg)) {
					return true
				}
			}
			return false
		}
	}
	switch v.Kind() {
	case reflect.Interface, reflect.Ptr:
		if !v.IsNil() {
			return unsafeExpression(v.Elem())
		}
	case reflect.Struct:
		// Only inspect GORM expression containers, never arbitrary model internals.
		if v.Type().PkgPath() == "gorm.io/gorm/clause" {
			for i := 0; i < v.NumField(); i++ {
				if unsafeExpression(v.Field(i)) {
					return true
				}
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if unsafeExpression(v.Index(i)) {
				return true
			}
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			if unsafeExpression(v.MapIndex(key)) {
				return true
			}
		}
	}
	return false
}

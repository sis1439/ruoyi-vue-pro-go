package infra

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	contract "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/infra"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	pkgcontext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"go.uber.org/zap"
)

func TestFileTenantPaths(t *testing.T) {
	s := &FileService{}
	a := pkgcontext.WithTenant(context.Background(), 1)
	b := pkgcontext.WithTenant(context.Background(), 2)
	path, err := s.GenerateUploadPath(a, "image.png", "products")
	if err != nil || !strings.HasPrefix(path, "tenant/1/products/") {
		t.Fatalf("path: %q %v", path, err)
	}
	for _, path := range []string{path, "../x.png", "/x.png", "tenant/2/../1/x.png", "tenant/2/%2e%2e/x.png", "tenant/2/a\\b.png"} {
		if _, err := s.tenantPath(b, path, false); err == nil {
			t.Errorf("accepted unsafe/cross-tenant path %q", path)
		}
	}
	if _, err := s.GenerateUploadPath(context.Background(), "x.png", ""); err == nil {
		t.Fatal("accepted missing tenant")
	}
}

func TestPostgresFileTenantIsolation(t *testing.T) {
	db := testutil.PostgreSQL(t)
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	q := query.Use(db)
	configs := NewFileConfigService(q)
	s := NewFileService(q, configs)
	a := pkgcontext.WithTenant(context.Background(), 101)
	b := pkgcontext.WithTenant(context.Background(), 102)
	for _, ctx := range []context.Context{a, b} {
		data, _ := json.Marshal(map[string]string{"basePath": t.TempDir(), "domain": "http://localhost:58090"})
		if err := db.WithContext(ctx).Create(&model.InfraFileConfig{Name: "local", Storage: 10, Master: true, Config: data}).Error; err != nil {
			t.Fatal(err)
		}
	}
	url, err := s.CreateFile(a, "product.txt", "products", []byte("tenant A"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	f := &model.InfraFile{}
	if err := db.WithContext(a).First(f).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(url, "tenant/101/") {
		t.Fatalf("unscoped URL %q", url)
	}
	if _, err := s.GetFile(b, f.ID); err == nil {
		t.Fatal("cross-tenant file record read")
	}
	if _, err := s.GetFileContent(b, f.ConfigId, f.Path); err == nil {
		t.Fatal("cross-tenant content read")
	}
	if err := s.DeleteFile(b, f.ID); err == nil {
		t.Fatal("cross-tenant delete")
	}
	if _, err := s.GetFilePresignedUrl(b, f.Path); err == nil {
		t.Fatal("cross-tenant presign")
	}
	if _, err := s.CreateFileCallback(b, &contract.FileCreateReq{ConfigID: f.ConfigId, Name: f.Name, Path: f.Path, URL: f.Url, Size: f.Size}); err == nil {
		t.Fatal("cross-tenant callback")
	}
	data, err := s.GetFileContent(a, f.ConfigId, "/"+f.Path)
	if err != nil || string(data) != "tenant A" {
		t.Fatalf("owner content: %q %v", data, err)
	}
	if err := s.DeleteFile(a, f.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetFileContent(a, f.ConfigId, f.Path); err == nil {
		t.Fatal("deleted content accessible")
	}
}

type tenantJobProbe struct {
	t      *testing.T
	tenant int64
	calls  int
}

func (p *tenantJobProbe) GetHandlerName() string { return "tenantProbe" }
func (p *tenantJobProbe) Execute(ctx context.Context, _ string) error {
	id, ok := pkgcontext.TenantID(ctx)
	if !ok || id != p.tenant || ctx.Err() != nil {
		p.t.Errorf("job context tenant=%d valid=%v error=%v", id, ok, ctx.Err())
	}
	p.calls++
	return nil
}

func TestPostgresJobRestoresPersistedTenant(t *testing.T) {
	db := testutil.PostgreSQL(t)
	if err := db.Create(&model.SystemTenant{ID: 101, Name: "jobs", Status: 0, ExpireDate: time.Now().Add(time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Use(&database.TenantPlugin{}); err != nil {
		t.Fatal(err)
	}
	s := &Scheduler{q: query.Use(db), log: zap.NewNop()}
	probe := &tenantJobProbe{t: t, tenant: 101}
	ctx, cancel := context.WithCancel(pkgcontext.WithTenant(context.Background(), 102))
	cancel()
	job := &model.InfraJob{ID: 91, HandlerName: "tenantProbe", TenantBaseDO: model.TenantBaseDO{TenantID: 101}}
	s.executeJob(ctx, job, probe)
	if probe.calls != 1 {
		t.Fatalf("expected job executed once, got %d", probe.calls)
	}
	job.TenantID = 0
	s.executeJob(context.Background(), job, probe)
	if probe.calls != 1 {
		t.Fatal("tenantless job executed")
	}
	var count int64
	if err := db.WithContext(pkgcontext.WithTenant(context.Background(), 102)).Model(&model.InfraJobLog{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("job logs leaked across tenants")
	}
}

package file

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FileClient 文件客户端接口
type FileClient interface {
	Upload(content []byte, path string) (string, error)
	Delete(path string) error
	GetContent(path string) ([]byte, error)
	GetURL(path string) string
	GetPresignedURL(path string) (string, error)
}

// ClientConfig 客户端配置通用结构 (用于解析 JSON)
type ClientConfig struct {
	Domain   string `json:"domain"`
	BasePath string `json:"basePath"` // Local 使用
	// S3 相关字段可按需添加
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	Bucket    string `json:"bucket"`
}

// LocalFileClient 本地文件客户端
type LocalFileClient struct {
	Config   ClientConfig
	ConfigID int64
}

func NewLocalFileClient(id int64, config json.RawMessage) (*LocalFileClient, error) {
	var cfg ClientConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(cfg.BasePath) {
		return nil, errors.New("local file basePath must be an absolute directory")
	}
	return &LocalFileClient{Config: cfg, ConfigID: id}, nil
}

func (c *LocalFileClient) Upload(content []byte, path string) (string, error) {
	if !filepath.IsLocal(path) {
		return "", errors.New("invalid storage path")
	}
	if err := os.MkdirAll(c.Config.BasePath, 0750); err != nil {
		return "", err
	}
	root, err := os.OpenRoot(c.Config.BasePath)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := root.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return "", err
	}
	if err := root.WriteFile(path, content, 0640); err != nil {
		return "", err
	}
	// 返回完整 URL
	return c.GetURL(path), nil
}

func (c *LocalFileClient) Delete(path string) error {
	root, err := os.OpenRoot(c.Config.BasePath)
	if err != nil {
		return err
	}
	defer root.Close()
	return root.Remove(path)
}

func (c *LocalFileClient) GetContent(path string) ([]byte, error) {
	root, err := os.OpenRoot(c.Config.BasePath)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.ReadFile(path)
}

func (c *LocalFileClient) GetURL(path string) string {
	// 对齐 Java: {}/admin-api/infra/file/{}/get/{}
	return fmt.Sprintf("%s/admin-api/infra/file/%d/get/%s", c.Config.Domain, c.ConfigID, path)
}

func (c *LocalFileClient) GetPresignedURL(path string) (string, error) {
	// Local 模式下不支持真正的预签名上传，返回上传接口地址
	// 前端需特殊处理：如果是 Local，直接调用 /upload
	return c.Config.Domain + "/admin-api/infra/file/upload", nil
}

// FileClientFactory 简单工厂
func NewFileClient(id int64, storage int32, config json.RawMessage) (FileClient, error) {
	switch storage {
	case 10: // Local
		return NewLocalFileClient(id, config)
	case 20: // S3
		return NewS3FileClient(id, config)
	default:
		return nil, errors.New("unknown storage type")
	}
}

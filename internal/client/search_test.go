package client

import (
	"os"
	"testing"

	"github.com/riba2534/feishu-cli/internal/config"
	"github.com/spf13/viper"
)

func TestBuildDocsURL_DefaultDocURL(t *testing.T) {
	// 隔离配置：重置 viper 并清除环境变量
	viper.Reset()
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	os.Unsetenv("FEISHU_DOC_URL")
	_ = config.Init("")

	tests := []struct {
		docsType string
		token    string
		expected string
	}{
		{"docx", "abc123", "https://feishu.cn/docx/abc123"},
		{"doc", "def456", "https://feishu.cn/docx/def456"},
		{"sheet", "ghi789", "https://feishu.cn/sheets/ghi789"},
		{"bitable", "jkl012", "https://feishu.cn/base/jkl012"},
		{"wiki", "mno345", "https://feishu.cn/wiki/mno345"},
		{"unknown_type", "pqr678", "https://feishu.cn/unknown_type/pqr678"},
	}

	for _, tt := range tests {
		t.Run(tt.docsType, func(t *testing.T) {
			result := buildDocsURL(tt.docsType, tt.token)
			if result != tt.expected {
				t.Errorf("buildDocsURL(%q, %q) = %q, want %q", tt.docsType, tt.token, result, tt.expected)
			}
		})
	}
}

func TestBuildDocsURL_CustomDocURL(t *testing.T) {
	// 设置自定义 DocURL
	viper.Reset()
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	os.Setenv("FEISHU_DOC_URL", "https://custom.larkoffice.com")
	defer os.Unsetenv("FEISHU_DOC_URL")
	_ = config.Init("")

	tests := []struct {
		docsType string
		token    string
		expected string
	}{
		{"docx", "abc123", "https://custom.larkoffice.com/docx/abc123"},
		{"doc", "def456", "https://custom.larkoffice.com/docx/def456"},
		{"wiki", "mno345", "https://custom.larkoffice.com/wiki/mno345"},
	}

	for _, tt := range tests {
		t.Run(tt.docsType, func(t *testing.T) {
			result := buildDocsURL(tt.docsType, tt.token)
			if result != tt.expected {
				t.Errorf("buildDocsURL(%q, %q) = %q, want %q", tt.docsType, tt.token, result, tt.expected)
			}
		})
	}
}

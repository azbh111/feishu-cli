package client

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
)

// cacheEntry 缓存条目，包含值和过期时间
type cacheEntry struct {
	Value     string    `json:"value"`
	ExpiresAt time.Time `json:"expires_at"`
}

// diskCache 实现 larkcore.Cache 接口，内存+磁盘两级缓存。
// 同进程内走内存，跨进程通过磁盘文件共享 token。
type diskCache struct {
	mu       sync.Mutex
	memory   map[string]cacheEntry
	filePath string
}

// 确保 diskCache 实现 larkcore.Cache 接口
var _ larkcore.Cache = (*diskCache)(nil)

// newDiskCache 创建磁盘缓存实例
func newDiskCache(filePath string) *diskCache {
	return &diskCache{
		memory:   make(map[string]cacheEntry),
		filePath: filePath,
	}
}

// defaultCachePath 返回默认缓存文件路径 (~/.feishu-cli/token_cache.json)
func defaultCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".feishu-cli", "token_cache.json")
}

func (c *diskCache) Get(ctx context.Context, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 内存缓存命中
	if entry, ok := c.memory[key]; ok {
		if time.Now().Before(entry.ExpiresAt) {
			return entry.Value, nil
		}
		// 过期，删除
		delete(c.memory, key)
	}

	// 内存未命中，尝试从磁盘加载
	entries := c.loadFromDisk()
	if entries == nil {
		return "", nil
	}

	entry, ok := entries[key]
	if !ok {
		return "", nil
	}
	if time.Now().After(entry.ExpiresAt) {
		// 磁盘上也过期了，清理
		delete(entries, key)
		c.saveToDisk(entries)
		return "", nil
	}

	// 回填内存缓存
	c.memory[key] = entry
	return entry.Value, nil
}

func (c *diskCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := cacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}

	// 写入内存
	c.memory[key] = entry

	// 写入磁盘（合并已有条目）
	entries := c.loadFromDisk()
	if entries == nil {
		entries = make(map[string]cacheEntry)
	}
	entries[key] = entry

	// 顺便清理已过期的条目
	now := time.Now()
	for k, e := range entries {
		if now.After(e.ExpiresAt) {
			delete(entries, k)
		}
	}

	return c.saveToDisk(entries)
}

// loadFromDisk 从磁盘读取缓存文件，解析失败返回 nil
func (c *diskCache) loadFromDisk() map[string]cacheEntry {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return nil
	}
	var entries map[string]cacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil
	}
	return entries
}

// saveToDisk 将缓存写入磁盘（0600 权限）
func (c *diskCache) saveToDisk(entries map[string]cacheEntry) error {
	dir := filepath.Dir(c.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	return os.WriteFile(c.filePath, data, 0600)
}

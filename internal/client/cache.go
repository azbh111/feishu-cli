package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
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
// 磁盘写入使用 tempfile+rename 保证原子性，文件锁防止跨进程竞争。
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

	// 文件锁保护 read-modify-write，防止跨进程竞争
	return c.lockedWriteToDisk(key, entry)
}

// lockFilePath 返回文件锁路径（与缓存文件同目录）
func (c *diskCache) lockFilePath() string {
	return c.filePath + ".lock"
}

// lockedWriteToDisk 在文件锁保护下执行 read-modify-write
func (c *diskCache) lockedWriteToDisk(key string, entry cacheEntry) error {
	dir := filepath.Dir(c.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// 打开锁文件（不存在则创建）
	lockFile, err := os.OpenFile(c.lockFilePath(), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("打开锁文件失败: %w", err)
	}
	defer lockFile.Close()

	// 获取排他锁（阻塞等待）
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("获取文件锁失败: %w", err)
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	// 在锁内读取最新磁盘数据
	entries := c.loadFromDisk()
	if entries == nil {
		entries = make(map[string]cacheEntry)
	}
	entries[key] = entry

	// 清理已过期的条目
	now := time.Now()
	for k, e := range entries {
		if now.After(e.ExpiresAt) {
			delete(entries, k)
		}
	}

	// 原子写入：tempfile + rename
	return c.atomicWriteToDisk(entries)
}

// loadFromDisk 从磁盘读取缓存文件，解析失败返回 nil
func (c *diskCache) loadFromDisk() map[string]cacheEntry {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return nil
	}
	var entries map[string]cacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		// 损坏的文件静默降级为空缓存
		return nil
	}
	return entries
}

// atomicWriteToDisk 通过临时文件+rename 原子写入，避免写入中途崩溃导致文件损坏
func (c *diskCache) atomicWriteToDisk(entries map[string]cacheEntry) error {
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}

	dir := filepath.Dir(c.filePath)
	base := filepath.Base(c.filePath)

	// 在同目录创建临时文件（同文件系统才能保证 rename 原子性）
	tmpFile, err := os.CreateTemp(dir, "."+base+".tmp.*")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmpFile.Name()

	// 写入数据并设置权限
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Chmod(0600); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// 原子替换目标文件
	if err := os.Rename(tmpPath, c.filePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("原子替换文件失败: %w", err)
	}
	return nil
}

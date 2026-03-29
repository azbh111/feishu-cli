package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestDiskCache_SetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	c := newDiskCache(filepath.Join(tmpDir, "cache.json"))

	ctx := context.Background()

	// 写入缓存
	err := c.Set(ctx, "test-key", "test-value", 10*time.Minute)
	if err != nil {
		t.Fatalf("Set() 返回错误: %v", err)
	}

	// 读取缓存
	val, err := c.Get(ctx, "test-key")
	if err != nil {
		t.Fatalf("Get() 返回错误: %v", err)
	}
	if val != "test-value" {
		t.Errorf("Get() = %q, 期望 %q", val, "test-value")
	}
}

func TestDiskCache_Expiry(t *testing.T) {
	tmpDir := t.TempDir()
	c := newDiskCache(filepath.Join(tmpDir, "cache.json"))

	ctx := context.Background()

	// 写入一个极短 TTL 的缓存
	err := c.Set(ctx, "expire-key", "expire-value", 1*time.Millisecond)
	if err != nil {
		t.Fatalf("Set() 返回错误: %v", err)
	}

	// 等待过期
	time.Sleep(5 * time.Millisecond)

	// 应返回空字符串
	val, err := c.Get(ctx, "expire-key")
	if err != nil {
		t.Fatalf("Get() 返回错误: %v", err)
	}
	if val != "" {
		t.Errorf("过期后 Get() = %q, 期望空字符串", val)
	}
}

func TestDiskCache_MultipleKeys(t *testing.T) {
	tmpDir := t.TempDir()
	c := newDiskCache(filepath.Join(tmpDir, "cache.json"))

	ctx := context.Background()

	// 写入多个 key
	c.Set(ctx, "key1", "value1", 10*time.Minute)
	c.Set(ctx, "key2", "value2", 10*time.Minute)

	val1, _ := c.Get(ctx, "key1")
	val2, _ := c.Get(ctx, "key2")

	if val1 != "value1" {
		t.Errorf("Get(key1) = %q, 期望 %q", val1, "value1")
	}
	if val2 != "value2" {
		t.Errorf("Get(key2) = %q, 期望 %q", val2, "value2")
	}
}

func TestDiskCache_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	c := newDiskCache(filepath.Join(tmpDir, "cache.json"))

	ctx := context.Background()

	// 写入后覆盖
	c.Set(ctx, "key", "old-value", 10*time.Minute)
	c.Set(ctx, "key", "new-value", 10*time.Minute)

	val, _ := c.Get(ctx, "key")
	if val != "new-value" {
		t.Errorf("Get() = %q, 期望 %q", val, "new-value")
	}
}

func TestDiskCache_PersistAcrossInstances(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "cache.json")

	ctx := context.Background()

	// 第一个实例写入
	c1 := newDiskCache(cacheFile)
	c1.Set(ctx, "persist-key", "persist-value", 10*time.Minute)

	// 第二个实例读取（模拟新进程）
	c2 := newDiskCache(cacheFile)
	val, err := c2.Get(ctx, "persist-key")
	if err != nil {
		t.Fatalf("新实例 Get() 返回错误: %v", err)
	}
	if val != "persist-value" {
		t.Errorf("新实例 Get() = %q, 期望 %q", val, "persist-value")
	}
}

func TestDiskCache_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "cache.json")

	ctx := context.Background()

	c := newDiskCache(cacheFile)
	c.Set(ctx, "key", "value", 10*time.Minute)

	info, err := os.Stat(cacheFile)
	if err != nil {
		t.Fatalf("Stat() 返回错误: %v", err)
	}

	// 文件权限应为 0600
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("文件权限 = %o, 期望 0600", perm)
	}
}

func TestDiskCache_MissingKey(t *testing.T) {
	tmpDir := t.TempDir()
	c := newDiskCache(filepath.Join(tmpDir, "cache.json"))

	ctx := context.Background()

	// 不存在的 key 应返回空字符串
	val, err := c.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get() 返回错误: %v", err)
	}
	if val != "" {
		t.Errorf("Get(不存在的key) = %q, 期望空字符串", val)
	}
}

func TestDiskCache_CorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "cache.json")

	// 写入损坏的 JSON
	os.WriteFile(cacheFile, []byte("{invalid json"), 0600)

	ctx := context.Background()
	c := newDiskCache(cacheFile)

	// 损坏文件不应导致 Get 崩溃，返回空字符串
	val, err := c.Get(ctx, "key")
	if err != nil {
		t.Fatalf("Get() 不应返回错误: %v", err)
	}
	if val != "" {
		t.Errorf("Get() = %q, 期望空字符串", val)
	}

	// Set 应能覆盖损坏文件
	err = c.Set(ctx, "key", "value", 10*time.Minute)
	if err != nil {
		t.Fatalf("Set() 返回错误: %v", err)
	}

	val, _ = c.Get(ctx, "key")
	if val != "value" {
		t.Errorf("恢复后 Get() = %q, 期望 %q", val, "value")
	}
}

func TestDiskCache_CleanExpired(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "cache.json")

	ctx := context.Background()
	c := newDiskCache(cacheFile)

	// 写入一个已过期和一个未过期的 key
	c.Set(ctx, "expired", "v1", 1*time.Millisecond)
	c.Set(ctx, "valid", "v2", 10*time.Minute)

	time.Sleep(5 * time.Millisecond)

	// 读取有效 key 应触发清理过期条目
	val, _ := c.Get(ctx, "valid")
	if val != "v2" {
		t.Errorf("Get(valid) = %q, 期望 %q", val, "v2")
	}

	// 过期的 key 应返回空
	val, _ = c.Get(ctx, "expired")
	if val != "" {
		t.Errorf("Get(expired) = %q, 期望空字符串", val)
	}
}

// TestDiskCache_AtomicWrite 验证 atomicWriteToDisk 使用原子写入（tempfile+rename），
// 写入后文件始终是有效 JSON，且不残留临时文件。
func TestDiskCache_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "cache.json")

	ctx := context.Background()
	c := newDiskCache(cacheFile)

	// 写入一个有效值
	c.Set(ctx, "key", "value", 10*time.Minute)

	// 验证文件是有效 JSON
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("ReadFile() 返回错误: %v", err)
	}
	var entries map[string]cacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("写入后文件不是有效 JSON: %v", err)
	}

	// 验证没有遗留临时文件
	matches, _ := filepath.Glob(filepath.Join(tmpDir, ".cache.json.tmp.*"))
	if len(matches) > 0 {
		t.Errorf("存在遗留临时文件: %v", matches)
	}
}

// TestDiskCache_ConcurrentSameFile 验证多个独立实例并发写同一个缓存文件时，
// 所有条目最终都不会丢失（通过文件锁保护 read-modify-write）。
func TestDiskCache_ConcurrentSameFile(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "cache.json")

	ctx := context.Background()
	const n = 20

	var wg sync.WaitGroup
	wg.Add(n)

	// 每个 goroutine 使用独立的 diskCache 实例，模拟不同进程
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			c := newDiskCache(cacheFile)
			key := fmt.Sprintf("key-%d", idx)
			val := fmt.Sprintf("value-%d", idx)
			if err := c.Set(ctx, key, val, 10*time.Minute); err != nil {
				t.Errorf("Set(%s) 失败: %v", key, err)
			}
		}(i)
	}

	wg.Wait()

	// 验证磁盘文件是有效 JSON
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("ReadFile() 返回错误: %v", err)
	}
	var entries map[string]cacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("并发写入后文件不是有效 JSON: %v\n文件内容: %s", err, string(data))
	}

	// 验证所有 key 都存在（文件锁保证不丢条目）
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("key-%d", i)
		entry, ok := entries[key]
		if !ok {
			t.Errorf("缺少 key %q（跨进程竞争导致条目丢失）", key)
			continue
		}
		expected := fmt.Sprintf("value-%d", i)
		if entry.Value != expected {
			t.Errorf("entries[%q].Value = %q, 期望 %q", key, entry.Value, expected)
		}
	}
}

// TestDiskCache_ConcurrentReadWrite 验证并发读写不会 panic 或产生损坏数据
func TestDiskCache_ConcurrentReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, "cache.json")

	ctx := context.Background()

	// 种一个初始值
	c := newDiskCache(cacheFile)
	c.Set(ctx, "init", "init-value", 10*time.Minute)

	const n = 20
	var wg sync.WaitGroup
	wg.Add(n * 2)

	// 并发写
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			ci := newDiskCache(cacheFile)
			ci.Set(ctx, fmt.Sprintf("w-%d", idx), fmt.Sprintf("v-%d", idx), 10*time.Minute)
		}(i)
	}

	// 并发读
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			ci := newDiskCache(cacheFile)
			ci.Get(ctx, "init")
		}()
	}

	wg.Wait()

	// 验证文件仍是有效 JSON
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("ReadFile() 返回错误: %v", err)
	}
	var entries map[string]cacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("并发读写后文件不是有效 JSON: %v", err)
	}
}

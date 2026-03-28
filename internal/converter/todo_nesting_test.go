package converter

import (
	"testing"
)

// TestTodoNestedChildren 验证 todo 列表项的嵌套子项能正确收集
func TestTodoNestedChildren(t *testing.T) {
	md := `- [ ] 12
- [ ] 13
    - [ ] 131
    - [ ] 132
- [ ] 14`

	conv := NewMarkdownToBlock([]byte(md), ConvertOptions{}, "")
	result, err := conv.ConvertWithTableData()
	if err != nil {
		t.Fatalf("ConvertWithTableData 失败: %v", err)
	}

	// 应该有 3 个顶层块: 12, 13, 14
	if len(result.BlockNodes) != 3 {
		t.Fatalf("期望 3 个顶层块，得到 %d", len(result.BlockNodes))
	}

	// 第二个块 (13) 应该有 2 个子项 (131, 132)
	node13 := result.BlockNodes[1]
	if len(node13.Children) != 2 {
		t.Errorf("块 '13' 期望 2 个子项，得到 %d", len(node13.Children))
	}

	// 验证子项都是 Todo 类型 (type=17)
	for i, child := range node13.Children {
		if child.Block.BlockType == nil || *child.Block.BlockType != int(BlockTypeTodo) {
			bt := 0
			if child.Block.BlockType != nil {
				bt = *child.Block.BlockType
			}
			t.Errorf("子项 %d 期望类型 %d (Todo)，得到 %d", i, BlockTypeTodo, bt)
		}
	}
}

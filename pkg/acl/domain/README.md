# Domain Matcher Package

高效的域名匹配器，基于 Succinct Trie 数据结构实现，适用于大规模域名列表的快速匹配。

## 特性

- ⚡ **O(k) 时间复杂度**: k 为域名长度，与规则数量无关
- 💾 **内存高效**: 使用位图压缩，内存占用降低 5-10 倍
- 🎯 **精确匹配**: 支持完全匹配和后缀匹配
- 🔄 **UTF-8 安全**: 正确处理国际化域名
- ✅ **零依赖**: 无需第三方库，复刻自 sing-box 官方实现

## 使用方法

### 基本用法

```go
package main

import (
    "fmt"
    "github.com/xflash-panda/acl-engine/pkg/acl/domain"
)

func main() {
    // 创建域名匹配器
    matcher := domain.NewMatcher(
        []string{"google.com"},        // 精确匹配列表
        []string{"facebook.com"},      // 后缀匹配列表
    )

    // 测试匹配
    fmt.Println(matcher.Match("google.com"))         // true (精确匹配)
    fmt.Println(matcher.Match("www.google.com"))     // false (精确匹配不含子域名)

    fmt.Println(matcher.Match("facebook.com"))       // true (后缀匹配)
    fmt.Println(matcher.Match("www.facebook.com"))   // true (后缀匹配包含子域名)
    fmt.Println(matcher.Match("twitter.com"))        // false (不在列表中)
}
```

### 后缀匹配的两种模式

#### 1. 不带前导点 (推荐)

```go
matcher := domain.NewMatcher(nil, []string{"google.com"})

matcher.Match("google.com")         // ✅ true  - 匹配根域名
matcher.Match("www.google.com")     // ✅ true  - 匹配子域名
matcher.Match("mail.google.com")    // ✅ true  - 匹配子域名
```

#### 2. 带前导点 (仅子域名)

```go
matcher := domain.NewMatcher(nil, []string{".google.com"})

matcher.Match("google.com")         // ❌ false - 不匹配根域名
matcher.Match("www.google.com")     // ✅ true  - 仅匹配子域名
matcher.Match("mail.google.com")    // ✅ true  - 仅匹配子域名
```

### 大规模域名列表

```go
// 1000+ 域名规则
domains := []string{"google.com", "facebook.com", ...} // 1000+ 个域名
matcher := domain.NewMatcher(nil, domains)

// 匹配性能: ~60ns/op (与域名数量无关!)
result := matcher.Match("test.google.com")
```

## 性能特征

### 时间复杂度

| 操作 | 复杂度 | 说明 |
|-----|--------|-----|
| 构建 | O(m log m) | m = 所有域名的总字符数 |
| 匹配 | O(k) | k = 查询域名的长度 |

### 基准测试结果

```
BenchmarkMatcher_Match_Hit_First    20M ops  58ns/op  16B/op
BenchmarkMatcher_Match_Hit_Middle   20M ops  61ns/op  24B/op
BenchmarkMatcher_Match_Miss         23M ops  52ns/op  16B/op
BenchmarkMatcher_Construction       518K ops 2.3μs/op 8.4KB/op
```

**关键洞察**:
- 匹配延迟稳定在 ~60ns，无论列表大小
- 内存分配极小 (16-24 字节)
- 构建时间：10域名=2.3μs，1000域名=207μs

## 实现细节

### Succinct Trie 数据结构

```go
type succinctSet struct {
    leaves      []uint64  // 叶子节点位图
    labelBitmap []uint64  // 节点边界位图
    labels      []byte    // 字符标签数组
    ranks       []int32   // Rank 辅助索引
    selects     []int32   // Select 辅助索引
}
```

### 核心算法

1. **域名反转**: "google.com" → "moc.elgoog"
2. **排序**: 字典序排序所有反转后的域名
3. **构建 Trie**: 使用 BFS 构建紧凑字典树
4. **位图压缩**: 使用位图代替指针存储
5. **Rank/Select**: 预计算索引实现 O(1) 导航

### 特殊标记

- `prefixLabel` ('\r'): 标记 ".example.com" 类型的规则
- `rootLabel` ('\n'): 标记 "example.com" 类型的规则

## 测试覆盖

- 单元测试覆盖率: 90.3%
- 包含边界条件、UTF-8、大小写等全面测试
- 所有测试通过

## 参考资料

- **sing-box 官方实现**: https://github.com/sagernet/sing/tree/main/common/domain
- **Succinct Data Structures**: https://github.com/openacid/succinct

## 许可

本实现基于 sing-box 官方代码复刻，遵循原项目许可协议。

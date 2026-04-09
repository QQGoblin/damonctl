# damon-ctl

管理内核 DAMON (Data Access Monitoring Framework) kdamond 实例的命令行工具.

通过预分配槽位 (Slot) 的策略, 实现对多个进程动态监控和内存回收的高效管理.

## 设计背景

- **一对一绑定**: 每个进程对应一个 kdamond 实例 (1 kdamond → 1 context → 1 target → N schemes).
- **槽位预分配**: 通过 `init` 命令提前创建固定数量的 sysfs 目录 (槽位). `start` 和 `stop` 操作仅针对特定槽位进行, 互不干扰.
- **低开销**: 未使用的槽位仅为 sysfs 中的 kobjects, 不会创建内核线程.
- **Context 限制**: 每个 kdamond 实例最多支持 1 个 context.
- **目录重构**: 写入 `nr_kdamonds` 会销毁并重新创建所有相关目录. 如果此时有任何 kdamond 正在运行, 内核将返回 `EBUSY`.
- **预分配的必要性**: 基于上述约束, 预分配槽位是避免运行时冲突、实现并发管理的唯一可靠路径.

## 配置文件

kdamond 配置文件格式如下：

```json
{
  "ops": "vaddr",
  "monitoring_attrs": {
    "sample_us": 5000,
    "aggr_us": 100000,
    "update_us": 1000000,
    "min_regions": 10,
    "max_regions": 1000
  },
  "schemes": [
    {
      "action": "pageout",
      "min_sz_bytes": 4096,
      "max_sz_bytes": 4611686018427387904,
      "min_nr_accesses": 0,
      "max_nr_accesses": 0,
      "min_age": 600,
      "max_age": 2147483647,
      "quota": {
        "ms": 10,
        "bytes": 134217728,
        "reset_interval_ms": 1000,
        "weight_sz": 0,
        "weight_accesses": 0,
        "weight_age": 1
      },
      "watermarks": {
        "metric": "free_mem_rate",
        "interval_us": 5000000,
        "high": 500,
        "mid": 400,
        "low": 200
      }
    }
  ]
}
```


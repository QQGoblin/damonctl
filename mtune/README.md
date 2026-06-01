# mtune

`mtune` 是一个常驻守护进程，根据当前主机的内存压力，**反馈式地自动调优内核 `damon_reclaim` 模块的参数**（主要是 `quota_sz`
），使主机的可用内存（`MemAvailable`）稳定维持在目标水位附近，从而在「内存回收力度」与「业务性能开销」之间取得平衡。

## 2. 配置文件

`mtune` 启动时读取 `/etc/mtune/config.json`（路径可由 `--config` 覆盖）。配置分为两部分：`reclaim`
（模块初始参数，启动时一次性写入）与 `tune`（调优控制器参数）。

```json
{
  "reclaim": {
    "min_age": "60000000",
    "min_nr_regions": "2048",
    "max_nr_regions": "8196",
    "sample_interval": "20000",
    "aggr_interval": "1000000",
    "quota_ms": "0",
    "quota_sz": "1073741824",
    "quota_reset_interval_ms": "1000",
    "wmarks_high": "600",
    "wmarks_mid": "500",
    "wmarks_low": "0",
    "monitor_region_start": "0",
    "monitor_region_end": ""
  },
  "tune": {
    "interval": 60,
    "available_bytes": 21474836480,
    "available_ratio": 0.10,
    "dead_ratio": 0.05,
    "quota_sz_min": 134217728,
    "quota_sz_max": 2147483648,
    "step": 268435456,
    "gain": 0.1
  }
}
```

## 3. 调优算法

`mtune` 通过控制内核 `damon_reclaim` 模块（`/sys/module/damon_reclaim/parameters/`）的 `quota_sz` 大小，使主机可用内存维持在期望大小。

### 1. 控制参数

| 字段                | 类型    | 默认                     | 说明                                                                 |
|-------------------|-------|------------------------|--------------------------------------------------------------------|
| `interval`        | int   | `5`                    | 每 N 个 aggr_interval 调整一次 quota_sz 参数                               |
| `available_bytes` | int   | `21474836480` (20 GiB) | 目标可用内存的上限                                                          |
| `available_ratio` | float | `0.10`                 | 目标可用内存占 `MemTotal` 的比例                                             |
| `dead_ratio`      | float | `0.05`                 | 死区比例，`MemAvailable` 落在 `target ± target*deadband_ratio` 内时不调参，抑制抖动 |
| `quota_sz_min`    | int   | `134217728` (128 MiB)  | `quota_sz` 下限（即使内存充裕也不低于此值，保证基础回收能力）                               |
| `quota_sz_max`    | int   | `8589934592` (8 GiB)   | `quota_sz` 上限（防止回收过猛拖垮业务）                                          |
| `step`            | int   | `268435456` (256 MiB)  | 单次调整的步长上限                                                          |
| `gain`            | float | `0.1`                  | 经验值取值为 0～1， 表示 damon_reclaim 回收成功比例， 通常情况下为 0.1。                   |

### 2 控制律

### 3 边界与约束

- `quota_sz` 始终被钳制在 `[quota_sz_min, quota_sz_max]`，杜绝「回收为 0」或「无限回收」两种极端；
- 每次写入 `quota_sz` 后**必须**写 `commit_inputs=Y`，否则内核不会重新读取新值；
- 若计算结果与当前生效值相同（含死区命中），跳过写入，减少无谓的 sysfs 操作与日志噪声。
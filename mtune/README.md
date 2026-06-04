# mtune

`mtune` 是一个常驻守护进程，根据当前主机的内存压力，**反馈式地自动调优内核 `damon_reclaim` 模块的参数**（主要是 `quota_sz`
）。它采用 hybrid 控制：用 `MemAvailable` 计算需要的回收力度，再用 `memory some PSI` 计算允许的回收上限，从而在「可用内存水位」与「业务 stall 开销」之间取得平衡。

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
    "wmarks_low": "0"
  },
  "tune": {
    "interval": 60,
    "available_bytes": 21474836480,
    "available_ratio": 0.10,
    "dead_ratio": 0.05,
    "quota_sz_min": 134217728,
    "quota_sz_max": 2147483648,
    "gain": 10,
    "some_psi_us": 600000,
    "psi_dead_ratio": 0.05
  }
}
```

## 3. 调优算法

`mtune` 通过控制内核 `damon_reclaim` 模块（`/sys/module/damon_reclaim/parameters/`）的 `quota_sz` 大小，使主机可用内存维持在期望大小，同时限制 DAMON_RECLAIM 自身带来的 memory stall。

### 1. 控制参数

| 字段                | 类型    | 默认                     | 说明                                                                 |
|-------------------|-------|------------------------|--------------------------------------------------------------------|
| `interval`        | int   | `60`                   | 每 N 个 aggr_interval 调整一次 quota_sz 参数                               |
| `available_bytes` | int   | `21474836480` (20 GiB) | 目标可用内存的上限                                                          |
| `available_ratio` | float | `0.10`                 | 目标可用内存占 `MemTotal` 的比例                                             |
| `dead_ratio`      | float | `0.05`                 | 死区比例，`MemAvailable` 落在 `target ± target*deadband_ratio` 内时不调参，抑制抖动 |
| `quota_sz_min`    | int   | `134217728` (128 MiB)  | `quota_sz` 下限（即使内存充裕也不低于此值，保证基础回收能力）                               |
| `quota_sz_max`    | int   | `2147483648` (2 GiB)   | `quota_sz` 上限（防止回收过猛拖垮业务）                                          |
| `gain`            | float | `10`                   | 含义 damon_reclaim 回收成功比例，即每成功回合 1GB 内存，需要对 gain 倍大小的内存尝试回收          |
| `some_psi_us`     | int   | `1000000`              | 每个调优周期内允许的 `/proc/pressure/memory` 中 `some total` 增量，单位 us                  |
| `psi_dead_ratio`  | float | `0.05`                 | PSI 目标死区比例，`some` PSI 落在 `target ± target*psi_dead_ratio` 内时保持 PSI 上限不变       |

### 2. Hybrid 控制

每个调优周期内，`mtune` 同时计算两个值：

- `available_quota`：由 `MemAvailable` 缺口计算出的回收需求；如果可用内存已经超过目标，则降到 `quota_sz_min`；如果位于死区内，则保持当前值。
- `psi_quota_cap`：由本周期 `memory some PSI` 增量计算出的回收上限；PSI 低于目标时上限放宽，PSI 高于目标时上限收缩，达到目标两倍及以上时降到 `quota_sz_min`。

最终写入值为：

```text
quota_sz = min(available_quota, psi_quota_cap)
```

因此，`MemAvailable` 负责表达“需要回收多少”，`memory some PSI` 负责表达“最多允许回收多猛”。

### 3 边界与约束

- `quota_sz` 始终被钳制在 `[quota_sz_min, quota_sz_max]`，杜绝「回收为 0」或「无限回收」两种极端；
- 每次写入 `quota_sz` 后**必须**写 `commit_inputs=Y`，否则内核不会重新读取新值；
- 若计算结果与当前生效值相同（含死区命中），跳过写入，减少无谓的 sysfs 操作与日志噪声。
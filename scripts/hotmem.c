/*
 * hotmem.c - 申请一块内存并反复读写，使其成为内存热点
 *
 * 编译: gcc -O2 -o hotmem hotmem.c
 * 用法: ./hotmem <size>
 *   size 支持单位后缀: K/M/G (大小写均可), 例如 64M, 1G, 512K, 1048576
 *
 * 行为:
 *   - 通过 mmap (匿名 + MAP_POPULATE) 申请指定大小的内存并预触达
 *   - 主线程在循环里对整块内存做顺序读写(累加+写回)，使其成为热点
 *   - 收到 SIGINT (Ctrl-C) 时优雅退出，打印总耗时与吞吐
 */

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <errno.h>
#include <unistd.h>
#include <signal.h>
#include <ctype.h>
#include <time.h>
#include <sys/mman.h>

static volatile sig_atomic_t g_stop = 0;

static void on_sigint(int sig) {
    (void)sig;
    g_stop = 1;
}

/* 解析带 K/M/G 后缀的大小字符串，返回字节数；失败返回 0 */
static size_t parse_size(const char *s) {
    if (!s || !*s) return 0;
    char *end = NULL;
    errno = 0;
    unsigned long long v = strtoull(s, &end, 10);
    if (errno != 0 || end == s) return 0;
    size_t mult = 1;
    if (end && *end) {
        char c = (char)toupper((unsigned char)*end);
        switch (c) {
            case 'K': mult = 1024ULL; break;
            case 'M': mult = 1024ULL * 1024ULL; break;
            case 'G': mult = 1024ULL * 1024ULL * 1024ULL; break;
            case '\0': mult = 1; break;
            default: return 0;
        }
        /* 允许 'KB'/'MB'/'GB' 这样的写法 */
        if (end[1] != '\0' && !(toupper((unsigned char)end[1]) == 'B' && end[2] == '\0')) {
            return 0;
        }
    }
    if (v == 0) return 0;
    if (v > (SIZE_MAX / mult)) return 0; /* 溢出保护 */
    return (size_t)(v * mult);
}

static double elapsed_sec(const struct timespec *a, const struct timespec *b) {
    return (b->tv_sec - a->tv_sec) + (b->tv_nsec - a->tv_nsec) / 1e9;
}

int main(int argc, char **argv) {
    if (argc != 2) {
        fprintf(stderr,
            "用法: %s <size>\n"
            "  size 支持单位 K/M/G, 例如: 64M, 1G, 512K, 1048576\n",
            argv[0]);
        return 1;
    }

    size_t size = parse_size(argv[1]);
    if (size == 0) {
        fprintf(stderr, "无效的 size 参数: %s\n", argv[1]);
        return 1;
    }

    /* 对齐到 8 字节, 方便按 uint64_t 访问 */
    size_t bytes = size & ~((size_t)7);
    if (bytes < 8) {
        fprintf(stderr, "size 太小, 至少 8 字节\n");
        return 1;
    }

    /* 申请内存: 匿名映射, MAP_POPULATE 预填充页表, 减少首次缺页开销 */
    void *p = mmap(NULL, bytes, PROT_READ | PROT_WRITE,
                   MAP_PRIVATE | MAP_ANONYMOUS | MAP_POPULATE, -1, 0);
    if (p == MAP_FAILED) {
        fprintf(stderr, "mmap 失败: %s\n", strerror(errno));
        return 1;
    }

    /* 建议内核把这块内存视为"将被频繁访问"(尽量保留在物理内存中) */
    (void)madvise(p, bytes, MADV_WILLNEED);

    /* 初始化一遍, 确保所有页都真正分配了物理页 */
    memset(p, 0x5A, bytes);

    /* 注册 SIGINT, 支持 Ctrl-C 优雅退出 */
    struct sigaction sa;
    memset(&sa, 0, sizeof(sa));
    sa.sa_handler = on_sigint;
    sigaction(SIGINT, &sa, NULL);
    sigaction(SIGTERM, &sa, NULL);

    uint64_t *buf = (uint64_t *)p;
    size_t n = bytes / sizeof(uint64_t);

    printf("[hotmem] pid=%d size=%zu bytes (%.2f MiB), elements=%zu\n",
           getpid(), bytes, bytes / 1048576.0, n);
    printf("[hotmem] 开始反复读写, Ctrl-C 退出...\n");
    fflush(stdout);

    struct timespec t0, t1;
    clock_gettime(CLOCK_MONOTONIC, &t0);

    uint64_t iter = 0;
    uint64_t sink = 0; /* 防止编译器优化掉读 */

    while (!g_stop) {
        /* 顺序读写一整轮: 累加并写回, 形成读+写的内存流量 */
        for (size_t i = 0; i < n && !g_stop; i++) {
            uint64_t v = buf[i];
            v = v + iter + i;
            buf[i] = v;
            sink ^= v;
        }
        iter++;

        /* 每若干轮打印一次进度 */
        if ((iter & 0xF) == 0) {
            clock_gettime(CLOCK_MONOTONIC, &t1);
            double sec = elapsed_sec(&t0, &t1);
            double gib = ((double)bytes * iter * 2.0) / (1024.0 * 1024.0 * 1024.0); /* 读+写 */
            printf("[hotmem] iter=%llu elapsed=%.2fs throughput=%.2f GiB/s\n",
                   (unsigned long long)iter, sec, gib / sec);
            fflush(stdout);
        }
    }

    clock_gettime(CLOCK_MONOTONIC, &t1);
    double sec = elapsed_sec(&t0, &t1);
    double gib = ((double)bytes * iter * 2.0) / (1024.0 * 1024.0 * 1024.0);
    printf("\n[hotmem] 退出: iter=%llu elapsed=%.2fs avg=%.2f GiB/s sink=0x%llx\n",
           (unsigned long long)iter, sec, sec > 0 ? gib / sec : 0.0,
           (unsigned long long)sink);

    munmap(p, bytes);
    return 0;
}
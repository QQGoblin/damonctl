/*
 * hotmem.c - 申请一块内存并反复读写，使其成为内存热点
 *
 * 编译: gcc -O2 -Wall -Wextra -o bin/hotmem hotmem.c
 * 用法: ./hotmem <size>
 *   size 支持单位后缀: K/M/G (大小写均可), 例如 64M, 1G, 512K, 1048576
 *
 * 行为:
 *   - 通过 mmap (匿名 + MAP_POPULATE) 申请指定大小的内存并预触达
 *   - 主线程在循环里按 4KB 步长仅写每页首字节，尽量用最小写放大维持页面热度
 *   - 收到 SIGINT (Ctrl-C) 时优雅退出，打印总耗时、热区扫描速率和触页速率
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

enum {
    HOT_PAGE_SIZE = 4096,
};

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

    size_t bytes = size;

    /* 申请内存: 匿名映射, MAP_POPULATE 预填充页表, 减少首次缺页开销 */
    void *p = mmap(NULL, bytes, PROT_READ | PROT_WRITE,
                   MAP_PRIVATE | MAP_ANONYMOUS | MAP_POPULATE, -1, 0);
    if (p == MAP_FAILED) {
        fprintf(stderr, "mmap 失败: %s\n", strerror(errno));
        return 1;
    }

    /* 建议内核把这块内存视为"将被频繁访问"(尽量保留在物理内存中) */
    (void)madvise(p, bytes, MADV_WILLNEED);

    /* 初始化每个 4KB 页的首字节, 确保所有页都真正分配了物理页 */
    uint8_t *buf = (uint8_t *)p;
    for (size_t offset = 0; offset < bytes; offset += HOT_PAGE_SIZE) {
        buf[offset] = 0x5A;
    }

    /* 注册 SIGINT, 支持 Ctrl-C 优雅退出 */
    struct sigaction sa;
    memset(&sa, 0, sizeof(sa));
    sa.sa_handler = on_sigint;
    sigaction(SIGINT, &sa, NULL);
    sigaction(SIGTERM, &sa, NULL);

    size_t pages = (bytes + HOT_PAGE_SIZE - 1) / HOT_PAGE_SIZE;

    printf("[hotmem] pid=%d size=%zu bytes (%.2f MiB), pages=%zu stride=%d\n",
        getpid(), bytes, bytes / 1048576.0, pages, HOT_PAGE_SIZE);
    printf("[hotmem] 开始按页触达, Ctrl-C 退出...\n");
    fflush(stdout);

    struct timespec t0, t1, last_report;
    clock_gettime(CLOCK_MONOTONIC, &t0);
    last_report = t0;

    uint64_t iter = 0;
    uint64_t sink = 0; /* 防止编译器优化掉读 */

    while (!g_stop) {
        /* 每轮仅触碰每个 4KB 页的首字节, 以最小写放大维持页面热度 */
        for (size_t offset = 0, page = 0; offset < bytes && !g_stop; offset += HOT_PAGE_SIZE, page++) {
            uint8_t v = buf[offset];
            v = (uint8_t)(v + iter + page + 1U);
            buf[offset] = v;
            sink ^= v;
        }
        iter++;

        /* 用时间门控输出, 避免小热区场景下日志成为瓶颈 */
        if ((iter & 0x3FFF) == 0) {
            clock_gettime(CLOCK_MONOTONIC, &t1);
            if (elapsed_sec(&last_report, &t1) >= 1.0) {
                double sec = elapsed_sec(&t0, &t1);
                double hot_gib = ((double)bytes * iter) / (1024.0 * 1024.0 * 1024.0);
                double touch_mpps = ((double)pages * iter) / (1000.0 * 1000.0);
                printf("[hotmem] iter=%llu elapsed=%.2fs hotscan=%.2f GiB/s touch=%.2f Mpages/s\n",
                       (unsigned long long)iter, sec,
                       sec > 0 ? hot_gib / sec : 0.0,
                       sec > 0 ? touch_mpps / sec : 0.0);
                fflush(stdout);
                last_report = t1;
            }
        }
    }

    clock_gettime(CLOCK_MONOTONIC, &t1);
    double sec = elapsed_sec(&t0, &t1);
    double hot_gib = ((double)bytes * iter) / (1024.0 * 1024.0 * 1024.0);
    double touch_mpps = ((double)pages * iter) / (1000.0 * 1000.0);
    printf("\n[hotmem] 退出: iter=%llu elapsed=%.2fs hotscan=%.2f GiB/s touch=%.2f Mpages/s sink=0x%llx\n",
           (unsigned long long)iter, sec,
           sec > 0 ? hot_gib / sec : 0.0,
           sec > 0 ? touch_mpps / sec : 0.0,
           (unsigned long long)sink);

    munmap(p, bytes);
    return 0;
}
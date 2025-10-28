//go:build ignore
#include "../../headers/vmlinux.h"
#include "../../headers/common.h"
#include "../../headers/cuda_types.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

char __license[] SEC("license") = "Dual MIT/GPL";

struct gpu_event_t {
  __u32 pid;
  u8 comm[TASK_COMM_SIZE];
  __u64 total_blocks;
  __u64 threads_block;
  __u64 total_threads;
};

struct
{
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} gpu_ringbuf SEC(".maps");


const struct gpu_event_t *unused __attribute__((unused));

SEC("uprobe/cudaLaunchKernel")
int handle_cuda_launchkernel(struct pt_regs *ctx){
  
  __u32 pid = bpf_get_current_pid_tgid() >> 32;

  struct gpu_event_t *e;
  e = bpf_ringbuf_reserve(&gpu_ringbuf,sizeof(struct gpu_event_t),0);
  if (!e) return 0;

  e->pid = pid;

  bpf_get_current_comm(&e->comm, TASK_COMM_SIZE);

  struct dim3 grid = {};
  struct dim3 block = {};

  bpf_probe_read_user(&grid,sizeof(grid), (void *)PT_REGS_PARM2(ctx));
  bpf_probe_read_user(&block,sizeof(block), (void *)PT_REGS_PARM3(ctx));

  e->total_blocks  = (__u64)grid.x * grid.y * grid.z;
  e->threads_block = (__u64)block.x * block.y * block.z;
  e->total_threads = e->total_blocks * e->threads_block;

  bpf_ringbuf_submit(e,0);
  return 0;
}

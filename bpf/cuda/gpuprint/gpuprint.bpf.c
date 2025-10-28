//go:build ignore
#include "../../headers/vmlinux.h"
#include "../../headers/common.h"
#include "../../headers/cuda_types.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

char __license[] SEC("license") = "Dual MIT/GPL";

struct gpu_kernel_launch_event_t {
  __u8 flag;
  u8 _pad[3];

  __u32 pid;
  u8 comm[TASK_COMM_SIZE];

  __u32 gridx ;
  __u32 gridy ;
  __u32 gridz ;
  __u32 blockx ;
  __u32 blocky ;
  __u32 blockz ;

  __u64 total_blocks;
  __u64 threads_block;
  __u64 total_threads;
};

struct gpu_memalloc_event_t{
  __u8 flag;
  u8 _pad[3];

  __u32 pid;
  u8 comm[TASK_COMM_SIZE];

  size_t byte_size;
};

struct
{
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} gpu_ringbuf SEC(".maps");


const struct gpu_kernel_launch_event_t *unused __attribute__((unused));

SEC("uprobe/cuLaunchKernel")
int BPF_KPROBE(handle_cuLaunchkernel,
    u64 func,
    u64 gridX, u64 gridY, u64 gridZ,
    u64 blockX, u64 blockY){
  __u32 pid = bpf_get_current_pid_tgid() >> 32;

  struct gpu_kernel_launch_event_t *e;
  e = bpf_ringbuf_reserve(&gpu_ringbuf,sizeof(struct gpu_kernel_launch_event_t),0);
  if (!e) return 0;

  e->pid = pid;

  e->flag = EVENT_GPU_KERNEL_LAUNCH;

  bpf_get_current_comm(&e->comm, sizeof(e->comm));

  u64 blockZ = 0;

  bpf_probe_read_user(&blockZ, sizeof(blockZ), (void *)(PT_REGS_SP(ctx) + 8));

  e->gridx = gridX;
  e->gridy = gridY;
  e->gridz = gridZ;
  e->blockx = blockX;
  e->blocky = blockY;
  e->blockz = blockZ;

  e->total_blocks  = (__u64)gridX * gridY * gridZ;
  e->threads_block = (__u64)blockX * blockY * blockZ;
  e->total_threads = e->total_blocks * e->threads_block;

  bpf_ringbuf_submit(e,0);

  return 0;
}

SEC("uprobe/cuMemAlloc")
int BPF_KPROBE(handle_cuMemAlloc, void **devptr, size_t bytesize){

  __u32 pid = bpf_get_current_pid_tgid() >> 32;

  struct gpu_memalloc_event_t *e;
  e = bpf_ringbuf_reserve(&gpu_ringbuf,sizeof(struct gpu_memalloc_event_t),0);
  if (!e) return 0;

  e->pid = pid;

  e->flag = EVENT_GPU_MALLOC;

  bpf_get_current_comm(&e->comm, sizeof(e->comm));

  e->byte_size = bytesize;

  bpf_ringbuf_submit(e,0);

  return 0;
}

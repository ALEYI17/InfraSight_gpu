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

struct gpu_memcpy_event_t{
  __u8 flag;
  u8 _pad[3];

  __u32 pid;
  u8 comm[TASK_COMM_SIZE];

  size_t byte_size;
  u8 kind;
};

struct gpu_stream_event_t{
  __u8 flag;
  u8 _pad[3];

  __u32 pid;
  u8 comm[TASK_COMM_SIZE];
  __u64 start_time;
  __u64 end_time;
  __u64 delta_ns;
};

struct ioctl_watchdog_event_t{
  __u64 ioctl_hit_count;
  __u64 uprobe_hit_count;
  __u64 first_seen_time;
};

struct{
  __uint(type, BPF_MAP_TYPE_HASH);
  __type(key, __u64);
  __type(value, __u64);
  __uint(max_entries, 1024);
} start_events_stream SEC(".maps");

struct{
  __uint(type, BPF_MAP_TYPE_PERCPU_HASH);
  __type(key, __u32);
  __type(value, struct ioctl_watchdog_event_t);
  __uint(max_entries, 1024);
} ioctl_watchdog_map SEC(".maps");

struct
{
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} gpu_ringbuf SEC(".maps");


const struct gpu_kernel_launch_event_t *unused __attribute__((unused));
const struct gpu_memalloc_event_t *unused2 __attribute__((unused));
const struct gpu_memcpy_event_t *unused3 __attribute__((unused));
const struct gpu_stream_event_t *unused4 __attribute__((unused));
const struct  ioctl_watchdog_event_t *unused5 __attribute__((unused));

static __always_inline struct ioctl_watchdog_event_t * get_or_init_ioctl(__u32 pid){

  struct ioctl_watchdog_event_t *e;

  e = bpf_map_lookup_elem(&ioctl_watchdog_map,&pid);

  if (e)
    return e;
  

  struct ioctl_watchdog_event_t zero ={
    .ioctl_hit_count = 0,
    .uprobe_hit_count = 0,
    .first_seen_time = bpf_ktime_get_ns()
  };
  bpf_map_update_elem(&ioctl_watchdog_map,&pid,&zero,BPF_NOEXIST);

  return bpf_map_lookup_elem(&ioctl_watchdog_map,&pid);
}

static __always_inline int add_to_counter(__u32 pid, __u32 hit_type){

  struct ioctl_watchdog_event_t *hit = get_or_init_ioctl(pid);

  if(hit){
    if (hit_type == UPROBE_HIT){
      __sync_fetch_and_add(&hit->uprobe_hit_count,1);
    }
    else if(hit_type == IOCTL_HIT){
      __sync_fetch_and_add(&hit->ioctl_hit_count,1);
    }

  }

  return 0;
}

static __always_inline int
is_nvidia_compute_device(__u32 major, __u32 minor)
{
    if (major == NVIDIA_UVM_MAJOR)
        return 1;
    if (major == NVIDIA_MAJOR && minor < NVIDIA_MODESET_MINOR)
        return 1;
    return 0;
}

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

  add_to_counter(pid,UPROBE_HIT);
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

  add_to_counter(pid,UPROBE_HIT);
  return 0;
}


SEC("uprobe/cuMemcpyHtoD")
int BPF_KPROBE(handle_cuMemcpy_htod, void **dst, const void *src, size_t bytesize){
  __u32 pid = bpf_get_current_pid_tgid() >> 32;

  struct gpu_memcpy_event_t *e;
  e = bpf_ringbuf_reserve(&gpu_ringbuf,sizeof(struct gpu_memcpy_event_t),0);
  if (!e) return 0;

  e->pid = pid;

  e->flag = EVENT_GPU_MEMCPY;

  bpf_get_current_comm(&e->comm, sizeof(e->comm));

  e->byte_size = bytesize;

  e->kind = DIR_HTOD;

  bpf_ringbuf_submit(e,0);

  add_to_counter(pid,UPROBE_HIT);
  return 0;
}

SEC("uprobe/cuMemcpyDtoH")
int BPF_KPROBE(handle_cuMemcpy_dtoh, void *dst, void **src, size_t bytesize){
  __u32 pid = bpf_get_current_pid_tgid() >> 32;

  struct gpu_memcpy_event_t *e;
  e = bpf_ringbuf_reserve(&gpu_ringbuf,sizeof(struct gpu_memcpy_event_t),0);
  if (!e) return 0;

  e->pid = pid;

  e->flag = EVENT_GPU_MEMCPY;

  bpf_get_current_comm(&e->comm, sizeof(e->comm));

  e->byte_size = bytesize;

  e->kind = DIR_DTOH;

  bpf_ringbuf_submit(e,0);

  add_to_counter(pid,UPROBE_HIT);
  return 0;
}

SEC("uprobe/cuMemcpyHtoDAsync")
int BPF_KPROBE(handle_cuMemcpy_htod_async, void **dst, const void *src, size_t bytesize, cudaStream_t hStream){
  __u32 pid = bpf_get_current_pid_tgid() >> 32;

  struct gpu_memcpy_event_t *e;
  e = bpf_ringbuf_reserve(&gpu_ringbuf,sizeof(struct gpu_memcpy_event_t),0);
  if (!e) return 0;

  e->pid = pid;

  e->flag = EVENT_GPU_MEMCPY;

  bpf_get_current_comm(&e->comm, sizeof(e->comm));

  e->byte_size = bytesize;

  e->kind = DIR_HTOD;

  bpf_ringbuf_submit(e,0);

  add_to_counter(pid,UPROBE_HIT);
  return 0;
}

SEC("uprobe/cuMemcpyDtoHAsync")
int BPF_KPROBE(handle_cuMemcpy_dtohAsync, void *dst, void **src, size_t bytesize, cudaStream_t hStream){
  __u32 pid = bpf_get_current_pid_tgid() >> 32;

  struct gpu_memcpy_event_t *e;
  e = bpf_ringbuf_reserve(&gpu_ringbuf,sizeof(struct gpu_memcpy_event_t),0);
  if (!e) return 0;

  e->pid = pid;

  e->flag = EVENT_GPU_MEMCPY;

  bpf_get_current_comm(&e->comm, sizeof(e->comm));

  e->byte_size = bytesize;

  e->kind = DIR_DTOH;

  bpf_ringbuf_submit(e,0);

  add_to_counter(pid,UPROBE_HIT);
  return 0;
}

SEC("uprobe/cuStreamSynchronize")
int BPF_KPROBE(handle_cuStreamSync, cudaStream_t hStream){

  __u64 id = bpf_get_current_pid_tgid(); 
  __u64 ts = bpf_ktime_get_ns();

  bpf_map_update_elem(&start_events_stream,&id,&ts,BPF_ANY);
  return 0;
}

SEC("uretprobe/cuStreamSynchronize")
int BPF_KRETPROBE(handle_cuStreamSynchronize_ret){

  struct gpu_stream_event_t *e;

  __u64 id = bpf_get_current_pid_tgid();

  __u64 *tsp = bpf_map_lookup_elem(&start_events_stream, &id);
  if (!tsp) {
      return 0;
  }

  e = bpf_ringbuf_reserve(&gpu_ringbuf, sizeof(*e), 0);
    if (!e) return 0;
  
  __u32 pid = id >> 32;
  e->pid = pid;

  e->start_time = *tsp;

  e->end_time = bpf_ktime_get_ns();

  e->delta_ns =  e->end_time - e->start_time;

  e->flag = EVENT_GPU_STREAM_SYNC;

  bpf_get_current_comm(&e->comm, sizeof(e->comm));

  bpf_map_delete_elem(&start_events_stream, &id);

  bpf_ringbuf_submit(e, 0);

  add_to_counter(pid,UPROBE_HIT);
  return 0;
}

SEC("uprobe/cuCtxSynchronize")
int BPF_KPROBE(handle_cuCtxSync){

  __u64 id = bpf_get_current_pid_tgid(); 
  __u64 ts = bpf_ktime_get_ns();

  bpf_map_update_elem(&start_events_stream,&id,&ts,BPF_ANY);
  return 0;
}

SEC("uretprobe/cuCtxSynchronize")
int BPF_KRETPROBE(handle_cuCtxSync_ret){

  struct gpu_stream_event_t *e;

  __u64 id = bpf_get_current_pid_tgid();

  __u64 *tsp = bpf_map_lookup_elem(&start_events_stream, &id);
  if (!tsp) {
      return 0;
  }

  e = bpf_ringbuf_reserve(&gpu_ringbuf, sizeof(*e), 0);
    if (!e) return 0;
  
  __u32 pid = id>> 32;
  e->pid = pid;

  e->start_time = *tsp;

  e->end_time = bpf_ktime_get_ns();

  e->delta_ns =  e->end_time - e->start_time;

  e->flag = EVENT_GPU_STREAM_SYNC;

  bpf_get_current_comm(&e->comm, sizeof(e->comm));

  bpf_map_delete_elem(&start_events_stream, &id);

  bpf_ringbuf_submit(e, 0);

  add_to_counter(pid, UPROBE_HIT);
  return 0;
}


SEC("tracepoint/syscalls/sys_enter_ioctl")
int watchdog_ioctl(struct trace_event_raw_sys_enter *ctx){

  __u32 pid = bpf_get_current_pid_tgid() >>32; 
  __u64 ts = bpf_ktime_get_ns();

  __u64 fd = (__u64)ctx->args[0];

  struct task_struct *task = (void *)bpf_get_current_task();

  struct fdtable *fdt    = BPF_CORE_READ(task, files, fdt);
  if (!fdt)
    return 0;

  struct file **fd_array = BPF_CORE_READ(fdt,fd);
  if (!fd_array)
    return 0;

  struct file *file = NULL;
  __u64 fd_addr = (__u64)fd_array + (__u64)fd * sizeof(struct file *);
  bpf_core_read(&file, sizeof(struct file *), (void *)fd_addr);
  if (!file)
    return 0;

  dev_t rdev = BPF_CORE_READ(file, f_inode, i_rdev);

  // kernel MINORBITS = 20
  __u32 major = rdev >> 20;
  __u32 minor = rdev & 0xfffff;

  if (!is_nvidia_compute_device(major, minor))
    return 0;

  add_to_counter(pid, IOCTL_HIT);
  return 0;
}

SEC("tracepoint/sched/sched_process_exit")
int handle_process_exit(struct trace_event_raw_sched_process_template *ctx){
  
  __u32 pid = bpf_get_current_pid_tgid() >> 32;
  bpf_map_delete_elem(&ioctl_watchdog_map, &pid);
  return 0;
}

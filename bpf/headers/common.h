#ifndef __COMMON_H__
#define __COMMON_H__

#ifndef TASK_COMM_SIZE
#define TASK_COMM_SIZE 150
#endif

#ifndef PATH_MAX
#define PATH_MAX 256
#endif

#ifndef PAGE_SHIFT
#define PAGE_SHIFT 12
#endif

#define EVENT_GPU_KERNEL_LAUNCH 1
#define EVENT_GPU_MALLOC 2
#define EVENT_GPU_MEMCPY 3
#define EVENT_GPU_STREAM_SYNC 4

#define DIR_HTOD 0
#define DIR_DTOH 1

#define IOCTL_HIT  20
#define UPROBE_HIT 10
#define NVIDIA_MAJOR       195
#define NVIDIA_UVM_MAJOR   511
#define NVIDIA_MODESET_MINOR 254  // exclude display
#endif /* __COMMON_H__ */


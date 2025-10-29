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
#define DIR_HTOD 0
#define DIR_DTOH 1
#endif /* __COMMON_H__ */


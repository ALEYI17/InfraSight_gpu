#ifndef __CUDA_TYPES_H__
#define __CUDA_TYPES_H__


struct dim3 {
    __u32 x;
    __u32 y;
    __u32 z;
};

typedef __u64 cudaStream_t;

#endif /* __CUDA_TYPES_H__ */


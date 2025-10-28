from numba import cuda
import numpy as np
import time

# Define a simple CUDA kernel
@cuda.jit
def add_kernel(a, b, c):
    idx = cuda.grid(1)
    if idx < a.size:
        c[idx] = a[idx] + b[idx]

# Prepare data
n = 1024 * 1024
a = np.ones(n, dtype=np.float32)
b = np.ones(n, dtype=np.float32)
c = np.zeros(n, dtype=np.float32)

# Copy to device
da = cuda.to_device(a)
db = cuda.to_device(b)
dc = cuda.to_device(c)

# Define grid and block sizes
threads_per_block = 256
blocks_per_grid = (n + threads_per_block - 1) // threads_per_block

# --- Launch kernel (this triggers cudaLaunchKernel) ---
add_kernel[blocks_per_grid, threads_per_block](da, db, dc)

# Wait a bit so you can observe
time.sleep(10)


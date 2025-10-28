package gpuprint

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -target amd64 -type gpu_kernel_launch_event_t -type gpu_memalloc_event_t Gpuprint gpuprint.bpf.c

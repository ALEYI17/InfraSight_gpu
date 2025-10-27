package gpuprint

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -target amd64 -type gpu_event_t Gpuprint gpuprint.bpf.c

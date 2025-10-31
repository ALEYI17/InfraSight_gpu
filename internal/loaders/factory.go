package loaders

import (
	"errors"

	"github.com/ALEYI17/InfraSight_gpu/pkg/types"
)

func NewEbpfGpuLoaders(programs string) (types.Gpu_loaders ,error){

  switch programs{
    case types.LoaderFingerprint:
      return NewGpuprinterLoader()
    default:
      return nil , errors.New("Unsuported or unknow program")
  }
}

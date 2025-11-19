package loaders

import (
	"errors"
	"time"

	"github.com/ALEYI17/InfraSight_gpu/internal/collector/aggregator"
	"github.com/ALEYI17/InfraSight_gpu/internal/collector/timeserie"
	"github.com/ALEYI17/InfraSight_gpu/internal/config"
	"github.com/ALEYI17/InfraSight_gpu/pkg/types"
)

func NewEbpfGpuLoaders(programs string, cfg *config.Programsconfig) (types.Gpu_loaders, error) {

	switch programs {
	case types.LoaderFingerprint:
		c1 := aggregator.NewGPUAggregator(2 * time.Second)
		c2 := timeserie.NewTimeSeriesCollector(5 * time.Second)
		return NewGpuprinterLoader(cfg,c1, c2)
	default:
		return nil, errors.New("Unsuported or unknow program")
	}
}

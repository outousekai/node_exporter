// Copyright 2016 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !nonvidiagpu
// +build !nonvidiagpu

package collector

import (
	"fmt"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/mindprince/gonvml"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
)

var (
	nvidiaGPULabelNames = []string{"ID", "gpu"}
)

type nvidiaGPUCollector struct {
	gpuTemperatureDesc *prometheus.Desc
	gpuUsageDesc       *prometheus.Desc
	gpuMemoryUsedDesc  *prometheus.Desc
	gpuMemoryTotalDesc *prometheus.Desc
	gpuPowerDesc       *prometheus.Desc
	logger             log.Logger
}

func init() {
	registerCollector("nvidia_gpu", defaultDisabled, NewNvidiaGPUCollector)
}

// NewNvidiaGPUCollector returns a new Collector exposing nvidia gpu.
func NewNvidiaGPUCollector(logger log.Logger) (Collector, error) {
	subsystem := "nvidia_gpu"

	gpuTemperatureDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, "temperature_celsius"),
		"nvidia gpu temperature celsius",
		nvidiaGPULabelNames, nil,
	)

	gpuUsageDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, "usage_percent"),
		"nvidia gpu usage percent",
		nvidiaGPULabelNames, nil,
	)

	gpuMemoryUsedDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, "memory_used"),
		"nvidia memory used ",
		nvidiaGPULabelNames, nil,
	)

	gpuMemoryTotalDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, "memory_total"),
		"gpu memory total",
		nvidiaGPULabelNames, nil,
	)

	gpuPowerDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, "power"),
		"gpu power",
		nvidiaGPULabelNames, nil,
	)

	return &nvidiaGPUCollector{
		gpuTemperatureDesc: gpuTemperatureDesc,
		gpuUsageDesc:       gpuUsageDesc,
		gpuMemoryUsedDesc:  gpuMemoryUsedDesc,
		gpuMemoryTotalDesc: gpuMemoryTotalDesc,
		gpuPowerDesc:       gpuPowerDesc,
		logger:             logger,
	}, nil
}

func Check(f func() error) {
	if err := f(); err != nil {
		fmt.Println("Received error:", err)
	}
}

func (c *nvidiaGPUCollector) Update(ch chan<- prometheus.Metric) error {
	// 初始化 NVML 库
	err := gonvml.Initialize()
	if err != nil {
		level.Error(c.logger).Log(
			"msg", "Failed to initialize NVML",
			"err", err)
		return nil
	}
	defer gonvml.Shutdown()

	// 获取 GPU 数量
	deviceCount, err := gonvml.DeviceCount()
	if err != nil {
		level.Error(c.logger).Log("Failed to get device count",
			"err", err)
	}
	// 遍历每个 GPU 设备
	for i := uint(0); i < deviceCount; i++ {
		device, err := gonvml.DeviceHandleByIndex(i)
		if err != nil {
			level.Error(c.logger).Log("Failed to get device handle for GPU %d: %v",
				"id", i,
				"err", err)
			continue
		}
		name, err := device.Name()
		if err != nil {
			level.Error(c.logger).Log("Get name for gpu error %d: %v",
				"id", i,
				"err", err)
			continue
		}
		// 获取 GPU 温度
		temperature, err := device.Temperature()
		if err != nil {
			level.Error(c.logger).Log("Failed to get temperature for GPU %d: %v",
				"id", i,
				"err", err)
			continue
		}

		// 获取 GPU 使用率
		utilization, _, err := device.UtilizationRates()
		if err != nil {
			level.Error(c.logger).Log("Failed to get utilization for GPU %d: %v",
				"id", i,
				"err", err)
			continue
		}

		// 获取 GPU 显存使用情况
		memory, used_memory, err := device.MemoryInfo()
		if err != nil {
			level.Error(c.logger).Log("Failed to get memory info for GPU %d: %v",
				"id", i,
				"err", err)
			continue
		}
		power, err := device.PowerUsage()
		if err != nil {
			level.Error(c.logger).Log("Failed to get memory info for GPU %d: %v", "id",
				"id", i,
				"err", err)
			continue
		}

		// 发送温度指标
		ch <- prometheus.MustNewConstMetric(
			c.gpuTemperatureDesc,
			prometheus.GaugeValue,
			float64(temperature),
			strconv.Itoa(int(i)), name,
		)

		// 发送 GPU 使用率指标
		ch <- prometheus.MustNewConstMetric(
			c.gpuUsageDesc,
			prometheus.GaugeValue,
			float64(utilization),
			strconv.Itoa(int(i)), name,
		)

		// 发送 GPU 显存使用量指标
		ch <- prometheus.MustNewConstMetric(
			c.gpuMemoryUsedDesc,
			prometheus.GaugeValue,
			float64(used_memory),
			strconv.Itoa(int(i)), name,
		)
		ch <- prometheus.MustNewConstMetric(
			c.gpuMemoryTotalDesc,
			prometheus.GaugeValue,
			float64(memory),
			strconv.Itoa(int(i)), name,
		)
		ch <- prometheus.MustNewConstMetric(
			c.gpuPowerDesc,
			prometheus.GaugeValue,
			float64(power),
			strconv.Itoa(int(i)), name,
		)
	}
	return nil
}

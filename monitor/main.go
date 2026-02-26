package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type Stats struct {
	Hostname    string  `json:"hostname"`
	OS          string  `json:"os"`
	Uptime      uint64  `json:"uptime"`
	CPUUsage    float64 `json:"cpu_usage"`
	Load1       float64 `json:"load1"`
	Load5       float64 `json:"load5"`
	Load15      float64 `json:"load15"`
	RAMTotal    uint64  `json:"ram_total"`
	RAMUsed     uint64  `json:"ram_used"`
	RAMFree     uint64  `json:"ram_free"`
	RAMPercent  float64 `json:"ram_percent"`
	DiskTotal   uint64  `json:"disk_total"`
	DiskUsed    uint64  `json:"disk_used"`
	DiskFree    uint64  `json:"disk_free"`
	DiskPercent float64 `json:"disk_percent"`
	NetSent     uint64  `json:"net_sent"`
	NetRecv     uint64  `json:"net_recv"`
	Timestamp   string  `json:"timestamp"`
}

var lastNetSent, lastNetRecv uint64
var lastNetTime time.Time

func getStats() Stats {
	h, _ := host.Info()
	c, _ := cpu.Percent(0, false)
	l, _ := load.Avg()
	m, _ := mem.VirtualMemory()
	d, _ := disk.Usage("/")
	n, _ := net.IOCounters(false)

	var cpuVal float64
	if len(c) > 0 {
		cpuVal = c[0]
	}

	currentTime := time.Now()
	var sentDelta, recvDelta uint64
	if !lastNetTime.IsZero() {
		duration := currentTime.Sub(lastNetTime).Seconds()
		if duration > 0 && len(n) > 0 {
			sentDelta = uint64(float64(n[0].BytesSent-lastNetSent) / duration)
			recvDelta = uint64(float64(n[0].BytesRecv-lastNetRecv) / duration)
		}
	}

	if len(n) > 0 {
		lastNetSent = n[0].BytesSent
		lastNetRecv = n[0].BytesRecv
	}
	lastNetTime = currentTime

	return Stats{
		Hostname:    h.Hostname,
		OS:          fmt.Sprintf("%s %s", h.OS, h.PlatformVersion),
		Uptime:      h.Uptime,
		CPUUsage:    cpuVal,
		Load1:       l.Load1,
		Load5:       l.Load5,
		Load15:      l.Load15,
		RAMTotal:    m.Total,
		RAMUsed:     m.Used,
		RAMFree:     m.Available,
		RAMPercent:  m.UsedPercent,
		DiskTotal:   d.Total,
		DiskUsed:    d.Used,
		DiskFree:    d.Free,
		DiskPercent: d.UsedPercent,
		NetSent:     sentDelta, // Bytes per second
		NetRecv:     recvDelta, // Bytes per second
		Timestamp:   currentTime.Format("15:04:05"),
	}
}

func main() {
	app := fiber.New()
	app.Use(cors.New())

	app.Get("/status", func(c *fiber.Ctx) error {
		return c.JSON(getStats())
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	log.Printf("Monitor service starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}

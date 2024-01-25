package collector

import (
	"flag"
	"github.com/prometheus/procfs"
	"path/filepath"
	"strings"
)

var (
	procPath   = flag.String("path.procfs", procfs.DefaultMountPoint, "procfs mountpoint.")
	rootfsPath = flag.String("path.rootfs", "/", "procfs mountpoint.")
	sysPath    = flag.String("path.sysfs", "/sys", "sysfs mountpoint.")
)

func procFilePath(name string) string {
	return filepath.Join(*procPath, name)
}

func rootfsFilePath(name string) string {
	return filepath.Join(*rootfsPath, name)
}

func rootfsStripPrefix(path string) string {
	if *rootfsPath == "/" {
		return path
	}
	stripped := strings.TrimPrefix(path, *rootfsPath)
	if stripped == "" {
		return "/"
	}
	return stripped
}

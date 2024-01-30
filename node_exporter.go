package main

import (
	"fmt"
	"github.com/zhaoqiang0201/node_exporter/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
	}
}

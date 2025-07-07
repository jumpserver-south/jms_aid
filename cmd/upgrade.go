package cmd

import (
	"jms_tools/pkg/service"
	"strings"

	"github.com/spf13/cobra"
)

var (
	filepath string
	workers  int
	process  string
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "v2.28.20+ 升级 v3.10.x 处理程序",
	Long:  "v2.28.20+ 升级 v3.10.x 处理程序",
	PreRun: func(cmd *cobra.Command, args []string) {
		
	},
	Run: func(cmd *cobra.Command, args []string) {
		js := service.NewJmsService(filepath, workers)
		switch strings.ToLower(process) {
		case "pre":
			js.PreProcessing()
		case "post":
			js.PostProcessing()
		default:
			js.AutoRun()
		}
	},
}

func init() {
	upgradeCmd.Flags().StringVarP(&filepath, "file", "f", "/opt/jumpserver/config/config.txt", "指定 JumpServer 的配置文件路径")
	upgradeCmd.Flags().IntVarP(&workers, "workers", "w", 10, "预处理账号合并时并发处理数，默认 10")
	upgradeCmd.Flags().StringVarP(&process, "process", "p", "", "处理流程, 可选值 pre: 升级预处理, post: 升级后处理, 为空时智能判断")

	rootCmd.AddCommand(upgradeCmd)
}

package cmd

import (
	"jms_tools/pkg/service"

	"github.com/spf13/cobra"
)

var (
	queryFilepath string
	queryWorkers  int
	queryStrategy string
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "数据查询工具",
	Long:  "数据查询工具，支持检测不活跃资产等",
}

var unActiveCmd = &cobra.Command{
	Use:   "unactive",
	Short: "检测不活跃资产（端口不可达）",
	Long:  "从数据库获取资产及端口信息，通过 telnet 方式检测端口连通性。strategy=export 时导出 CSV 文件，strategy=disabled 时通过 SQL 禁用资产",
	Run: func(cmd *cobra.Command, args []string) {
		qs := service.NewQueryService(queryFilepath, queryWorkers, queryStrategy)
		qs.ListUnActiveAssets()
	},
}

func init() {
	queryCmd.PersistentFlags().StringVarP(&queryFilepath, "file", "f", "/opt/jumpserver/config/config.txt", "指定 JumpServer 的配置文件路径")
	queryCmd.PersistentFlags().IntVarP(&queryWorkers, "workers", "w", 10, "并发处理数，默认 10")
	unActiveCmd.Flags().StringVarP(&queryStrategy, "strategy", "s", "export", "处理策略：export 导出 CSV 文件，disabled 禁用资产")

	queryCmd.AddCommand(unActiveCmd)
	rootCmd.AddCommand(queryCmd)
}

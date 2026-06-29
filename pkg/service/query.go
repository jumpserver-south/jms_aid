package service

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
)

type AssetPort struct {
	ID      int64
	Name    string
	Address string
	Ports   string
}

type UnActiveAsset struct {
	ID      int64
	Name    string
	Address string
	Port    string
}

// QueryService 资产查询服务
type QueryService struct {
	db       *gorm.DB
	worker   int
	strategy string
}

func NewQueryService(configpath string, worker int, strategy string) *QueryService {
	initLogger()
	initConfig(configpath)
	return &QueryService{
		db:       initDB(),
		worker:   worker,
		strategy: strategy,
	}
}

func (s *QueryService) getAssetBaseSQL() string {
	if appConfig.IsPostgreSQL() {
		return `SELECT a.id, a.name, a.address, STRING_AGG(DISTINCT p.port::text, ',') AS ports
FROM assets_asset AS a
INNER JOIN assets_protocol AS p ON a.id = p.asset_id
GROUP BY a.id, a.name, a.address
ORDER BY a.id`
	}
	return `SELECT a.id, a.name, a.address, GROUP_CONCAT(DISTINCT p.port ORDER BY p.port ASC SEPARATOR ',') AS ports
FROM assets_asset AS a
INNER JOIN assets_protocol AS p ON a.id = p.asset_id
GROUP BY a.id, a.name, a.address
ORDER BY a.id`
}

func (s *QueryService) fetchAssetsBatch(ch chan<- []AssetPort) {
	defer close(ch)

	batchSize := 100
	offset := 0
	baseSQL := s.getAssetBaseSQL()

	for {
		sql := fmt.Sprintf("%s LIMIT %d OFFSET %d", baseSQL, batchSize, offset)
		rows, err := s.db.Raw(sql).Rows()
		if err != nil {
			slog.Error(fmt.Sprintf("查询资产端口信息失败：%s", err.Error()))
			return
		}

		batch := make([]AssetPort, 0, batchSize)
		for rows.Next() {
			var asset AssetPort
			if err := rows.Scan(&asset.ID, &asset.Name, &asset.Address, &asset.Ports); err != nil {
				slog.Error(fmt.Sprintf("扫描资产行失败：%s", err.Error()))
				continue
			}
			batch = append(batch, asset)
		}
		rows.Close()

		if len(batch) == 0 {
			return
		}
		ch <- batch

		if len(batch) < batchSize {
			return
		}
		offset += batchSize
	}
}

func isPortReachable(address, port string) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(address, port), 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// ListUnActiveAssets 列举不活跃资产，通过 telnet 方式检测资产端口连通性
// strategy=export 时导出 CSV 文件，strategy=disabled 时通过 SQL 禁用资产
func (s *QueryService) ListUnActiveAssets() {
	start := time.Now()
	fmt.Println("开始检测不活跃资产...")

	assetCh := make(chan []AssetPort, 10)
	go s.fetchAssetsBatch(assetCh)

	unActiveCh := make(chan UnActiveAsset, 100)
	var totalAssets int64
	var totalUnActive int64

	done := make(chan struct{})
	go func() {
		defer close(done)
		if s.strategy == "disabled" {
			count := s.disableAssets(unActiveCh)
			atomic.StoreInt64(&totalUnActive, count)
		} else {
			csvPath := getFilepath("unactive_assets.csv")
			count, err := writeCSV(csvPath, unActiveCh)
			if err != nil {
				slog.Error(fmt.Sprintf("写入 CSV 文件失败：%s", err.Error()))
				return
			}
			atomic.StoreInt64(&totalUnActive, count)
			if count > 0 {
				fmt.Printf("结果已保存至：%s\n", csvPath)
			}
		}
	}()

	for batch := range assetCh {
		for _, asset := range batch {
			atomic.AddInt64(&totalAssets, 1)
			ports := strings.Split(asset.Ports, ",")
			var wg sync.WaitGroup
			for _, port := range ports {
				port = strings.TrimSpace(port)
				if port == "" {
					continue
				}
				wg.Add(1)
				go func(id int64, name, address, p string) {
					defer wg.Done()
					if !isPortReachable(address, p) {
						unActiveCh <- UnActiveAsset{
							ID:      id,
							Name:    name,
							Address: address,
							Port:    p,
						}
					}
				}(asset.ID, asset.Name, asset.Address, port)
			}
			wg.Wait()
		}
	}
	close(unActiveCh)
	<-done

	assets := atomic.LoadInt64(&totalAssets)
	unActive := atomic.LoadInt64(&totalUnActive)
	duration := time.Since(start).Seconds()

	if unActive == 0 {
		fmt.Println("所有资产端口均可达，无不活跃资产。")
	} else {
		action := "不可达端口"
		if s.strategy == "disabled" {
			action = "已禁用资产"
		}
		fmt.Printf("检测完成，共扫描 %d 个资产，发现 %d 个%s，耗时 %.2fs\n", assets, unActive, action, duration)
	}
}

func (s *QueryService) disableAssets(ch <-chan UnActiveAsset) int64 {
	seen := make(map[int64]bool)
	ids := make([]int64, 0)
	for a := range ch {
		if !seen[a.ID] {
			seen[a.ID] = true
			ids = append(ids, a.ID)
		}
	}
	if len(ids) == 0 {
		return 0
	}
	const disableComment = "Disabled: Port unreachable detected."

	var concatExpr string
	if appConfig.IsPostgreSQL() {
		concatExpr = "comment || ' " + disableComment + "'"
	} else {
		concatExpr = "CONCAT(comment, ' " + disableComment + "')"
	}
	sql := fmt.Sprintf("UPDATE assets_asset SET is_active = false, comment = %s WHERE id IN ? AND comment NOT LIKE ?", concatExpr)

	result := s.db.Exec(sql, ids, "%"+disableComment+"%")
	if result.Error != nil {
		slog.Error(fmt.Sprintf("禁用资产失败：%s", result.Error.Error()))
		return 0
	}
	fmt.Printf("已禁用 %d 个资产\n", result.RowsAffected)
	return int64(len(ids))
}

func writeCSV(filepath string, ch <-chan UnActiveAsset) (int64, error) {
	f, err := os.Create(filepath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	f.WriteString("\xEF\xBB\xBF")

	w := csv.NewWriter(f)
	if err := w.Write([]string{"资产名称", "资产地址", "不可达端口"}); err != nil {
		return 0, err
	}

	var count int64
	for a := range ch {
		if err := w.Write([]string{a.Name, a.Address, a.Port}); err != nil {
			return count, err
		}
		count++
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return count, err
	}

	if count == 0 {
		os.Remove(filepath)
	}
	return count, nil
}

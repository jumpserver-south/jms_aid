package service

import (
	"fmt"
	"io"
	"jms_tools/pkg/common"
	"jms_tools/pkg/config"
	"jms_tools/pkg/sdk/utils/crypto"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	appConfig *config.AppConfig
	//logger    *common.Logger
	js *JmsService
)

func initConfig(filepath string) {
	// 默认文件 /opt/jumpserver/config/config.txt
	configMap, err := common.ConfigFileToMap(filepath)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	appConfig = new(config.AppConfig)
	appConfig.SecretKey = configMap["SECRET_KEY"]
	appConfig.DBHost = getHostFromDocker(configMap["DB_HOST"])
	appConfig.DBPort = configMap["DB_PORT"]
	appConfig.DBUser = configMap["DB_USER"]
	appConfig.DBPassword = configMap["DB_PASSWORD"]
	appConfig.DBName = configMap["DB_NAME"]
}

func getHostFromDocker(host string) string {
	container := ""
	if host == "mysql" {
		container = "jms_mysql"
	}
	if container != "" {
		finCommand := fmt.Sprintf("docker inspect -f '{{.NetworkSettings.Networks.jms_net.IPAddress}}' %s", container)
		cmd := exec.Command("sh", "-c", finCommand)
		if ret, err := cmd.CombinedOutput(); err == nil {
			ipv4Regex := regexp.MustCompile(`([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})`)
			matches := ipv4Regex.FindStringSubmatch(string(ret))
			if len(matches) > 1 {
				host = matches[1]
			}
		}
	}
	return host
}

func initDB() (db *gorm.DB) {
	var err error
	db, err = gorm.Open(mysql.Open(appConfig.DBUri()), &gorm.Config{
		// 关闭所有日志（包括 Slow SQL）
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		slog.Error(fmt.Sprintf("数据库初始化失败：%s", err.Error()))
		os.Exit(1)
	}
	return db
}

func initLogger() {
	exepath, err := os.Executable()
	if err != nil {
		slog.Error(fmt.Sprintf("获取可执行文件路径失败：%s", err.Error()))
		os.Exit(1)
	}
	//logger = common.GetLogger()
	logfile := filepath.Join(filepath.Dir(exepath), "jumpserver.log")
	f, err := os.Create(logfile)
	if err != nil {
		panic(err)
	}
	multiWriter := io.MultiWriter(os.Stdout, f)
	handler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		AddSource:   true,
		Level:       slog.LevelInfo,
		ReplaceAttr: nil,
	})
	slog.SetDefault(slog.New(handler))
}

type JmsService struct {
	db     *gorm.DB
	parser crypto.ICrypto
}

func initJmsService() *JmsService {
	s := new(JmsService)
	s.db = initDB()
	s.parser = crypto.NewAESGcmCrypto(appConfig.SecretKey)
	return s
}

func NewJmsService(configpath string, workers int) *JmsService {
	initLogger()
	initConfig(configpath)
	appConfig.Workers = workers
	if js == nil {
		js = initJmsService()
	}
	return js
}

// AutoRun 执行 JMS 服务的主要流程：
// 1. 检查标记文件是否存在，若不存在或路径为目录则执行预处理
// 2. 若标记文件存在且为普通文件则执行后处理
// 3. 遇到非"文件不存在"错误时记录日志并终止程序
func (s *JmsService) AutoRun() {
	fi, err := os.Stat(getTagFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			s.PreProcessing()
			return
		} else {
			slog.Error(fmt.Sprintf("标记文件 .update 状态信息获取异常：%s", err.Error()))
			os.Exit(1)
		}
	}
	if fi.IsDir() {
		s.PreProcessing()
	} else {
		s.PostProcessing()
	}
}

// PreProcessing 初始化并运行 JMS 服务的预处理任务，包括合并账户和备份应用资产，在执行升级 v3 操作之前执行
func (s *JmsService) PreProcessing() {
	// 创建 .update 文件标记为预处理状态
	if err := os.WriteFile(getTagFilePath(), []byte("prepare"), 0644); err != nil {
		slog.Error(fmt.Sprintf("标记文件 .update 创建失败：%s", err.Error()))
		os.Exit(1)
	}
	start := time.Now()
	fmt.Println("开始预处理...")

	// 合并同名账号
	mergeAccount := NewJMSMergeAccountService(s.db)
	mergeAccount.Run()

	// 应用资产数据迁移准备，DB/Cloud/Web 应用备份、授权备份
	migrateApp := NewJmsMigrateAppAssets(s.db)
	migrateApp.Prepare()

	// ACL 规则迁移准备
	migrateAcl := NewJmsMigrateAclAssets(s.db)
	migrateAcl.Prepare()

	// 系统用户迁移准备
	migrateAccount := NewMigrateAccountService(s.db)
	migrateAccount.Prepare()

	duration := time.Since(start).Seconds()
	fmt.Printf("预处理完成, 耗时 %fs, 请完成升级后再次运行本脚本程序执行后处理！\n", duration)
}

// postProcessing 应用迁移，在执行升级 v3 操作之后处理逻辑
func (s *JmsService) PostProcessing() {
	start := time.Now()
	fmt.Println("开始升级后处理...")

	migrateApp := NewJmsMigrateAppAssets(s.db)
	migrateApp.Migrate(s.parser)

	migrateAcl := NewJmsMigrateAclAssets(s.db)
	migrateAcl.Post()

	migrateAccount := NewMigrateAccountService(s.db)
	migrateAccount.Post()

	duration := time.Since(start).Seconds()
	fmt.Printf("升级后处理完成, 耗时 %fs.\n", duration)
}

func getTagFilePath() string {
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	} else {
		dir := filepath.Dir(exePath)
		return filepath.Join(dir, ".update")
	}
}

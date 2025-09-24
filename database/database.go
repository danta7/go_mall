package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/danta7/go_mall/internal/config"
	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

// DB 封装数据库连接
type DB struct {
	*sql.DB
	logger *zap.Logger
}

// New 创建数据库连接
func New(cfg *config.Config, logger *zap.Logger) (*DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
	)

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)

	// 测试链接
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	logger.Info("database connected",
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
		zap.String("database", cfg.Database.DBName),
	)

	return &DB{DB: sqlDB, logger: logger}, nil
}

// RunMigrations 执行数据库迁移
// 数据库迁移是一种管理数据库结构变更的版本控制机制，通过SQL文件定义数据库模式变更
// 主要作用是：
// 1. 确保所有环境（开发、测试、生产）使用相同的数据库结构
// 2. 跟踪数据库结构的变更历史
// 3. 支持向前（应用新变更）和向后（回滚）操作
// 4. 多人协作开发时避免数据库结构不一致
func (db *DB) RunMigrations(migrationsDir string) error {
	// 创建 migrations 表来记录已执行的迁移
	if err := db.createMigrationsTable(); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	// 获取已经执行的迁移
	executed, err := db.getExecuteMigrations()
	if err != nil {
		return fmt.Errorf("get executed migraions: %w", err)
	}

	// 读取迁移文件
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("read migration files: %w", err)
	}

	sort.Strings(files)

	// 执行未执行的迁移
	// 迁移文件命名采用时间戳前缀（如 20241001_001_create_users_table.sql）确保按顺序执行
	// 这种命名约定很重要，因为它保证了迁移按预期的顺序应用
	for _, file := range files {
		filename := filepath.Base(file)
		if executed[filename] {
			db.logger.Debug("migration already executed", zap.String("file", filename))
			continue
		}

		if err := db.executeMigration(file); err != nil {
			return fmt.Errorf("execute migration %s: %w", filename, err)
		}
		db.logger.Info("migration executed", zap.String("file", filename))
	}

	return nil
}

func (db *DB) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS migrations (
			id INT AUTO_INCREMENT PRIMARY KEY,
			filename VARCHAR(255) NOT NULL UNIQUE,
			executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`
	_, err := db.Exec(query)
	return err
}

// 查询已经执行过的迁移表文件
func (db *DB) getExecuteMigrations() (map[string]bool, error) {
	executed := make(map[string]bool)
	rows, err := db.Query("SELECT filename FROM migrations")
	if err != nil {
		return executed, err
	}
	defer func() { _ = rows.Close() }()

	// 遍历查询结果
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return executed, err
		}
		executed[filename] = true
	}
	return executed, rows.Err()
}

func (db *DB) executeMigration(filepath string) error {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	filename := filepath[strings.LastIndex(filepath, "/")+1:]

	// 分割 SQL 语句并逐条执行
	// 使用分号(;)作为语句分隔符，但需要处理注释和字符串中的分号
	sqlStatements := strings.Split(string(content), ";")

	// 开始事务以确保所有SQL语句要么全部成功，要么全部失败
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	// 	执行每条 sql 语句
	for _, stmt := range sqlStatements {
		// 去除语句前后的空白字符
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(strings.TrimLeft(stmt, " \t"), "--") {
			continue
		}

		if _, execErr := tx.Exec(stmt); execErr != nil {
			err = fmt.Errorf("exec SQL: %w", execErr)
			return err
		}
	}

	// 记录迁移过程
	// 通过在migrations表中记录已执行的迁移文件名，确保相同的迁移不会被重复执行
	if _, err = tx.Exec("INSERT INTO migrations (filename) VALUES (?)", filename); err != nil {
		err = fmt.Errorf("record migration: %w", err)
		return err
	}

	return nil
}

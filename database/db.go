package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"ssl_mode"`
	DSN      string `json:"dsn,omitempty"`
}

type DBStats struct {
	Status     string  `json:"status"`
	Storage    string  `json:"storage_used"`
	SizeMB     float64 `json:"storage_mb"`
	SizeBytes  int64   `json:"storage_bytes"`
	TableCount int64   `json:"tables"`
}

var (
	mu      sync.RWMutex
	clients = make(map[string]*gorm.DB)
)

const DefaultName = "default"

func (c Config) BuildDSN() string {
	if strings.TrimSpace(c.DSN) != "" {
		return strings.TrimSpace(c.DSN)
	}
	port := strings.TrimSpace(c.Port)
	if port == "" {
		port = "5432"
	}
	sslMode := strings.TrimSpace(c.SSLMode)
	if sslMode == "" {
		sslMode = "require"
	}
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		strings.Trim(c.Host, "'\""),
		strings.Trim(c.User, "'\""),
		strings.Trim(c.Password, "'\""),
		strings.Trim(c.Database, "'\""),
		strings.Trim(port, "'\""),
		strings.Trim(sslMode, "'\""),
	)
}

func ConfigFromEnv(prefix string) (Config, error) {
	key, err := ResolveMasterKey()
	if err != nil {
		return Config{}, err
	}

	strictEncrypted := os.Getenv("ALLOW_PLAINTEXT_DB_ENV") != "true"

	readField := func(name string, requireEncrypted bool) (string, error) {
		val := strings.Trim(strings.TrimSpace(os.Getenv(prefix+name)), "'\"")
		if val == "" {
			return "", nil
		}
		if strings.HasPrefix(val, EncPrefix) {
			return DecryptValue(val, key)
		}
		if strictEncrypted && requireEncrypted {
			return "", fmt.Errorf("security violation: %s%s is plain-text; encrypt it with ENC: using 'go run ./cmd/vault'", prefix, name)
		}
		return val, nil
	}

	host, err := readField("PGHOST", true)
	if err != nil {
		return Config{}, err
	}
	user, err := readField("PGUSER", true)
	if err != nil {
		return Config{}, err
	}
	pass, err := readField("PGPASSWORD", true)
	if err != nil {
		return Config{}, err
	}
	dbName, err := readField("PGDATABASE", true)
	if err != nil {
		return Config{}, err
	}
	dsn, err := readField("DATABASE_URL", true)
	if err != nil {
		return Config{}, err
	}
	port, err := readField("PGPORT", false)
	if err != nil {
		return Config{}, err
	}
	sslMode, err := readField("PGSSLMODE", false)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Host:     host,
		Port:     port,
		User:     user,
		Password: pass,
		Database: dbName,
		SSLMode:  sslMode,
		DSN:      dsn,
	}, nil
}

func inspectDBStorage(db *gorm.DB) DBStats {
	stats := DBStats{
		Status:  "connected",
		Storage: "0 B",
	}
	sqlDB, err := db.DB()
	if err != nil || sqlDB.Ping() != nil {
		stats.Status = "disconnected"
		return stats
	}

	var row struct {
		SizeBytes  int64
		SizePretty string
		TableCount int64
	}

	err = db.Raw(`
		SELECT 
			pg_database_size(current_database()) AS size_bytes,
			pg_size_pretty(pg_database_size(current_database())) AS size_pretty,
			(SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public') AS table_count
	`).Scan(&row).Error

	if err == nil {
		stats.SizeBytes = row.SizeBytes
		stats.SizeMB = math.Round((float64(row.SizeBytes)/(1024*1024))*100) / 100
		stats.Storage = fmt.Sprintf("%.2f MB (%s)", stats.SizeMB, row.SizePretty)
		stats.TableCount = row.TableCount
	}

	return stats
}

func Connect(name string, cfg Config) (*gorm.DB, error) {
	if name == "" {
		name = DefaultName
	}

	dsn := cfg.BuildDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	dsn = ""
	cfg.Password = ""
	cfg.DSN = ""

	if err != nil {
		return nil, fmt.Errorf("db [%s] connection failed", name)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("db [%s] sql pool error", name)
	}

	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("db [%s] ping failed", name)
	}

	mu.Lock()
	clients[name] = db
	mu.Unlock()

	stats := inspectDBStorage(db)
	log.Printf("[DB] Connected: %q | Storage: %s | Tables: %d", name, stats.Storage, stats.TableCount)
	return db, nil
}

func ConnectEnv(name, prefix string) (*gorm.DB, error) {
	cfg, err := ConfigFromEnv(prefix)
	if err != nil {
		return nil, err
	}
	return Connect(name, cfg)
}

func InitAll() error {
	key, err := ResolveMasterKey()
	if err != nil {
		return err
	}

	vaultPath := os.Getenv("ALAB_VAULT_FILE")
	if vaultPath == "" {
		vaultPath = DefaultVaultFile
	}

	if _, statErr := os.Stat(vaultPath); statErr == nil {
		configs, err := LoadVault(vaultPath, key)
		if err != nil {
			return fmt.Errorf("failed to unlock database vault: %w", err)
		}
		if len(configs) == 0 {
			return errors.New("database vault is empty")
		}
		for name, cfg := range configs {
			if _, err := Connect(name, cfg); err != nil {
				return err
			}
		}
		return nil
	}

	_, err = ConnectEnv(DefaultName, "")
	return err
}

func Get(name ...string) *gorm.DB {
	target := DefaultName
	if len(name) > 0 && name[0] != "" {
		target = name[0]
	}
	mu.RLock()
	defer mu.RUnlock()
	return clients[target]
}

func SQL(name ...string) (*sql.DB, error) {
	db := Get(name...)
	if db == nil {
		return nil, fmt.Errorf("database connection not found")
	}
	return db.DB()
}

func GetStorageStats(name ...string) DBStats {
	db := Get(name...)
	if db == nil {
		return DBStats{Status: "not_found"}
	}
	return inspectDBStorage(db)
}

func PingAll() map[string]DBStats {
	mu.RLock()
	defer mu.RUnlock()

	statuses := make(map[string]DBStats, len(clients))
	for name, db := range clients {
		statuses[name] = inspectDBStorage(db)
	}
	return statuses
}

func CloseAll() {
	mu.Lock()
	defer mu.Unlock()

	for name, db := range clients {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
			log.Printf("[DB] Closed connection: %q", name)
		}
	}
}

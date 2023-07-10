
package builder

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type dbConfig struct {
	Host         string `default:"localhost"`
	Port         int    `default:"5432"`
	DBName       string `default:"l2"`
	User         string `default:"l2"`
	Password     string `default:"password"`
	ConnLifeTime int    `default:"300" split_words:"true"`
	ConnTimeOut  int    `default:"30" split_words:"true"`
	MaxIdleConns int    `default:"10" split_words:"true"`
	MaxOpenConns int    `default:"80" split_words:"true"`
	LogLevel     int    `default:"4" split_words:"true"`
}

func (c *dbConfig) DNSMySQL() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&timeout=%ds",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.ConnTimeOut,
	)
}

func (c *dbConfig) DNSPostgres() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s connect_timeout=%d",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.ConnTimeOut,
	)
}

func NewMySQL(cfg *dbConfig) (*gorm.DB, error) {
	db, err := gorm.Open(
		mysql.Open(cfg.DNSMySQL()),
		&gorm.Config{
			Logger: logger.Default.LogMode(logger.LogLevel(cfg.LogLevel)),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection, err: %v", err)
	}

	return configDB(db, cfg)
}

func NewPostgres(cfg *dbConfig) (*gorm.DB, error) {
	db, err := gorm.Open(
		postgres.Open(cfg.DNSPostgres()),
		&gorm.Config{
			Logger: logger.Default.LogMode(logger.LogLevel(cfg.LogLevel)),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection, err: %v", err)
	}

	return configDB(db, cfg)
}

func configDB(db *gorm.DB, cfg *dbConfig) (*gorm.DB, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get *sql.DB, err: %v", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnLifeTime) * time.Second)

	return db, sqlDB.Ping()
}

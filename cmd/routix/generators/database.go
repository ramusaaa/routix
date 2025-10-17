package generators

import (
	"path/filepath"
)

func GenerateDatabaseFiles(projectName string, config ProjectConfig) {
	generateDatabaseConnection(projectName, config)
	generateMigrationSystem(projectName)
	generateSeederSystem(projectName)
}

func generateDatabaseConnection(projectName string, config ProjectConfig) {
	var content string

	switch config.DatabaseType {
	case "postgres":
		content = generatePostgresConnection(projectName)
	case "mysql":
		content = generateMySQLConnection(projectName)
	case "sqlite":
		content = generateSQLiteConnection(projectName)
	default:
		content = generatePostgresConnection(projectName)
	}

	writeFile(filepath.Join(projectName, "database", "connection.go"), content)
}

func generatePostgresConnection(projectName string) string {
	return `package database

import (
	"fmt"
	"log"

	"` + projectName + `/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(cfg *config.Config) *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.Database.Host,
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Database,
		cfg.Database.Port,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("✅ Database connected successfully")
	return DB
}

func Close(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Error getting database instance: %v", err)
		return
	}
	
	if err := sqlDB.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}
}`
}

func generateMySQLConnection(projectName string) string {
	return `package database

import (
	"fmt"
	"log"

	"` + projectName + `/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(cfg *config.Config) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("✅ Database connected successfully")
	return DB
}

func Close(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Error getting database instance: %v", err)
		return
	}
	
	if err := sqlDB.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}
}`
}

func generateSQLiteConnection(projectName string) string {
	return `package database

import (
	"log"

	"` + projectName + `/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(cfg *config.Config) *gorm.DB {
	var err error
	DB, err = gorm.Open(sqlite.Open(cfg.Database.Database), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("✅ Database connected successfully")
	return DB
}

func Close(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Error getting database instance: %v", err)
		return
	}
	
	if err := sqlDB.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}
}`
}

func generateMigrationSystem(projectName string) {
	migrationContent := `package migrations

import (
	"gorm.io/gorm"
)

type Migration struct {
	ID   string
	Up   func(*gorm.DB) error
	Down func(*gorm.DB) error
}

var migrations []*Migration

func RegisterMigration(migration *Migration) {
	migrations = append(migrations, migration)
}

func GetMigrations() []*Migration {
	return migrations
}

func RunMigrations(db *gorm.DB) error {
	for _, migration := range migrations {
		if err := migration.Up(db); err != nil {
			return err
		}
	}
	return nil
}

func RollbackMigration(db *gorm.DB, id string) error {
	for _, migration := range migrations {
		if migration.ID == id {
			return migration.Down(db)
		}
	}
	return nil
}`

	writeFile(filepath.Join(projectName, "database", "migrations", "migration.go"), migrationContent)
}

func generateSeederSystem(projectName string) {
	seederContent := `package seeders

import (
	"gorm.io/gorm"
)

type Seeder interface {
	Run(*gorm.DB) error
}

var seeders []Seeder

func RegisterSeeder(seeder Seeder) {
	seeders = append(seeders, seeder)
}

func RunSeeders(db *gorm.DB) error {
	for _, seeder := range seeders {
		if err := seeder.Run(db); err != nil {
			return err
		}
	}
	return nil
}`

	writeFile(filepath.Join(projectName, "database", "seeders", "seeder.go"), seederContent)
}
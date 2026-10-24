package generators

import (
	"path/filepath"
	"strings"
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
	"net"
 "net/url"
	"log"
 "os"

	"` + projectName + `/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(cfg *config.Config) *gorm.DB {
 connection:=url.URL{Scheme:"postgres",User:url.UserPassword(cfg.Database.Username,cfg.Database.Password),Host:net.JoinHostPort(cfg.Database.Host,cfg.Database.Port),Path:cfg.Database.Database}
 query:=connection.Query();query.Set("sslmode",cfg.Database.SSLMode);connection.RawQuery=query.Encode()
 dsn:=connection.String()

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.New(log.New(os.Stderr,"",log.LstdFlags),logger.Config{LogLevel:logger.Warn,ParameterizedQueries:true}),
 TranslateError: true,
	})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

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
 "net"
	"log"
 "os"

	"` + projectName + `/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(cfg *config.Config) *gorm.DB {
	network,address:="tcp",net.JoinHostPort(cfg.Database.Host,cfg.Database.Port)
 if cfg.Database.Socket!="" {network,address="unix",cfg.Database.Socket}
 dsn:=fmt.Sprintf("%s:%s@%s(%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC",cfg.Database.Username,cfg.Database.Password,network,address,cfg.Database.Database)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.New(log.New(os.Stderr,"",log.LstdFlags),logger.Config{LogLevel:logger.Warn,ParameterizedQueries:true}),
 TranslateError: true,
	})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

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
 "os"
 "strings"

	"` + projectName + `/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(cfg *config.Config) *gorm.DB {
 dsn:=cfg.Database.Database
 separator:="?"
 if strings.Contains(dsn,"?") {separator="&"}
 dsn+=separator+"_foreign_keys=on&_busy_timeout=5000&_txlock=immediate"
 var err error
 DB,err=gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.New(log.New(os.Stderr,"",log.LstdFlags),logger.Config{LogLevel:logger.Warn,ParameterizedQueries:true}),
 TranslateError: true,
	})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

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
	content := `package migrations

import (
 "fmt"
 "crypto/sha256"
 "sort"
 "sync"
 "time"

 "gorm.io/gorm"
 "gorm.io/gorm/clause"
)

type Migration struct {
 ID string
 Up func(*gorm.DB) error
 Down func(*gorm.DB) error
}

type Record struct {
 ID string ` + "`" + `gorm:"primaryKey;size:255"` + "`" + `
 Batch int ` + "`" + `gorm:"not null;index"` + "`" + `
 AppliedAt time.Time ` + "`" + `gorm:"not null"` + "`" + `
 Dirty bool ` + "`" + `gorm:"not null;default:false"` + "`" + `
}

func (Record) TableName() string {return "routix_migrations"}

type migrationLock struct {
 ID int ` + "`" + `gorm:"primaryKey"` + "`" + `
 Version int ` + "`" + `gorm:"not null"` + "`" + `
}

func (migrationLock) TableName() string {return "routix_migration_locks"}

type Status struct {ID string;Applied bool;Missing bool;Dirty bool;Batch int}

var migrations []*Migration
 var mysqlMigrationMu sync.Mutex

func RegisterMigration(migration *Migration) {
 if migration==nil || migration.ID=="" || migration.Up==nil || migration.Down==nil {panic("invalid migration registration")}
 for _,existing:=range migrations {if existing.ID==migration.ID {panic("duplicate migration: "+migration.ID)}}
 copy:=*migration
 migrations=append(migrations,&copy)
}

func GetMigrations() []*Migration {
 out:=make([]*Migration,0,len(migrations))
 for _,migration:=range migrations {copy:=*migration;out=append(out,&copy)}
 sort.Slice(out,func(i,j int)bool{return out[i].ID<out[j].ID})
 return out
}

func withLock(db *gorm.DB,fn func(*gorm.DB)error) error {
 if db==nil {return fmt.Errorf("database is required")}
 if db.Dialector.Name()=="mysql" {
  mysqlMigrationMu.Lock()
  defer mysqlMigrationMu.Unlock()
  sqlDB,err:=db.DB();if err!=nil {return err}
  if sqlDB.Stats().MaxOpenConnections==1 {return fmt.Errorf("MySQL migrations require at least two database connections")}
 }
 return db.Connection(func(connection *gorm.DB)error {
 connection=connection.Session(&gorm.Session{NewDB:true})
  if connection.Dialector.Name()=="mysql" {
   var database string
   if err:=connection.Raw("SELECT DATABASE()").Scan(&database).Error;err!=nil {return err}
   key:=fmt.Sprintf("routix:%x",sha256.Sum256([]byte(database)))[:63]
   var acquired int
   if err:=connection.Raw("SELECT GET_LOCK(?, 10)",key).Scan(&acquired).Error;err!=nil {return err}
   if acquired!=1 {return fmt.Errorf("could not acquire migration lock")}
   defer connection.Exec("SELECT RELEASE_LOCK(?)",key)
   operations:=db.Session(&gorm.Session{NewDB:true})
   if err:=operations.AutoMigrate(&Record{});err!=nil {return err}
   return fn(operations)
  }
  return connection.Transaction(func(tx *gorm.DB)error {
   if tx.Dialector.Name()=="postgres" {
    if err:=tx.Exec("SELECT pg_advisory_xact_lock(781205141)").Error;err!=nil {return err}
   }
   if err:=tx.Exec("CREATE TABLE IF NOT EXISTS routix_migration_locks (id INTEGER PRIMARY KEY, version INTEGER NOT NULL)").Error;err!=nil {return err}
   if err:=tx.Clauses(clause.OnConflict{DoNothing:true}).Create(&migrationLock{ID:1}).Error;err!=nil {return err}
   if err:=tx.Model(&migrationLock{}).Where("id = ?",1).Update("version",gorm.Expr("version + 1")).Error;err!=nil {return err}
   if err:=tx.AutoMigrate(&Record{});err!=nil {return err}
   return fn(tx)
  })
 })
}

func run(db *gorm.DB) error {
 var records []Record
 if err:=db.Find(&records).Error;err!=nil {return err}
 applied:=make(map[string]bool,len(records))
 batch:=0
 for _,record:=range records {if record.Dirty {return fmt.Errorf("dirty migration %s: repair the database before retrying",record.ID)};applied[record.ID]=true;if record.Batch>batch {batch=record.Batch}}
 for _,migration:=range GetMigrations() {
  if applied[migration.ID] {continue}
  if err:=db.Create(&Record{ID:migration.ID,Batch:batch+1,AppliedAt:time.Now().UTC(),Dirty:true}).Error;err!=nil {return err}
  if err:=migration.Up(db);err!=nil {return fmt.Errorf("migration %s: %w",migration.ID,err)}
  if err:=db.Model(&Record{}).Where("id = ?",migration.ID).Update("dirty",false).Error;err!=nil {return err}
 }
 return nil
}

func RunMigrations(db *gorm.DB) error {return withLock(db,run)}

func rollback(db *gorm.DB,all bool,id string) error {
 var records []Record
 query:=db.Order("batch DESC").Order("id DESC")
 if id!="" {query=query.Where("id = ?",id)}
 if err:=query.Find(&records).Error;err!=nil {return err}
 if id!="" && len(records)==0 {return fmt.Errorf("applied migration not found: %s",id)}
 if !all && len(records)>0 {
  batch:=records[0].Batch
  end:=0
  for end<len(records) && records[end].Batch==batch {end++}
  records=records[:end]
 }
 registered:=make(map[string]*Migration,len(migrations))
 for _,migration:=range GetMigrations() {registered[migration.ID]=migration}
 for _,record:=range records {if record.Dirty {return fmt.Errorf("dirty migration %s: repair the database before retrying",record.ID)};if registered[record.ID]==nil {return fmt.Errorf("migration source missing: %s",record.ID)}}
 for _,record:=range records {
  if err:=db.Model(&Record{}).Where("id = ?",record.ID).Update("dirty",true).Error;err!=nil {return err}
  if err:=registered[record.ID].Down(db);err!=nil {return fmt.Errorf("rollback %s: %w",record.ID,err)}
  if err:=db.Delete(&Record{},"id = ?",record.ID).Error;err!=nil {return err}
 }
 return nil
}

func RollbackMigration(db *gorm.DB,id string) error {
 return withLock(db,func(tx *gorm.DB)error{return rollback(tx,false,id)})
}

func ResetMigrations(db *gorm.DB) error {
 return withLock(db,func(tx *gorm.DB)error{return rollback(tx,true,"")})
}

func FreshMigrations(db *gorm.DB) error {
 return withLock(db,func(tx *gorm.DB)error {
  if err:=rollback(tx,true,"");err!=nil {return err}
  return run(tx)
 })
}

func MigrationStatus(db *gorm.DB) ([]Status,error) {
 if db==nil {return nil,fmt.Errorf("database is required")}
 records:=make(map[string]Record)
 if db.Migrator().HasTable(&Record{}) {
  var applied []Record
  if err:=db.Find(&applied).Error;err!=nil {return nil,err}
  for _,record:=range applied {records[record.ID]=record}
 }
 result:=make([]Status,0,len(migrations)+len(records))
 for _,migration:=range GetMigrations() {
  record,ok:=records[migration.ID]
  result=append(result,Status{ID:migration.ID,Applied:ok,Dirty:record.Dirty,Batch:record.Batch})
  delete(records,migration.ID)
 }
 for _,record:=range records {result=append(result,Status{ID:record.ID,Applied:true,Missing:true,Dirty:record.Dirty,Batch:record.Batch})}
 sort.Slice(result,func(i,j int)bool{return result[i].ID<result[j].ID})
 return result,nil
}

func RepairMigration(db *gorm.DB,id string,applied bool) error {
 if id=="" {return fmt.Errorf("migration ID is required")}
 return withLock(db,func(tx *gorm.DB)error {
  var record Record
  if err:=tx.Where("id = ? AND dirty = ?",id,true).First(&record).Error;err!=nil {return fmt.Errorf("dirty migration not found: %s: %w",id,err)}
  if applied {return tx.Model(&Record{}).Where("id = ?",id).Update("dirty",false).Error}
  return tx.Delete(&Record{},"id = ?",id).Error
 })
}
`
	writeFile(filepath.Join(projectName, "database", "migrations", "migration.go"), content)
}

func generateSeederSystem(projectName string) {
	content := `package seeders

import (
 "fmt"
 "gorm.io/gorm"
)

type Seeder interface {Run(*gorm.DB)error}
var seeders []Seeder

func RegisterSeeder(seeder Seeder) {
 if seeder==nil {panic("nil seeder")}
 seeders=append(seeders,seeder)
}

func RunSeeders(db *gorm.DB) error {
 if db==nil {return fmt.Errorf("database is required")}
 return db.Transaction(func(tx *gorm.DB)error {
  for _,seeder:=range seeders {if err:=seeder.Run(tx);err!=nil {return err}}
  return nil
 })
}
`
	writeFile(filepath.Join(projectName, "database", "seeders", "seeder.go"), content)
}
func GenerateDatabaseRunner(projectName string) string {
	return strings.ReplaceAll(`package main

import (
 "fmt"
 "os"
 "{{module}}/config"
 "{{module}}/database"
 "{{module}}/database/migrations"
 "{{module}}/database/seeders"
)

func run() error {
 db:=database.Connect(config.Load())
 defer database.Close(db)
 if len(os.Args)<2 {return fmt.Errorf("database action required")}
 switch os.Args[1] {
 case "migrate":return migrations.RunMigrations(db)
 case "repair":
  if len(os.Args)!=4 || (os.Args[3]!="--applied" && os.Args[3]!="--pending") {return fmt.Errorf("usage: repair <id> --applied|--pending")}
  return migrations.RepairMigration(db,os.Args[2],os.Args[3]=="--applied")
 case "rollback":return migrations.RollbackMigration(db,"")
 case "reset":return migrations.ResetMigrations(db)
 case "fresh":return migrations.FreshMigrations(db)
 case "seed":return seeders.RunSeeders(db)
 case "status":
  items,err:=migrations.MigrationStatus(db)
  if err!=nil {return err}
  fmt.Printf("%-55s %-10s %s\n","Migration","Status","Batch")
  for _,item:=range items {
   state:="Pending"
   if item.Applied {state="Ran"}
   if item.Missing {state="Missing"}
   if item.Dirty {state="Dirty"}
   fmt.Printf("%-55s %-10s %d\n",item.ID,state,item.Batch)
  }
  return nil
 default:return fmt.Errorf("unknown database action: %s",os.Args[1])
 }
}

func main() {
 if err:=run();err!=nil {fmt.Fprintln(os.Stderr,err);os.Exit(1)}
}
`, "{{module}}", projectName)
}

package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var (
	envLoaded bool
	envOnce   sync.Once
)

// DBType represents the type of database
type DBType string

const (
	DBTypeSQLite   DBType = "sqlite"
	DBTypeMySQL    DBType = "mysql"
	DBTypePostgres DBType = "postgres"
)

// loadEnv loads environment variables from .env file in the examples directory
// This function locates the .env file by finding the examples directory
func loadEnv() {
	envOnce.Do(func() {
		// Get the directory where this file (helpers.go) is located
		// runtime.Caller(0) gives us the location of this function (loadEnv)
		// which is in helpers.go in the testutil package
		_, currentFile, _, ok := runtime.Caller(0)
		if !ok {
			return
		}

		// Get the examples directory (parent of testutil)
		// currentFile is something like: /path/to/examples/testutil/helpers.go
		testutilDir := filepath.Dir(currentFile)
		examplesDir := filepath.Dir(testutilDir)

		// Load .env file from examples directory
		envPath := filepath.Join(examplesDir, ".env")
		if err := godotenv.Load(envPath); err != nil {
			// .env file is optional, so we don't fail if it doesn't exist
			// Environment variables can still be set directly via system environment
			return
		}
		envLoaded = true
	})
}

// getDBType returns the database type from environment variable or defaults to sqlite
// It loads .env file from examples directory if not already loaded
func getDBType() DBType {
	loadEnv()
	dbType := os.Getenv("TEST_DB_TYPE")
	switch dbType {
	case "mysql":
		return DBTypeMySQL
	case "postgres", "postgresql":
		return DBTypePostgres
	default:
		return DBTypeSQLite
	}
}

// getDBDir returns the db directory path where SQLite files should be stored
// This function locates the examples/db directory relative to the examples directory
func getDBDir() (string, error) {
	// Get the directory where this file (helpers.go) is located
	// This file is in examples/testutil directory
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrInvalid
	}

	// Get the examples directory (parent of testutil)
	testutilDir := filepath.Dir(currentFile)
	examplesDir := filepath.Dir(testutilDir)

	// The db directory is examples/db
	dbDir := filepath.Join(examplesDir, "db")

	// Ensure db directory exists
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return "", err
	}

	return dbDir, nil
}

// NewTestDB creates a test database connection
// For SQLite: files are stored in the db directory (examples/db) with "sqlite" in the filename
// For MySQL/PostgreSQL: uses connection strings from environment variables
func NewTestDB(t *testing.T, filename string) *gorm.DB {
	t.Helper()

	dbType := getDBType()
	var db *gorm.DB
	var err error

	switch dbType {
	case DBTypeSQLite:
		db, err = newSQLiteDB(t, filename)
	case DBTypeMySQL:
		db, err = newMySQLDB(t)
	case DBTypePostgres:
		db, err = newPostgresDB(t)
	default:
		t.Fatalf("unsupported database type: %s", dbType)
	}

	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	// Get the underlying *sql.DB to configure connection pool settings
	// Connection pool settings are configured on the underlying database connection,
	// not in gorm.Config, because they are database-specific settings
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get generic db: %v", err)
	}

	// Connection pool configuration:
	// - SetMaxIdleConns: Maximum number of idle connections in the pool
	//   (connections that are open but not currently in use)
	// - SetMaxOpenConns: Maximum number of open connections to the database
	//   (total connections, including idle and in-use)
	// - SetConnMaxLifetime: Maximum amount of time a connection may be reused
	//   (prevents using stale connections)
	sqlDB.SetMaxIdleConns(2)                   // Keep 2 idle connections ready
	sqlDB.SetMaxOpenConns(5)                   // Allow up to 5 concurrent connections
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // Reuse connections for up to 30 minutes

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}

// newSQLiteDB creates a SQLite database connection
// The database file is stored in the db directory (examples/db) with "sqlite" in the filename
func newSQLiteDB(t *testing.T, filename string) (*gorm.DB, error) {
	// Get the db directory where SQLite files should be stored
	dbDir, err := getDBDir()
	if err != nil {
		return nil, err
	}

	// Ensure filename contains "sqlite"
	if filename == "" {
		filename = "test.sqlite.db"
	} else {
		// Extract base name and extension
		ext := filepath.Ext(filename)
		if ext == "" {
			ext = ".db"
		}
		base := filename[:len(filename)-len(ext)]
		if base == "" {
			base = "test"
		}
		// Check if "sqlite" is already in the filename (case-insensitive check)
		baseLower := strings.ToLower(base)
		if !strings.Contains(baseLower, "sqlite") {
			filename = base + "_sqlite" + ext
		} else {
			filename = base + ext
		}
	}

	// Database file will be stored in db directory (examples/db)
	dbPath := filepath.Join(dbDir, filename)

	// Configure GORM with:
	// 1. Logger: Control SQL logging level
	//    - Silent: No logs
	//    - Error: Only errors
	//    - Warn: Errors and warnings
	//    - Info: All SQL queries (default)
	// 2. NamingStrategy: Customize table and column naming
	//    - TableName: How struct names map to table names
	//    - ColumnName: How field names map to column names
	//    - JoinTableName: How join table names are generated
	//    - SchemaName: Schema name for databases that support it
	return gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		// Logger configuration
		Logger: logger.Default.LogMode(logger.Info), // Silent for tests, use logger.Info for development

		// NamingStrategy: Customize how GORM names tables and columns
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",    // Prefix for all table names (e.g., "app_")
			SingularTable: false, // Use singular table names (User -> user instead of users)
			NoLowerCase:   false, // Disable automatic lowercasing
			NameReplacer:  nil,   // Custom name replacer function
		},
	})
}

// newMySQLDB creates a MySQL database connection
// Connection string is read from TEST_MYSQL_DSN environment variable or .env file
// Format: user:password@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local
func newMySQLDB(t *testing.T) (*gorm.DB, error) {
	loadEnv()
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
		t.Logf("using default MySQL DSN, set TEST_MYSQL_DSN in .env file or environment variable to override")
	}

	return gorm.Open(mysql.Open(dsn), &gorm.Config{
		// Logger: Set to logger.Info to see all SQL queries in development
		Logger: logger.Default.LogMode(logger.Silent),

		// NamingStrategy: Customize table and column naming
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",
			SingularTable: false,
			NoLowerCase:   false,
		},
	})
}

// newPostgresDB creates a PostgreSQL database connection
// Connection string is read from TEST_POSTGRES_DSN environment variable or .env file
// Format: host=localhost user=postgres password=password dbname=testdb port=5432 sslmode=disable TimeZone=Asia/Shanghai
func newPostgresDB(t *testing.T) (*gorm.DB, error) {
	loadEnv()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=password dbname=testdb port=5432 sslmode=disable TimeZone=Asia/Shanghai"
		t.Logf("using default PostgreSQL DSN, set TEST_POSTGRES_DSN in .env file or environment variable to override")
	}

	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		// Logger: Set to logger.Info to see all SQL queries in development
		Logger: logger.Default.LogMode(logger.Silent),

		// NamingStrategy: Customize table and column naming
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",
			SingularTable: false,
			NoLowerCase:   false,
		},
	})
}

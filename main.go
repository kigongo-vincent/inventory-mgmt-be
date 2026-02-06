package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/kigongo-vincent/inventory-mgmt-be.git/config"
	"github.com/kigongo-vincent/inventory-mgmt-be.git/modules/Company"
	"github.com/kigongo-vincent/inventory-mgmt-be.git/modules/Notification"
	"github.com/kigongo-vincent/inventory-mgmt-be.git/modules/Expense"
	"github.com/kigongo-vincent/inventory-mgmt-be.git/modules/Product"
	"github.com/kigongo-vincent/inventory-mgmt-be.git/modules/Sale"
	"github.com/kigongo-vincent/inventory-mgmt-be.git/modules/User"
	"gorm.io/gorm"
)

func main() {
	// Load environment variables
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	envFile := ".env." + env
	if err := godotenv.Load(envFile); err != nil {
		log.Printf("Warning: Error loading %s file: %v", envFile, err)
		// Try loading .env as fallback
		if err := godotenv.Load(); err != nil {
			log.Printf("Warning: Error loading .env file: %v", err)
		}
	}

	// Initialize database connection
	db, err := config.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Auto-migrate database schemas
	if err := migrateDatabase(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize services with database connection (order matters)
	Company.InitializeService(db)
	User.InitializeService(db) // This also initializes Branch service (moved to User package)
	Notification.InitializeService(db)
	Product.InitializeService(db)
	Sale.InitializeService(db)
	Expense.InitializeService(db)

	// Initialize Gin router
	r := gin.Default()

	// CORS configuration
	config := cors.DefaultConfig()
	corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if corsOrigins == "" || corsOrigins == "*" {
		config.AllowOrigins = []string{"*"}
	} else {
		config.AllowOrigins = []string{corsOrigins}
	}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Public routes (no auth required)
	api := r.Group("/api/v1")
	{
		User.RegisterRoutes(api)
	}

	// Protected routes (auth required)
	protected := api.Group("")
	protected.Use(User.AuthMiddleware())
	{
		User.RegisterBranchRoutes(protected)
		Notification.RegisterRoutes(protected)
		Product.RegisterRoutes(protected)
		Sale.RegisterRoutes(protected)
		Expense.RegisterRoutes(protected)
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// migrateDatabase runs auto-migration for all models
// Order matters: Company -> Branch -> User -> Product -> Sale
func migrateDatabase(db *gorm.DB) error {
	log.Println("Running database migrations...")

	// First, migrate column types from UUID to integer if needed
	if err := migrateColumnTypes(db); err != nil {
		log.Printf("Warning: Column type migration had issues: %v", err)
		// Continue with AutoMigrate even if column migration fails
	}

	// 1. Company (no dependencies)
	if err := db.AutoMigrate(&Company.Company{}); err != nil {
		return err
	}

	// 2. Branch (depends on Company, now in User package)
	if err := db.AutoMigrate(&User.Branch{}); err != nil {
		return err
	}

	// 3. User (depends on Company and Branch)
	if err := db.AutoMigrate(&User.UserModel{}); err != nil {
		return err
	}

	// 3.5. NotificationPreferences (depends on User)
	if err := db.AutoMigrate(&User.NotificationPreferences{}); err != nil {
		return err
	}

	// 3.6. Notification (depends on User)
	if err := db.AutoMigrate(&Notification.Notification{}); err != nil {
		return err
	}

	// 4. Product (depends on Company)
	if err := db.AutoMigrate(&Product.Product{}); err != nil {
		return err
	}

	// 5. Sale (depends on User and Product)
	if err := db.AutoMigrate(&Sale.Sale{}); err != nil {
		return err
	}

	// 6. Expense (depends on User and Branch)
	if err := db.AutoMigrate(&Expense.Expense{}); err != nil {
		return err
	}

	// Verify that all ID columns are integer type (not UUID)
	if err := verifyIDTypes(db); err != nil {
		log.Printf("Warning: ID type verification failed: %v", err)
		// Don't fail the migration, but log the warning
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// migrateColumnTypes converts UUID columns to integer by dropping and recreating tables
// WARNING: This will delete all existing data. Only use in development!
func migrateColumnTypes(db *gorm.DB) error {
	log.Println("Checking for UUID columns and migrating to integer...")

	// First, check what tables actually exist and their column types
	var tableInfo []struct {
		TableName  string
		ColumnName string
		DataType   string
		Default    *string
	}

	hasUUID := false
	// Check both ID columns and foreign key columns for UUID/text types
	infoQuery := `
		SELECT table_name, column_name, data_type, column_default
		FROM information_schema.columns 
		WHERE table_name IN ('companies', 'branches', 'user_models', 'notifications', 'products', 'sales', 'expenses')
		AND (column_name = 'id' 
		     OR column_name LIKE '%_id' 
		     OR column_name = 'company_id' 
		     OR column_name = 'branch_id' 
		     OR column_name = 'seller_id' 
		     OR column_name = 'product_id')
		ORDER BY table_name, column_name
	`

	if err := db.Raw(infoQuery).Scan(&tableInfo).Error; err != nil {
		log.Printf("Warning: Could not check table info: %v", err)
		// If we can't check, assume we need to drop tables
		hasUUID = true
	} else {
		log.Printf("Found %d columns to check", len(tableInfo))
		for _, info := range tableInfo {
			defaultStr := ""
			if info.Default != nil {
				defaultStr = *info.Default
			}
			log.Printf("  Table: %s, Column: %s, Type: %s, default: %s", info.TableName, info.ColumnName, info.DataType, defaultStr)
			// Check if it's UUID type or text type (which might contain UUIDs)
			if info.DataType == "uuid" || info.DataType == "text" ||
				(info.Default != nil && (*info.Default == "gen_random_uuid()" || *info.Default == "uuid_generate_v4()")) {
				hasUUID = true
				log.Printf("  -> UUID/text detected in %s.%s", info.TableName, info.ColumnName)
			}
		}
	}

	if !hasUUID {
		log.Println("No UUID columns found, tables should use integer IDs from gorm.Model")
		return nil
	}

	log.Println("WARNING: Found UUID columns. Dropping all tables to ensure clean migration to integer IDs...")
	log.Println("All existing data will be lost!")

	// Drop all tables in reverse dependency order (CASCADE handles everything)
	tables := []string{"expenses", "sales", "products", "notifications", "user_models", "branches", "companies"}
	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)).Error; err != nil {
			log.Printf("Warning: Error dropping table %s: %v", table, err)
		} else {
			log.Printf("Dropped table: %s", table)
		}
	}

	// Drop any sequences that might exist (they'll be recreated by GORM with proper types)
	sequenceQuery := `
		SELECT sequence_name 
		FROM information_schema.sequences 
		WHERE sequence_name LIKE '%_id_seq'
	`
	var sequences []string
	if err := db.Raw(sequenceQuery).Scan(&sequences).Error; err == nil {
		for _, seq := range sequences {
			if err := db.Exec(fmt.Sprintf("DROP SEQUENCE IF EXISTS %s CASCADE", seq)).Error; err != nil {
				log.Printf("Warning: Error dropping sequence %s: %v", seq, err)
			}
		}
	}

	log.Println("Tables and sequences dropped. AutoMigrate will recreate them with integer IDs using gorm.Model.")

	// Verify after AutoMigrate that IDs are integer type
	// This will be checked after AutoMigrate runs in migrateDatabase
	return nil
}

// verifyIDTypes checks that all ID columns are integer type (not UUID)
func verifyIDTypes(db *gorm.DB) error {
	var tableInfo []struct {
		TableName string
		DataType  string
	}

	verifyQuery := `
		SELECT table_name, data_type
		FROM information_schema.columns 
		WHERE table_name IN ('companies', 'branches', 'user_models', 'notifications', 'products', 'sales', 'expenses')
		AND column_name = 'id'
		ORDER BY table_name
	`

	if err := db.Raw(verifyQuery).Scan(&tableInfo).Error; err != nil {
		return fmt.Errorf("failed to verify ID types: %w", err)
	}

	allInteger := true
	for _, info := range tableInfo {
		log.Printf("Verifying %s.id: type = %s", info.TableName, info.DataType)
		if info.DataType != "integer" && info.DataType != "bigint" {
			log.Printf("ERROR: %s.id is %s, expected integer/bigint!", info.TableName, info.DataType)
			allInteger = false
		}
	}

	if !allInteger {
		return fmt.Errorf("some ID columns are not integer type - migration may have failed")
	}

	log.Println("All ID columns verified as integer type ✓")
	return nil
}

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"alabapi/database"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "keygen":
		keyHex, err := database.GenerateMasterKeyHex()
		if err != nil {
			log.Fatalf("Key generation failed: %v", err)
		}
		fmt.Printf("ALAB_MASTER_KEY='%s'\n", keyHex)

	case "encrypt":
		if len(os.Args) < 3 {
			log.Fatal("Usage: go run ./cmd/vault encrypt <plaintext>")
		}
		key, err := database.ResolveMasterKey()
		if err != nil {
			log.Fatal(err)
		}
		enc, err := database.EncryptValue(os.Args[2], key)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(enc)

	case "add":
		addCmd := flag.NewFlagSet("add", flag.ExitOnError)
		name := addCmd.String("name", "default", "Database connection name (e.g. default, auth, analytics)")
		host := addCmd.String("host", "", "PostgreSQL Host")
		port := addCmd.String("port", "5432", "PostgreSQL Port")
		user := addCmd.String("user", "", "PostgreSQL User")
		pass := addCmd.String("pass", "", "PostgreSQL Password")
		dbName := addCmd.String("db", "", "PostgreSQL Database Name")
		ssl := addCmd.String("ssl", "require", "SSL Mode (require, verify-full)")
		vaultFile := addCmd.String("vault", database.DefaultVaultFile, "Path to encrypted vault file")
		_ = addCmd.Parse(os.Args[2:])

		if *host == "" || *user == "" || *pass == "" || *dbName == "" {
			log.Fatal("Required flags: -host, -user, -pass, -db")
		}

		key, err := database.ResolveMasterKey()
		if err != nil {
			log.Fatal(err)
		}

		configs, err := database.LoadVault(*vaultFile, key)
		if err != nil {
			configs = make(map[string]database.Config)
		}

		configs[*name] = database.Config{
			Host:     *host,
			Port:     *port,
			User:     *user,
			Password: *pass,
			Database: *dbName,
			SSLMode:  *ssl,
		}

		if err := database.SaveVault(*vaultFile, configs, key); err != nil {
			log.Fatalf("Failed to save vault: %v", err)
		}
		fmt.Printf("[VAULT] Encrypted and saved database %q into %s (AES-256-GCM)\n", *name, *vaultFile)

	case "list":
		key, err := database.ResolveMasterKey()
		if err != nil {
			log.Fatal(err)
		}
		configs, err := database.LoadVault(database.DefaultVaultFile, key)
		if err != nil {
			log.Fatalf("Failed to read vault: %v", err)
		}
		fmt.Printf("Encrypted Databases in %s (%d total):\n", database.DefaultVaultFile, len(configs))
		for name, cfg := range configs {
			maskedHost := maskString(cfg.Host)
			fmt.Printf("  - %s (host=%s, db=%s, ssl=%s)\n", name, maskedHost, maskString(cfg.Database), cfg.SSLMode)
		}

	case "test":
		if err := database.InitAll(); err != nil {
			log.Fatalf("[FAIL] Database connection failed: %v", err)
		}
		defer database.CloseAll()

		for name, stats := range database.PingAll() {
			fmt.Printf("[OK] DB %q -> status=%s | storage=%s (%d bytes) | tables=%d\n",
				name, stats.Status, stats.Storage, stats.SizeBytes, stats.TableCount)
		}

	default:
		printUsage()
	}
}

func maskString(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

func printUsage() {
	fmt.Println("AlabAPI Encrypted DB Vault CLI (AES-256-GCM)")
	fmt.Println("Commands:")
	fmt.Println("  go run ./cmd/vault test")
	fmt.Println("  go run ./cmd/vault list")
	fmt.Println("  go run ./cmd/vault add -name=default -host=... -db=... -user=... -pass=...")
	fmt.Println("  go run ./cmd/vault encrypt <plaintext>")
	fmt.Println("  go run ./cmd/vault keygen")
}

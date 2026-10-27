package config

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type Config struct {
	AppPort           string
	Environment       string
	DBHost            string
	DBPort            string
	DBName            string
	DBUser            string
	DBPassword        string
	DBSSLMode         string
	LogLevel          string
	JWTSecret         string
	ProductServiceURL string
}

func loadDotEnv(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			v = strings.Trim(v, `"'`)
			os.Setenv(k, v)
		}
	}
}

func LoadConfig() *Config {
	loadDotEnv(".env")

	cfg := &Config{
		AppPort:           getEnv("APP_PORT", "8080"),
		Environment:       getEnv("ENVIRONMENT", "development"),
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBName:            getEnv("DB_NAME", "orderdb"),
		DBUser:            getEnv("DB_USER", "postgres"),
		DBPassword:        getEnv("DB_PASSWORD", ""),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		JWTSecret:         getEnv("JWT_SECRET", "supersecretjwtkey_do_not_share_in_git"),
		ProductServiceURL: getEnv("PRODUCT_SERVICE_URL", "http://localhost:8081"),
	}

	secretName := os.Getenv("AWS_SECRET_NAME")
	if secretName != "" && cfg.DBPassword == "" {
		fetchSecretsManager(secretName, cfg)
	}

	return cfg
}

func fetchSecretsManager(secretName string, cfg *Config) {
	region := getEnv("AWS_REGION", "us-east-1")
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(region))
	if err != nil {
		log.Printf("WARN: Failed to load AWS SDK config: %v", err)
		return
	}

	smClient := secretsmanager.NewFromConfig(awsCfg)
	res, err := smClient.GetSecretValue(context.TODO(), &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		log.Printf("WARN: Failed to retrieve secret from Secrets Manager: %v", err)
		return
	}

	if res.SecretString != nil {
		var secData map[string]string
		if err := json.Unmarshal([]byte(*res.SecretString), &secData); err == nil {
			if v, ok := secData["DB_PASSWORD"]; ok {
				cfg.DBPassword = v
			}
			if v, ok := secData["DB_USER"]; ok {
				cfg.DBUser = v
			}
			if v, ok := secData["DB_HOST"]; ok {
				cfg.DBHost = v
			}
			if v, ok := secData["JWT_SECRET"]; ok {
				cfg.JWTSecret = v
			}
		}
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

#!/bin/bash
set -e

echo "Building Order Microservice..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o oms main.go

echo "Building Product Microservice..."
cd product-service
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o product-svc main.go
cd ..

echo "All microservices compiled successfully!"

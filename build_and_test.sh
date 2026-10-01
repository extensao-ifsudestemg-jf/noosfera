#!/bin/bash
set -e

echo "Iniciando pipeline Noosfera..."
echo "1. Executando testes com deteccao de Data Race (-race)..."
go test -v -race ./...

echo -e "\n2. Verificando cobertura global de codigo..."
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
rm coverage.out

echo -e "\n3. Compilando binarios com otimizacao (-s -w)..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o noosfera-linux main.go
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o noosfera-win.exe main.go

echo -e "\nSucesso! Binarios gerados:"
ls -lh noosfera-linux noosfera-win.exe

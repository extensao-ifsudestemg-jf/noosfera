package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"noosfera/internal/domain"
	"noosfera/internal/infrastructure/providers"
	"noosfera/internal/presentation/handlers"
	"noosfera/internal/usecase"
)

//go:embed web/static/*
var staticFiles embed.FS

const (
	serverPort = ":8080"
	serverURL  = "http://localhost:8080"
)

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("sistema operacional não suportado para abertura automática do navegador: %s", runtime.GOOS)
	}

	return cmd.Start()
}

func main() {

	subFS, err := fs.Sub(staticFiles, "web/static")
	if err != nil {
		log.Fatalf("Erro ao carregar arquivos estáticos embarcados: %v", err)
	}
	http.Handle("/", http.FileServer(http.FS(subFS)))

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	apiKey := os.Getenv("OPENALEX_API_KEY")
	openAlexProvider := providers.NewOpenAlexProvider(httpClient, apiKey)
	crossRefProvider := providers.NewCrossRefProvider(httpClient)

	searchUseCase := usecase.NewSearchArticlesUseCase([]domain.ArticleProvider{openAlexProvider, crossRefProvider})

	articleHandler := handlers.NewArticleHandler(searchUseCase)

	http.HandleFunc("/api/v1/search", articleHandler.HandleSearch)
	http.HandleFunc("/api/v1/export", articleHandler.HandleExport)

	go func() {
		time.Sleep(100 * time.Millisecond)
		log.Printf("Abrindo o navegador padrão em %s...", serverURL)
		if err := openBrowser(serverURL); err != nil {
			log.Printf("Aviso: Não foi possível abrir o navegador automaticamente: %v", err)
		}
	}()

	log.Printf("Servidor HTTP iniciado com sucesso. Acesse: %s", serverURL)
	if err := http.ListenAndServe(serverPort, nil); err != nil {
		log.Fatalf("Erro fatal ao iniciar servidor HTTP: %v", err)
	}
}

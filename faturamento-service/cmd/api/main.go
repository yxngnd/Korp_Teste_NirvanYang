// Comando principal do faturamento-service: carrega config, aplica
// migrations, conecta no banco, monta o client do estoque-service e as
// camadas internas, e sobe o servidor HTTP.
package main

import (
	"log"

	"github.com/candidato/faturamento-service/internal/client"
	"github.com/candidato/faturamento-service/internal/config"
	"github.com/candidato/faturamento-service/internal/db"
	"github.com/candidato/faturamento-service/internal/handler"
	"github.com/candidato/faturamento-service/internal/middleware"
	"github.com/candidato/faturamento-service/internal/repository"
	"github.com/candidato/faturamento-service/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("erro ao carregar configuração: %v", err)
	}

	if err := db.RunMigrations(cfg); err != nil {
		log.Fatalf("erro ao aplicar migrations: %v", err)
	}
	log.Println("migrations aplicadas com sucesso")

	dbConn, err := db.NewPostgresConnection(cfg)
	if err != nil {
		log.Fatalf("erro ao conectar no banco: %v", err)
	}
	defer dbConn.Close()

	estoqueClient := client.NewEstoqueClient(cfg.EstoqueServiceURL)
	log.Printf("client do estoque-service configurado para %s", cfg.EstoqueServiceURL)

	notaRepo := repository.NewNotaRepository(dbConn)
	notaService := service.NewNotaService(notaRepo, estoqueClient)
	notaHandler := handler.NewNotaHandler(notaService)

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4200"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Idempotency-Key"},
		AllowCredentials: true,
	}))
	router.Use(middleware.ErrorHandlerMiddleware())

	router.GET("/health", handler.Health)

	v1 := router.Group("/api/v1")
	{
		notas := v1.Group("/notas")
		{
			notas.POST("", notaHandler.CriarNota)
			notas.GET("", notaHandler.ListarNotas)
			notas.GET("/:numero", notaHandler.BuscarNota)
			notas.POST("/:numero/imprimir", notaHandler.ImprimirNota)
		}
	}

	log.Printf("faturamento-service ouvindo na porta %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("erro ao subir servidor: %v", err)
	}
}

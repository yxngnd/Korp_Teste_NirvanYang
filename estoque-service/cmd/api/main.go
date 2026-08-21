// Comando principal do estoque-service: carrega config, aplica migrations,
// conecta no banco, monta as camadas (repository -> service -> handler) e
// sobe o servidor HTTP.
package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/yxngnd/estoque-service/internal/config"
	"github.com/yxngnd/estoque-service/internal/db"
	"github.com/yxngnd/estoque-service/internal/handler"
	"github.com/yxngnd/estoque-service/internal/middleware"
	"github.com/yxngnd/estoque-service/internal/repository"
	"github.com/yxngnd/estoque-service/internal/service"
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

	// Composição das camadas: repository -> service -> handler.
	// Cada camada só conhece a interface da camada anterior, nunca a
	// implementação concreta — facilita substituir por mocks em testes.
	produtoRepo := repository.NewProdutoRepository(dbConn)
	produtoService := service.NewProdutoService(produtoRepo)
	produtoHandler := handler.NewProdutoHandler(produtoService)

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
		produtos := v1.Group("/produtos")
		{
			produtos.POST("", produtoHandler.CriarProduto)
			produtos.GET("", produtoHandler.ListarProdutos)
			produtos.GET("/:codigo", produtoHandler.BuscarProduto)
			produtos.PATCH("/:codigo/saldo", produtoHandler.AtualizarSaldo)
		}
	}

	log.Printf("estoque-service ouvindo na porta %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("erro ao subir servidor: %v", err)
	}
}

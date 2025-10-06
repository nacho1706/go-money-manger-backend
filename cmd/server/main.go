package main

import (
	"log"
	"net/http"
	"os"
	"test-go2/ent"
	routes "test-go2/internal/http"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func main() {
	mysqlenv := os.Getenv("DATABASE_URL")

	if mysqlenv == "" {
		err := godotenv.Load()
		if err != nil {
			log.Print("Advertencia: No se encontró el archivo .env")
		}
		mysqlenv = os.Getenv("DATABASE_URL")
	}
	if mysqlenv == "" {
		log.Fatalf("Error al conectar la DB: La variable DATABASE_URL no está configurada en el entorno ni en el archivo .env")
	}

	client, err := ent.Open("mysql", mysqlenv)
	if err != nil {
		log.Fatal("Error loading credentials: ", mysqlenv)
	}
	defer client.Close()

	r := gin.Default()
	routes.SetupRoutes(r, client)

	log.Println("Servidor corriendo en :8080")
	if err := r.Run(":8080"); err != nil && err != http.ErrServerClosed {
		log.Fatalf("❌ error al iniciar server: %v", err)
	}
}

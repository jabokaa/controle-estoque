// Package database cuida da conexão com o banco de dados e das migrations.
package database

import (
	"embed"
	"fmt"
	"time"

	"github.com/glebarez/sqlite" // driver SQLite para o GORM, escrito em Go puro (sem CGO)
	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Connect abre a conexão com o banco usando o GORM.
func Connect(driver, dsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch driver {
	case "sqlite":
		dialector = sqlite.Open(dsn)
	default:
		return nil, fmt.Errorf("driver de banco não suportado: %s", driver)
	}

	connection, err := gorm.Open(dialector, &gorm.Config{
		// Grava created_at/updated_at sempre em UTC, igual ao CURRENT_TIMESTAMP do SQLite.
		// A conversão para o fuso local acontece ao ler (ver toEntity nos repositórios).
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, err
	}
	return connection, nil
}

// Close fecha a conexão com o banco.
func Close(connection *gorm.DB) error {
	sqlConnection, err := connection.DB()
	if err != nil {
		return err
	}
	return sqlConnection.Close()
}

// Migrate executa as migrations pendentes (embutidas no binário) com o goose.
func Migrate(connection *gorm.DB) error {
	sqlConnection, err := connection.DB()
	if err != nil {
		return err
	}

	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	return goose.Up(sqlConnection, "migrations")
}

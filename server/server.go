package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Estrutua para mapear a resposta da API
type CotacaoResponse struct {
	USDBRL struct {
		Bid string `json:"bid"`
	} `json:"USDBRL"`
}

func main() {
	http.HandleFunc("/cotacao", cotacaoHandler)

	fmt.Println("Servidor rodando na porta 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Erro ao iniciar o servidor:", err)
	}
}

func cotacaoHandler(w http.ResponseWriter, r *http.Request) {
	// Cria um context com timeout de 200ms
	ctx, cancel := context.WithTimeout(r.Context(), 200*time.Millisecond)
	defer cancel()

	bid, err := buscarCotacao(ctx)
	if err != nil {
		http.Error(w, "Erro ao buscar cotação: "+err.Error(), http.StatusRequestTimeout)
		fmt.Println("Erro ao buscar cotação:", err)
		return
	}

	// Abre conexão com o banco de dados SQLite
	db, err := sql.Open("sqlite3", "./cotacoes.db")
	if err != nil {
		http.Error(w, "Erro ao conectar ao banco de dados", http.StatusInternalServerError)
		fmt.Println("Erro ao conectar ao banco de dados:", err)
		return
	}
	defer db.Close()

	// Timeout de 10ms para a operação de inserção
	ctxBD, cancelBD := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancelBD()

	if err := salvarCotacaoNoBD(ctxBD, db, bid); err != nil {
		fmt.Println("Erro ao salvar cotação no banco de dados:", err)
	}

	// Retorna o valor "bid" como resposta JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"bid": bid})
}

func buscarCotacao(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return "", err
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var cotacao CotacaoResponse
	if err := json.NewDecoder(resp.Body).Decode(&cotacao); err != nil {
		return "", err
	}

	return cotacao.USDBRL.Bid, nil
}

func salvarCotacaoNoBD(ctx context.Context, db *sql.DB, bid string) error {
	ctxBD, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()

	// Prepara a inserção no banco de dados
	stmt, err := db.PrepareContext(ctxBD, "INSERT INTO cotacoes (bid, data) VALUES (?, datetime('now'))")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctxBD, bid)
	return err
}

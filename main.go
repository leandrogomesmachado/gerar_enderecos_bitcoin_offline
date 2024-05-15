package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/ethereum/go-ethereum/crypto"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/ripemd160"
)

func CreateTablesIfNotExist(db *sql.DB) {
	createTableSQL := `CREATE TABLE IF NOT EXISTS bitcoin (
		"id" INTEGER PRIMARY KEY AUTOINCREMENT,
		"endereco" TEXT NOT NULL,
		"chave_privada" TEXT NOT NULL
	);`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Erro ao criar tabela: %v", err)
	}

	fmt.Println("Tabelas criadas com sucesso caso não existissem tabelas anteriores.")
}

func GenerateKeyPair() (string, string) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatalf("Erro ao gerar a chave privada: %v", err)
	}
	privateKeyBytes := crypto.FromECDSA(privateKey)
	publicKey := privateKey.PublicKey
	publicKeyBytes := crypto.FromECDSAPub(&publicKey)

	compressedPublicKey := append([]byte{0x02 + byte(publicKey.Y.Bit(0))}, publicKeyBytes[1:33]...)

	sha256Hash := sha256.Sum256(compressedPublicKey)

	ripemd160Hasher := ripemd160.New()
	ripemd160Hasher.Write(sha256Hash[:])
	hashedPublicKey := ripemd160Hasher.Sum(nil)

	versionedHashedPublicKey := append([]byte{0x00}, hashedPublicKey...)

	sha256Hash1 := sha256.Sum256(versionedHashedPublicKey)
	sha256Hash2 := sha256.Sum256(sha256Hash1[:])

	checksum := sha256Hash2[:4]

	binAddress := append(versionedHashedPublicKey, checksum...)

	bitcoinAddress := base58.Encode(binAddress)

	privateKeyHex := fmt.Sprintf("%x", privateKeyBytes)

	return bitcoinAddress, privateKeyHex
}

func SaveInfo(db *sql.DB, address string, pk string) {
	insertSQL := `INSERT INTO bitcoin (endereco, chave_privada) VALUES (?, ?)`
	_, err := db.Exec(insertSQL, address, pk)
	if err != nil {
		log.Fatalf("Erro ao inserir dados: %v", err)
	}
	fmt.Println("Novo endereço Bitcoin criado e salvo no Banco de dados!\nPara parar de gerar novos endereços, aperte CTRL + C")
}

func main() {
	dbDirectory := "db"
	databasePath := dbDirectory + "/dados.db"

	if _, err := os.Stat(dbDirectory); os.IsNotExist(err) {
		err := os.Mkdir(dbDirectory, 0755)
		if err != nil {
			log.Fatalf("Erro ao criar diretório: %v", err)
		}
		fmt.Println("Diretório 'db' criado com sucesso.")
	}

	db, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		log.Fatalf("Erro ao abrir o Banco de Dados: %v", err)
	}
	defer db.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nSaindo...")
		db.Close()
		os.Exit(0)
	}()

	for {
		var choice int
		fmt.Println("SELECIONE UMA OPÇÃO ABAIXO:")
		fmt.Println("1. Criar Banco de Dados")
		fmt.Println("2. Gerar Endereços")
		fmt.Println("3. Sair")
		fmt.Scan(&choice)

		switch choice {
		case 1:
			CreateTablesIfNotExist(db)
		case 2:
			fmt.Println("Gerando Endereços... (Pressione CTRL + C para parar)")
			go func() {
				for {
					address, privateKey := GenerateKeyPair()
					SaveInfo(db, address, privateKey)
					time.Sleep(time.Millisecond * 500)
				}
			}()
		case 3:
			fmt.Println("Saindo...")
			return
		default:
			fmt.Println("Opção inválida. Selecione uma das mencionadas acima.")
		}
	}
}

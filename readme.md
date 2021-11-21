# Golang Blockchain

## Usage

- Inicializar a blockchain:

```cmd
    go run main.go createblockchain -address "Satoshi"
```

- Mostrar saldo de uma carteira:

```cmd
    go run main.go getbalance -address "Satoshi"
```

- Enviar moedas de uma carteira para outra:

```cmd
    go run main.go send -from "Satoshi" -to "John" -amount 50
```

- Mostrar todos os blocos:

```cmd
    go run main.go printchain
```

## Tutoriais 

- [Youtube](https://www.youtube.com/playlist?list=PLpP5MQvVi4PGmNYGEsShrlvuE2B33xV1L)

## BadgerDB

- [Github](https://github.com/dgraph-io/badger)
- [Documentação](https://dgraph.io/docs/badger/get-started/)
- [Introdução](https://dgraph.io/blog/post/badger/)
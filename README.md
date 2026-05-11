# Delta Bank Backend

API RESTful para um sistema bancário digital construída em **Golang** com **Clean Architecture** e **PostgreSQL**. O projeto implementa operações bancárias essenciais como criação de contas, autenticação, transferências via PIX, depósitos, saques e gerenciamento de chaves PIX — tudo com validações robustas de regras de negócio e segurança com hash de senhas via bcrypt.

> **API em produção:** [https://delta-bank-backend-in-golang-production.up.railway.app](https://delta-bank-backend-in-golang-production.up.railway.app)

## Sumário

- [Deploy](#deploy)
- [Tecnologias](#tecnologias)
- [Arquitetura](#arquitetura)
- [Pré-requisitos](#pré-requisitos)
- [Instalação e Execução](#instalação-e-execução)
- [Variáveis de Ambiente](#variáveis-de-ambiente)
- [Estrutura do Banco de Dados](#estrutura-do-banco-de-dados)
- [Endpoints da API](#endpoints-da-api)
- [Exemplos de Requisição](#exemplos-de-requisição)
- [Validações e Regras de Negócio](#validações-e-regras-de-negócio)
- [Tratamento de Erros](#tratamento-de-erros)
- [Estrutura de Diretórios](#estrutura-de-diretórios)

## Deploy

A API está hospedada em produção no **Railway** e pode ser acessada diretamente:

```
https://delta-bank-backend-in-golang-production.up.railway.app
```

Todas as rotas estão disponíveis sob o prefixo `/api`. Exemplo rápido:

```bash
# Criar uma conta na API de produção
curl -X POST https://delta-bank-backend-in-golang-production.up.railway.app/api/contas \
  -H "Content-Type: application/json" \
  -d '{"nome": "João Silva", "cpf": "12345678909", "senha": "minhasenha", "saldo_inicial": 100.00}'

# Consultar saldo
curl https://delta-bank-backend-in-golang-production.up.railway.app/api/saldo/12345678909
```

O deploy é feito automaticamente via Railway a partir do repositório GitHub, com as variáveis de ambiente (`DATABASE_URL` e `PORT`) configuradas no painel da plataforma.

## Tecnologias

| Tecnologia | Versão | Descrição |
|------------|--------|-----------|
| **Go** | 1.26.2 | Linguagem principal do backend |
| **Gorilla Mux** | v1.8.1 | Router HTTP com suporte a parâmetros e sub-routers |
| **pgx** | v5.9.2 | Driver PostgreSQL nativo de alto desempenho |
| **bcrypt** (golang.org/x/crypto) | v0.51.0 | Hash seguro de senhas |
| **godotenv** | v1.5.1 | Carregamento de variáveis de ambiente via arquivo `.env` |
| **PostgreSQL** | — | Banco de dados relacional |

## Arquitetura

O projeto segue o padrão **Clean Architecture**, separando as responsabilidades em camadas bem definidas com dependências unidirecionais (as camadas internas não conhecem as externas):

```
┌──────────────────────────────────────────────────┐
│                  Handler (API)                    │  ← Recebe HTTP, parseia DTOs
├──────────────────────────────────────────────────┤
│                  Use Case (Service)               │  ← Regras de negócio / orquestração
├──────────────────────────────────────────────────┤
│                  Domain (Entities + Errors)       │  ← Entidades e erros do domínio
├──────────────────────────────────────────────────┤
│             Repository (Interfaces + Postgres)    │  ← Acesso a dados / persistência
├──────────────────────────────────────────────────┤
│                Validation (Pure Functions)        │  ← Validações puras (CPF, email, senha)
└──────────────────────────────────────────────────┘
```

**Fluxo de dependência:** `main → Handler → UseCase → Repository → Database`

A camada de **domínio** é o centro da aplicação — ela não depende de nenhuma outra camada. As camadas externas dependem das internas, nunca o contrário. Isso garante que as regras de negócio sejam independentes de frameworks, banco de dados e detalhes de infraestrutura.

## Pré-requisitos

- **Go** 1.26.2 ou superior instalado
- **PostgreSQL** 12+ em execução
- Banco de dados criado com as tabelas necessárias (veja a seção abaixo)

## Instalação e Execução

### 1. Clone o repositório

```bash
git clone https://github.com/Joshua-likesjs/Delta-Bank-Backend-In-Golang.git
cd Delta-Bank-Backend-In-Golang
```

### 2. Instale as dependências

```bash
go mod tidy
```

### 3. Configure o banco de dados

Crie o banco de dados PostgreSQL e execute os seguintes comandos SQL para criar as tabelas:

```sql
CREATE TABLE contas (
    cpf            VARCHAR(11) PRIMARY KEY,
    owner          VARCHAR(100) NOT NULL,
    balance        INTEGER NOT NULL DEFAULT 0,
    senha          VARCHAR(255) NOT NULL,
    limite_diario  INTEGER NOT NULL DEFAULT 5000000,
    created_at     TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP
);

CREATE TABLE transacoes (
    id                      SERIAL PRIMARY KEY,
    conta_cpf_remetente     VARCHAR(11) NOT NULL REFERENCES contas(cpf),
    conta_cpf_destinatario  VARCHAR(11) NOT NULL REFERENCES contas(cpf),
    tipo                    VARCHAR(20) NOT NULL,
    valor_centavos          INTEGER NOT NULL,
    created_at              TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE chaves_pix (
    id         SERIAL PRIMARY KEY,
    conta_cpf  VARCHAR(11) NOT NULL REFERENCES contas(cpf),
    tipo       VARCHAR(20) NOT NULL,
    valor      VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### 4. Configure as variáveis de ambiente

```bash
export DATABASE_URL="postgres://usuario:senha@localhost:5432/nome_do_banco"
export PORT="8080"
```

Ou crie um arquivo `.env` na raiz do projeto (a importação do `godotenv` está comentada no `main.go`, mas pode ser ativada):

```env
DATABASE_URL=postgres://usuario:senha@localhost:5432/nome_do_banco
PORT=8080
```

### 5. Execute o servidor

```bash
go run cmd/server/main.go
```

O servidor iniciará na porta configurada (padrão: `8080`):

```
 🌐 API REST - CLEAN ARCHITECTURE v9.0 

🚀 Servidor em :8080
```

## Variáveis de Ambiente

| Variável | Obrigatória | Padrão | Descrição |
|----------|-------------|--------|-----------|
| `DATABASE_URL` | Sim | — | String de conexão PostgreSQL (`postgres://user:pass@host:port/db`) |
| `PORT` | Não | `8080` | Porta em que o servidor HTTP escuta |

## Estrutura do Banco de Dados

O sistema utiliza **3 tabelas** principais:

### `contas` — Contas bancárias
| Coluna | Tipo | Descrição |
|--------|------|-----------|
| `cpf` | VARCHAR(11) PK | CPF do titular (sem pontuação) |
| `owner` | VARCHAR(100) | Nome do titular |
| `balance` | INTEGER | Saldo em centavos (R$ 1,00 = 100) |
| `senha` | VARCHAR(255) | Hash bcrypt da senha |
| `limite_diario` | INTEGER | Limite diário de PIX em centavos |
| `created_at` | TIMESTAMP | Data de criação da conta |
| `updated_at` | TIMESTAMP | Data da última atualização |

### `transacoes` — Movimentações financeiras
| Coluna | Tipo | Descrição |
|--------|------|-----------|
| `id` | SERIAL PK | ID auto-incremento |
| `conta_cpf_remetente` | VARCHAR(11) FK | CPF de quem enviou |
| `conta_cpf_destinatario` | VARCHAR(11) FK | CPF de quem recebeu |
| `tipo` | VARCHAR(20) | Tipo: `TRANSACAO_PIX`, `DEPOSITO`, `SAQUE` |
| `valor_centavos` | INTEGER | Valor em centavos |
| `created_at` | TIMESTAMP | Data/hora da transação |

### `chaves_pix` — Chaves PIX cadastradas
| Coluna | Tipo | Descrição |
|--------|------|-----------|
| `id` | SERIAL PK | ID auto-incremento |
| `conta_cpf` | VARCHAR(11) FK | CPF do titular da chave |
| `tipo` | VARCHAR(20) | Tipo: `CPF`, `Email`, `Telefone`, `Aleatoria` |
| `valor` | VARCHAR(100) UNIQUE | Valor da chave PIX |
| `created_at` | TIMESTAMP | Data de cadastro |

## Endpoints da API

Todos os endpoints estão sob o prefixo `/api`.

### Contas

| Método | Rota | Descrição |
|--------|------|-----------|
| `POST` | `/api/contas` | Criar nova conta bancária |
| `POST` | `/api/login` | Autenticar login por CPF e senha |
| `GET` | `/api/saldo/{cpf}` | Consultar saldo da conta |
| `GET` | `/api/extrato/{cpf}` | Obter extrato de transações |
| `PUT` | `/api/dados` | Editar nome e limite diário |
| `PUT` | `/api/senha` | Alterar senha da conta |

### Operações Financeiras

| Método | Rota | Descrição |
|--------|------|-----------|
| `POST` | `/api/pix` | Realizar transferência PIX |
| `POST` | `/api/depositar` | Realizar depósito |
| `POST` | `/api/sacar` | Realizar saque |

### Chaves PIX

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/api/chaves-pix/{cpf}` | Listar chaves PIX da conta |
| `POST` | `/api/chaves-pix` | Adicionar nova chave PIX |
| `DELETE` | `/api/chaves-pix` | Remover chave PIX |

## Exemplos de Requisição

### Criar Conta

```bash
curl -X POST https://delta-bank-backend-in-golang-production.up.railway.app/api/contas \
  -H "Content-Type: application/json" \
  -d '{
    "nome": "João Silva",
    "cpf": "12345678909",
    "senha": "minhasenha",
    "saldo_inicial": 100.00
  }'
```

**Resposta:**
```json
{
  "sucesso": true,
  "mensagem": "Conta criada!",
  "dados": {
    "cpf": "12345678909",
    "nome": "João Silva",
    "saldo_centavos": 10000,
    "limite_diario": 5000000,
    "criada_em": "2025-01-15T10:30:00Z"
  }
}
```

### Login

```bash
curl -X POST https://delta-bank-backend-in-golang-production.up.railway.app/api/login \
  -H "Content-Type: application/json" \
  -d '{
    "cpf": "12345678909",
    "senha": "minhasenha"
  }'
```

**Resposta:**
```json
{
  "sucesso": true,
  "mensagem": "Login OK!",
  "dados": {
    "cpf": "12345678909",
    "nome": "João Silva",
    "saldo_centavos": 10000
  }
}
```

### Consultar Saldo

```bash
curl https://delta-bank-backend-in-golang-production.up.railway.app/api/saldo/12345678909
```

**Resposta:**
```json
{
  "sucesso": true,
  "dados": {
    "Conta": {
      "cpf": "12345678909",
      "nome": "João Silva",
      "saldo_centavos": 10000,
      "limite_diario": 5000000
    },
    "QtdChaves": 2
  }
}
```

### Fazer PIX

```bash
curl -X POST https://delta-bank-backend-in-golang-production.up.railway.app/api/pix \
  -H "Content-Type: application/json" \
  -d '{
    "cpf_origem": "12345678909",
    "cpf_destino": "98765432100",
    "valor": 50.00
  }'
```

**Resposta:**
```json
{
  "sucesso": true,
  "mensagem": "PIX realizado!",
  "dados": {
    "id": 15,
    "tipo": "TRANSACAO_PIX",
    "valor_centavos": 5000,
    "cpf_remetente": "12345678909",
    "cpf_destinatario": "98765432100",
    "nome_remetente": "João Silva",
    "nome_destinatario": "Maria Souza",
    "data_hora": "2025-01-15T11:00:00Z"
  }
}
```

### Depositar

```bash
curl -X POST https://delta-bank-backend-in-golang-production.up.railway.app/api/depositar \
  -H "Content-Type: application/json" \
  -d '{
    "cpf": "12345678909",
    "valor": 200.00
  }'
```

### Sacar

```bash
curl -X POST https://delta-bank-backend-in-golang-production.up.railway.app/api/sacar \
  -H "Content-Type: application/json" \
  -d '{
    "cpf": "12345678909",
    "valor": 50.00
  }'
```

### Consultar Extrato

```bash
curl https://delta-bank-backend-in-golang-production.up.railway.app/api/extrato/12345678909
```

**Resposta:**
```json
{
  "sucesso": true,
  "dados": {
    "transacoes": [
      {
        "id": 15,
        "tipo": "TRANSACAO_PIX",
        "valor_centavos": 5000,
        "cpf_remetente": "12345678909",
        "cpf_destinatario": "98765432100",
        "nome_remetente": "João Silva",
        "nome_destinatario": "Maria Souza",
        "data_hora": "2025-01-15T11:00:00Z"
      }
    ],
    "saldo_atual": 5000
  }
}
```

### Adicionar Chave PIX

```bash
curl -X POST https://delta-bank-backend-in-golang-production.up.railway.app/api/chaves-pix \
  -H "Content-Type: application/json" \
  -d '{
    "cpf": "12345678909",
    "tipo": "Email",
    "valor": "joao@email.com"
  }'
```

### Remover Chave PIX

```bash
curl -X DELETE https://delta-bank-backend-in-golang-production.up.railway.app/api/chaves-pix \
  -H "Content-Type: application/json" \
  -d '{
    "cpf": "12345678909",
    "tipo": "Email",
    "valor": "joao@email.com"
  }'
```

### Alterar Senha

```bash
curl -X PUT https://delta-bank-backend-in-golang-production.up.railway.app/api/senha \
  -H "Content-Type: application/json" \
  -d '{
    "cpf": "12345678909",
    "senha_atual": "minhasenha",
    "nova_senha": "novasenha123"
  }'
```

### Editar Dados da Conta

```bash
curl -X PUT https://delta-bank-backend-in-golang-production.up.railway.app/api/dados \
  -H "Content-Type: application/json" \
  -d '{
    "cpf": "12345678909",
    "novo_nome": "João Silva Santos",
    "novo_limite_diario": 3000.00
  }'
```

## Validações e Regras de Negócio

O sistema aplica diversas validações e regras de negócio para garantir a integridade das operações:

### Contas
- **Nome**: mínimo de 3 caracteres
- **CPF**: validação completa com dígitos verificadores (algoritmo oficial da Receita Federal)
- **Senha**: mínimo de 6 caracteres; armazenada como hash bcrypt (nunca em texto plano)
- **Saldo inicial**: mínimo de R$ 10,00 para abertura de conta
- **CPF duplicado**: não é permitido cadastrar o mesmo CPF mais de uma vez

### PIX
- **Valor mínimo**: R$ 1,00 por transferência
- **Limite diário**: cada conta possui um limite diário de PIX (padrão: R$ 50.000,00)
- **Auto-transferência**: não é permitido enviar PIX para a própria conta
- **Saldo insuficiente**: a transferência é bloqueada se o saldo não cobrir o valor
- **Limite de chaves**: cada conta pode ter no máximo 5 chaves PIX cadastradas
- **Chave única**: cada valor de chave PIX é único no sistema (não pode ser duplicado)

### Depósitos
- Valor deve estar entre R$ 0,01 e R$ 10.000,00

### Saques
- Valor deve estar entre R$ 0,01 e R$ 2.000,00
- O valor do saque deve ser múltiplo de R$ 10,00 (notas de caixa eletrônico)
- Saldo deve ser suficiente para cobrir o saque

### Chaves PIX
- Tipos suportados: `CPF`, `Email`, `Telefone`, `Aleatoria`
- Validação de email com regex e verificação de formato
- Validação de telefone brasileiro com verificação de DDD válido
- Geração de chave aleatória no formato UUID v4

## Tratamento de Erros

Todos os erros seguem um formato padrão de resposta, facilitando o consumo da API por clientes:

```json
{
  "sucesso": false,
  "mensagem": "descrição do erro"
}
```

### Códigos de Erro do Domínio

| Erro | HTTP | Descrição |
|------|------|-----------|
| `conta não encontrada` | 404 | CPF não existe no sistema |
| `CPF já cadastrado` | 409 | CPF já possui conta |
| `CPF inválido` | 400 | CPF falhou na validação dos dígitos |
| `nome deve ter no mínimo 3 caracteres` | 400 | Nome muito curto |
| `senha deve ter no mínimo 6 caracteres` | 400 | Senha muito curta |
| `senha incorreta` | 401 | Senha não confere com o hash |
| `conta bloqueada por muitas tentativas` | 423 | Conta temporariamente bloqueada |
| `saldo insuficiente` | 400 | Saldo menor que o valor da operação |
| `limite diário excedido` | 400 | Limite diário de PIX atingido |
| `valor inválido` | 400 | Valor fora do intervalo permitido |
| `destinatário não encontrado` | 404 | CPF do destinatário não existe |
| `não pode transferir para si mesmo` | 400 | PIX para o próprio CPF |
| `chave PIX já cadastrada no sistema` | 409 | Valor da chave já existe |
| `limite de 5 chaves PIX atingido` | 400 | Conta já tem 5 chaves |
| `chave PIX não encontrada` | 404 | Chave não existe para a conta |
| `erro interno do servidor` | 500 | Erro inesperado no servidor |
| `não autorizado` | 403 | Acesso não autorizado |

## Estrutura de Diretórios

```
Delta-Bank-Backend-In-Golang/
├── cmd/
│   └── server/
│       └── main.go                 # Ponto de entrada da aplicação
├── internal/
│   ├── domain/
│   │   ├── entities.go             # Entidades: Conta, Transacao, ChavePix
│   │   └── errors.go               # Erros de domínio padronizados
│   ├── handler/
│   │   ├── api_handler.go          # Handlers HTTP (controllers)
│   │   └── dto.go                  # DTOs de entrada/saída da API
│   ├── usecase/
│   │   ├── service.go              # Lógica de negócio (use cases)
│   │   └── ports.go                # Interfaces e structs de input/output
│   ├── repository/
│   │   ├── repository.go           # Interfaces dos repositórios
│   │   └── postgres.go             # Implementação PostgreSQL
│   └── validation/
│       └── validators.go           # Funções puras de validação
├── go.mod                          # Módulo e dependências Go
├── go.sum                          # Checksums das dependências
└── README.md                       # Este arquivo
```

### Descrição dos Pacotes

- **`cmd/server`** — Contém o `main.go`, responsável por inicializar a conexão com o banco, instanciar as camadas (repository → usecase → handler), configurar as rotas e iniciar o servidor HTTP.

- **`internal/domain`** — O coração da aplicação. Define as entidades de negócio (`Conta`, `Transacao`, `ChavePix`) e os erros de domínio. Esta camada não possui nenhuma dependência externa.

- **`internal/handler`** — Camada de apresentação. Os handlers recebem requisições HTTP, decodificam os DTOs de entrada, chamam os use cases e formatam as respostas padronizadas (`RespostaAPI`). Inclui suporte a CORS para consumo por frontends.

- **`internal/usecase`** — Camada de aplicação. Contém toda a orquestração das regras de negócio, como validação de dados, verificação de limites, cálculos financeiros e coordenação das operações entre entidades. Define as interfaces (ports) que desacoplam o service dos handlers.

- **`internal/repository`** — Camada de dados. Define as interfaces de repositório (`ContaRepository`, `TransacaoRepository`, `ChavePixRepository`) e sua implementação concreta com PostgreSQL via pgx.

- **`internal/validation`** — Funções puras de validação sem efeitos colaterais: validação de CPF (com dígitos verificadores), email, telefone brasileiro, senhas, e utilitários de hash bcrypt. Independente de qualquer outra camada.

---

Feito com Go e Clean Architecture.

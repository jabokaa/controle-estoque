# De Laravel para Clean Architecture

Guia rápido para quem conhece Laravel no padrão **MVC + Service + Repository** e vai trabalhar neste projeto Go organizado em **Clean Architecture**.

## De-para

| Laravel | Neste projeto | Pasta | Camada |
| --- | --- | --- | --- |
| `routes/web.php` | Router | `internal/delivery/http/router` | Adaptadores |
| Controller | Handler | `internal/delivery/http/handler` | Adaptadores |
| FormRequest (validação) | Handler (formato dos dados) + Entidade (regras) | `handler` / `internal/domain/entity` | Adaptadores / Domínio |
| Service | Caso de uso (um por ação) | `internal/usecase` | Casos de Uso |
| Model (atributos e regras) | Entidade | `internal/domain/entity` | Domínio |
| Interface do Repository | Interface do repositório | `internal/domain/repository` | Domínio |
| Repository com Eloquent | Repositório SQL | `internal/infrastructure/repository` | Adaptadores |
| `config/database.php` | Conexão com o banco | `internal/infrastructure/database` | Adaptadores |
| `database/migrations` | Migrations SQL (goose) | `internal/infrastructure/database/migrations` | Adaptadores |
| Blade (`resources/views`) | Templates `html/template` | `web/templates` | Frameworks e Drivers |
| `public/` (CSS, JS) | Arquivos estáticos | `web/static` | Frameworks e Drivers |
| `.env` + `config/*.php` | `.env` + pacote `config` | `internal/config` | Frameworks e Drivers |
| Service Container / `AppServiceProvider` | `main.go` (composition root) | `cmd/server` | Frameworks e Drivers |

## O que muda de verdade

**1. O Model é dividido em dois.**
No Laravel, o Model Eloquent guarda os dados **e** sabe se salvar (`$categoria->save()`). Aqui, a **entidade** é uma struct pura, sem nenhum acesso ao banco, e quem salva é o **repositório**.

**2. A interface do repositório fica no domínio.**
No Laravel, a interface costuma ficar ao lado da implementação, em `app/Repositories`. Aqui, ela fica em `internal/domain/repository`, porque quem define o contrato é quem usa (o caso de uso), não quem implementa (a infraestrutura).

**3. Um caso de uso por ação.**
No Laravel, é comum um `CategoriaService` com vários métodos. Aqui, cada ação é um caso de uso separado: `ListarCategorias`, `CadastrarCategoria` e assim por diante. Cada arquivo fica pequeno e recebe só as dependências de que precisa.

**4. A injeção de dependência é explícita.**
No Laravel, o container resolve as dependências sozinho pelo type-hint do construtor. Em Go, você monta tudo à mão no `main.go`:

```go
categoryRepo := repository.NewCategoryRepositorySQL(db)
listCategories := usecase.NewListCategories(categoryRepo)
categoryHandler := handler.NewCategoryHandler(listCategories)
```

É o equivalente a `$this->app->bind(CategoriaRepositoryInterface::class, CategoriaRepository::class)`.

**5. As dependências só apontam para dentro.**

```
cmd / web  →  delivery · infrastructure  →  usecase  →  domain
```

O domínio não importa nenhuma outra camada. O caso de uso não sabe que existe HTTP, HTML ou SQL.

## Exemplo lado a lado: listar categorias

| Passo | Laravel | Go (este projeto) |
| --- | --- | --- |
| Rota | `Route::get('/categorias', [CategoriaController::class, 'index']);` | `mux.HandleFunc("GET /categories", a.CategoryHandler.List)` |
| Controller / Handler | `return view('categorias.index', ['categorias' => $this->service->listar()]);` | `categorias, err := h.listCategories.Execute(r.Context())` e depois `templates.ExecuteTemplate(w, "categories.html", dados)` |
| Service / Caso de uso | `return $this->repo->all();` | `return uc.repo.List(ctx)` |
| Interface | `interface CategoriaRepositoryInterface { public function all(); }` | `type CategoryRepository interface { List(ctx) ([]entity.Category, error) }` |
| Repository | `return Categoria::orderBy('nome')->get();` | `r.db.QueryContext(ctx, "SELECT ... FROM categories ORDER BY name")` |
| Model / Entidade | `class Categoria extends Model {}` | `type Category struct { ID int64; Name string; ... }` |
| View | `@foreach($categorias as $c) {{ $c->nome }} @endforeach` | `{{range .Categories}} {{.Name}} {{end}}` |

## Equivalência de comandos

| Laravel | Go |
| --- | --- |
| `php artisan serve` | `go run ./cmd/server` |
| `composer require pacote` | `go get pacote` |
| `composer.json` / `composer.lock` | `go.mod` / `go.sum` |
| `php artisan migrate` | `goose up` (ou automático ao subir o servidor) |
| `php artisan migrate:rollback` | `goose down` |
| `php artisan make:migration` | `goose create nome sql` |

## Onde colocar cada coisa (regra rápida)

- **É uma regra que vale sempre?** Por exemplo, "o nome é obrigatório". Vai na **entidade**.
- **É o passo a passo de uma ação do usuário?** Vai no **caso de uso**.
- **Fala com o banco?** Vai no **repositório SQL**.
- **Lê a requisição ou decide a resposta HTTP?** Vai no **handler**.
- **É HTML ou CSS?** Vai em **`web/`**.
- **Liga uma peça na outra?** Vai no **`main.go`**.

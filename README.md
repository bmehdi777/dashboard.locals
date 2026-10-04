# dashboard.locals

## Serveur

Le serveur Go est le point d'accès local de l'application. Il initialise SQLite,
expose l'API versionnée sous `/api/v1`, orchestre `ripgrep`, l'ouverture dans
l'éditeur et la synchronisation optionnelle avec OpenCode.

```sh
go run ./cmd/server
```

La base est créée par défaut dans :

```text
~/.config/dashboard.locals/database.sqlite
```

`XDG_CONFIG_HOME` et les variables `DASHBOARD_LOCALS_*` permettent de définir
les chemins et les timeouts opérationnels.

## Vérifications

```sh
make quality
make build-server
```

La collection Bruno contenant les requêtes de l'API se trouve dans
`.bruno`. L'environnement `local` utilise par défaut
`http://127.0.0.1:8080`.

## CLI

La CLI communique avec le daemon uniquement via l'API HTTP. Le daemon doit être
démarré avant d'utiliser les commandes :

```sh
go run ./cmd/server
go run ./cmd/dashboard server health
```

L'URL du daemon peut être modifiée avec `--server-url` ou avec la variable
`DASHBOARD_LOCALS_SERVER_URL`. Les commandes qui produisent des données
supportent `--json` pour l'automatisation.

Quelques exemples :

```sh
dashboard config get
dashboard search roots list --include-disabled
dashboard search run "TODO" --root <root-id>
dashboard stats sync --from 2026-01-01T00:00:00Z
dashboard stats compact --before 2026-01-01T00:00:00Z --dry-run
dashboard stats compact --before 2026-01-01T00:00:00Z --drop-raw
```

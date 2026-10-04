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

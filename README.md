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

## Raccourcis

Les raccourcis sont enregistrés dans SQLite et peuvent être gérés depuis la
page **Raccourcis** de l'interface. L'accueil affiche automatiquement les cinq
liens les plus ouverts. L'API correspondante est :

```text
GET    /api/v1/shortcuts?sort=recent|popular&limit=5
POST   /api/v1/shortcuts
PATCH  /api/v1/shortcuts/{shortcutID}
DELETE /api/v1/shortcuts/{shortcutID}
POST   /api/v1/shortcuts/{shortcutID}/use
```

Les URLs sont validées côté serveur et doivent utiliser `http` ou `https`.
L'appel `use` est effectué lorsque l'utilisateur ouvre un panneau afin de
maintenir le classement des liens les plus utilisés.

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

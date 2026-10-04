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

## Logs

Les logs sont écrits par défaut dans :

```text
/var/log/dashboard.locals/access.log
/var/log/dashboard.locals/error.log
```

L'unité systemd provisionne automatiquement le répertoire de logs. Pour une
exécution manuelle ou un autre emplacement, utilisez :

```sh
DASHBOARD_LOCALS_LOG_DIR="$HOME/.local/state/dashboard.locals/log" \
  go run ./cmd/server
```

Les requêtes d'accès n'enregistrent pas les query strings, les corps JSON, les
tokens ou les credentials.

## Vérifications

```sh
make quality
make build-server
```

## Installation

L'installation par défaut est prévue pour l'utilisateur courant et utilise
`systemd --user`. Elle ne nécessite pas `sudo` : le serveur doit conserver le
même utilisateur que la CLI afin d'accéder à la base SQLite, aux racines de
recherche et à l'état local d'OpenCode.

```sh
make install
```

Les exécutables sont installés dans `~/.local/bin` et le service dans
`~/.config/systemd/user/dashboard.locals.service`. Ajoutez le répertoire des
exécutables au `PATH` si nécessaire :

```sh
export PATH="$HOME/.local/bin:$PATH"
```

Vérifier le service :

```sh
systemctl --user status dashboard.locals.service
dashboard server health
journalctl --user -u dashboard.locals.service
```

Les variables `PREFIX`, `BINDIR`, `SYSTEMD_USER_DIR`, `SERVICE_NAME`,
`INSTALL` et `SYSTEMCTL` peuvent être surchargées lors de l'installation. Par
exemple :

```sh
make install PREFIX="$HOME/.local"
```

Le fichier optionnel `~/.config/dashboard.locals/service.env` peut contenir
des variables d'environnement opérationnelles, par exemple
`DASHBOARD_LOCALS_LISTEN_ADDR=127.0.0.1:9090`. Il est chargé par systemd et
n'est pas remplacé par les mises à jour.

Pour mettre à jour l'installation, il suffit de reconstruire le projet puis de
relancer la même cible :

```sh
git pull
make install
```

Les binaires sont remplacés atomiquement, le service est rechargé puis
redémarré, et les données de `~/.config/dashboard.locals` ne sont pas
supprimées. Le démarrage automatique sans session utilisateur ouverte peut
être activé séparément avec `loginctl enable-linger "$USER"` si cela est
nécessaire.

La collection Bruno contenant les requêtes de l'API se trouve dans
`.bruno`. L'environnement `local` utilise par défaut
`http://127.0.0.1:8443`.

## Raccourcis

Les raccourcis sont enregistrés dans SQLite et peuvent être gérés depuis la
page **Raccourcis** de l'interface. L'accueil affiche automatiquement les cinq
liens les plus ouverts. L'API correspondante est :

```text
GET    /api/v1/shortcuts?sort=recent|popular|custom&limit=5
POST   /api/v1/shortcuts
PATCH  /api/v1/shortcuts/{shortcutID}
DELETE /api/v1/shortcuts/{shortcutID}
PUT    /api/v1/shortcuts/order
POST   /api/v1/shortcuts/{shortcutID}/use
GET    /api/v1/shortcuts/{shortcutID}/favicon
GET    /api/v1/shortcut-folders
POST   /api/v1/shortcut-folders
PATCH  /api/v1/shortcut-folders/{folderID}
DELETE /api/v1/shortcut-folders/{folderID}
```

Les URLs sont validées côté serveur et doivent utiliser `http` ou `https`.
L'appel `use` est effectué lorsque l'utilisateur ouvre un panneau afin de
maintenir le classement des liens les plus utilisés. Le classement populaire de
l'accueil reste indépendant des dossiers et de l'ordre personnalisé de la page
**Raccourcis**.

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

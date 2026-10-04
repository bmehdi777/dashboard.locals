# Instructions pour les agents

## Présentation du projet

`dashboard.locals` est une application locale composée de trois parties :

1. un serveur Go qui expose l'API, sert l'interface web et orchestre les
   intégrations locales et externes ;
2. un client React/TypeScript utilisant shadcn/ui ;
3. une CLI Go utilisant Cobra et communiquant avec le serveur.

Le projet a deux objectifs principaux :

- fournir une interface de recherche de code local inspirée du visualiseur de
  code de GitHub, en utilisant `ripgrep` ;
- collecter et visualiser des statistiques d'utilisation de l'IA depuis les
  endpoints proposés par OpenCode.

L'application est conçue pour fonctionner localement. Le serveur Go est le
seul point d'accès du client web.

## Règles impératives

- Le client web communique exclusivement avec le serveur Go.
- Le client ne doit jamais appeler directement OpenCode, `ripgrep`, `xdg-open`
  ou un autre service externe.
- Toutes les routes HTTP d'API doivent être préfixées par `/api/v1`.
- Le serveur Go sert l'application web sur `/`.
- La CLI communique avec le daemon via l'API HTTP et ne doit pas accéder
  directement à la base de données du serveur.
- Les settings applicatifs sont stockés dans SQLite.
- Les commandes externes doivent être exécutées sans shell lorsque cela est
  possible, avec des arguments séparés et contrôlés.
- Les chemins fournis par l'utilisateur doivent être validés et rester dans la
  racine de recherche concernée.
- Ne jamais supprimer ou réinitialiser des données utilisateur sans demande
  explicite.
- Ne jamais exposer de secrets, de tokens OpenCode ou de chemins sensibles dans
  les réponses de l'API, les logs ou l'interface web.

## Structure cible

La structure doit rester organisée par responsabilité :

```text
cmd/
  server/
    main.go             # Point d'entrée du serveur
  dashboard/
    main.go             # Point d'entrée de la CLI Cobra

internal/
  config/               # Configuration et settings persistés
  http/                 # Serveur HTTP, middleware et handlers
    api/
      v1/               # Routes et modèles de l'API version 1
  search/               # Recherche ripgrep et validation des chemins
  opencode/             # Détection, client et synchronisation OpenCode
  store/                # SQLite, GORM, modèles et migrations
  launcher/             # Ouverture des fichiers dans l'éditeur
  stats/                # Collecte, agrégation et compactage des statistiques
  cli/                  # Commandes et clients HTTP de la CLI

assets/
  dashboard-ui/         # Client React/TypeScript/Vite
```

Les noms exacts peuvent évoluer, mais les couches doivent rester séparées.
Les handlers HTTP ne doivent pas contenir directement la logique métier ni
accéder directement à GORM.

## Serveur Go

Le serveur doit :

- initialiser la configuration ;
- créer le répertoire de configuration si nécessaire ;
- ouvrir et migrer la base SQLite ;
- initialiser les services de recherche, statistiques et éditeur ;
- démarrer le serveur HTTP ;
- servir les fichiers frontend compilés à la racine `/` ;
- démarrer la détection OpenCode sans rendre son absence bloquante.

Le client web doit être compilé avant le serveur et embarqué dans le binaire Go
avec `go:embed` pour les builds de production. Le serveur ne doit pas dépendre
d'un serveur Vite en production.

Les routes inconnues sous `/api/v1` doivent retourner une erreur API structurée.
Les routes frontend doivent permettre le fonctionnement d'une SPA, notamment
le fallback vers `index.html` pour les routes non-API appropriées.

## API HTTP

Toutes les routes d'API utilisent le préfixe :

```text
/api/v1
```

Exemples de domaines fonctionnels prévus :

```text
GET    /api/v1/health
GET    /api/v1/settings
PATCH  /api/v1/settings

GET    /api/v1/search-roots
POST   /api/v1/search-roots
PATCH  /api/v1/search-roots/{rootID}
DELETE /api/v1/search-roots/{rootID}

POST   /api/v1/search
POST   /api/v1/files/open

GET    /api/v1/stats
POST   /api/v1/stats/sync
POST   /api/v1/stats/compact
```

Ces routes sont des conventions initiales. Toute nouvelle route doit :

- respecter le préfixe `/api/v1` ;
- utiliser des modèles de requête et de réponse explicites ;
- valider les entrées ;
- retourner des erreurs JSON cohérentes ;
- définir clairement les codes HTTP ;
- ne pas exposer directement les modèles GORM ou les réponses brutes d'un
  service externe.

Le client frontend doit centraliser tous les appels HTTP dans un client API
TypeScript. Les composants React ne doivent pas disperser des appels `fetch`
directs dans l'interface.

## Configuration et settings

La base de données par défaut est :

```text
~/.config/dashboard.locals/database.sqlite
```

Si `XDG_CONFIG_HOME` est défini, le chemin doit devenir :

```text
$XDG_CONFIG_HOME/dashboard.locals/database.sqlite
```

Les settings doivent être stockés dans SQLite, notamment :

- les racines de recherche ;
- les valeurs par défaut de recherche ;
- l'éditeur de code ;
- les paramètres de connexion ou de détection OpenCode ;
- les préférences de synchronisation et d'agrégation.

Les variables d'environnement et les options de démarrage peuvent fournir des
overrides opérationnels, mais ne doivent pas remplacer silencieusement les
settings persistés.

### Racines de recherche

Plusieurs racines de recherche sont supportées. Chaque racine doit posséder au
minimum :

- un identifiant stable ;
- un nom lisible ;
- un chemin absolu normalisé ;
- un état activé/désactivé ;
- des dates de création et de modification.

Une recherche doit référencer une racine par son identifiant. Le serveur doit
valider que la racine existe, qu'elle est accessible et qu'elle est un
répertoire.

Les chemins de résultats doivent être normalisés et ne doivent pas permettre de
sortir de la racine sélectionnée par l'intermédiaire de `..` ou de liens
symboliques.

## Recherche avec ripgrep

La recherche est exécutée par `ripgrep` via `exec.CommandContext` ou un
mécanisme équivalent. Ne jamais construire une commande shell à partir de la
requête utilisateur.

Une requête de recherche peut notamment contrôler :

- la racine sélectionnée ;
- le texte ou motif recherché ;
- le respect des fichiers `.gitignore` ;
- l'ignorance des fichiers binaires.

Les options de recherche visibles dans le client sont transmises à l'API puis
validées côté serveur. Les règles suivantes sont obligatoires :

- les liens symboliques ne sont pas suivis ;
- la recherche reste limitée à la racine sélectionnée ;
- les fichiers binaires peuvent être ignorés selon l'option choisie ;
- le respect des `.gitignore` peut être activé ou désactivé selon l'option
  choisie ;
- le contexte d'exécution doit être annulable ;
- une limite de durée et, si nécessaire, une limite de résultats doivent être
  appliquées ;
- le code de sortie indiquant simplement « aucun résultat » n'est pas une
  erreur serveur ;
- les erreurs de permission et d'exécution doivent être présentées comme des
  erreurs structurées ;
- les résultats doivent contenir suffisamment d'informations pour afficher le
  chemin, la ligne, la colonne et l'extrait correspondant lorsque disponibles.

Les arguments de `ripgrep` doivent être construits sous forme de liste. Toute
option ajoutée doit être documentée et couverte par un test.

## Ouverture dans un éditeur

Un navigateur ne peut pas lancer de manière fiable `xdg-open` sur l'ordinateur
local. L'ouverture d'un résultat est donc une responsabilité du serveur.

La configuration de l'éditeur doit être structurée. Elle peut contenir :

- un nom ;
- un exécutable ;
- une liste d'arguments ;
- les placeholders autorisés pour le fichier, la ligne et la colonne.

Exemple conceptuel :

```json
{
  "name": "Visual Studio Code",
  "command": "code",
  "arguments": [
    "--reuse-window",
    "{file}",
    "--goto",
    "{file}:{line}:{column}"
  ]
}
```

Le serveur doit lancer l'exécutable avec des arguments séparés, sans shell.
Seuls les placeholders définis par l'application peuvent être remplacés.
L'éditeur et le chemin final doivent être validés avant l'exécution.

## Base de données et GORM

SQLite est la base de données applicative et GORM est la couche d'accès.

Règles :

- encapsuler l'accès aux données dans `internal/store` ;
- ne pas retourner de modèles GORM directement par l'API ;
- utiliser des migrations explicites et versionnées ;
- créer les index nécessaires aux recherches par date, racine, modèle et
  période ;
- utiliser des transactions pour les synchronisations et les compactages ;
- rendre les opérations de synchronisation et de compactage idempotentes ;
- tester les migrations et les opérations critiques avec SQLite ;
- configurer proprement les timeouts et la concurrence SQLite.

Ne pas utiliser SQLite comme canal de communication entre le serveur et la
CLI. La CLI doit utiliser l'API HTTP du daemon.

## Intégration OpenCode

OpenCode est une dépendance optionnelle. L'absence d'OpenCode ne doit pas
empêcher la recherche locale ou le démarrage de l'interface.

La couche `internal/opencode` doit encapsuler la détection et les appels
externes. Elle doit pouvoir :

1. vérifier si l'exécutable `opencode` est installé ;
2. découvrir une instance locale disponible ;
3. vérifier l'instance via `GET /api/info` ;
4. utiliser son mécanisme d'authentification au lieu de supposer une API
   anonyme ;
5. récupérer les statistiques ;
6. retourner un état explicite lorsque l'instance est absente, arrêtée,
   incompatible ou non authentifiée.

La documentation OpenCode V2 indique notamment :

- l'API HTTP locale expose `GET /api/info` ;
- les statistiques sont disponibles via
  `GET /api/experimental/session/stats` ;
- les paramètres de statistiques incluent `from`, `to`, `project`, `timezone`
  et `tools` ;
- le service local est enregistré sous
  `~/.local/state/opencode/service.json` dans l'installation standard ;
- la CLI officielle peut utiliser `opencode api` avec la découverte et
  l'authentification du service.

Références officielles :

- https://opencode.ai/v2/docs/api
- https://opencode.ai/v2/docs/troubleshooting
- https://opencode.ai/v2/docs/build/client
- https://opencode.ai/v2/openapi.json

Le endpoint `/api/experimental/session/stats` étant expérimental, son format
doit être converti vers des modèles internes versionnés. Ne pas propager ses
types bruts dans le reste de l'application.

Les statistiques récupérées peuvent contenir notamment :

- sessions ;
- sous-agents ;
- prompts ;
- étapes ;
- tokens d'entrée, sortie et raisonnement ;
- cache lu et écrit ;
- coûts ;
- statistiques d'outils ;
- activité par jour ;
- modèles utilisés ;
- jours actifs et séries d'activité.

Les timeouts réseau, les erreurs d'authentification, les versions incompatibles
et les réponses invalides doivent être gérés explicitement. Les credentials et
headers d'authentification ne doivent jamais être enregistrés dans les logs ou
retournés par l'API publique de dashboard.locals.

## Historique, synchronisation et statistiques

L'historique d'utilisation doit être conservé pour permettre des graphiques
sur de longues périodes. Les données doivent être stockées avec leur période,
leur source et leur granularité.

La synchronisation doit éviter les doublons grâce à une clé d'idempotence
appropriée, par exemple une combinaison de :

- source OpenCode ;
- projet ;
- période ;
- granularité ;
- version du format ;
- empreinte du contenu lorsque nécessaire.

Prévoir au minimum une séparation entre :

- les enregistrements issus de la synchronisation ;
- les agrégats journaliers ;
- les agrégats mensuels ;
- les métadonnées de synchronisation.

Les agrégats doivent préserver les métriques nécessaires aux graphiques :
tokens, coût, sessions, prompts, étapes, modèles et outils lorsque disponibles.

## Compactage via la CLI

La CLI doit fournir une commande de compactage des statistiques, par exemple :

```text
dashboard stats sync
dashboard stats compact
dashboard stats compact --before <date>
dashboard stats compact --granularity daily|monthly
dashboard stats compact --dry-run
dashboard stats compact --drop-raw
```

Le compactage doit :

- être demandé via le serveur ;
- être exécuté dans une transaction ;
- produire les agrégats avant toute suppression ;
- être relançable sans produire de doublons ;
- exposer un mode dry-run ;
- fournir un résumé du nombre de lignes lues, créées et supprimées.

Par défaut, le compactage ne doit pas supprimer les données détaillées. La
suppression doit nécessiter une option explicite telle que `--drop-raw`, car
elle réduit la précision disponible même si l'historique agrégé reste conservé.

## Client React/TypeScript

Le client utilise React, TypeScript, Vite et shadcn/ui.

Principes d'interface :

- interface simple, épurée et accessible ;
- composants shadcn/ui réutilisables ;
- états de chargement, erreur, vide et succès toujours gérés ;
- aucune logique d'accès à SQLite, OpenCode ou au système local ;
- aucun appel direct à une API externe ;
- appels serveur centralisés dans une couche API ;
- types de requêtes et de réponses alignés avec les contrats HTTP ;
- options de recherche clairement visibles, notamment `.gitignore` et
  fichiers binaires ;
- sélection explicite de la racine de recherche ;
- copie du chemin sans exposer de comportement dépendant du système au client.

Le client ne doit pas supposer que le serveur est disponible sur une origine
différente. En production, il est servi depuis le même serveur Go.

## CLI Go et Cobra

La CLI est écrite en Go et utilise Cobra.

Les commandes doivent :

- utiliser un client HTTP dédié ;
- afficher des erreurs lisibles ;
- retourner des codes d'erreur adaptés ;
- ne pas accéder directement à SQLite ;
- respecter les timeouts HTTP ;
- proposer une sortie JSON lorsque cela est utile pour l'automatisation.

Les groupes de commandes attendus incluent :

```text
dashboard server ...
dashboard config ...
dashboard search ...
dashboard stats sync
dashboard stats compact
```

Les noms et options peuvent évoluer, mais chaque commande doit être
documentée et testée.

## Build et développement

Le build de production doit compiler le frontend avant le binaire Go. Le
`Makefile` doit exprimer explicitement cette dépendance.

Ordre attendu :

```text
installation des dépendances frontend
→ build TypeScript/Vite
→ vérifications Go
→ compilation du serveur et de la CLI
```

Les commandes de qualité doivent inclure, selon les outils installés :

```text
gofmt
go vet ./...
go test ./...
npm run lint
npm run build
```

Toute modification doit préserver le build frontend embarqué. Les fichiers
générés du frontend ne doivent pas être édités manuellement.

## Tests

Prévoir des tests pour :

- la résolution du chemin SQLite ;
- la création et la migration de la base ;
- les racines de recherche ;
- la validation des chemins ;
- le refus des liens symboliques ;
- la construction des arguments `ripgrep` ;
- les recherches sans résultat ;
- les timeouts et annulations de recherche ;
- la configuration structurée de l'éditeur ;
- l'exécution sans shell ;
- les handlers `/api/v1` ;
- la détection d'OpenCode lorsqu'il est absent ou disponible ;
- le décodage des statistiques OpenCode ;
- l'idempotence des synchronisations ;
- les compactages et leurs modes dry-run ;
- les migrations et transactions SQLite ;
- les commandes Cobra.

Les tests ne doivent pas lancer de commandes destructives sur le système de
l'utilisateur. Les tests d'exécution utilisent des binaires factices ou des
interfaces injectables.

## Sécurité et robustesse

Les points suivants sont particulièrement sensibles :

- injection de commande dans `ripgrep` ou l'éditeur ;
- sortie de la racine de recherche par `..` ou liens symboliques ;
- exposition de fichiers arbitraires par l'API ;
- fuite de credentials OpenCode ;
- absence de timeout pour les processus ou appels réseau ;
- suppression irréversible lors d'un compactage ;
- blocage du démarrage lorsque OpenCode n'est pas installé ;
- concurrence et corruption SQLite ;
- réponses externes OpenCode non conformes.

Toute nouvelle fonctionnalité touchant l'un de ces points doit ajouter des
tests de régression et documenter son comportement.

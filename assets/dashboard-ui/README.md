# dashboard.locals — web-ui

Interface React/TypeScript/Vite du projet `dashboard.locals`.

L’application est conçue pour être servie par le serveur Go depuis la même
origine. Elle communique exclusivement avec les routes versionnées :

```text
/api/v1
```

Le frontend ne lance aucun processus local, ne lit pas SQLite et ne contacte
pas directement OpenCode, `ripgrep` ou un éditeur.

## Fonctionnalités

- vue d’ensemble de l’état du serveur et de l’usage IA ;
- recherche dans une racine configurée avec options `.gitignore` et fichiers
  binaires ;
- copie d’un chemin de résultat et demande d’ouverture via l’API serveur ;
- statistiques quotidiennes, modèles, outils, tokens et coûts ;
- synchronisation des statistiques ;
- configuration des racines, de l’éditeur et des préférences ;
- états de chargement, erreur, vide et succès ;
- interface responsive et navigation clavier.

## Développement

```bash
npm install
npm run dev
```

Le serveur API attendu est l’origine courante. Aucun mock n’est activé en
production : lorsque le serveur n’est pas lancé, l’interface affiche son état
d’erreur.

## Vérifications

```bash
npm run lint
npm test
npm run build
```

Les réponses du serveur sont converties dans `src/api/normalize.ts` afin de
conserver les composants indépendants des formes brutes de l’API externe.

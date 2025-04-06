#!/bin/bash

# Script pour générer un prompt LLM décrivant le projet courant
# et le copier dans le presse-papiers (via xsel).
#
# Options:
#   -i <pattern> : Pattern (glob) pour INCLURE des fichiers/dossiers (défaut: '*')
#                  Peut être utilisé plusieurs fois (implicitement combiné).
#                  Le pattern est passé à 'git ls-files'.
#   -e <pattern> : Pattern (glob) pour EXCLURE des fichiers/dossiers.
#                  Peut être utilisé plusieurs fois.
#                  Utilise l'option --exclude de 'git ls-files'.
#   -q "question" : Spécifie la question à poser au LLM.
#   -h            : Affiche cette aide.
#
# PRÉREQUIS: git, tree, xsel, file

# --- Variables par défaut ---
INCLUDE_PATTERNS=()
EXCLUDE_OPTS=()     
LLM_QUESTION="[VOTRE QUESTION ICI]"
TREE_IGNORE_DIRS=".git|node_modules|vendor|dist|build" 
TREE_MAX_DEPTH="" 

# --- Fonction d'aide ---
show_usage() {
  echo "Usage: $(basename "$0") [-i <include_pattern>] [-e <exclude_pattern>] [-q \"question\"] [-h]"
  echo ""
  echo "Options:"
  echo "  -i <pattern> : Pattern (glob) pour INCLURE des fichiers/dossiers (défaut: '*' si aucun -i n'est fourni)."
  echo "                 Peut être utilisé plusieurs fois (ex: -i 'src/*' -i '*.py')."
  echo "  -e <pattern> : Pattern (glob) pour EXCLURE des fichiers/dossiers (ex: -e '*.log' -e 'tests/data/*')."
  echo "                 Peut être utilisé plusieurs fois."
  echo "  -q \"question\" : Spécifie la question pour le LLM."
  echo "  -h            : Affiche cette aide."
  echo ""
  echo "Exemple: $(basename "$0") -i 'src/**/*.js' -e '**/__tests__/*' -q \"Refactorise ce code React pour utiliser des Hooks.\""
  exit 1
}

# --- Analyse des options de la ligne de commande ---
while getopts "hi:e:q:" opt; do
  case $opt in
    h) show_usage ;;
    i) INCLUDE_PATTERNS+=("$OPTARG") ;;
    e) EXCLUDE_OPTS+=("--exclude=${OPTARG}") ;; # Prépare directement l'option pour git ls-files
    q) LLM_QUESTION="$OPTARG" ;;
    \?) echo "Option invalide: -$OPTARG" >&2; show_usage ;;
    :) echo "L'option -$OPTARG requiert un argument." >&2; show_usage ;;
  esac
done
shift $((OPTIND-1)) # Retire les options traitées

if [ ${#INCLUDE_PATTERNS[@]} -eq 0 ]; then
  INCLUDE_PATTERNS=(".")
fi

# --- Vérification des commandes nécessaires ---
command -v git >/dev/null 2>&1 || { echo >&2 "ERREUR: Commande 'git' non trouvée."; exit 1; }
command -v tree >/dev/null 2>&1 || { echo >&2 "ERREUR: Commande 'tree' non trouvée. Installez-la."; exit 1; }
command -v xsel >/dev/null 2>&1 || { echo >&2 "ERREUR: Commande 'xsel' non trouvée. Installez-la."; exit 1; }
command -v file >/dev/null 2>&1 || { echo >&2 "ERREUR: Commande 'file' non trouvée. Installez-la."; exit 1; }

# --- Vérification si nous sommes dans un dépôt Git ---
if ! git rev-parse --is-inside-work-tree > /dev/null 2>&1; then
  echo >&2 "ERREUR: Vous n'êtes pas dans un dépôt Git."
  echo >&2 "Ce script utilise 'git ls-files' pour lister les fichiers et respecter .gitignore."
  exit 1
fi

# --- Construction du Prompt ---
echo "Génération du prompt..."
echo "Inclusion: ${INCLUDE_PATTERNS[@]}"
if [ ${#EXCLUDE_OPTS[@]} -gt 0 ]; then
    # Extrait juste les patterns pour l'affichage
    exclude_patterns_display=$(printf "%s\n" "${EXCLUDE_OPTS[@]}" | sed 's/^--exclude=//')
    echo "Exclusion: $exclude_patterns_display"
fi
echo "Question: $LLM_QUESTION"

PROMPT_CONTENT="Voici le contexte de mon projet actuel. Analyse la structure et le contenu des fichiers fournis pour répondre à ma question.\n\n"

# 2. Structure du projet via 'tree'
PROMPT_CONTENT+="--- STRUCTURE DU PROJET (basée sur 'tree', peut différer légèrement des fichiers inclus) ---\n"
TREE_CMD="tree"
if [ -n "$TREE_MAX_DEPTH" ]; then
  TREE_CMD+=" $TREE_MAX_DEPTH"
fi

if [ -n "$TREE_IGNORE_DIRS" ]; then
  if tree --help | grep -q "\-I pattern"; then
    TREE_CMD+=" -I '$TREE_IGNORE_DIRS'"
  else
    echo >&2 "Attention: Votre version de 'tree' ne supporte peut-être pas -I. Tentative sans exclusion de répertoires spécifiques pour tree."
  fi
fi
PROJECT_TREE=$(eval $TREE_CMD 2>/dev/null || echo "Erreur lors de l'exécution de tree.")
PROMPT_CONTENT+="$PROJECT_TREE\n\n"

# 3. Contenu des fichiers pertinents
PROMPT_CONTENT+="--- CONTENU DES FICHIERS (basé sur git ls-files, respectant .gitignore et les options -i/-e) ---\n"
FILE_COUNTER=0
TOTAL_SIZE=0 

# Construit la commande git ls-files
# -c: cached (tracked) / -o: others (untracked but not ignored)
# --exclude-standard: respecte .gitignore, .git/info/exclude, etc.
# "${EXCLUDE_OPTS[@]}": ajoute les options --exclude=... spécifiées par l'utilisateur
# -- : sépare les options des patterns de chemin
# "${INCLUDE_PATTERNS[@]}": les patterns à inclure spécifiés par l'utilisateur
GIT_LS_FILES_CMD=(git ls-files -co --exclude-standard "${EXCLUDE_OPTS[@]}" -- "${INCLUDE_PATTERNS[@]}")

# Débogage : Affiche la commande git ls-files qui sera exécutée
# echo "Commande git ls-files:"
# printf "%q " "${GIT_LS_FILES_CMD[@]}"; echo

"${GIT_LS_FILES_CMD[@]}" | while IFS= read -r file; do
   if [ ! -f "$file" ] || [ ! -r "$file" ]; then
    echo >&2 "Attention: Fichier '$file' listé par git mais non trouvé ou illisible. Ignoré."
    continue
  fi

  MIME_TYPE=$(file -b --mime-type "$file")
   if [[ "$MIME_TYPE" != text/* && \
         "$MIME_TYPE" != application/json && \
         "$MIME_TYPE" != application/xml && \
         "$MIME_TYPE" != application/javascript && \
         "$MIME_TYPE" != application/x-sh && \
         "$MIME_TYPE" != application/x-shellscript && \
         "$MIME_TYPE" != application/x-python* && \
         "$MIME_TYPE" != application/x-php && \
         "$MIME_TYPE" != application/x-ruby && \
         "$MIME_TYPE" != application/toml && \
         "$MIME_TYPE" != application/yaml ]]; then
     echo "Info: Fichier '$file' ignoré (type MIME non textuel: $MIME_TYPE)"
     continue
  fi

  FILE_SIZE=$(stat -c%s "$file")
  if [ "$FILE_SIZE" -gt 1048576 ]; then # 1 Mo
    echo "Info: Fichier '$file' ignoré car trop volumineux (> 1MB)."
    continue
  fi

  PROMPT_CONTENT+="\n--- FICHIER: $file ---\n"
  FILE_CONTENT=$(cat "$file" 2>/dev/null)
  if [ $? -ne 0 ]; then
      echo >&2 "Attention: Échec de la lecture du contenu de '$file' avec cat. Ignoré."
      PROMPT_CONTENT="${PROMPT_CONTENT%--- FICHIER: $file ---*}"
      continue
  fi
  PROMPT_CONTENT+="$FILE_CONTENT"
  PROMPT_CONTENT+="\n--- FIN FICHIER: $file ---\n"
  ((FILE_COUNTER++))
done

PROMPT_CONTENT+="\n--- FIN DU CONTENU DES FICHIERS ---\n"

# 4. Question finale
PROMPT_CONTENT+="\nBasé sur le contexte fourni ci-dessus, réponds à la question suivante :\n\n$LLM_QUESTION\n"

# --- Copie dans le presse-papiers ---
printf "%s" "$PROMPT_CONTENT" | xsel -ib

# --- Feedback Utilisateur ---
echo "-------------------------------------"
echo "Prompt généré et copié dans le presse-papiers !"
echo "Nombre de fichiers inclus : $FILE_COUNTER"
# TODO: afficher la taille totale ici 
if [[ "$LLM_QUESTION" == "[VOTRE QUESTION ICI]" ]]; then
    echo "NOTE : Aucune question spécifiée avec -q. N'oubliez pas de remplacer '[VOTRE QUESTION ICI]'."
fi
echo "Collez (Ctrl+Shift+V ou clic milieu) dans votre LLM."
echo "-------------------------------------"

exit 0

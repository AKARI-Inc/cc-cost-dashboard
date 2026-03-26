#!/bin/bash
set -e
cd "$(dirname "$0")/../client" && npm run build
cd ../appscript && npx clasp push -f && npx clasp deploy --description "v$(date +%Y%m%d-%H%M%S)"
echo "Deploy complete!"

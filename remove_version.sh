#!/bin/bash

# Find all docker-compose files
compose_files=$(find . -name "*.yaml" -path "*/compose/*" | grep -v node_modules | grep -E 'compose-|docker-compose-')

for file in $compose_files; do
    echo "Processing $file..."
    # Create a temporary file
    sed '/^version:/d' "$file" > "$file.tmp"
    # Replace the original file with the modified content
    mv "$file.tmp" "$file"
    echo "Removed version attribute from $file"
done

echo "All done! Removed version attributes from all docker-compose files."

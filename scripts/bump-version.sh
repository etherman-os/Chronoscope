#!/bin/bash
set -e

NEW_VERSION=$1
if [ -z "$NEW_VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.2.0"
    exit 1
fi

echo "Bumping version to $NEW_VERSION"

# Update Go modules
cd services/ingestion
sed -i "s/version = \"[^\"]*\"/version = \"$NEW_VERSION\"/" go.mod 2>/dev/null || true
cd ../analytics
sed -i "s/version = \"[^\"]*\"/version = \"$NEW_VERSION\"/" go.mod 2>/dev/null || true

# Update Rust crates
cd ../processor
cargo set-version "$NEW_VERSION" 2>/dev/null || sed -i "s/^version = \"[^\"]*\"/version = \"$NEW_VERSION\"/" Cargo.toml

cd ../privacy-engine
cargo set-version "$NEW_VERSION" 2>/dev/null || sed -i "s/^version = \"[^\"]*\"/version = \"$NEW_VERSION\"/" Cargo.toml

cd ../../packages/sdk-linux
cargo set-version "$NEW_VERSION" 2>/dev/null || sed -i "s/^version = \"[^\"]*\"/version = \"$NEW_VERSION\"/" Cargo.toml

# Update package.json files
cd ../../services/web
npm version "$NEW_VERSION" --no-git-tag-version 2>/dev/null || true

cd ../landing
npm version "$NEW_VERSION" --no-git-tag-version 2>/dev/null || true

# Update Swift package
cd ../../packages/sdk-macos
sed -i "s/version: \"[^\"]*\"/version: \"$NEW_VERSION\"/" Package.swift 2>/dev/null || true

echo "Version bumped to $NEW_VERSION"
echo "Don't forget to update CHANGELOG.md and commit the changes!"

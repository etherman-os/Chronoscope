#!/bin/bash
set -e

VERSION=$1
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    exit 1
fi

echo "🚀 Releasing Chronoscope $VERSION"

# Run tests
echo "Running tests..."
cd services/privacy-engine && cargo test && cd ../..

# Bump version
echo "Bumping version..."
./scripts/bump-version.sh "$VERSION"

# Update CHANGELOG
echo "Update CHANGELOG.md manually, then press Enter to continue"
read

# Commit
git add -A
git commit -m "chore(release): prepare $VERSION"

# Tag
git tag -a "v$VERSION" -m "Release v$VERSION"

# Pull latest changes before pushing
git pull origin main --rebase

# Push
git push origin main
git push origin "v$VERSION"

echo "✅ Release v$VERSION pushed! GitHub Actions will create the release."

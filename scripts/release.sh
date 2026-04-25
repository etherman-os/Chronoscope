#!/bin/bash
set -e

VERSION=$1
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    exit 1
fi

echo "🚀 Releasing Chronoscope $VERSION"

# Run tests
echo "Running Go tests..."
go test ./pkg/middleware/... ./services/ingestion/... ./services/analytics/...

echo "Running frontend tests..."
cd web && npm test && cd ..

echo "Running Rust processor tests..."
cd services/privacy-engine && cargo test && cd ../..

echo "Checking for uncommitted secrets..."
git diff --cached --name-only | xargs grep -ilE '\.env|SECRET|PASSWORD|TOKEN' || true
if git diff --cached --name-only | grep -qE '\.env$'; then
    echo "ERROR: .env file detected in staging. Aborting."
    exit 1
fi

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

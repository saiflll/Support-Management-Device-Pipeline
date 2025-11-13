#!/bin/bash
# prepare-deploy.sh - Generate vendor directories for deployment

echo "==================================="
echo "Generating vendor directories..."
echo "==================================="

# Forward service
echo ""
echo "📦 Processing forward service..."
cd forward
if [ -f "go.mod" ]; then
    echo "  Running go mod vendor..."
    go mod vendor
    if [ -d "vendor" ]; then
        echo "  ✅ forward/vendor generated successfully"
    else
        echo "  ❌ Failed to generate forward/vendor"
        exit 1
    fi
else
    echo "  ❌ go.mod not found in forward/"
    exit 1
fi
cd ..

# OTA service
echo ""
echo "📦 Processing ota service..."
cd ota
if [ -f "go.mod" ]; then
    echo "  Running go mod vendor..."
    go mod vendor
    if [ -d "vendor" ]; then
        echo "  ✅ ota/vendor generated successfully"
    else
        echo "  ❌ Failed to generate ota/vendor"
        exit 1
    fi
else
    echo "  ❌ go.mod not found in ota/"
    exit 1
fi
cd ..

echo ""
echo "==================================="
echo "✅ All vendor directories generated"
echo "==================================="
echo ""
echo "Next steps:"
echo "1. Review the changes: git status"
echo "2. Add to git: git add forward/vendor ota/vendor"
echo "3. Commit: git commit -m 'Add vendor directories for deployment'"
echo "4. Push: git push"
echo "5. Deploy on server: docker-compose up -d --build"

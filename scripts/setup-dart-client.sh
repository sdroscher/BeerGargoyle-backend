#!/bin/bash
set -e

DART_CLIENT_DIR="dart-client"

echo "Setting up Dart client package structure..."

# Create pubspec.yaml
cat > "${DART_CLIENT_DIR}/pubspec.yaml" <<EOF
name: beergargoyle_client
description: Local development package for BeerGargoyle gRPC client
version: 1.0.0-local
publish_to: 'none'

environment:
  sdk: '>=3.2.6 <4.0.0'

dependencies:
  grpc: ^4.0.1
  protobuf: ^3.1.0
EOF

# Create main library export
cat > "${DART_CLIENT_DIR}/lib/beergargoyle_client.dart" <<EOF
library beergargoyle_client;

// Export all generated protobuf files
export 'src/api/v1/beer.pb.dart';
export 'src/api/v1/beer.pbenum.dart';
export 'src/api/v1/beer.pbgrpc.dart';
export 'src/api/v1/beer.pbjson.dart';

export 'src/api/v1/cellar.pb.dart';
export 'src/api/v1/cellar.pbenum.dart';
export 'src/api/v1/cellar.pbgrpc.dart';
export 'src/api/v1/cellar.pbjson.dart';

export 'src/api/v1/user.pb.dart';
export 'src/api/v1/user.pbenum.dart';
export 'src/api/v1/user.pbgrpc.dart';
export 'src/api/v1/user.pbjson.dart';

export 'src/google/protobuf/timestamp.pb.dart';
export 'src/google/protobuf/timestamp.pbenum.dart';
export 'src/google/protobuf/timestamp.pbjson.dart';
EOF

# Create backward-compatible re-export for google/protobuf
mkdir -p "${DART_CLIENT_DIR}/lib/google/protobuf"
cat > "${DART_CLIENT_DIR}/lib/google/protobuf/timestamp.pb.dart" <<EOF
// Re-export to maintain backward compatibility with original package structure
export '../../src/google/protobuf/timestamp.pb.dart';
EOF

# Create README
cat > "${DART_CLIENT_DIR}/README.md" <<EOF
# Local BeerGargoyle Client Development

This package wrapper allows you to use locally generated protobuf files for development.

## Structure

- \`lib/src/api/\` - Generated API protobuf files (beer, cellar, user)
- \`lib/src/google/\` - Generated Google protobuf dependencies
- \`lib/beergargoyle_client.dart\` - Main library export
- \`lib/google/protobuf/\` - Backward compatibility re-exports

## Usage

In your Flutter app's \`pubspec.yaml\`:

\`\`\`yaml
dependencies:
  beergargoyle_client:
    path: ../BeerGargoyle-backend/dart-client
\`\`\`

## Switching Back to GitHub

To switch back to the published GitHub package:

\`\`\`yaml
dependencies:
  beergargoyle_client:
    git:
      url: https://github.com/sdroscher/beergargoyle-dart-client.git
      ref: v1.0.2
\`\`\`

## Regenerating

This package structure is automatically set up when running \`task dart-client\` in the backend.
EOF

echo "✅ Dart client package structure created successfully"

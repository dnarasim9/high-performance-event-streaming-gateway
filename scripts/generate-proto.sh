#!/bin/bash

##
# Protocol Buffer Code Generation Script
# Regenerates all Go code from .proto files
##

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="${PROJECT_ROOT}/api/proto"
GEN_DIR="${PROJECT_ROOT}/internal/generated"
THIRD_PARTY_DIR="${PROJECT_ROOT}/api/third_party"

# Tool paths
PROTOC="${PROTOC:-protoc}"
PROTOC_GEN_GO="${PROTOC_GEN_GO:-$(go env GOPATH)/bin/protoc-gen-go}"
PROTOC_GEN_GO_GRPC="${PROTOC_GEN_GO_GRPC:-$(go env GOPATH)/bin/protoc-gen-go-grpc}"

# Check if protoc is installed
check_protoc() {
    if ! command -v "$PROTOC" &> /dev/null; then
        echo -e "${RED}Error: protoc not found${NC}"
        echo "Install protoc from: https://github.com/protocolbuffers/protobuf/releases"
        exit 1
    fi
    echo -e "${GREEN}✓ protoc found: $($PROTOC --version)${NC}"
}

# Check if Go protobuf plugin is installed
check_protoc_gen_go() {
    if [ ! -f "$PROTOC_GEN_GO" ]; then
        echo -e "${YELLOW}Installing protoc-gen-go...${NC}"
        go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    fi
    echo -e "${GREEN}✓ protoc-gen-go found${NC}"
}

# Check if gRPC plugin is installed
check_protoc_gen_go_grpc() {
    if [ ! -f "$PROTOC_GEN_GO_GRPC" ]; then
        echo -e "${YELLOW}Installing protoc-gen-go-grpc...${NC}"
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    fi
    echo -e "${GREEN}✓ protoc-gen-go-grpc found${NC}"
}

# Validate proto directory exists
check_proto_dir() {
    if [ ! -d "$PROTO_DIR" ]; then
        echo -e "${RED}Error: Proto directory not found: $PROTO_DIR${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Proto directory found: $PROTO_DIR${NC}"
}

# Create output directory
create_output_dir() {
    mkdir -p "$GEN_DIR"
    echo -e "${GREEN}✓ Output directory ready: $GEN_DIR${NC}"
}

# Find all proto files
find_proto_files() {
    find "$PROTO_DIR" -name "*.proto" | sort
}

# Generate code from proto files
generate_code() {
    local proto_files=()
    local count=0

    echo -e "\n${BLUE}Scanning for .proto files...${NC}"

    while IFS= read -r proto_file; do
        proto_files+=("$proto_file")
        ((count++))
    done < <(find_proto_files)

    if [ $count -eq 0 ]; then
        echo -e "${YELLOW}Warning: No .proto files found in $PROTO_DIR${NC}"
        return 0
    fi

    echo -e "${GREEN}Found $count .proto file(s)${NC}\n"

    # Build include paths
    local include_paths="-I=$PROTO_DIR"
    if [ -d "$THIRD_PARTY_DIR" ]; then
        include_paths="$include_paths -I=$THIRD_PARTY_DIR"
    fi

    # Add standard includes
    local go_include_path="$(go env GOPATH)/pkg/mod/github.com/grpc-ecosystem/grpc-gateway"
    if [ -d "$go_include_path" ]; then
        include_paths="$include_paths -I=$go_include_path"
    fi

    local google_api_path="$(go env GOPATH)/pkg/mod/github.com/googleapis/googleapis"
    if [ -d "$google_api_path" ]; then
        include_paths="$include_paths -I=$google_api_path"
    fi

    # Generate Go code for each proto file
    for proto_file in "${proto_files[@]}"; do
        local relative_path="${proto_file#$PROTO_DIR/}"
        local output_file="${GEN_DIR}/${relative_path%.proto}.pb.go"
        local output_dir=$(dirname "$output_file")

        mkdir -p "$output_dir"

        echo -e "${BLUE}Processing: $relative_path${NC}"

        # Generate protobuf code
        if $PROTOC $include_paths \
            --go_out="$GEN_DIR" \
            --go_opt="module=github.com/event-gateway/internal/generated" \
            "$proto_file"; then
            echo -e "${GREEN}  ✓ Generated: $relative_path.pb.go${NC}"
        else
            echo -e "${RED}  ✗ Failed to generate: $relative_path${NC}"
            return 1
        fi

        # Check if it's a service definition and generate gRPC code
        if grep -q "^service " "$proto_file"; then
            if $PROTOC $include_paths \
                --go-grpc_out="$GEN_DIR" \
                --go-grpc_opt="module=github.com/event-gateway/internal/generated" \
                "$proto_file"; then
                echo -e "${GREEN}  ✓ Generated: $relative_path.grpc.pb.go${NC}"
            else
                echo -e "${RED}  ✗ Failed to generate gRPC service: $relative_path${NC}"
                return 1
            fi
        fi
    done

    return 0
}

# Format generated code
format_generated() {
    echo -e "\n${BLUE}Formatting generated code...${NC}"
    if command -v goimports &> /dev/null; then
        goimports -w "$GEN_DIR" 2>/dev/null || true
        echo -e "${GREEN}✓ Code formatted${NC}"
    else
        go fmt ./... 2>/dev/null || true
        echo -e "${YELLOW}Note: goimports not found, using go fmt${NC}"
    fi
}

# Run vet on generated code
vet_generated() {
    echo -e "\n${BLUE}Running go vet on generated code...${NC}"
    if go vet "$GEN_DIR/..."; then
        echo -e "${GREEN}✓ Vet check passed${NC}"
    else
        echo -e "${YELLOW}Warning: Vet check failed (this may be expected)${NC}"
    fi
}

# Display summary
summary() {
    echo -e "\n${BLUE}═══════════════════════════════════════════${NC}"
    echo -e "${GREEN}Proto generation complete!${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    echo -e "Proto directory: ${BLUE}$PROTO_DIR${NC}"
    echo -e "Output directory: ${BLUE}$GEN_DIR${NC}"
    echo ""
    echo -e "${YELLOW}Next steps:${NC}"
    echo "1. Review generated files in: $GEN_DIR"
    echo "2. Run tests: make test"
    echo "3. Commit changes: git add -A && git commit"
}

# Main execution
main() {
    echo -e "${BLUE}Protocol Buffer Code Generator${NC}\n"

    check_protoc
    check_protoc_gen_go
    check_protoc_gen_go_grpc
    check_proto_dir
    create_output_dir

    if generate_code; then
        format_generated
        vet_generated
        summary
    else
        echo -e "\n${RED}Error: Code generation failed${NC}"
        exit 1
    fi
}

# Run main
main "$@"
